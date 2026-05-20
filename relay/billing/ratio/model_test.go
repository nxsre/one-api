package ratio

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
)

// withTempModelRatio 在测试中临时注入额外的 ModelRatio 条目，结束后恢复。
func withTempModelRatio(t *testing.T, entries map[string]float64) {
	t.Helper()
	modelRatioLock.Lock()
	restore := make(map[string]float64, len(entries))
	for k, v := range entries {
		if old, ok := ModelRatio[k]; ok {
			restore[k] = old
		}
		ModelRatio[k] = v
	}
	modelRatioLock.Unlock()
	t.Cleanup(func() {
		modelRatioLock.Lock()
		defer modelRatioLock.Unlock()
		for k := range entries {
			if v, ok := restore[k]; ok {
				ModelRatio[k] = v
			} else {
				delete(ModelRatio, k)
			}
		}
	})
}

// withTempCompletionRatio 同上，针对 CompletionRatio。
func withTempCompletionRatio(t *testing.T, entries map[string]float64) {
	t.Helper()
	restore := make(map[string]float64, len(entries))
	for k, v := range entries {
		if old, ok := CompletionRatio[k]; ok {
			restore[k] = old
		}
		CompletionRatio[k] = v
	}
	t.Cleanup(func() {
		for k := range entries {
			if v, ok := restore[k]; ok {
				CompletionRatio[k] = v
			} else {
				delete(CompletionRatio, k)
			}
		}
	})
}

func TestGetModelRatio_ChannelSlugLookup(t *testing.T) {
	const model = "unit-test-model-x"
	withTempModelRatio(t, map[string]float64{
		model + "@aws-claude": 11,
		model:                 33,
	})

	if got := GetModelRatio("", model, channeltype.AwsClaude); got != 11 {
		t.Fatalf("应命中 @slug 写法，期望 11，实际 %v", got)
	}
}

func TestMigrateLegacyParenRatioKeys(t *testing.T) {
	m := map[string]float64{
		"gpt-4(1)":        10,
		"claude-3(24)":    20,
		"already@openai":  3,
		"plain-model":     5,
		"unknown-channel(99999)": 7,
	}
	migrateLegacyParenRatioKeys(m)
	if _, ok := m["gpt-4(1)"]; ok {
		t.Fatal("旧键 gpt-4(1) 应被移除")
	}
	if got := m["gpt-4@openai"]; got != 10 {
		t.Fatalf("gpt-4(1) 应迁移为 gpt-4@openai=10，实际 %v", got)
	}
	if got := m["claude-3@aws-claude"]; got != 20 {
		t.Fatalf("claude-3(24) 应迁移为 claude-3@aws-claude=20，实际 %v", got)
	}
	if got := m["plain-model"]; got != 5 {
		t.Fatalf("裸名应保留，实际 %v", got)
	}
	if _, ok := m["unknown-channel(99999)"]; ok {
		t.Fatal("无法映射 slug 的旧键应被删除")
	}
}

func TestGetModelRatio_FallsBackToBareName(t *testing.T) {
	const model = "unit-test-model-bare"
	withTempModelRatio(t, map[string]float64{
		model: 5,
	})

	if got := GetModelRatio("", model, channeltype.Groq); got != 5 {
		t.Fatalf("应回退到裸名，期望 5，实际 %v", got)
	}
}

func TestGetCompletionRatio_ChannelSlugLookup(t *testing.T) {
	const model = "unit-test-completion-x"
	withTempCompletionRatio(t, map[string]float64{
		model + "@aws-claude": 2.5,
	})

	if got := GetCompletionRatio("", model, channeltype.AwsClaude); got != 2.5 {
		t.Fatalf("CompletionRatio 应命中 @slug 写法，期望 2.5，实际 %v", got)
	}
}

func TestUpdateModelRatioByJSONString_MigratesLegacyKeys(t *testing.T) {
	modelRatioLock.Lock()
	backup := ModelRatio
	modelRatioLock.Unlock()
	t.Cleanup(func() {
		modelRatioLock.Lock()
		ModelRatio = backup
		modelRatioLock.Unlock()
	})

	const payload = `{"legacy-model(24)": 4.5, "bare": 1}`
	if err := UpdateModelRatioByJSONString(payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := ModelRatio["legacy-model(24)"]; ok {
		t.Fatal("加载后不应保留 (id) 键")
	}
	if got := ModelRatio["legacy-model@aws-claude"]; got != 4.5 {
		t.Fatalf("加载时应迁移为 @slug，期望 4.5，实际 %v", got)
	}
}

func TestGetModelRatio_StripsUpstreamAliasWhenOnlyMapped(t *testing.T) {
	const official = "claude-sonnet-4-6"
	withTempModelRatio(t, map[string]float64{
		official: 2.5,
	})
	if got := GetModelRatio("", "pa/"+official, channeltype.Anthropic); got != 2.5 {
		t.Fatalf("应剥掉 pa/ 后命中官方名倍率，期望 2.5，实际 %v", got)
	}
}
