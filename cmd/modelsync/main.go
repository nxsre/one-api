// Command modelsync 调用三大原厂（OpenAI / Anthropic / Gemini）官方 Models API，
// 拉取最新模型 ID 列表，并自动回写各适配器的 constants.go 中的 ModelList。
//
// 设计要点：
//   - 直接调官方 /models 接口（返回精确、权威的模型 ID），而非抓文档页（易被反爬拦截）。
//   - 支持 HTTP/HTTPS 代理（-proxy 或环境变量 HTTPS_PROXY/HTTP_PROXY）。
//   - 默认携带 Chrome 浏览器 User-Agent，规避部分网关/反爬对非浏览器 UA 的拦截。
//
// 用法示例：
//
//	export OPENAI_API_KEY=sk-...
//	export ANTHROPIC_API_KEY=sk-ant-...
//	export GEMINI_API_KEY=AIza...
//	go run ./cmd/modelsync -proxy http://127.0.0.1:7890            # 回写 constants.go
//	go run ./cmd/modelsync -providers openai,gemini -dry-run       # 只打印不写文件
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// defaultChromeUA 默认使用的 Chrome User-Agent。
const defaultChromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var modelListRe = regexp.MustCompile(`var ModelList = \[\]string\{[^}]*\}`)

type options struct {
	providers  string
	source     string // api（官方，需 key）| litellm | openrouter（均免 key）
	proxy      string
	ua         string
	repoRoot   string
	timeout    time.Duration
	dryRun     bool
	exclude    string
	dashesOnly bool

	openaiBase    string
	anthropicBase string
	geminiBase    string

	litellmURL    string
	openrouterURL string
}

// provider 描述一家原厂的拉取方式与回写目标。
type provider struct {
	name        string
	constPath   string // 相对 repoRoot 的 constants.go 路径
	fetch       func(ctx context.Context, c *client, base string) ([]string, error)
	defaultBase string
}

func main() {
	var o options
	flag.StringVar(&o.providers, "providers", "openai,anthropic,gemini", "要更新的厂商，逗号分隔：openai,anthropic,gemini")
	flag.StringVar(&o.source, "source", "api", "模型列表来源，可逗号分隔多个取并集：api(官方,需 key) | litellm(免 key) | openrouter(免 key)")
	flag.StringVar(&o.proxy, "proxy", "", "HTTP/HTTPS 代理地址，如 http://127.0.0.1:7890（留空则用 HTTPS_PROXY/HTTP_PROXY 环境变量）")
	flag.StringVar(&o.ua, "ua", defaultChromeUA, "请求使用的 User-Agent（默认 Chrome）")
	flag.StringVar(&o.repoRoot, "repo", ".", "仓库根目录（用于定位各适配器 constants.go）")
	flag.DurationVar(&o.timeout, "timeout", 30*time.Second, "单次 HTTP 请求超时")
	flag.BoolVar(&o.dryRun, "dry-run", false, "只打印拉取到的模型列表，不回写文件")
	flag.StringVar(&o.exclude, "exclude", "", "排除匹配的模型，逗号分隔；支持 * 通配（如 *-tts）或子串（如 ft:、dall-e）")
	flag.BoolVar(&o.dashesOnly, "dashes-only", false, "仅对 anthropic：剔除含点号的非原厂 ID 变体（如 OpenRouter 的 claude-opus-4.8）")
	flag.StringVar(&o.openaiBase, "openai-base", envOr("OPENAI_BASE_URL", "https://api.openai.com"), "OpenAI API Base URL")
	flag.StringVar(&o.anthropicBase, "anthropic-base", envOr("ANTHROPIC_BASE_URL", "https://api.anthropic.com"), "Anthropic API Base URL")
	flag.StringVar(&o.geminiBase, "gemini-base", envOr("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"), "Gemini API Base URL")
	flag.StringVar(&o.litellmURL, "litellm-url", "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json", "LiteLLM 模型价目 JSON 地址（免 key 源）")
	flag.StringVar(&o.openrouterURL, "openrouter-url", "https://openrouter.ai/api/v1/models", "OpenRouter 模型列表地址（免 key 源）")
	flag.Parse()

	c, err := newClient(o.proxy, o.ua, o.timeout)
	if err != nil {
		fatal("初始化 HTTP 客户端失败: %v", err)
	}

	all := map[string]provider{
		"openai":    {name: "openai", constPath: "relay/adaptor/openai/constants.go", fetch: fetchOpenAI, defaultBase: o.openaiBase},
		"anthropic": {name: "anthropic", constPath: "relay/adaptor/anthropic/constants.go", fetch: fetchAnthropic, defaultBase: o.anthropicBase},
		"gemini":    {name: "gemini", constPath: "relay/adaptor/geminiv2/constants.go", fetch: fetchGemini, defaultBase: o.geminiBase},
	}

	sources := splitCSV(o.source)
	for _, s := range sources {
		if s != "api" && s != "litellm" && s != "openrouter" {
			fatal("未知 -source %q（可选 api|litellm|openrouter，可逗号分隔）", s)
		}
	}

	excludePatterns := splitCSV(o.exclude)

	ctx := context.Background()
	agg := &aggregator{c: c, litellmURL: o.litellmURL, openrouterURL: o.openrouterURL}
	exitCode := 0
	for _, name := range splitCSV(o.providers) {
		p, ok := all[name]
		if !ok {
			warn("未知厂商 %q，跳过", name)
			exitCode = 1
			continue
		}
		// 按 -source 列出的多个来源取并集；只要有一个成功就用，全失败才记错误。
		var models []string
		okCount := 0
		for _, s := range sources {
			var part []string
			var err error
			switch s {
			case "api":
				part, err = p.fetch(ctx, c, p.defaultBase)
			case "litellm":
				part, err = agg.litellm(ctx, p.name)
			case "openrouter":
				part, err = agg.openrouter(ctx, p.name)
			}
			if err != nil {
				warn("[%s] 源 %s 拉取失败: %v", p.name, s, err)
				continue
			}
			fmt.Printf("[%s] 源 %s: %d 个\n", p.name, s, len(part))
			models = append(models, part...)
			okCount++
		}
		if okCount == 0 {
			warn("[%s] 所有来源均失败，跳过", p.name)
			exitCode = 1
			continue
		}
		raw := len(models)
		models = applyFilters(models, p.name, excludePatterns, o.dashesOnly)
		models = normalize(models)
		if dropped := raw - len(models); dropped > 0 {
			fmt.Printf("[%s] 过滤/去重后剩 %d 个模型（剔除 %d）\n", p.name, len(models), dropped)
		} else {
			fmt.Printf("[%s] 拉取到 %d 个模型\n", p.name, len(models))
		}
		if o.dryRun {
			for _, m := range models {
				fmt.Printf("    %s\n", m)
			}
			continue
		}
		path := filepath.Join(o.repoRoot, p.constPath)
		if err := rewriteModelList(path, models); err != nil {
			warn("[%s] 回写 %s 失败: %v", p.name, path, err)
			exitCode = 1
			continue
		}
		fmt.Printf("[%s] 已更新 %s\n", p.name, path)
	}
	os.Exit(exitCode)
}

