// Package agentdetect 识别 LLM 请求来自哪个 agent 客户端
// （Claude Code、Codex CLI、Gemini CLI、openclaw、hermes、Cursor、Cline 等）。
//
// 单靠 User-Agent 不可靠（易伪造、很多 agent 直接用官方 SDK 默认 UA），
// 因此采用多信号分层识别：先看低开销的头部信号（UA / x-app / anthropic-beta /
// originator / 厂商专有头），头部不确定时再看请求体（system prompt 品牌句、
// metadata.user_id 形态、工具集指纹）二次确认。
//
// 规则表 rules 是数据驱动的，且必须与 CLIProxyAPI/internal/agentdetect/detect.go
// 保持一致（相同指纹、相同优先级）。
//
// Result.Client 为空表示未识别。Version 可能为空：仅当版本号紧跟在命中的 UA
// 产品名之后才提取，避免把通用 SDK 版本误报成 agent 版本。
package agentdetect

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// Result 为识别结果。Client 为空表示未识别。Via 标记命中来源（header/body），用于审计置信度。
type Result struct {
	Client  string `json:"client"`
	Version string `json:"version,omitempty"`
	Via     string `json:"via,omitempty"`
}

// rule 描述一个 agent 客户端的指纹。各字段为"任一命中"语义（toolsAll 与 userIDRe 除外）。
type rule struct {
	client string
	// uaSub: User-Agent 的小写子串
	uaSub []string
	// xApp: x-app 头的小写精确值（Anthropic 系客户端）
	xApp []string
	// betaSub: anthropic-beta 头的小写子串
	betaSub []string
	// originatorSub: originator 头的小写子串（Codex 系列用）
	originatorSub []string
	// headerPresent: 只要其中任一头存在（非空）即命中（如 Codex 的 x-codex-* 专有头）
	headerPresent []string
	// systemSub: system prompt / instructions 的小写子串。
	// 只用品牌句（"you are <name>"），不要用裸产品名，否则 prompt 里提到别家
	// agent 的名字就会误判。
	systemSub []string
	// toolsAll: 必须同时出现的工具名（区分大小写；大小写本身就是信号，
	// 如 Claude Code 的 "Bash" vs opencode 的 "bash"）
	toolsAll []string
	// userIDRe: 匹配请求体 metadata.user_id（Claude Code 形态）
	userIDRe *regexp.Regexp
}

// claudeUserIDRe 匹配 Claude Code 的 metadata.user_id：
// user_<64 hex>_account_<uuid>_session_<uuid>
var claudeUserIDRe = regexp.MustCompile(`^user_[a-fA-F0-9]{64}_account_[0-9a-f-]{36}_session_[0-9a-f-]{36}$`)

