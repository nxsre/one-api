package controller

import (
	"testing"
)

func TestRowsFromAnyfastPricingPayload(t *testing.T) {
	body := []byte(`{
		"success": true,
		"vendors": [{"id": 1, "name": "OpenAI"}],
		"data": [
			{
				"model_name": "gpt-test",
				"vendor_id": 1,
				"quota_type": 1,
				"model_price": 1.5,
				"completion_ratio": 2,
				"billing_config": {
					"usePrice": true,
					"rules": [{"type": "tier", "inputPrice": 1.5, "outputPrice": 3}]
				}
			},
			{
				"model_name": "ratio-model",
				"quota_type": 0,
				"model_ratio": 2,
				"completion_ratio": 3
			}
		]
	}`)

	rows, err := rowsFromAnyfastPricingPayload(body)
	if err != nil {
		t.Fatalf("rowsFromAnyfastPricingPayload failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].ModelId != "gpt-test" || rows[0].CostInput != 1.5 || rows[0].CostOutput != 3 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
	if rows[0].ProviderDisplay != "OpenAI" {
		t.Fatalf("expected vendor OpenAI, got %q", rows[0].ProviderDisplay)
	}
	if rows[1].ModelId != "ratio-model" || rows[1].Notes == "" {
		t.Fatalf("unexpected ratio row: %+v", rows[1])
	}
}

func TestAnyfastPricingURL(t *testing.T) {
	if got := anyfastPricingURL(""); got != "https://www.anyfast.ai/api/pricing" {
		t.Fatalf("default url: %s", got)
	}
	if got := anyfastPricingURL("https://mirror.example.com/api/pricing"); got != "https://mirror.example.com/api/pricing" {
		t.Fatalf("full pricing url: %s", got)
	}
	if got := anyfastPricingURL("https://mirror.example.com"); got != "https://mirror.example.com/api/pricing" {
		t.Fatalf("base url: %s", got)
	}
}