// client 封装带代理与 Chrome UA 的 HTTP 客户端。
type client struct {
	hc *http.Client
	ua string
}

func newClient(proxy, ua string, timeout time.Duration) (*client, error) {
	tr := &http.Transport{
		// 默认沿用环境变量代理（HTTPS_PROXY/HTTP_PROXY/NO_PROXY）。
		Proxy: http.ProxyFromEnvironment,
	}
	if proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("解析代理地址失败: %w", err)
		}
		tr.Proxy = http.ProxyURL(u)
	}
	return &client{
		hc: &http.Client{Transport: tr, Timeout: timeout},
		ua: ua,
	}, nil
}

// getJSON 发起 GET 请求并把 JSON 解码到 out，自动带上 Chrome UA 及自定义 header。
func (c *client) getJSON(ctx context.Context, rawURL string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return nil
}

// ---- 各厂商拉取实现 ----

func fetchOpenAI(ctx context.Context, c *client, base string) ([]string, error) {
	key := firstEnv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("缺少环境变量 OPENAI_API_KEY")
	}
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	// OpenAI /v1/models 一次性返回全部，不分页。
	err := c.getJSON(ctx, strings.TrimRight(base, "/")+"/v1/models",
		map[string]string{"Authorization": "Bearer " + key}, &resp)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		if m.ID != "" {
			out = append(out, m.ID)
		}
	}
	return out, nil
}

