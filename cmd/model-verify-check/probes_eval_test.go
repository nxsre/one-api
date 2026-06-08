package main

import "testing"

func TestCityMatchesProbeTarget(t *testing.T) {
	tests := []struct {
		city string
		want bool
	}{
		{city: "北京", want: true},
		{city: "Beijing", want: true},
		{city: "beijing", want: true},
		{city: "上海", want: false},
		{city: "", want: false},
	}
	for _, tt := range tests {
		if got := cityMatchesProbeTarget(tt.city); got != tt.want {
			t.Fatalf("cityMatchesProbeTarget(%q) = %v, want %v", tt.city, got, tt.want)
		}
	}
}

func TestEvaluateToolCallRoundtripBeijingAlias(t *testing.T) {
	toolUse := anthropicContentBlock{
		Type:  "tool_use",
		ID:    "toolu_test",
		Name:  toolCallWeatherName,
		Input: map[string]any{"city": "Beijing"},
	}
	msg1 := anthropicMessageResponse{StopReason: "tool_use", Content: []anthropicContentBlock{toolUse}}
	msg2 := anthropicMessageResponse{
		StopReason: "end_turn",
		Content:    []anthropicContentBlock{{Type: "text", Text: "Beijing is sunny, 25°C."}},
	}

	got := evaluateToolCallRoundtrip(msg1, msg2, toolUse)
	if !got.Success || !got.Pass {
		t.Fatalf("evaluateToolCallRoundtrip() = %+v, want pass", got)
	}
}

func TestOAuthModelNameUnavailableReason(t *testing.T) {
	text := "Claude Code is built on Claude, but I don't have a way to verify my exact underlying model name."
	reason, ok := oauthModelNameUnavailableReason(text)
	if !ok {
		t.Fatalf("oauthModelNameUnavailableReason() = ok=%v, want true", ok)
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}
