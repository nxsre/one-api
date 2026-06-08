package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/cmd/internal/apitest"
)

const (
	toolCallMaxTokens   = 1024
	toolCallWeatherName = "get_weather"
	toolCallUserPrompt  = "北京天气怎么样？"
	toolCallToolResult  = "晴，25°C，湿度 60%"
)

func weatherToolDefinition() map[string]any {
	return map[string]any{
		"name":        toolCallWeatherName,
		"description": "Get weather for a city",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string"},
			},
			"required": []string{"city"},
		},
	}
}

func buildToolCallRound1Body(model string) map[string]any {
	return map[string]any{
		"model":      model,
		"max_tokens": toolCallMaxTokens,
		"stream":     false,
		"tools":      []map[string]any{weatherToolDefinition()},
		"messages": []map[string]string{
			{"role": "user", "content": toolCallUserPrompt},
		},
	}
}

func buildToolCallRound2Body(model, toolUseID string, round1Assistant []anthropicContentBlock) map[string]any {
	messages := []any{
		map[string]string{"role": "user", "content": toolCallUserPrompt},
		map[string]any{
			"role":    "assistant",
			"content": round1Assistant,
		},
		map[string]any{
			"role": "user",
			"content": []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": toolUseID,
					"content":     toolCallToolResult,
				},
			},
		},
	}
	return map[string]any{
		"model":      model,
		"max_tokens": toolCallMaxTokens,
		"stream":     false,
		"tools":      []map[string]any{weatherToolDefinition()},
		"messages":   messages,
	}
}