func fetchAnthropic(ctx context.Context, c *client, base string) ([]string, error) {
	key := firstEnv("ANTHROPIC_API_KEY", "CLAUDE_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("缺少环境变量 ANTHROPIC_API_KEY")
	}
	headers := map[string]string{
		"x-api-key":         key,
		"anthropic-version": "2023-06-01",
	}
	var out []string
	afterID := ""
	for {
		u := strings.TrimRight(base, "/") + "/v1/models?limit=1000"
		if afterID != "" {
			u += "&after_id=" + url.QueryEscape(afterID)
		}
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := c.getJSON(ctx, u, headers, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Data {
			if m.ID != "" {
				out = append(out, m.ID)
			}
		}
		if !resp.HasMore || resp.LastID == "" {
			break
		}
		afterID = resp.LastID
	}
	return out, nil
}

func fetchGemini(ctx context.Context, c *client, base string) ([]string, error) {
	key := firstEnv("GEMINI_API_KEY", "GOOGLE_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("缺少环境变量 GEMINI_API_KEY")
	}
	var out []string
	pageToken := ""
	for {
		u := strings.TrimRight(base, "/") + "/v1beta/models?pageSize=1000&key=" + url.QueryEscape(key)
		if pageToken != "" {
			u += "&pageToken=" + url.QueryEscape(pageToken)
		}
		var resp struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := c.getJSON(ctx, u, nil, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Models {
			// name 形如 "models/gemini-2.5-pro"，去掉前缀。
			out = append(out, strings.TrimPrefix(m.Name, "models/"))
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// ---- 免 key 聚合源 ----

// aggregator 拉取第三方聚合源（LiteLLM / OpenRouter），按厂商过滤出模型 ID。
// 聚合源整体只拉一次，多个厂商共享缓存。
type aggregator struct {
	c             *client
	litellmURL    string
	openrouterURL string

	liteCache map[string][]string // provider -> models
	orCache   map[string][]string
}

// litellmProviderKey 把本工具的厂商名映射为 LiteLLM 的 litellm_provider 取值。
var litellmProviderKey = map[string]string{
	"openai":    "openai",
	"anthropic": "anthropic",
	"gemini":    "gemini",
}

func (a *aggregator) litellm(ctx context.Context, provider string) ([]string, error) {
	if a.liteCache == nil {
		// 值里只取 litellm_provider；用 RawMessage 容忍非对象条目（如 "sample_spec"）。
		var raw map[string]struct {
			LiteLLMProvider string `json:"litellm_provider"`
		}
		if err := a.c.getJSON(ctx, a.litellmURL, nil, &raw); err != nil {
			return nil, err
		}
		a.liteCache = map[string][]string{}
		for id, v := range raw {
			// 跳过带 "/" 的复合 key（如 "1024-x-1024/dall-e-2"、bedrock 区域键）。
			if strings.Contains(id, "/") {
				continue
			}
			for ourName, lp := range litellmProviderKey {
				if v.LiteLLMProvider == lp {
					a.liteCache[ourName] = append(a.liteCache[ourName], id)
				}
			}
		}
	}
	out := a.liteCache[provider]
	if len(out) == 0 {
		return nil, fmt.Errorf("LiteLLM 源中未找到 %s 的模型（litellm_provider=%s）", provider, litellmProviderKey[provider])
	}
	return out, nil
}

// openrouterPrefix 把厂商名映射为 OpenRouter id 的 "provider/" 前缀。
var openrouterPrefix = map[string]string{
	"openai":    "openai/",
	"anthropic": "anthropic/",
	"gemini":    "google/",
}

func (a *aggregator) openrouter(ctx context.Context, provider string) ([]string, error) {
	if a.orCache == nil {
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := a.c.getJSON(ctx, a.openrouterURL, nil, &resp); err != nil {
			return nil, err
		}
		a.orCache = map[string][]string{}
		for _, m := range resp.Data {
			for ourName, pre := range openrouterPrefix {
				if strings.HasPrefix(m.ID, pre) {
					// 去掉 "provider/" 前缀；可能还带 ":free" 等后缀变体，一并去掉。
					id := strings.TrimPrefix(m.ID, pre)
					if i := strings.IndexByte(id, ':'); i >= 0 {
						id = id[:i]
					}
					a.orCache[ourName] = append(a.orCache[ourName], id)
				}
			}
		}
	}
	out := a.orCache[provider]
	if len(out) == 0 {
		return nil, fmt.Errorf("OpenRouter 源中未找到前缀 %q 的模型", openrouterPrefix[provider])
	}
	return out, nil
}

// ---- 回写 constants.go ----

// rewriteModelList 用拉取到的模型列表替换文件中的 `var ModelList = []string{...}` 块。
func rewriteModelList(path string, models []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !modelListRe.Match(raw) {
		return fmt.Errorf("未在文件中找到 `var ModelList = []string{...}` 块")
	}
	var b strings.Builder
	b.WriteString("var ModelList = []string{\n")
	for _, m := range models {
		b.WriteString("\t\"")
		b.WriteString(m)
		b.WriteString("\",\n")
	}
	b.WriteString("}")
	out := modelListRe.ReplaceAll(raw, []byte(b.String()))
	return os.WriteFile(path, out, 0o644)
}

// ---- 工具函数 ----

// applyFilters 按 -exclude 模式与 -dashes-only 规则过滤模型 ID。
func applyFilters(in []string, provider string, exclude []string, dashesOnly bool) []string {
	out := in[:0:0]
	for _, m := range in {
		if matchAny(m, exclude) {
			continue
		}
		// dashes-only 仅对 anthropic 生效：原厂 ID 用横杠，含点号的是 OpenRouter 等的非原厂变体。
		if dashesOnly && provider == "anthropic" && strings.Contains(m, ".") {
			continue
		}
		out = append(out, m)
	}
	return out
}

// matchAny 判断 id 是否命中任一排除模式：含 * 走通配匹配，否则按子串包含。
func matchAny(id string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(p, "*") {
			if ok, _ := path.Match(p, id); ok {
				return true
			}
		} else if strings.Contains(id, p) {
			return true
		}
	}
	return false
}

// normalize 去重 + 排序，保证回写结果稳定。
func normalize(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func warn(format string, a ...any)  { fmt.Fprintf(os.Stderr, "WARN  "+format+"\n", a...) }
func fatal(format string, a ...any) { fmt.Fprintf(os.Stderr, "FATAL "+format+"\n", a...); os.Exit(1) }
