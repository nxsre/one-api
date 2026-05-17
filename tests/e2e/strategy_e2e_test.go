//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 端到端：依赖 docker-compose.e2e 栈（MySQL + Redis + mock-upstream + one-api）。
// 运行：go test -tags/e2e -v ./tests/e2e/ -timeout 15m
// 集群前置 Nginx：E2E_ONE_API_URL=http://127.0.0.1:28080

func baseURL() string {
	if b := strings.TrimSpace(os.Getenv("E2E_ONE_API_URL")); b != "" {
		return strings.TrimRight(b, "/")
	}
	return "http://127.0.0.1:13000"
}

func mockStatsURL() string {
	if u := strings.TrimSpace(os.Getenv("E2E_MOCK_STATS_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://127.0.0.1:18080/__debug/stats"
}

func mockUpstreamForChannels() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("E2E_MOCK_UPSTREAM_URL")), "/")
}

func openAIModel() string {
	if m := strings.TrimSpace(os.Getenv("E2E_OPENAI_MODEL")); m != "" {
		return m
	}
	return "gpt-4o-mini"
}

type apiResp struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// 与 config.e2e.toml 中 initial_root_access_token 保持一致（32 字符，与 users.access_token char(32) 一致）。
const defaultE2ERootAccessToken = "12345678901234567890123456789012"

func rootAccessToken() string {
	if v := strings.TrimSpace(os.Getenv("E2E_ROOT_ACCESS_TOKEN")); v != "" {
		return v
	}
	return defaultE2ERootAccessToken
}

type accessTokenTransport struct {
	token string
	next  http.RoundTripper
}

func (a *accessTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := a.next
	if next == nil {
		next = http.DefaultTransport
	}
	if req.Header.Get("Authorization") != "" {
		return next.RoundTrip(req)
	}
	r2 := req.Clone(req.Context())
	r2.Header.Set("Authorization", a.token)
	return next.RoundTrip(r2)
}

func newAccessTokenHTTPClient(token string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &accessTokenTransport{
			token: strings.TrimSpace(token),
			next:  http.DefaultTransport,
		},
	}
}

func verifyAdminAuth(ctx context.Context, client *http.Client, base string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/user/self", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	return out.Success
}

type e2eEnv struct {
	Client  *http.Client
	Base    string
	Token   string
	OAI     string
	RLModel string
}

var (
	envOnce sync.Once
	envVal  *e2eEnv
	envErr  error
)

func mustEnv(t *testing.T) *e2eEnv {
	t.Helper()
	envOnce.Do(func() { envVal, envErr = bootstrapE2E() })
	if envErr != nil {
		t.Fatalf("e2e bootstrap: %v", envErr)
	}
	return envVal
}

func bootstrapE2E() (*e2eEnv, error) {
	base := baseURL()
	mockHost := mockUpstreamForChannels()
	if mockHost == "" {
		mockHost = "http://mock-upstream:8080"
	}

	ctx := context.Background()
	if err := waitStatus(ctx, http.DefaultClient, base, 3*time.Minute); err != nil {
		return nil, err
	}

	accessTok := rootAccessToken()
	adminViaToken := newAccessTokenHTTPClient(accessTok, 60*time.Second)
	var client *http.Client
	if verifyAdminAuth(ctx, adminViaToken, base) {
		client = adminViaToken
	} else {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		client = &http.Client{Jar: jar, Timeout: 60 * time.Second}
		if err := loginRoot(ctx, client, base); err != nil {
			return nil, fmt.Errorf("根访问令牌无效且 root 登录失败（请确认 initial_root_access_token/E2E_ROOT_ACCESS_TOKEN 与库中 root 一致，可对 MySQL 卷执行 docker compose down -v）: %w", err)
		}
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	oai := openAIModel()
	rlModel := "e2e-rl-model-" + suffix

	channels := []struct {
		typ    int
		name   string
		key    string
		models string
		config string
	}{
		{1, "e2e-openai-" + suffix, "sk-mock-openai", oai, ""},
		{14, "e2e-anthropic-" + suffix, "sk-mock-anthropic", "claude-3-haiku-20240307", ""},
		{24, "e2e-gemini-" + suffix, "mock-google-api-key", "gemini-2.0-flash", `{"api_version":"v1beta"}`},
		{1, "e2e-rl-" + suffix, "sk-mock-rl", rlModel, ""},
	}
	for _, ch := range channels {
		if err := addChannel(ctx, client, base, ch.typ, ch.name, ch.key, ch.models, mockHost, ch.config); err != nil {
			return nil, err
		}
	}
	time.Sleep(4 * time.Second)

	tok, err := addToken(ctx, client, base, "e2e-token-"+suffix)
	if err != nil {
		return nil, err
	}
	return &e2eEnv{Client: client, Base: base, Token: tok, OAI: oai, RLModel: rlModel}, nil
}

func waitStatus(ctx context.Context, client *http.Client, base string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	var last error
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/status", nil)
		resp, err := client.Do(req)
		if err == nil {
			var body apiResp
			_ = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if body.Success {
				return nil
			}
		} else {
			last = err
		}
		time.Sleep(2 * time.Second)
	}
	if last != nil {
		return fmt.Errorf("one-api not ready at %s: %w", base, last)
	}
	return fmt.Errorf("one-api not ready at %s", base)
}

