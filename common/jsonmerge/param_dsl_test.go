package jsonmerge

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyParamOverrideDeepMergeNoOperations(t *testing.T) {
	base := []byte(`{"a":1,"b":{"c":2}}`)
	patch := map[string]interface{}{"b": map[string]interface{}{"d": 3}}
	out, err := ApplyParamOverride(base, patch, nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	_ = json.Unmarshal(out, &m)
	b := m["b"].(map[string]interface{})
	if int(b["c"].(float64)) != 2 || int(b["d"].(float64)) != 3 {
		t.Fatalf("unexpected merge: %s", string(out))
	}
}

func TestOperationsSetAndDelete(t *testing.T) {
	base := []byte(`{"model":"x","temperature":0.5}`)
	patch := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{"mode": "set", "path": "model", "value": "gpt-4"},
			map[string]interface{}{"mode": "delete", "path": "temperature"},
		},
	}
	out, err := ApplyParamOverride(base, patch, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"model":"gpt-4"`) {
		t.Fatal(string(out))
	}
	if strings.Contains(string(out), "temperature") {
		t.Fatal(string(out))
	}
}

func TestOperationsWithConditionsAndReturnError(t *testing.T) {
	base := []byte(`{"x":1}`)
	patch := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode": "return_error",
				"path": "",
				"value": map[string]interface{}{
					"message":     "blocked",
					"status_code": 403,
					"code":        "forbidden",
				},
				"conditions": []interface{}{
					map[string]interface{}{"path": "x", "mode": "full", "value": float64(1)},
				},
				"logic": "AND",
			},
		},
	}
	_, err := ApplyParamOverride(base, patch, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	re, ok := AsReturnError(err)
	if !ok || re.StatusCode != 403 || re.Message != "blocked" {
		t.Fatalf("got %v %v", err, ok)
	}
}

func TestLegacyKeysShallowWithOperations(t *testing.T) {
	base := []byte(`{"a":1}`)
	patch := map[string]interface{}{
		"b": float64(2),
		"operations": []interface{}{
			map[string]interface{}{"mode": "set", "path": "c", "value": float64(3)},
		},
	}
	out, err := ApplyParamOverride(base, patch, nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	_ = json.Unmarshal(out, &m)
	if m["a"] == nil || m["b"] == nil || m["c"] == nil {
		t.Fatalf("%s", string(out))
	}
}

func TestSetHeaderInContext(t *testing.T) {
	ctx := BuildParamOverrideContext("/v1/chat", map[string]string{"Authorization": "Bearer client"}, map[string]string{"X-Old": "1"}, "m1", "m2")
	base := []byte(`{}`)
	patch := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{"mode": "set_header", "path": "X-Test", "value": "yes"},
		},
	}
	_, err := ApplyParamOverride(base, patch, ctx)
	if err != nil {
		t.Fatal(err)
	}
	h := RuntimeHeaderOverrideFromContext(ctx)
	if h["x-test"] != "yes" {
		t.Fatalf("%v", h)
	}
}
