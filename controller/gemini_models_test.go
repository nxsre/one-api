package controller

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToGeminiModel_googleShape(t *testing.T) {
	gm := toGeminiModel(OpenAIModels{Id: "gemini-2.0-flash", OwnedBy: "google · Gemini 2.0 Flash"})
	if gm.Name != "models/gemini-2.0-flash" {
		t.Fatalf("name: %q", gm.Name)
	}
	if gm.DisplayName != "Gemini 2.0 Flash" {
		t.Fatalf("displayName: %q", gm.DisplayName)
	}
	if len(gm.SupportedGenerationMethods) == 0 {
		t.Fatal("expected supportedGenerationMethods")
	}
	raw, err := json.Marshal(gm)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"name":"models/gemini-2.0-flash"`) {
		t.Fatalf("unexpected json: %s", raw)
	}
}

func TestListGeminiModels_responseEnvelope(t *testing.T) {
	payload := ginEnvelopeForTest()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	models, ok := probe["models"].([]any)
	if !ok || models == nil {
		t.Fatalf("models must be array, got %T", probe["models"])
	}
}

func ginEnvelopeForTest() map[string]any {
	return map[string]any{
		"models": []GeminiModel{
			toGeminiModel(OpenAIModels{Id: "gemini-2.0-flash"}),
		},
		"nextPageToken": "",
	}
}
