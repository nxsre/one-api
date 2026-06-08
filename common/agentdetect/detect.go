// Package agentdetect 识别 LLM 请求来自哪个 agent 客户端（如 Claude Code、openclaw、hermes）。
//
// 单靠 User-Agent 不可靠（易伪造、CLI 常走官方 SDK 默认 UA），因此采用多信号分层识别：
// 头部信号（User-Agent / x-app / anthropic-beta）优先快速命中，命中不确定时再看请求体的
// system prompt 前缀与工具集（functional content，几乎不可伪造）。
//
// 规则表 rules 是数据驱动的：新增 agent 只需补一条规则。openclaw / hermes 的精确指纹
// 建议先用真实日志（logs.other.client_request_headers）确认后再细化 UASub。
package agentdetect

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// Result 为识别结果。Client 为空表示未识别。Via 标记命中来源（header/body），用于审计置信度。
type Result struct {
	Client string `json:"client"`
	Via    string `json:"via,omitempty"`
}

// rule 描述一个 agent 客户端的指纹。各字段为"任一命中"语义（ToolsAll 除外）。
type rule struct {
	client string
	// uaSub: User-Agent 的小写子串
	uaSub []string
	// xApp: x-app 头的小写精确值
	xApp []string
	// betaSub: anthropic-beta 头的小写子串
	betaSub []string
	// originatorSub: originator 头的小写子串（Codex 系列用）
	originatorSub []string
	// headerPresent: 只要其中任一头存在（非空）即命中（如 Codex 的 x-codex-* 专有头）
	headerPresent []string
	// systemSub: system prompt 的小写子串
	systemSub []string
	// toolsAll: 必须同时出现的工具名（区分大小写）；用于体内二次确认
	toolsAll []string
}

// rules 按优先级排序，靠前者先匹配。
var rules = []rule{
	{
		client:    "claude-code",
		uaSub:     []string{"claude-cli"},
		xApp:      []string{"cli"},
		betaSub:   []string{"claude-code"},
		systemSub: []string{"you are claude code"},
		toolsAll:  []string{"Bash", "Read", "Edit"},
	},
	{
		// openclaw：实测其出站请求 UA 为通用 OpenAI Node SDK（"OpenAI/JS x.y" + x-stainless-* 头），
		// 无法靠 UA 区分；可靠指纹在请求体——system prompt 含 "running inside OpenClaw"，
		// 以及其专有工具集（exec / web_fetch / sessions_spawn 等）。uaSub 仅作自定义 UA 的兜底。
		client:    "openclaw",
		uaSub:     []string{"openclaw"},
		systemSub: []string{"running inside openclaw", "you are openclaw"},
		// 共享工具集兜底(hermes 与 openclaw 同源，工具集相同)：仅当 system prompt 品牌标识缺失
		// (如自定义了 identity)时才命中，此时归类到 openclaw（同一代码家族）。
		toolsAll: []string{"exec", "web_fetch", "sessions_spawn"},
	},
	{
		// hermes(NousResearch/hermes-agent)：与 openclaw 同源代码，出站 UA 同为通用 OpenAI SDK。
		// 实测源码默认 identity："You are Hermes Agent ... created by Nous Research" / "You run on Hermes Agent"。
		// 两遍匹配下，品牌 system prompt 先于共享工具集，故能与 openclaw 区分。
		client:    "hermes",
		uaSub:     []string{"hermes"},
		systemSub: []string{"you are hermes agent", "you run on hermes agent", "created by nous research"},
	},
	{
		// codex(OpenAI Codex CLI, Rust)：实测——交互 TUI 用 UA/originator "codex_cli_rs"，
		// 而 headless `codex exec` 用 "codex_exec"(UA 形如 "codex_exec/<ver> (...)")；
		// 二者都带 originator 头(含 "codex")与专有 x-codex-* 头(x-codex-turn-metadata 等)。
		// 走 OpenAI Responses API,system 提示在 body 的 instructions 里("running in the Codex CLI")。
		client:        "codex",
		uaSub:         []string{"codex_cli_rs", "codex_exec", "codex_vscode"},
		originatorSub: []string{"codex"},
		headerPresent: []string{"x-codex-turn-metadata", "x-codex-window-id"},
		systemSub:     []string{"running in the codex cli", "codex cli, a terminal-based"},
	},
	{
		// gemini-cli(google-gemini/gemini-cli)：实测 v0.45 出站 UA 形如
		// "GeminiCLI-tui/<ver>/<model> (linux; arm64; terminal)"(也有 "GeminiCLI/..."、
		// VS Code 代理路径含 "proxy_client=geminicli")，并带专有头 x-gemini-api-privileged-user-id；
		// 走 Gemini 原生 API(/v1beta/...:generateContent)。system prompt 含 "Gemini CLI"。
		client:        "gemini-cli",
		uaSub:         []string{"geminicli", "google-gemini-cli"},
		headerPresent: []string{"x-gemini-api-privileged-user-id"},
		systemSub:     []string{"you are gemini cli", "this is the gemini cli"},
		toolsAll:      []string{"run_shell_command", "replace", "read_many_files"},
	},
}

