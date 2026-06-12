//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// 端到端：覆盖两项改动
//   1) 渠道密钥默认脱敏：GET /api/channel/:id 返回掩码串（非空、非完整、含 ****）。
//   2) 高德代理白名单：POST /amap 放行 /v3/geocode/，拒绝未列入的前缀。
// 运行：go test -tags=e2e -run 'TestChannelKeyMasked|TestAmapGeocodeWhitelist' -v ./tests/e2e/ -timeout 5m

// getAPIResp 用 admin 客户端（自动注入 access token）GET 并解析统一信封。
func getAPIResp(tb testing.TB, client *http.Client, url string) apiResp {
	tb.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		tb.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		tb.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		tb.Fatalf("decode %s: %v", url, err)
	}
	return out
}

// findChannelIDByName 经 /api/channel/search 按名称定位渠道 ID。
func findChannelIDByName(tb testing.TB, env *e2eEnv, name string) int {
	tb.Helper()
	out := getAPIResp(tb, env.Client, env.Base+"/api/channel/search?keyword="+name)
	if !out.Success {
		tb.Fatalf("search channels: %s", out.Message)
	}
	var list []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out.Data, &list); err != nil {
		tb.Fatalf("decode channel list: %v", err)
	}
	for _, ch := range list {
		if ch.Name == name {
			return ch.Id
		}
	}
	tb.Fatalf("channel %q not found in search results (%d items)", name, len(list))
	return 0
}

// TestChannelKeyMasked 验证 GET /api/channel/:id 默认返回脱敏密钥（首尾各 4 位 + 中间 ****）。
func TestChannelKeyMasked(t *testing.T) {
	env := mustEnv(t)
	ctx := context.Background()

	const fullKey = "sk-e2eMASK0123456789abcd" // 首4 "sk-e"，末4 "abcd"
	name := fmt.Sprintf("e2e-maskkey-%d", time.Now().UnixNano()%1_000_000_000)

	if err := addChannel(ctx, env.Client, env.Base, 1, name, fullKey, "gpt-4o-mini", "https://api.example.com", ""); err != nil {
		t.Fatalf("add channel: %v", err)
	}

	id := findChannelIDByName(t, env, name)
	out := getAPIResp(t, env.Client, fmt.Sprintf("%s/api/channel/%d", env.Base, id))
	if !out.Success {
		t.Fatalf("get channel: %s", out.Message)
	}
	var ch struct {
		Id  int    `json:"id"`
		Key string `json:"key"`
	}
	if err := json.Unmarshal(out.Data, &ch); err != nil {
		t.Fatalf("decode channel: %v", err)
	}

	if ch.Key == "" {
		t.Fatal("masked key is empty; expected a masked preview")
	}
	if ch.Key == fullKey {
		t.Fatalf("GET channel leaked the full key: %q", ch.Key)
	}
	if !strings.Contains(ch.Key, "****") {
		t.Fatalf("masked key missing ****: %q", ch.Key)
	}
	if !strings.HasPrefix(ch.Key, "sk-e") || !strings.HasSuffix(ch.Key, "abcd") {
		t.Fatalf("masked key not first4/last4 of original: %q", ch.Key)
	}
	if ch.Key != "sk-e****abcd" {
		t.Fatalf("unexpected mask: got %q, want %q", ch.Key, "sk-e****abcd")
	}
}

// postAmap 用用户 Token（TokenAuth）POST /amap，返回状态码与响应体。
func postAmap(t *testing.T, env *e2eEnv, body map[string]any) (int, string) {
	t.Helper()
	resp := postJSON(t, env.Client, env.Base+"/amap", map[string]string{
		"Authorization": "Bearer " + env.Token,
	}, body)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// TestAmapGeocodeWhitelist 验证 /v3/geocode/ 已进入代理白名单，而未列入的前缀仍被拒绝。
func TestAmapGeocodeWhitelist(t *testing.T) {
	env := mustEnv(t)

	const whitelistRejection = "不在允许列表内"

	// 1) 放行：/v3/geocode/geo 不应再被白名单拒绝。
	//    本环境未配置高德 Key，故请求会停在「Key 未配置」而非白名单拒绝——这恰好证明 path 已通过校验。
	status, bodyGeo := postAmap(t, env, map[string]any{
		"method": "GET",
		"path":   "/v3/geocode/geo",
		"query":  map[string]string{"address": "北京"},
	})
	if strings.Contains(bodyGeo, whitelistRejection) {
		t.Fatalf("/v3/geocode/geo 被白名单拒绝（HTTP %d）: %s", status, bodyGeo)
	}
	if !strings.Contains(bodyGeo, "未配置") {
		// 不是硬性失败信号，但有助于诊断：理论上应到达「高德 Key 未配置」分支。
		t.Logf("geocode 响应非预期的 Key 未配置（HTTP %d）: %s", status, bodyGeo)
	}

	// 2) 逆地理编码同前缀，同样应放行。
	if _, body := postAmap(t, env, map[string]any{
		"method": "GET",
		"path":   "/v3/geocode/regeo",
		"query":  map[string]string{"location": "116.481488,39.990464"},
	}); strings.Contains(body, whitelistRejection) {
		t.Fatalf("/v3/geocode/regeo 被白名单拒绝: %s", body)
	}

	// 3) 控制组：未列入白名单的前缀必须被拒绝（证明白名单确实在生效）。
	status, bodyDeny := postAmap(t, env, map[string]any{
		"method": "GET",
		"path":   "/v5/inputtips/inputtips",
		"query":  map[string]string{"keywords": "北京"},
	})
	if !strings.Contains(bodyDeny, whitelistRejection) {
		t.Fatalf("未列入白名单的 /v5/inputtips/ 未被拒绝（HTTP %d）: %s", status, bodyDeny)
	}
}
