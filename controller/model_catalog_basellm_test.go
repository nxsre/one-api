package controller

import "testing"

func TestBasellmAPIURL(t *testing.T) {
	if got := basellmAPIURL(""); got != "https://basellm.github.io/llm-metadata/api/all.json" {
		t.Fatalf("default: %s", got)
	}
	if got := basellmAPIURL("https://llm-metadata.pages.dev"); got != "https://llm-metadata.pages.dev/api/all.json" {
		t.Fatalf("mirror root: %s", got)
	}
	if got := basellmAPIURL("https://llm-metadata.pages.dev/api/all.json"); got != "https://llm-metadata.pages.dev/api/all.json" {
		t.Fatalf("full all.json: %s", got)
	}
}

func TestRowsFromBasellmPayload(t *testing.T) {
	body := []byte(`{
		"openai": {
			"id": "openai",
			"name": "OpenAI",
			"api": "https://api.openai.com/v1",
			"doc": "https://platform.openai.com/docs/models",
			"models": {
				"gpt-test": {
					"id": "gpt-test",
					"name": "GPT Test",
					"family": "gpt",
					"reasoning": true,
					"tool_call": true,
					"attachment": false,
					"temperature": true,
					"modalities": { "input": ["text"], "output": ["text"] },
					"limit": { "context": 128000, "output": 4096 },
					"cost": { "input": 2.5, "output": 10, "cache_read": 1.25 }
				}
			}
		}
	}`)
	rows, err := rowsFromProviderTreePayload(body, "basellm", "basellm")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ModelId != "gpt-test" || rows[0].Source != "basellm" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	if rows[0].ProviderKey != "openai" || rows[0].ProviderDisplay != "OpenAI" {
		t.Fatalf("provider fields: %+v", rows[0])
	}
	if rows[0].CostInput != 2.5 || rows[0].Reasoning != true {
		t.Fatalf("cost/reasoning: %+v", rows[0])
	}
}
