package billing_setting

import (
	"testing"

	"github.com/songquanpeng/one-api/common/config"
)

func TestMergePatchIntoBlockPayload(t *testing.T) {
	// 依赖 config.OptionMap，仅测 JSON 合并逻辑需先写入内存
	const key = "ModelRatio"
	old := `{"gpt-4":1}`
	config.OptionMapRWMutex.Lock()
	if config.OptionMap == nil {
		config.OptionMap = make(map[string]string)
	}
	config.OptionMap[key] = old
	config.OptionMapRWMutex.Unlock()

	payload, err := MergePatchIntoBlockPayload("model_ratio", map[string]any{
		"gpt-4":  2.0,
		"claude": 3.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" {
		t.Fatal("empty payload")
	}
	if payload == old {
		t.Fatalf("expected merged payload, got %s", payload)
	}
}

func TestRecordSavedOptionAsVersion(t *testing.T) {
	RegisterPricingVersionStoreSaver = func(key, value string) error {
		config.OptionMapRWMutex.Lock()
		if config.OptionMap == nil {
			config.OptionMap = make(map[string]string)
		}
		config.OptionMap[key] = value
		config.OptionMapRWMutex.Unlock()
		return nil
	}
	t.Cleanup(func() {
		RegisterPricingVersionStoreSaver = nil
	})

	payload := `{"gpt-4":2}`
	id1, err := RecordSavedOptionAsVersion("ModelRatio", payload)
	if err != nil {
		t.Fatal(err)
	}
	if id1 < 1 {
		t.Fatalf("expected version id >= 1, got %d", id1)
	}
	id2, err := RecordSavedOptionAsVersion("ModelRatio", payload)
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id1 {
		t.Fatalf("duplicate payload should not bump version, got %d vs %d", id2, id1)
	}
	payload2 := `{"gpt-4":3}`
	id3, err := RecordSavedOptionAsVersion("ModelRatio", payload2)
	if err != nil {
		t.Fatal(err)
	}
	if id3 <= id1 {
		t.Fatalf("expected new version after change, got %d", id3)
	}
}
