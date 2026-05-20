package controller

import (
	"encoding/json"
	"testing"
)

func TestToAnthropicModelInfo_sonnet46(t *testing.T) {
	info := toAnthropicModelInfo(OpenAIModels{Id: "claude-sonnet-4-6", Created: 1626777600})
	if info.ID != "claude-sonnet-4-6" || info.Type != "model" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.MaxInputTokens != 1000000 || info.MaxTokens != 64000 {
		t.Fatalf("tokens: in=%d out=%d", info.MaxInputTokens, info.MaxTokens)
	}
	if !info.Capabilities.Thinking.Supported {
		t.Fatal("expected thinking support")
	}
}

func TestAnthropicModelInfo_jsonShape(t *testing.T) {
	info := toAnthropicModelInfo(OpenAIModels{Id: "claude-haiku-4-5-20251001", Created: 1626777600})
	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "type", "display_name", "capabilities"} {
		if _, ok := probe[key]; !ok {
			t.Fatalf("missing key %s in %s", key, string(raw))
		}
	}
}