// rules 按优先级排序，靠前者先匹配。品牌 fork 必须排在其上游之前
// （qwen-code 在 gemini-cli 之前）；品牌 system prompt 先于共享工具集匹配（见 Detect）。
var rules = []rule{
	{
		// Claude Code（Anthropic 官方 CLI）。实测指纹：
		// UA "claude-cli/1.0.44 (external, cli)"、头 "x-app: cli"、
		// "anthropic-beta: claude-code-20250219,..."、system prompt 开头
		// "You are Claude Code, Anthropic's official CLI for Claude."、
		// metadata.user_id "user_<64hex>_account_<uuid>_session_<uuid>"、
		// 首字母大写工具集（Bash/Read/Edit/Glob/Grep/Write/TodoWrite）。
		client:    "claude-code",
		uaSub:     []string{"claude-cli", "claude-code", "claudecode"},
		xApp:      []string{"cli"},
		betaSub:   []string{"claude-code"},
		systemSub: []string{"you are claude code", "anthropic's official cli for claude"},
		toolsAll:  []string{"Bash", "Read", "Edit"},
		userIDRe:  claudeUserIDRe,
	},
	{
		// OpenAI Codex CLI（Rust）。实测：交互 TUI 的 UA/originator 为
		// "codex_cli_rs"，headless exec 为 "codex_exec"，IDE 为 "codex_vscode"；
		// 带专有 x-codex-* 头；走 Responses API 时提示词在 body 的
		// instructions 里（"running in the Codex CLI"）。
		client:        "codex",
		uaSub:         []string{"codex_cli_rs", "codex_exec", "codex_vscode", "codex_ide"},
		originatorSub: []string{"codex"},
		headerPresent: []string{"x-codex-turn-metadata", "x-codex-window-id"},
		systemSub:     []string{"running in the codex cli", "codex cli, a terminal-based"},
	},
	{
		// Qwen Code（阿里的 gemini-cli fork）。必须排在 gemini-cli 之前，
		// 使其品牌句优先于共用的 fork prompt / 工具集。
		client:    "qwen-code",
		uaSub:     []string{"qwencode", "qwen-code"},
		systemSub: []string{"you are qwen code"},
	},
	{
		// Google Gemini CLI。实测：UA "GeminiCLI/<ver>/<model> (...)" 或
		// "GeminiCLI-tui/<ver>/..."，VS Code 代理路径 UA 含
		// "proxy_client=geminicli"；带专有头 x-gemini-api-privileged-user-id；
		// snake_case 工具集（run_shell_command/replace/read_many_files）。
		client:        "gemini-cli",
		uaSub:         []string{"geminicli", "google-gemini-cli", "gemini-cli"},
		headerPresent: []string{"x-gemini-api-privileged-user-id"},
		systemSub: []string{
			"you are gemini cli",
			"this is the gemini cli",
			"you are an interactive cli agent specializing in software engineering",
		},
		toolsAll: []string{"run_shell_command", "replace", "read_many_files"},
	},
	{
		// openclaw：实测其出站请求 UA 为通用 OpenAI Node SDK
		// （"OpenAI/JS x.y" + x-stainless-* 头），无法靠 UA 区分；
		// 可靠指纹在请求体——system prompt 含 "running inside OpenClaw"，
		// 以及其专有工具集（exec / web_fetch / sessions_spawn）。
		client:    "openclaw",
		uaSub:     []string{"openclaw"},
		systemSub: []string{"running inside openclaw", "you are openclaw"},
		// 共享工具集兜底（hermes 与 openclaw 同源，工具集相同）：仅当 system
		// prompt 品牌标识缺失（如自定义了 identity）时才命中，此时归类到
		// openclaw（同一代码家族）。品牌句一遍先跑，见 Detect。
		toolsAll: []string{"exec", "web_fetch", "sessions_spawn"},
	},
	{
		// hermes（NousResearch/hermes-agent）：与 openclaw 同源、工具集相同，
		// 出站 UA 同为通用 OpenAI SDK，只能靠品牌 identity prompt 区分。
		// UA 子串收窄（裸 "hermes" 太泛，无关软件也在用）。
		client:    "hermes",
		uaSub:     []string{"hermes-agent", "hermes_agent"},
		systemSub: []string{"you are hermes agent", "you run on hermes agent", "created by nous research"},
	},
	{
		// Cursor（IDE agent / cursor-agent CLI）。品牌句 "You operate in Cursor"。
		client:    "cursor",
		uaSub:     []string{"cursor-agent", "cursorai", "cursor/"},
		systemSub: []string{"you operate in cursor", "powered by cursor"},
	},
	{
		// GitHub Copilot（chat / CLI）。copilot-integration-id 为专有头。
		client:        "copilot",
		uaSub:         []string{"githubcopilot", "github-copilot", "copilot-cli"},
		headerPresent: []string{"copilot-integration-id"},
		systemSub:     []string{"you are github copilot"},
	},
	{
		// Cline（VS Code 扩展）。identity prompt："You are Cline, ..."。
		client:    "cline",
		uaSub:     []string{"cline/"},
		systemSub: []string{"you are cline"},
	},
	{
		// Roo Code（Cline fork）。identity prompt："You are Roo, ..." / "You are Roo Code"。
		client:    "roo-code",
		uaSub:     []string{"roo-code", "roocode"},
		systemSub: []string{"you are roo,", "you are roo code"},
	},
	{
		// Kilo Code（Roo/Cline fork）。
		client:    "kilo-code",
		uaSub:     []string{"kilo-code", "kilocode"},
		systemSub: []string{"you are kilo code"},
	},
	{
		// Windsurf / Cascade（Codeium）。identity prompt："You are Cascade, ...
		// designed by the Codeium engineering team"。
		client:    "windsurf",
		uaSub:     []string{"windsurf"},
		systemSub: []string{"you are cascade", "codeium engineering team"},
	},
	{
		// Aider。主 prompt 以 "Act as an expert software developer" 开头。
		client:    "aider",
		uaSub:     []string{"aider/"},
		systemSub: []string{"act as an expert software developer"},
	},
	{
		// opencode（sst/opencode）。小写工具名可与 Claude Code 的大写集区分。
		client:    "opencode",
		uaSub:     []string{"opencode"},
		systemSub: []string{"you are opencode"},
	},
	{
		// Crush（charmbracelet/crush）。
		client:    "crush",
		uaSub:     []string{"crush/"},
		systemSub: []string{"you are crush"},
	},
	{
		// Goose（block/goose）。identity prompt："a general-purpose AI agent called Goose"。
		// UA 子串收窄：裸 "goose/" 会误命中如 "Mongoose/x"。
		client:    "goose",
		uaSub:     []string{"goose-cli", "goose_cli", "block-goose"},
		systemSub: []string{"you are goose", "called goose"},
	},
	{
		// Droid（Factory AI）。identity prompt："You are Droid, ... built by Factory"。
		// UA 子串收窄：裸 "droid/" 会误命中 Android UA。
		client:    "droid",
		uaSub:     []string{"factory-droid", "factorydroid", "droid-cli"},
		systemSub: []string{"you are droid", "built by factory"},
	},
	{
		// Amp（Sourcegraph）。UA 子串收窄；裸 "amp/" 太泛。
		client:    "amp",
		uaSub:     []string{"ampcode", "amp-cli"},
		systemSub: []string{"you are amp, a", "built by sourcegraph"},
	},
}

