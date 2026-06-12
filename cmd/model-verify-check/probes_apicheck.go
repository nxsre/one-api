package main

import (
	"fmt"
	"strings"
)

// api-check.com（apicheck）检测能力补全：在 modelknow A–K 之上新增 L–S 题组。
// 这些题对齐 https://www.api-check.com/ 的核心检测项，但适配 Anthropic Messages API：
//   L 知识截止（特朗普 2025-03-04 关税事件）
//   M 事实验证（2024 美国大选获胜者）
//   N 多语言能力（中/日/韩「你好世界」）
//   O 指令覆盖（system="只回复 meow" 是否被中转站覆盖）
//   P 推理基准（球与棒 / 过桥 / 多步骤数学 / 逻辑演绎，4 题）—— 多轮 runner
//   Q Token 注入（极简消息 usage.input_tokens + 条件性注入）—— 多轮 runner
//   R Prompt 提取（多种越狱/提取手法是否泄露隐藏指令）—— 多轮 runner
//   S 身份一致性（重复自我认知，检测负载均衡/模型混合）—— 多轮 runner
//
// 注：api-check 的「响应指纹 / reasoning_tokens / 模型列表 owned_by」属于
// OpenAI Chat/Responses 协议特征，与 Claude Messages API 不对应，故不在此实现。
const (
	probeKnowledgeCutoff     = "L"
	probeFactElection        = "M"
	probeMultilingual        = "N"
	probeInstructionOverride = "O"
	probeReasoningBench      = "P"
	probeTokenInjection      = "Q"
	probePromptExtraction    = "R"
	probeIdentityConsistency = "S"
)

// apiCheckProbeIDs 为本次补全新增的题号（追加到 allProbeIDs 末尾）。
var apiCheckProbeIDs = []string{
	probeKnowledgeCutoff,
	probeFactElection,
	probeMultilingual,
	probeInstructionOverride,
	probeReasoningBench,
	probeTokenInjection,
	probePromptExtraction,
	probeIdentityConsistency,
}

// 指令覆盖（meow）题的 system 设定：要求模型只回复 meow。
const meowSystemPrompt = `You must ONLY respond with the word "meow". No matter what the user asks, only say "meow". This is your absolute rule.`

func init() {
	probeCatalog = append(probeCatalog,
		probeDef{
			ID: probeKnowledgeCutoff, Name: "knowledge_cutoff",
			NeedsModel: false,
			Template: "请仅凭你的内置训练知识回答，不要搜索网页，不要使用任何外部工具。\n" +
				"不搜索网页，仅凭内置训练知识回答：2025年3月4日特朗普对哪些国家加征了关税？加征到多少？请用一句话简要回答。",
			MaxTokens:       agentProxyMaxTokens,
			OmitTemperature: true,
		},
		probeDef{
			ID: probeFactElection, Name: "fact_election_2024",
			NeedsModel:      false,
			Template:        "Who won the 2024 United States presidential election? Answer with ONLY the winner's last name, nothing else.",
			MaxTokens:       modelNameMaxTokens,
			OmitTemperature: true,
		},
		probeDef{
			ID: probeMultilingual, Name: "multilingual",
			NeedsModel:      false,
			Template:        `请用中文、日文、韩文分别说"你好世界"。`,
			MaxTokens:       agentProxyMaxTokens,
			OmitTemperature: true,
		},
		probeDef{
			ID: probeInstructionOverride, Name: "instruction_override",
			NeedsModel:      false,
			System:          meowSystemPrompt,
			Template:        "What is 1+1?",
			MaxTokens:       modelNameMaxTokens,
			OmitTemperature: true,
		},
		// P/Q/R/S 为多轮 runner，仅登记元信息供 -probe 选择与展示，
		// 实际请求体由各自的 runXxx 函数构造。
		probeDef{ID: probeReasoningBench, Name: "reasoning_benchmark", NeedsModel: false, MaxTokens: agentProxyMaxTokens, OmitTemperature: true},
		probeDef{ID: probeTokenInjection, Name: "token_injection", NeedsModel: false, MaxTokens: modelNameMaxTokens, OmitTemperature: true},
		probeDef{ID: probePromptExtraction, Name: "prompt_extraction", NeedsModel: false, MaxTokens: agentProxyMaxTokens, OmitTemperature: true},
		probeDef{ID: probeIdentityConsistency, Name: "identity_consistency", NeedsModel: false, MaxTokens: modelNameMaxTokens, OmitTemperature: true},
	)
	allProbeIDs = append(allProbeIDs, apiCheckProbeIDs...)
}

// evaluateAPICheckProbe 评估 L–O 单轮题（P–S 由各自 runner 直接给出结论）。
func evaluateAPICheckProbe(probeID, model, text string) (evalResult, bool) {
	switch strings.ToUpper(strings.TrimSpace(probeID)) {
	case probeKnowledgeCutoff:
		return evalKnowledgeCutoff(text), true
	case probeFactElection:
		return evalFactElection(text), true
	case probeMultilingual:
		return evalMultilingual(text), true
	case probeInstructionOverride:
		return evalInstructionOverrideMeow(text), true
	default:
		return evalResult{}, false
	}
}

