package main

import "testing"

func TestModelSupportsTemperature(t *testing.T) {
	cases := map[string]bool{
		"":                              true, // 未知模型保守发送
		"claude-opus-4-6":               true,
		"claude-opus-4-5":               true,
		"claude-sonnet-4-6":             true,
		"claude-haiku-4-5":              true,
		"claude-opus-4-7":               false,
		"claude-opus-4-8":               false,
		"anthropic.claude-opus-4-8":     false,
		"claude-opus-4-8-20251101":      false,
		"CLAUDE-OPUS-4-7":               false,
		"claude-fable-5":                false,
		"claude-opus-5-0":               false,
	}
	for model, want := range cases {
		if got := modelSupportsTemperature(model); got != want {
			t.Errorf("modelSupportsTemperature(%q) = %v, want %v", model, got, want)
		}
	}
}

func TestBuildRequestBodyDropsTemperatureForUnsupportedModel(t *testing.T) {
	def := probeDef{MaxTokens: defaultMaxTokens} // OmitTemperature 默认为 false

	if body := buildRequestBody("claude-opus-4-6", "hi", def); body["temperature"] == nil {
		t.Error("expected temperature for claude-opus-4-6")
	}
	if body := buildRequestBody("claude-opus-4-8", "hi", def); body["temperature"] != nil {
		t.Error("did not expect temperature for claude-opus-4-8")
	}
}