// uaVersionAfterToken 提取紧跟在命中的 UA 产品名之后的版本号，
// 如 "claude-cli/1.0.44"、"geminicli-tui/0.45"。
var uaVersionAfterToken = regexp.MustCompile(`^[a-z0-9_-]*[/\s]*v?([0-9][0-9a-zA-Z_.-]*)`)

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
			return Result{Client: r.client, Version: extractVersion(ua, r), Via: "header"}
		}
	}
	return Result{}
}

// Detect 在头部识别基础上，未命中时再解析请求体做二次确认。body 可为 nil（退化为 DetectHeader）。
//
// 体内匹配按特异性从高到低跑三遍：
//  1. 品牌 system prompt（可区分同源 fork，如 hermes vs openclaw）
//  2. metadata.user_id 形态（Claude Code）
//  3. 共享工具集兜底
func Detect(h http.Header, body []byte) Result {
	if res := DetectHeader(h); res.Client != "" {
		return res
	}
	if len(body) == 0 {
		return Result{}
	}
	sys, tools, userID := probeBody(body)
	if sys == "" && len(tools) == 0 && userID == "" {
		return Result{}
	}
	ua := strings.ToLower(h.Get("User-Agent"))
	lowSys := strings.ToLower(sys)
	for _, r := range rules {
		if containsAny(lowSys, r.systemSub) {
			return Result{Client: r.client, Version: extractVersion(ua, r), Via: "body"}
		}
	}
	if userID != "" {
		for _, r := range rules {
			if r.userIDRe != nil && r.userIDRe.MatchString(userID) {
				return Result{Client: r.client, Version: extractVersion(ua, r), Via: "body"}
			}
		}
	}
	for _, r := range rules {
		if len(r.toolsAll) > 0 && hasAllTools(tools, r.toolsAll) {
			return Result{Client: r.client, Version: extractVersion(ua, r), Via: "body"}
		}
	}
	return Result{}
}

// probeBody 宽松解析请求体，提取 system 文本、工具名集合与 metadata.user_id，
// 兼容 Anthropic（system/messages）、OpenAI chat（messages）、OpenAI Responses（instructions）三种结构。
func probeBody(body []byte) (system string, tools map[string]struct{}, userID string) {
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
		Metadata struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return "", nil, ""
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
	return system, tools, strings.TrimSpace(p.Metadata.UserID)
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

// extractVersion 提取紧跟在规则 UA 产品名之后的版本号；UA 中不含产品名时返回空，
// 避免把无关的 "name/version"（通常是 SDK 版本）误报为 agent 版本。
func extractVersion(lowerUA string, r rule) string {
	for _, sub := range r.uaSub {
		idx := strings.Index(lowerUA, sub)
		if idx < 0 {
			continue
		}
		rest := lowerUA[idx+len(sub):]
		if m := uaVersionAfterToken.FindStringSubmatch(rest); len(m) > 1 {
			return m[1]
		}
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
	for _, v := range vals {
		if v != "" && s == v {
			return true
		}
	}
	return false
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