// KnownClients 返回所有可识别的客户端类型标识（按规则表顺序，去重），供后台配置白名单时展示。
func KnownClients() []string {
	out := make([]string, 0, len(rules))
	seen := map[string]struct{}{}
	for _, r := range rules {
		if _, ok := seen[r.client]; ok {
			continue
		}
		seen[r.client] = struct{}{}
		out = append(out, r.client)
	}
	return out
}

// DetectHeader 仅按 HTTP 头识别，开销低，适合请求早期（选路/限流）使用。
func DetectHeader(h http.Header) Result {
	if h == nil {
		return Result{}
	}
	ua := strings.ToLower(h.Get("User-Agent"))
	xapp := strings.ToLower(strings.TrimSpace(h.Get("x-app")))
	beta := strings.ToLower(h.Get("anthropic-beta"))
	originator := strings.ToLower(strings.TrimSpace(h.Get("originator")))
	for _, r := range rules {
		if containsAny(ua, r.uaSub) ||
			equalsAny(xapp, r.xApp) ||
			containsAny(beta, r.betaSub) ||
			containsAny(originator, r.originatorSub) ||
			anyHeaderPresent(h, r.headerPresent) {
			return Result{Client: r.client, Via: "header"}
		}
	}
	return Result{}
}

// Detect 在头部识别基础上，未命中时再解析请求体的 system / tools 做二次确认。
// body 可为 nil（退化为 DetectHeader）。兼容 Anthropic 与 OpenAI 两种请求体形态。
func Detect(h http.Header, body []byte) Result {
	if res := DetectHeader(h); res.Client != "" {
		return res
	}
	if len(body) == 0 {
		return Result{}
	}
	sys, tools := probeBody(body)
	if sys == "" && len(tools) == 0 {
		return Result{}
	}
	lowSys := strings.ToLower(sys)
	// 两遍匹配：先按 system prompt 的品牌标识(可区分同源 fork，如 hermes vs openclaw)，
	// 再退回到共享工具集。否则同源家族共用的工具集会让先列出的规则抢先误判。
	for _, r := range rules {
		if containsAny(lowSys, r.systemSub) {
			return Result{Client: r.client, Via: "body"}
		}
	}
	for _, r := range rules {
		if len(r.toolsAll) > 0 && hasAllTools(tools, r.toolsAll) {
			return Result{Client: r.client, Via: "body"}
		}
	}
	return Result{}
}

// probeBody 宽松解析请求体，提取 system 文本与工具名集合，兼容 Anthropic / OpenAI 两种结构。
func probeBody(body []byte) (system string, tools map[string]struct{}) {
	var p struct {
		System       json.RawMessage `json:"system"`       // anthropic: string 或 content block 数组
		Instructions json.RawMessage `json:"instructions"` // openai Responses API（Codex 等）
		Messages     []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
		Tools []struct {
			Name     string `json:"name"` // anthropic / openai responses
			Function struct {
				Name string `json:"name"` // openai chat
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return "", nil
	}
	system = rawTextOrBlocks(p.System)
	if system == "" {
		system = rawTextOrBlocks(p.Instructions)
	}
	if system == "" {
		// OpenAI 形态：从 role=system/developer 的首条消息取文本
		for _, m := range p.Messages {
			if m.Role == "system" || m.Role == "developer" {
				if t := rawTextOrBlocks(m.Content); t != "" {
					system = t
					break
				}
			}
		}
	}
	if len(p.Tools) > 0 {
		tools = make(map[string]struct{}, len(p.Tools))
		for _, t := range p.Tools {
			name := t.Name
			if name == "" {
				name = t.Function.Name
			}
			if name != "" {
				tools[name] = struct{}{}
			}
		}
	}
	return system, tools
}

// rawTextOrBlocks 解析既可能是字符串、也可能是 content block 数组（[{type,text}]）的字段。
func rawTextOrBlocks(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if t := strings.TrimSpace(b.Text); t != "" {
				if sb.Len() > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString(t)
			}
		}
		return sb.String()
	}
	return ""
}

func containsAny(s string, subs []string) bool {
	if s == "" {
		return false
	}
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func equalsAny(s string, vals []string) bool {
	if s == "" {
		return false
	}
	return slices.Contains(vals, s)
}

func anyHeaderPresent(h http.Header, names []string) bool {
	for _, n := range names {
		if n != "" && strings.TrimSpace(h.Get(n)) != "" {
			return true
		}
	}
	return false
}

func hasAllTools(have map[string]struct{}, want []string) bool {
	if len(have) == 0 {
		return false
	}
	for _, w := range want {
		if _, ok := have[w]; !ok {
			return false
		}
	}
	return true
}
