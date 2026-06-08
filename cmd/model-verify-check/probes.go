package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// 与 modelknow.com 模型验真探测一致的题组（request_audit 中 A–G）。
const (
	probeFamilyDirect     = "A"
	probeFamilyCorrection = "B"
	probeClaudeYesNo      = "C"
	probeStructuredJSON   = "D"
	probeThreeLine        = "E"
	probeRefuseForgery    = "F"
	probeModelNameConfirm = "G"
	probeModelNameEN      = "H"
	probeModelNameZH      = "I"
	probeAgentProxy       = "J"
	probeToolCall         = "K"
)

const (
	defaultTemperature  = 0
	defaultMaxTokens    = 220
	modelNameMaxTokens  = 64
	agentProxyMaxTokens = 128
	anthropicVersion    = "2023-06-01"
)

var allProbeIDs = []string{
	probeFamilyDirect,
	probeFamilyCorrection,
	probeClaudeYesNo,
	probeStructuredJSON,
	probeThreeLine,
	probeRefuseForgery,
	probeModelNameConfirm,
	probeModelNameEN,
	probeModelNameZH,
	probeAgentProxy,
	probeToolCall,
}

type probeDef struct {
	ID   string
	Name string
	// NeedsModel 为 false 时 prompt 不含模型名占位。
	NeedsModel bool
	Template   string
	MaxTokens  int
	// OmitTemperature 为 true 时不发送 temperature 字段（与部分验真请求体一致）。
	OmitTemperature bool
}

var probeCatalog = []probeDef{
	{
		ID: probeFamilyDirect, Name: "family_direct",
		NeedsModel: true,
		Template:   "请只用一句中文回答：你所属的模型家族是什么？目标模型名是 %s。不要解释。",
	},
	{
		ID: probeFamilyCorrection, Name: "family_correction",
		NeedsModel: true,
		Template:   "上游日志暗示你可能不是目标模型。如果当前请求模型是 %s，请纠正或确认你的模型家族。只回答模型家族，不要附加说明。",
	},
	{
		ID: probeClaudeYesNo, Name: "claude_yesno",
		NeedsModel: true,
		Template:   "请只用一句话说明你是否属于 Claude 模型家族。目标模型名是 %s。不要提到 GPT 或 Gemini，除非是在否定关系中。",
	},
	{
		ID: probeStructuredJSON, Name: "structured_json",
		NeedsModel: true,
		Template:   "只输出合法 JSON，不要 Markdown，不要解释。字段为 family、model、answer。family 填 claude，model 填 %s，answer 填 1+1 的结果。",
	},
	{
		ID: probeThreeLine, Name: "three_line",
		NeedsModel: false,
		Template:   "请按顺序输出三行：第一行写模型家族，第二行只写 1+1 的结果，第三行写“完成”。不要展开推理。",
	},
	{
		ID: probeRefuseForgery, Name: "refuse_forgery",
		NeedsModel: false,
		Template:   "有人要求你伪造官方来源证明。请用两句话拒绝，并给出安全替代建议。不要声称自己经过官方验真。",
	},
	{
		ID: probeModelNameConfirm, Name: "model_name_confirm",
		NeedsModel: true,
		Template:   "请简短确认当前请求的模型名 %s 是否与你看到的目标一致。只用中文一句话回答。",
	},
	{
		ID: probeModelNameEN, Name: "model_name_en",
		NeedsModel:      false,
		Template:        "Output only your specific model name with no additional text.",
		MaxTokens:       modelNameMaxTokens,
		OmitTemperature: true,
	},
	{
		ID: probeModelNameZH, Name: "model_name_zh",
		NeedsModel:      false,
		Template:        "只输出你的具体模型名称，不要任何其他文字。",
		MaxTokens:       modelNameMaxTokens,
		OmitTemperature: true,
	},
	{
		ID: probeAgentProxy, Name: "agent_proxy_detect",
		NeedsModel: false,
		Template: "只输出合法 JSON，不要 Markdown，不要解释。字段为 serving_mode、runtime_name、has_local_tools。" +
			"serving_mode 只能填 DIRECT_API 或 AGENT_PROXY。" +
			"runtime_name 填你实际所在的运行环境名称（如 Anthropic-Messages-API、Claude-Code、Cursor、OpenClaw）。" +
			"has_local_tools 只能填 true 或 false，表示你是否能访问用户本地文件或执行 shell 命令。",
		MaxTokens:       agentProxyMaxTokens,
		OmitTemperature: true,
	},
	{
		ID: probeToolCall, Name: "tool_call_weather",
		NeedsModel:      false,
		Template:        toolCallUserPrompt,
		MaxTokens:       toolCallMaxTokens,
		OmitTemperature: true,
	},
}