func loginRoot(ctx context.Context, client *http.Client, base string) error {
	b, _ := json.Marshal(map[string]string{"username": "root", "password": "123456"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/user/login", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.Success {
		return fmt.Errorf("login: %s", out.Message)
	}
	return nil
}

func addChannel(ctx context.Context, client *http.Client, base string, typ int, name, key, models, mockBase, configJSON string) error {
	payload := map[string]any{
		"type":     typ,
		"name":     name,
		"key":      key,
		"models":   models,
		"group":    "default",
		"base_url": mockBase,
		"status":   1,
	}
	if configJSON != "" {
		payload["config"] = configJSON
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/channel/", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.Success {
		return fmt.Errorf("add channel %s: %s", name, out.Message)
	}
	return nil
}

func addToken(ctx context.Context, client *http.Client, base, name string) (string, error) {
	payload := map[string]any{
		"name":             name,
		"remain_quota":     50_000_000,
		"unlimited_quota":  true,
		"expired_time":     -1,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/token/", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if !out.Success {
		return "", fmt.Errorf("add token: %s", out.Message)
	}
	var data struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(out.Data, &data); err != nil {
		return "", err
	}
	return data.Key, nil
}

func putOption(ctx context.Context, client *http.Client, base, key, value string) error {
	raw, _ := json.Marshal(map[string]string{"key": key, "value": value})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, base+"/api/option/", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.Success {
		return fmt.Errorf("option %s: %s", key, out.Message)
	}
	return nil
}

func postJSON(tb testing.TB, client *http.Client, url string, headers map[string]string, body any) *http.Response {
	tb.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		tb.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		tb.Fatalf("post %s: %v", url, err)
	}
	return resp
}

func TestOpenAIChatViaMock(t *testing.T) {
	env := mustEnv(t)
	resp := postJSON(t, env.Client, env.Base+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + env.Token,
	}, map[string]any{
		"model": env.OAI,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 32,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Choices) == 0 || !strings.HasPrefix(out.Choices[0].Message.Content, "mock-openai:") {
		t.Fatalf("unexpected body: %+v", out)
	}
	checkMockStat(t, "openai_chat_completions")
}

func TestAnthropicNativeViaMock(t *testing.T) {
	env := mustEnv(t)
	resp := postJSON(t, env.Client, env.Base+"/v1/messages", map[string]string{
		"Authorization":     "Bearer " + env.Token,
		"anthropic-version": "2023-06-01",
		"Content-Type":      "application/json",
	}, map[string]any{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 64,
		"messages":   []map[string]any{{"role": "user", "content": "hello"}},
	})
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, string(body))
	}
	var ar struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &ar); err != nil {
		t.Fatal(err)
	}
	if len(ar.Content) == 0 || !strings.Contains(ar.Content[0].Text, "mock-anthropic:") {
		t.Fatalf("unexpected anthropic body: %s", string(body))
	}
	checkMockStat(t, "anthropic_messages")
}

