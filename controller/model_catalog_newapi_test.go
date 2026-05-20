package controller

import (
	"testing"
)

func TestRowsFromNewAPIModelsPayload(t *testing.T) {
	body := []byte(`{
		"success": true,
		"data": {
			"1": ["gpt-4o", "gpt-4o-mini"],
			"24": ["gpt-4o", "gemini-2.0-flash"]
		}
	}`)

	rows, err := rowsFromNewAPIModelsPayload(body)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 unique models, got %d", len(rows))
	}
	byID := make(map[string]string)
	for _, r := range rows {
		byID[r.ModelId] = r.Tags
	}
	if byID["gpt-4o"] == "" {
		t.Fatalf("expected gpt-4o to have channel_types tag, row tags=%q", byID["gpt-4o"])
	}
}

func TestNewAPIModelsURL(t *testing.T) {
	if got := newAPIModelsURL(""); got != "https://www.anyfast.ai/api/models" {
		t.Fatalf("default: %s", got)
	}
	if got := newAPIModelsURL("https://mirror.example.com"); got != "https://mirror.example.com/api/models" {
		t.Fatalf("base: %s", got)
	}
}

func TestNewAPIAuthorizationHeader(t *testing.T) {
	if got := newAPIAuthorizationHeader("abc"); got != "abc" {
		t.Fatalf("plain token: %q", got)
	}
	if got := newAPIAuthorizationHeader("Bearer xyz"); got != "Bearer xyz" {
		t.Fatalf("bearer token: %q", got)
	}
}