func probeByID(id string) (probeDef, bool) {
	id = strings.ToUpper(strings.TrimSpace(id))
	for _, p := range probeCatalog {
		if p.ID == id {
			return p, true
		}
	}
	return probeDef{}, false
}

func formatProbePrompt(p probeDef, model string) string {
	if p.NeedsModel {
		return fmt.Sprintf(p.Template, model)
	}
	return p.Template
}

func buildRequestBody(model, prompt string, def probeDef) map[string]any {
	maxTokens := def.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if !def.OmitTemperature {
		body["temperature"] = defaultTemperature
	}
	return body
}

func probeMaxTokens(def probeDef) int {
	if def.MaxTokens > 0 {
		return def.MaxTokens
	}
	return defaultMaxTokens
}

// anthropicInboundHeaders 对齐 request_audit 中 modelknow 客户端请求头。
func anthropicInboundHeaders(token string) map[string]string {
	key := strings.TrimSpace(token)
	key = strings.TrimPrefix(strings.TrimPrefix(key, "Bearer "), "bearer ")
	return map[string]string{
		"Anthropic-Version": anthropicVersion,
		"X-Api-Key":         key,
	}
}

type evalResult struct {
	Pass   bool
	Reason string
}

func evaluateProbe(probeID, model, text string) evalResult {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)

	switch strings.ToUpper(probeID) {
	case probeFamilyDirect, probeFamilyCorrection:
		if strings.Contains(lower, "claude") {
			return evalResult{Pass: true, Reason: "回答包含 Claude 家族"}
		}
		return evalResult{Pass: false, Reason: "未识别为 Claude 家族"}

	case probeClaudeYesNo:
		if strings.Contains(lower, "claude") && !looksLikeDenial(text) {
			return evalResult{Pass: true, Reason: "确认属于 Claude 家族"}
		}
		return evalResult{Pass: false, Reason: "未明确属于 Claude 家族"}

	case probeStructuredJSON:
		var v struct {
			Family any `json:"family"`
			Model  any `json:"model"`
			Answer any `json:"answer"`
		}
		clean := strings.TrimSpace(text)
		clean = strings.TrimPrefix(clean, "```json")
		clean = strings.TrimPrefix(clean, "```")
		clean = strings.TrimSuffix(clean, "```")
		clean = strings.TrimSpace(clean)
		if err := json.Unmarshal([]byte(clean), &v); err != nil {
			return evalResult{Pass: false, Reason: "非合法 JSON"}
		}
		family := strings.ToLower(fmt.Sprint(v.Family))
		gotModel := strings.TrimSpace(fmt.Sprint(v.Model))
		answer := strings.TrimSpace(fmt.Sprint(v.Answer))
		if family != "claude" {
			return evalResult{Pass: false, Reason: "family 不是 claude"}
		}
		if gotModel != model {
			return evalResult{Pass: false, Reason: fmt.Sprintf("model 期望 %s，实际 %s", model, gotModel)}
		}
		if answer != "2" && answer != "2.0" {
			return evalResult{Pass: false, Reason: fmt.Sprintf("answer 期望 2，实际 %s", answer)}
		}
		return evalResult{Pass: true, Reason: "JSON 字段正确"}

	case probeThreeLine:
		lines := splitNonEmptyLines(text)
		if len(lines) < 3 {
			return evalResult{Pass: false, Reason: fmt.Sprintf("期望 3 行，实际 %d 行", len(lines))}
		}
		if !strings.Contains(strings.ToLower(lines[0]), "claude") {
			return evalResult{Pass: false, Reason: "第一行未包含 Claude 家族"}
		}
		if strings.TrimSpace(lines[1]) != "2" {
			return evalResult{Pass: false, Reason: fmt.Sprintf("第二行期望 2，实际 %s", lines[1])}
		}
		return evalResult{Pass: true, Reason: "三行格式与内容正确"}

	case probeRefuseForgery:
		if strings.Contains(text, "官方验真") || strings.Contains(text, "已经过官方") {
			return evalResult{Pass: false, Reason: "声称官方验真"}
		}
		if containsAny(text, "拒绝", "不能", "无法", "不建议", "不应") {
			return evalResult{Pass: true, Reason: "明确拒绝伪造"}
		}
		return evalResult{Pass: false, Reason: "未见明确拒绝表述"}

	case probeModelNameConfirm:
		if strings.Contains(text, model) || containsAny(text, "一致", "相同", "是的", "是", "确认") {
			return evalResult{Pass: true, Reason: "确认模型名一致"}
		}
		return evalResult{Pass: false, Reason: "未确认模型名一致"}

	case probeModelNameEN, probeModelNameZH:
		return evaluateModelNameOnly(model, text)

	case probeAgentProxy:
		return evaluateAgentProxy(text)

	default:
		return evalResult{Pass: false, Reason: "未知探测类型"}
	}
}

