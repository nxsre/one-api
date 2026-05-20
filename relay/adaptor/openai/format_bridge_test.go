package openai

import (
	"encoding/json"
	"testing"
)

func TestNormalizeContentPartsImageURL(t *testing.T) {
	parts := normalizeContentParts([]interface{}{
		map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": "https://example.com/a.png",
			},
		},
	})
	if len(parts) != 1 {
		t.Fatalf("len=%d", len(parts))
	}
	m, ok := parts[0].(map[string]interface{})
	if !ok || m["type"] != "input_image" {
		t.Fatalf("got=%v", parts[0])
	}
}

func TestTryNormalizeRealtimeSessionRequestModalities(t *testing.T) {
	body := []byte(`{"model":"gpt-4o-realtime-preview"}`)
	out, ok := TryNormalizeRealtimeSessionRequest(body)
	if !ok {
		t.Fatal("expected change")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	mod, ok := m["modalities"].([]interface{})
	if !ok || len(mod) != 2 {
		t.Fatalf("modalities=%v", m["modalities"])
	}
}