func TestGeminiNativeViaMock(t *testing.T) {
	env := mustEnv(t)
	resp := postJSON(t, env.Client, env.Base+"/v1beta/models/gemini-2.0-flash:generateContent", map[string]string{
		"Authorization": "Bearer " + env.Token,
	}, map[string]any{
		"contents": []map[string]any{{
			"role":  "user",
			"parts": []map[string]string{{"text": "e2e ping"}},
		}},
	})
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, string(body))
	}
	var gr struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &gr); err != nil {
		t.Fatal(err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 ||
		!strings.Contains(gr.Candidates[0].Content.Parts[0].Text, "mock-gemini:") {
		t.Fatalf("unexpected gemini body: %s", string(body))
	}
	checkMockStat(t, "gemini_generate_content")
}

func TestModelRateLimitRedis(t *testing.T) {
	env := mustEnv(t)
	ctx := context.Background()
	pol := `{"enabled":true,"by_token":{"*":{"qps":2,"burst":2,"concurrency":0,"daily_quota":0}},"by_user":{},"by_group":{}}`
	if err := putOption(ctx, env.Client, env.Base, "ModelRateLimitPolicy", pol); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = putOption(context.Background(), env.Client, env.Base, "ModelRateLimitPolicy",
			`{"enabled":false,"by_token":{},"by_user":{},"by_group":{}}`)
	})

	var ok, limited, unexpected atomic.Int32
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 60 * time.Second}
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := map[string]any{
				"model": env.RLModel,
				"messages": []map[string]string{
					{"role": "user", "content": "hello"},
				},
				"max_tokens": 32,
			}
			b, _ := json.Marshal(payload)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
				env.Base+"/v1/chat/completions", bytes.NewReader(b))
			if err != nil {
				unexpected.Add(1)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+env.Token)
			resp, err := client.Do(req)
			if err != nil {
				unexpected.Add(1)
				return
			}
			rb, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				ok.Add(1)
			case http.StatusTooManyRequests:
				var wrap struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				_ = json.Unmarshal(rb, &wrap)
				if wrap.Error.Code == "model_rate_limited" {
					limited.Add(1)
				}
			default:
				unexpected.Add(1)
			}
		}()
	}
	wg.Wait()
	if unexpected.Load() > 0 {
		t.Fatalf("unexpected status/code errors in load test (see one-api logs)")
	}
	if ok.Load() < 1 {
		t.Fatalf("expected some 200, got ok=%d limited=%d", ok.Load(), limited.Load())
	}
	if limited.Load() < 1 {
		t.Fatalf("expected model_rate_limited with Redis, got ok=%d limited=%d", ok.Load(), limited.Load())
	}
}

func TestModelAliasChain(t *testing.T) {
	env := mustEnv(t)
	ctx := context.Background()
	alias := "e2e-alias-" + env.OAI
	pol := fmt.Sprintf(`{"chains":{%q:%q},"aliases":{},"defaults":{},"group_overrides":{},"max_chain_steps":12}`,
		alias, env.OAI)
	if err := putOption(ctx, env.Client, env.Base, "ModelAliasPolicy", pol); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = putOption(context.Background(), env.Client, env.Base, "ModelAliasPolicy", "{}")
	})

	resp := postJSON(t, env.Client, env.Base+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + env.Token,
	}, map[string]any{
		"model": alias,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 32,
	})
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	txt := out.Choices[0].Message.Content
	if !strings.Contains(txt, "mock-openai") && !strings.Contains(txt, env.OAI) {
		t.Fatalf("unexpected completion: %q", txt)
	}
}

func checkMockStat(t *testing.T, key string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, mockStatsURL(), nil)
	if err != nil {
		t.Logf("mock stats skip: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("mock stats unavailable (%v), skip counter check", err)
		return
	}
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Logf("mock stats decode: %v", err)
		return
	}
	v, ok := m[key]
	if !ok {
		return
	}
	switch n := v.(type) {
	case float64:
		if n < 1 {
			t.Fatalf("mock %s expected >=1 got %v", key, n)
		}
	default:
		t.Logf("mock stat %s type %T", key, v)
	}
}
