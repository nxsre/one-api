package main

import (
	"testing"
)

func TestEvaluateToolCallRoundtripPass(t *testing.T) {
	toolUse := anthropicContentBlock{
		Type:  "tool_use",
		ID:    "toolu_test",
		Name:  toolCallWeatherName,
		Input: map[string]any{"city": "北京"},
	}
	msg1 := anthropicMessageResponse{StopReason: "tool_use", Content: []anthropicContentBlock{toolUse}}
	msg2 := anthropicMessageResponse{
		StopReason: "end_turn",
		Content:    []anthropicContentBlock{{Type: "text", Text: "北京目前天气晴朗，气温 25°C，湿度 60%。"}},
	}

	got := evaluateToolCallRoundtrip(msg1, msg2, toolUse)
	if !got.Success || !got.Pass {
		t.Fatalf("evaluateToolCallRoundtrip() = %+v, want pass", got)
	}
}

func TestEvaluateToolCallRoundtripMissingToolUse(t *testing.T) {
	msg1 := anthropicMessageResponse{
		StopReason: "end_turn",
		Content:    []anthropicContentBlock{{Type: "text", Text: "hello"}},
	}
	toolUse := anthropicContentBlock{Name: toolCallWeatherName, Input: map[string]any{"city": "北京"}}

	got := evaluateToolCallRoundtrip(msg1, anthropicMessageResponse{}, toolUse)
	if got.Pass {
		t.Fatalf("evaluateToolCallRoundtrip() = %+v, want fail", got)
	}
}

func TestBuildToolCallRound2Body(t *testing.T) {
	assistant := []anthropicContentBlock{{
		Type:  "tool_use",
		ID:    "toolu_123",
		Name:  toolCallWeatherName,
		Input: map[string]any{"city": "北京"},
	}}
	body := buildToolCallRound2Body("claude-sonnet-4-5-20250929", "toolu_123", assistant)
	if body["stream"] != false {
		t.Fatalf("stream = %v, want false", body["stream"])
	}
	tools, ok := body["tools"].([]map[string]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v", body["tools"])
	}
}