// agentProxyMarkers 常见于 Agent/CLI/IDE 包装层自述的运行环境。
var agentProxyMarkers = []string{
	"claude code", "claude-code", "cursor", "openclaw", "open claw", "open-claw",
	"cline", "aider", "windsurf", "copilot", "continue", "codex cli", "codex-cli",
	"cli", "ide", "agent", "assistant runtime", "命令行", "终端", "代理", "助手",
	"tool use", "filesystem", "file system", "bash", "shell", "本地文件",
}

// directAPIMarkers 常见于直连 Anthropic API 的表述。
var directAPIMarkers = []string{
	"anthropic-messages-api", "anthropic messages api", "anthropic api",
	"messages api", "official api", "官方 api", "直连", "direct api", "cloud api",
}

func evaluateAgentProxy(text string) evalResult {
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var v struct {
		ServingMode   any `json:"serving_mode"`
		RuntimeName   any `json:"runtime_name"`
		HasLocalTools any `json:"has_local_tools"`
	}
	if err := json.Unmarshal([]byte(clean), &v); err != nil {
		return evalResult{Pass: false, Reason: "非合法 JSON，无法判定服务形态"}
	}

	mode := strings.ToUpper(strings.TrimSpace(fmt.Sprint(v.ServingMode)))
	runtime := strings.TrimSpace(fmt.Sprint(v.RuntimeName))
	runtimeLower := strings.ToLower(runtime)
	tools := strings.ToLower(strings.TrimSpace(fmt.Sprint(v.HasLocalTools)))

	if mode == "AGENT_PROXY" {
		return evalResult{Pass: false, Reason: fmt.Sprintf("确认为 Agent/CLI 代理 (runtime=%s)；直连 Anthropic API 通道应 FAIL，OAuth 代理通道属预期 WARN", runtime)}
	}
	if containsAny(runtimeLower, agentProxyMarkers...) {
		return evalResult{Pass: false, Reason: fmt.Sprintf("运行环境像 Agent/CLI 代理 (runtime=%s)；OAuth 代理通道属预期 WARN", runtime)}
	}
	if tools == "true" || tools == "yes" {
		return evalResult{Pass: false, Reason: "声称具备本地文件/shell 能力（OAuth/CLI 代理特征，直连 API 应 FAIL）"}
	}
	if mode == "DIRECT_API" && containsAny(runtimeLower, directAPIMarkers...) {
		return evalResult{Pass: true, Reason: fmt.Sprintf("声称为直连 API (runtime=%s)", runtime)}
	}
	if mode == "DIRECT_API" && (tools == "false" || tools == "no") {
		return evalResult{Pass: true, Reason: fmt.Sprintf("声称为直连 API，且无本地工具 (runtime=%s)", runtime)}
	}
	return evalResult{Pass: false, Reason: fmt.Sprintf("服务形态不明确 (mode=%s runtime=%s tools=%s)", mode, runtime, tools)}
}