type anthropicContentBlock struct {
	Type  string         `json:"type"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
	Text  string         `json:"text,omitempty"`
}

type anthropicMessageResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	StopReason string                  `json:"stop_reason"`
	Content    []anthropicContentBlock `json:"content"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func runToolCallProbe(cli *apitest.Client, headers map[string]string, model string) probeOutcome {
	round1Body := buildToolCallRound1Body(model)
	round1Raw, _ := json.Marshal(round1Body)

	start := time.Now()
	ex1, err := postAnthropicJSON(cli, "/v1/messages", round1Body, headers)
	durationMs := time.Since(start).Milliseconds()

	result := probeOutcome{
		ProbeID:        probeToolCall,
		ProbeName:      "tool_call_weather",
		Model:          model,
		Prompt:         toolCallUserPrompt,
		Expected:       probeExpectation(probeToolCall),
		MaxTokens:      toolCallMaxTokens,
		DurationMs:     durationMs,
		RequestBody:    string(round1Raw),
		HasTemperature: false,
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.HTTPStatus = ex1.StatusCode
	result.RequestHeaders = ex1.RequestHeaders
	result.ResponseHeaders = ex1.ResponseHeaders
	result.ResponseBody = ex1.ResponseBody

	if ex1.StatusCode != 200 {
		result.Error = truncate(ex1.ResponseBody, 500)
		return result
	}

	msg1, errParse1 := parseAnthropicMessageResponse(ex1.ResponseBody)
	if errParse1 != nil {
		result.Error = errParse1.Error()
		return result
	}
	if msg1.Error != nil && strings.TrimSpace(msg1.Error.Message) != "" {
		result.Error = fmt.Sprintf("round1 api error: %s", msg1.Error.Message)
		return result
	}

	toolUse, errTool := extractToolUse(msg1)
	if errTool != nil {
		result.Success = true
		result.Pass = false
		result.Reason = errTool.Error()
		result.Snippet = anthropicText(msg1)
		return result
	}

	round2Body := buildToolCallRound2Body(model, toolUse.ID, msg1.Content)
	round2Raw, _ := json.Marshal(round2Body)
	ex2, err := postAnthropicJSON(cli, "/v1/messages", round2Body, headers)
	if err != nil {
		result.Error = err.Error()
		result.ResponseBody = formatToolCallReport(string(round1Raw), string(round2Raw), ex1, nil)
		return result
	}

	result.HTTPStatus = ex2.StatusCode
	result.ResponseHeaders = ex2.ResponseHeaders
	result.ResponseBody = formatToolCallReport(string(round1Raw), string(round2Raw), ex1, ex2)

	if ex2.StatusCode != 200 {
		result.Error = truncate(ex2.ResponseBody, 500)
		return result
	}

	msg2, errParse2 := parseAnthropicMessageResponse(ex2.ResponseBody)
	if errParse2 != nil {
		result.Error = errParse2.Error()
		return result
	}

	eval := evaluateToolCallRoundtrip(msg1, msg2, toolUse)
	result.Success = eval.Success
	result.Pass = eval.Pass
	result.Reason = eval.Reason
	result.Snippet = eval.Snippet
	if eval.Error != "" {
		result.Error = eval.Error
	}
	return result
}

type toolCallEval struct {
	Success bool
	Pass    bool
	Reason  string
	Snippet string
	Error   string
}

func evaluateToolCallRoundtrip(msg1, msg2 anthropicMessageResponse, toolUse anthropicContentBlock) toolCallEval {
	if msg1.StopReason != "tool_use" {
		return toolCallEval{
			Success: true,
			Pass:    false,
			Reason:  fmt.Sprintf("round1 stop_reason 期望 tool_use，实际 %q", msg1.StopReason),
			Snippet: summarizeToolUse(toolUse),
		}
	}
	if toolUse.Name != toolCallWeatherName {
		return toolCallEval{
			Success: true,
			Pass:    false,
			Reason:  fmt.Sprintf("round1 tool 期望 %s，实际 %s", toolCallWeatherName, toolUse.Name),
			Snippet: summarizeToolUse(toolUse),
		}
	}
	city := strings.TrimSpace(fmt.Sprint(toolUse.Input["city"]))
	if city == "" || !cityMatchesProbeTarget(city) {
		return toolCallEval{
			Success: true,
			Pass:    false,
			Reason:  fmt.Sprintf("round1 tool input.city 期望指北京（北京/Beijing），实际 %q", city),
			Snippet: summarizeToolUse(toolUse),
		}
	}

	if msg2.Error != nil && strings.TrimSpace(msg2.Error.Message) != "" {
		return toolCallEval{
			Success: false,
			Error:   fmt.Sprintf("round2 api error: %s", msg2.Error.Message),
			Snippet: summarizeToolUse(toolUse),
		}
	}
	if msg2.StopReason != "end_turn" {
		return toolCallEval{
			Success: true,
			Pass:    false,
			Reason:  fmt.Sprintf("round2 stop_reason 期望 end_turn，实际 %q", msg2.StopReason),
			Snippet: summarizeToolCallRoundtrip(toolUse, anthropicText(msg2)),
		}
	}

	finalText := anthropicText(msg2)
	if finalText == "" {
		return toolCallEval{
			Success: true,
			Pass:    false,
			Reason:  "round2 未返回 text 内容",
			Snippet: summarizeToolUse(toolUse),
		}
	}

	lower := strings.ToLower(finalText)
	if !containsAny(lower, "北京", "天气", "25", "晴") {
		return toolCallEval{
			Success: true,
			Pass:    false,
			Reason:  "round2 回复未体现天气结果",
			Snippet: summarizeToolCallRoundtrip(toolUse, finalText),
		}
	}

	return toolCallEval{
		Success: true,
		Pass:    true,
		Reason:  "round1 返回 tool_use，round2 基于 tool_result 生成最终回复",
		Snippet: summarizeToolCallRoundtrip(toolUse, finalText),
	}
}

func extractToolUse(msg anthropicMessageResponse) (anthropicContentBlock, error) {
	for _, block := range msg.Content {
		if block.Type == "tool_use" {
			if strings.TrimSpace(block.ID) == "" {
				return anthropicContentBlock{}, fmt.Errorf("round1 tool_use 缺少 id")
			}
			return block, nil
		}
	}
	return anthropicContentBlock{}, fmt.Errorf("round1 未返回 tool_use")
}

func anthropicText(msg anthropicMessageResponse) string {
	var b strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

// cityMatchesProbeTarget accepts common city spellings for the Beijing weather probe.
func cityMatchesProbeTarget(city string) bool {
	city = strings.TrimSpace(city)
	if city == "" {
		return false
	}
	if strings.Contains(city, "北京") {
		return true
	}
	switch strings.ToLower(city) {
	case "beijing", "peking":
		return true
	}
	return false
}

func summarizeToolUse(toolUse anthropicContentBlock) string {
	city := strings.TrimSpace(fmt.Sprint(toolUse.Input["city"]))
	return fmt.Sprintf("tool_use %s(id=%s, city=%q)", toolUse.Name, toolUse.ID, city)
}

func summarizeToolCallRoundtrip(toolUse anthropicContentBlock, finalText string) string {
	return fmt.Sprintf("%s => %s", summarizeToolUse(toolUse), oneLine(finalText))
}

func parseAnthropicMessageResponse(raw string) (anthropicMessageResponse, error) {
	var msg anthropicMessageResponse
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return anthropicMessageResponse{}, fmt.Errorf("parse anthropic message: %w", err)
	}
	return msg, nil
}

func formatToolCallReport(round1Req, round2Req string, ex1, ex2 *httpExchange) string {
	var b strings.Builder
	b.WriteString("--- round 1 request ---\n")
	b.WriteString(round1Req)
	b.WriteString("\n\n--- round 1 response ---\n")
	if ex1 != nil {
		b.WriteString(ex1.ResponseBody)
	}
	b.WriteString("\n\n--- round 2 request ---\n")
	b.WriteString(round2Req)
	b.WriteString("\n\n--- round 2 response ---\n")
	if ex2 != nil {
		b.WriteString(ex2.ResponseBody)
	}
	return b.String()
}
