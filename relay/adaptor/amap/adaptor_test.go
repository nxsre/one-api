package amap

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/songquanpeng/one-api/relay/model"
)

func TestBuildPOIRequestFromJSONMessage(t *testing.T) {
	req := &model.GeneralOpenAIRequest{
		Messages: []model.Message{{
			Role:    "user",
			Content: `{"location":"116.473168,39.993015","keywords":"咖啡","radius":1000,"page_size":5}`,
		}},
	}

	params, err := buildPOIRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if params.Location != "116.473168,39.993015" {
		t.Fatalf("unexpected location: %s", params.Location)
	}
	if params.Keywords != "咖啡" {
		t.Fatalf("unexpected keywords: %s", params.Keywords)
	}
	if params.Radius != "1000" {
		t.Fatalf("unexpected radius: %s", params.Radius)
	}
	if params.PageSize != "5" {
		t.Fatalf("unexpected page_size: %s", params.PageSize)
	}
}

func TestBuildPOIRequestFromMetadataAndText(t *testing.T) {
	req := &model.GeneralOpenAIRequest{
		Metadata: map[string]any{
			"amap_poi": map[string]any{
				"location": "116.473168,39.993015",
				"radius":   float64(2000),
			},
		},
		Messages: []model.Message{{
			Role:    "user",
			Content: "肯德基",
		}},
	}

	params, err := buildPOIRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if params.Location != "116.473168,39.993015" {
		t.Fatalf("unexpected location: %s", params.Location)
	}
	if params.Keywords != "肯德基" {
		t.Fatalf("unexpected keywords: %s", params.Keywords)
	}
	if params.Radius != "2000" {
		t.Fatalf("unexpected radius: %s", params.Radius)
	}
}

func TestBuildPOIRequestFromMetadataAmapKey(t *testing.T) {
	req := &model.GeneralOpenAIRequest{
		Metadata: map[string]any{
			"amap": map[string]any{
				"location": "116.473168,39.993015",
				"radius":   float64(2000),
			},
		},
		Messages: []model.Message{{
			Role:    "user",
			Content: "肯德基",
		}},
	}

	params, err := buildPOIRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if params.Location != "116.473168,39.993015" {
		t.Fatalf("unexpected location: %s", params.Location)
	}
	if params.Keywords != "肯德基" {
		t.Fatalf("unexpected keywords: %s", params.Keywords)
	}
}

func TestBuildPOIRequestRequiresLocation(t *testing.T) {
	req := &model.GeneralOpenAIRequest{
		Messages: []model.Message{{
			Role:    "user",
			Content: "咖啡",
		}},
	}

	if _, err := buildPOIRequest(req); err == nil {
		t.Fatal("expected missing location error")
	}
}

func TestNewAmapRequest(t *testing.T) {
	req, err := newAmapRequest(defaultBaseURL+"/v5/place/around", "test-key", poiRequest{
		Location: "116.473168,39.993015",
		Keywords: "咖啡",
	})
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, values, "key", "test-key")
	assertQuery(t, values, "location", "116.473168,39.993015")
	assertQuery(t, values, "keywords", "咖啡")
	assertQuery(t, values, "radius", "5000")
	assertQuery(t, values, "page_size", "10")
}

func TestPOIItemsUnmarshalWrapped(t *testing.T) {
	var resp amapResponse
	if err := json.Unmarshal([]byte(`{"status":"1","pois":{"poi":[{"name":"A","id":"B"}]}}`), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.POIs) != 1 || resp.POIs[0].Name != "A" {
		t.Fatalf("unexpected pois: %+v", resp.POIs)
	}
}

func assertQuery(t *testing.T, values url.Values, key, want string) {
	t.Helper()
	if got := values.Get(key); got != want {
		t.Fatalf("query %s = %q, want %q", key, got, want)
	}
}
