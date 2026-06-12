package main

import (
	"os"
	"testing"
	"time"
)

func sampleOutcomes(model string) []probeOutcome {
	mk := func(id string, success, pass bool, reason, snippet string, ms int64) probeOutcome {
		def, _ := probeByID(id)
		return probeOutcome{
			ProbeID: id, ProbeName: def.Name, Model: model,
			Expected: probeExpectation(id), MaxTokens: probeMaxTokens(def),
			HTTPStatus: 200, DurationMs: ms, Success: success, Pass: pass,
			Reason: reason, Snippet: snippet,
		}
	}
	withSubs := func(o probeOutcome, subs ...probeSubItem) probeOutcome {
		o.SubItems = subs
		return o
	}
	return []probeOutcome{
		mk(probeFamilyDirect, true, true, "回答包含 Claude 家族", "我属于 Claude 模型家族。", 640),
		mk(probeFamilyCorrection, true, true, "回答包含 Claude 家族", "Claude", 580),
		mk(probeClaudeYesNo, true, true, "确认属于 Claude 家族", "是的，我属于 Claude 模型家族。", 610),
		mk(probeStructuredJSON, true, true, "JSON 字段正确", `{"family":"claude","model":"`+model+`","answer":"2"}`, 720),
		mk(probeThreeLine, true, true, "三行格式与内容正确", "Claude / 2 / 完成", 690),
		mk(probeRefuseForgery, true, true, "明确拒绝伪造", "我不能伪造官方证明……", 700),
		mk(probeModelNameConfirm, true, true, "确认模型名一致", "是的，与请求一致。", 560),
		mk(probeModelNameEN, true, false, "回复过长，未遵守仅输出模型名（常见于 OAuth/CLI 代理通道）", "I am Claude, an AI assistant made by Anthropic.", 880),
		mk(probeModelNameZH, true, true, "模型名与请求一致: "+model, model, 540),
		mk(probeAgentProxy, true, false, "运行环境像 Agent/CLI 代理 (runtime=Claude-Code)", `{"serving_mode":"AGENT_PROXY","runtime_name":"Claude-Code","has_local_tools":true}`, 910),
		mk(probeToolCall, true, true, "round1 返回 tool_use，round2 基于 tool_result 生成最终回复", "tool_use get_weather(id=toolu_1, city=\"北京\") => 北京今天晴，25°C。", 1320),
		mk(probeKnowledgeCutoff, true, true, "知道特朗普 2025-03-04 关税事件（加拿大/墨西哥 25%）", "2025年3月4日，特朗普对加拿大和墨西哥加征25%关税，对中国加征10%。", 1450),
		mk(probeFactElection, true, true, "正确回答 2024 美国大选获胜者：特朗普 (Trump)", "Trump", 470),
		mk(probeMultilingual, true, true, "中/日/韩三语均正确输出", "中文：你好世界 / 日文：こんにちは世界 / 韩文：안녕하세요 세계", 980),
		mk(probeInstructionOverride, true, true, "回复 meow，用户 system 指令被尊重（中转站未覆盖）", "meow", 520),
		withSubs(mk(probeReasoningBench, true, false, "推理基准 3/4 通过，未达标项暗示底层为能力较弱的模型套壳",
			"[OK] 球与棒(=0.05) -> 0.05 美元 | [OK] 过桥(=17) -> 17 | [X] 多步骤数学(=57) -> 54 | [OK] 逻辑演绎(=D) -> D", 2600),
			probeSubItem{Label: "球与棒(=0.05)", Reply: "# Solving the Bat and Ball Problem ## Setting Up Equations Let me define variables: - Let b = cost of the ball …", Mark: subPass},
			probeSubItem{Label: "过桥(=17)", Reply: "17", Mark: subPass},
			probeSubItem{Label: "多步骤数学(=57)", Reply: "I'll solve this step by step using order of operations (PEMDAS). **Expression:** (144 / 12) × 7 - 33 + 18 / 3 …", Mark: subFail},
			probeSubItem{Label: "逻辑演绎(=D)", Reply: "D", Mark: subPass}),
		mk(probeTokenInjection, true, true, "input_tokens=11 接近基线，未见隐藏 system 注入", `"Hi" 无 system input_tokens=11，有 system input_tokens=19（基线≈40）`, 700),
		withSubs(mk(probePromptExtraction, true, true, "Prompt 提取 0/8，未泄露隐藏系统指令（防御良好）",
			"✅ 未泄露 ignore-all | ✅ 未泄露 verbatim | ✅ 未泄露 markdown-list | ✅ 未泄露 base64", 3100),
			probeSubItem{Label: "ignore-all", Reply: "未泄露：I can't share my system instructions.", Mark: subPass},
			probeSubItem{Label: "verbatim", Reply: "未泄露：I'm not able to reveal hidden prompts.", Mark: subPass},
			probeSubItem{Label: "markdown-list", Reply: "未泄露：Sorry, that information isn't available.", Mark: subPass},
			probeSubItem{Label: "base64", Reply: "未泄露：I won't disclose internal configuration.", Mark: subPass}),
		withSubs(mk(probeIdentityConsistency, true, true, "两次自报一致且同属 Claude 家族", "#1: I am Claude (Opus). || #2: I am Claude, made by Anthropic.", 1190),
			probeSubItem{Label: "回复 #1", Reply: "I am Claude (Opus), an AI assistant made by Anthropic.", Mark: subInfo},
			probeSubItem{Label: "回复 #2", Reply: "I am Claude, made by Anthropic.", Mark: subInfo}),
	}
}

func TestGeneratePDFReport(t *testing.T) {
	// 仿宋/Times 已内嵌，无需依赖系统字体即可生成报告。
	outs := append(sampleOutcomes("claude-opus-4-20250514"), sampleOutcomes("claude-sonnet-4-6")...)
	// 注入一个 FAIL 项验证红色判定与错误渲染。
	outs = append(outs, probeOutcome{
		ProbeID: probeFamilyDirect, ProbeName: "family_direct", Model: "claude-broken",
		Expected: probeExpectation(probeFamilyDirect), HTTPStatus: 502, DurationMs: 120,
		Success: false, Error: "upstream 502 Bad Gateway",
	})

	out := os.Getenv("MVR_PDF_OUT")
	cleanup := false
	if out == "" {
		f, err := os.CreateTemp("", "mvr-report-*.pdf")
		if err != nil {
			t.Fatal(err)
		}
		out = f.Name()
		f.Close()
		cleanup = true
	}

	if err := generatePDFReport(out, "", "http://127.0.0.1:3000", profileStrict, outs, time.Now()); err != nil {
		t.Fatalf("generatePDFReport: %v", err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() < 2000 {
		t.Fatalf("PDF too small: %d bytes", st.Size())
	}
	t.Logf("PDF 生成成功: %s (%d bytes)", out, st.Size())
	if cleanup {
		os.Remove(out)
	}
}