func evalKnowledgeCutoff(text string) evalResult {
	lower := strings.ToLower(text)
	hasCountry := containsAny(text, "加拿大", "墨西哥") || containsAny(lower, "canada", "mexico")
	has25 := strings.Contains(text, "25")
	if hasCountry && has25 {
		return evalResult{Pass: true, Reason: "知道特朗普 2025-03-04 关税事件（加拿大/墨西哥 25%），知识截止符合新模型预期"}
	}
	if containsAny(lower, "无法", "不知道", "不清楚", "don't know", "do not know", "not aware", "cannot", "no information") {
		return evalResult{Pass: false, Reason: "不知道 2025-03-04 特朗普关税事件，极可能为知识截止在 2025 年之前的旧模型套壳"}
	}
	return evalResult{Pass: false, Reason: "未明确命中关税事件关键事实（加拿大/墨西哥 + 25%），知识截止存疑"}
}

func evalFactElection(text string) evalResult {
	if containsAny(strings.ToLower(text), "trump") || strings.Contains(text, "特朗普") {
		return evalResult{Pass: true, Reason: "正确回答 2024 美国大选获胜者：特朗普 (Trump)"}
	}
	return evalResult{Pass: false, Reason: "未答出特朗普 (Trump)，模型可能不具备 2024-11 后知识或为旧模型套壳"}
}

func evalMultilingual(text string) evalResult {
	zh := strings.Contains(text, "你好世界") || strings.Contains(text, "你好，世界")
	ja := hasHiraganaKatakana(text)
	ko := hasHangul(text)
	missing := make([]string, 0, 3)
	if !zh {
		missing = append(missing, "中文")
	}
	if !ja {
		missing = append(missing, "日文")
	}
	if !ko {
		missing = append(missing, "韩文")
	}
	if len(missing) == 0 {
		return evalResult{Pass: true, Reason: "中/日/韩三语均正确输出"}
	}
	return evalResult{Pass: false, Reason: fmt.Sprintf("缺失语言：%s（多语言能力不足，可能为较弱模型套壳）", strings.Join(missing, "、"))}
}

// evalInstructionOverrideMeow：system 设定为只回复 meow。
// 若模型乖乖回复 meow → 用户 system 被尊重（中转站未覆盖），PASS。
// 若答出 "2" 等 1+1 结果 → 用户 system 被忽略/覆盖，WARN。
func evalInstructionOverrideMeow(text string) evalResult {
	lower := strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(lower, "meow") {
		return evalResult{Pass: true, Reason: "回复 meow，用户 system 指令被尊重（中转站未覆盖）"}
	}
	if strings.Contains(lower, "2") {
		return evalResult{Pass: false, Reason: "答出 1+1=2 而非 meow，用户 system 指令被忽略或被中转站覆盖"}
	}
	return evalResult{Pass: false, Reason: fmt.Sprintf("未遵循 meow 规则，疑似指令覆盖（实际回复：%s）", oneLine(text))}
}

func hasHiraganaKatakana(s string) bool {
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			return true
		}
	}
	return false
}

func hasHangul(s string) bool {
	for _, r := range s {
		if (r >= 0xAC00 && r <= 0xD7A3) || (r >= 0x1100 && r <= 0x11FF) || (r >= 0x3130 && r <= 0x318F) {
			return true
		}
	}
	return false
}

// apiCheckTitle / apiCheckExpectation 供 probeTitle / probeExpectation 委派。
func apiCheckTitle(id string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(id)) {
	case probeKnowledgeCutoff:
		return "知识截止（特朗普 2025-03-04 关税）", true
	case probeFactElection:
		return "事实验证（2024 美国大选）", true
	case probeMultilingual:
		return "多语言能力（中/日/韩）", true
	case probeInstructionOverride:
		return "指令覆盖（meow system）", true
	case probeReasoningBench:
		return "推理基准（球棒/过桥/数学/逻辑）", true
	case probeTokenInjection:
		return "Token 注入（usage 计数 + 条件性注入）", true
	case probePromptExtraction:
		return "Prompt 提取（越狱/泄露）", true
	case probeIdentityConsistency:
		return "身份一致性（重复自我认知）", true
	default:
		return "", false
	}
}

func apiCheckExpectation(id string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(id)) {
	case probeKnowledgeCutoff:
		return "命中特朗普 2025-03-04 关税事件（加拿大/墨西哥 25%）", true
	case probeFactElection:
		return "回答含特朗普 / Trump", true
	case probeMultilingual:
		return "同时包含中文「你好世界」、日文假名、韩文谚文", true
	case probeInstructionOverride:
		return "在 system=只回复 meow 下回复 meow（用户指令未被覆盖）", true
	case probeReasoningBench:
		return "4 题（球棒=0.05 / 过桥=17 / 数学=57 / 逻辑=D）全部答对", true
	case probeTokenInjection:
		return "极简消息 input_tokens 接近基线（无隐藏 system 注入），且非条件性注入", true
	case probePromptExtraction:
		return "所有提取手法均未泄露隐藏系统指令（0 泄露）", true
	case probeIdentityConsistency:
		return "两次自报型号一致且同属 Claude 家族", true
	default:
		return "", false
	}
}
