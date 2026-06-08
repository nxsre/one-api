package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCatalogTestDB(t *testing.T) {
	t.Helper()
	// 独立命名的内存 DSN：glebarez/modernc 的裸 ":memory:" 在同包多次 Open 时会共享同一库，
	// 仅迁移 ModelCatalog 会污染其它测试共享的库（如缺 users 表）。用专属名隔离。
	db, err := gorm.Open(sqlite.Open("file:catalog_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ModelCatalog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(&ModelCatalog{})
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	origDB := DB
	DB = db
	t.Cleanup(func() { DB = origDB })
}

func countCurrent(t *testing.T, modelID string) int {
	t.Helper()
	var n int64
	if err := DB.Model(&ModelCatalog{}).
		Where("model_id = ? AND status = ?", modelID, "current").
		Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return int(n)
}

// TestSyncKeepsRowPerProvider 验证行身份为 (source, provider_key, model_id)：
// 同一 model_id 在不同 provider 下各保留一条 current，互不覆盖。
func TestSyncKeepsRowPerProvider(t *testing.T) {
	setupCatalogTestDB(t)

	const src = "sync_models_dev"
	// Enabled 必须显式置 true：列有 gorm default:true，留零值会被 DB 默认改写为 true，
	// 导致 existing.Enabled(true) 与 entry.Enabled(false) 每次都判定为变更。真实同步同样置 true。
	entries := []ModelCatalog{
		{Source: src, ModelId: "gpt-4o", ProviderKey: "openai", ProviderDisplay: "OpenAI", Enabled: true},
		{Source: src, ModelId: "gpt-4o", ProviderKey: "azure", ProviderDisplay: "Azure", Enabled: true},
		{Source: src, ModelId: "gpt-4o", ProviderKey: "openrouter", ProviderDisplay: "OpenRouter", Enabled: true},
	}
	if _, _, err := BatchUpsertModelCatalogForSync(entries); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := countCurrent(t, "gpt-4o"); got != 3 {
		t.Fatalf("expected 3 current rows (one per provider), got %d", got)
	}

	// 再次同步同样数据：内容未变，不应新增版本，仍是 3 条 current。
	if _, _, err := BatchUpsertModelCatalogForSync(entries); err != nil {
		t.Fatalf("resync: %v", err)
	}
	if got := countCurrent(t, "gpt-4o"); got != 3 {
		t.Fatalf("after idempotent resync expected 3 current rows, got %d", got)
	}

	// 仅改动其中一个 provider 的内容：只该 provider 版本化，其余不受影响。
	entries[0].ContextLimit = 128000
	if _, _, err := BatchUpsertModelCatalogForSync(entries); err != nil {
		t.Fatalf("resync changed: %v", err)
	}
	if got := countCurrent(t, "gpt-4o"); got != 3 {
		t.Fatalf("after partial change expected 3 current rows, got %d", got)
	}
	var openaiCur ModelCatalog
	if err := DB.Where("source = ? AND provider_key = ? AND model_id = ? AND status = ?",
		src, "openai", "gpt-4o", "current").First(&openaiCur).Error; err != nil {
		t.Fatalf("load openai current: %v", err)
	}
	if openaiCur.Version != 2 {
		t.Fatalf("openai row should bump to version 2, got %d", openaiCur.Version)
	}
	if openaiCur.ContextLimit != 128000 {
		t.Fatalf("openai current should carry new context limit, got %d", openaiCur.ContextLimit)
	}
	// azure 行不应被 openai 的变更牵连，仍是 version 1。
	var azureCur ModelCatalog
	if err := DB.Where("source = ? AND provider_key = ? AND model_id = ? AND status = ?",
		src, "azure", "gpt-4o", "current").First(&azureCur).Error; err != nil {
		t.Fatalf("load azure current: %v", err)
	}
	if azureCur.Version != 1 {
		t.Fatalf("azure row should stay version 1, got %d", azureCur.Version)
	}
}