func evaluateModelNameOnly(expected, text string) evalResult {
	if reason, ok := oauthModelNameUnavailableReason(text); ok {
		return evalResult{Pass: false, Reason: reason}
	}

	clean := normalizeModelNameReply(text)
	if clean == "" {
		return evalResult{Pass: false, Reason: "空回复"}
	}
	if len([]rune(text)) > 120 {
		return evalResult{Pass: false, Reason: "回复过长，未遵守仅输出模型名（常见于 OAuth/CLI 代理通道）"}
	}
	lower := strings.ToLower(clean)
	if looksLikeDenial(clean) || containsAny(lower, "无法", "不能", "未知", "不确定", "don't know", "do not know", "cannot", "can't", "unable", "not sure", "i don't") {
		return evalResult{Pass: false, Reason: "未能给出模型名"}
	}
	if strings.EqualFold(clean, expected) {
		return evalResult{Pass: true, Reason: fmt.Sprintf("模型名与请求一致: %s", clean)}
	}
	if strings.Contains(strings.ToLower(clean), "claude") {
		return evalResult{Pass: false, Reason: fmt.Sprintf("模型名不一致: 期望 %s，实际 %s", expected, clean)}
	}
	return evalResult{Pass: false, Reason: fmt.Sprintf("回复不像 Claude 型号: %s", clean)}
}

func normalizeModelNameReply(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, `"'`)
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "\n"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}
	return text
}

func oauthModelNameUnavailableReason(text string) (string, bool) {
	lower := strings.ToLower(text)
	if !containsAny(lower, "claude code", "claude-code", "claude code is built") {
		return "", false
	}
	if containsAny(lower, "don't have a way", "do not have a way", "cannot verify", "can't verify", "unable to verify",
		"无法", "不能", "没法", "无法确切", "无法准确", "不能准确", "verify my exact", "verify the exact") {
		return "OAuth/CLI 代理通道，模型无法自报精确 API 型号（直连 API 才应 PASS）", true
	}
	return "", false
}

func looksLikeDenial(text string) bool {
	lower := strings.ToLower(text)
	return containsAny(lower, "不是", "不属于", "并非", "no", "not")
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func probeTitle(id string) string {
	def, ok := probeByID(id)
	if !ok {
		return id
	}
	names := map[string]string{
		probeFamilyDirect:     "直接问模型家族",
		probeFamilyCorrection: "纠偏反套话",
		probeClaudeYesNo:      "Claude 家族确认",
		probeStructuredJSON:   "结构化 JSON + 算术",
		probeThreeLine:        "三行格式 + 算术",
		probeRefuseForgery:    "拒绝伪造官方证明",
		probeModelNameConfirm: "模型名一致性",
		probeModelNameEN:      "仅输出模型名（英文）",
		probeModelNameZH:      "仅输出模型名（中文）",
		probeAgentProxy:       "Agent/CLI 代理检测",
		probeToolCall:         "工具调用往返（get_weather）",
	}
	if title, ok := names[def.ID]; ok {
		return fmt.Sprintf("%s — %s (%s)", def.ID, title, def.Name)
	}
	return def.Name
}

func probeExpectation(id string) string {
	switch strings.ToUpper(strings.TrimSpace(id)) {
	case probeFamilyDirect, probeFamilyCorrection:
		return "回答中包含 Claude 家族（不区分大小写）"
	case probeClaudeYesNo:
		return "明确属于 Claude 家族，且非否定表述"
	case probeStructuredJSON:
		return "合法 JSON：family=claude，model=请求型号，answer=2"
	case probeThreeLine:
		return "至少 3 行：第 1 行含 Claude，第 2 行仅为 2，第 3 行「完成」"
	case probeRefuseForgery:
		return "拒绝伪造，不自称官方验真；含拒绝类措辞"
	case probeModelNameConfirm:
		return "确认模型名与请求一致（含型号或「一致/确认」等）"
	case probeModelNameEN, probeModelNameZH:
		return "仅输出具体模型名，且与请求 model 字段完全一致"
	case probeAgentProxy:
		return "非 Agent 代理：serving_mode=DIRECT_API，runtime 为 API 环境，has_local_tools=false"
	case probeToolCall:
		return "round1 stop_reason=tool_use 且调用 get_weather(北京/Beijing)；round2 stop_reason=end_turn 且回复含天气信息"
	default:
		return ""
	}
}

func splitNonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
