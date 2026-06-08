package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
)

// TestPreConsumeTokenQuotaDecrements 验证 PreConsumeTokenQuota 在 Redis 关闭时
// （CacheGetUserQuota 回退到 DB）仍正确扣减 token 与 user 额度，且额度不足时报错。
// 守护把 GetUserQuota 换成 CacheGetUserQuota 的改动不破坏配额正确性。
func TestPreConsumeTokenQuotaDecrements(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Token{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB, origRedis := DB, common.RedisEnabled
	DB, common.RedisEnabled = db, false // Redis 关闭 → CacheGetUserQuota 回退 DB
	t.Cleanup(func() { DB, common.RedisEnabled = origDB, origRedis })

	db.Create(&User{Id: 1, Username: "u1", AccessToken: "at1", AffCode: "af1", Quota: 1000})
	db.Create(&Token{Id: 10, UserId: 1, Key: "k10", Name: "t10", RemainQuota: 500})

	if err := PreConsumeTokenQuota(10, 200); err != nil {
		t.Fatalf("pre-consume 200 should succeed: %v", err)
	}
	var tok Token
	var usr User
	db.First(&tok, 10)
	db.First(&usr, 1)
	if tok.RemainQuota != 300 {
		t.Errorf("token remain_quota: want 300, got %d", tok.RemainQuota)
	}
	if usr.Quota != 800 {
		t.Errorf("user quota: want 800, got %d", usr.Quota)
	}

	// token 额度不足应报错且不扣减
	if err := PreConsumeTokenQuota(10, 400); err == nil {
		t.Error("expected error when token quota insufficient")
	}
	db.First(&tok, 10)
	if tok.RemainQuota != 300 {
		t.Errorf("token remain_quota must be unchanged after failed pre-consume, got %d", tok.RemainQuota)
	}
}
