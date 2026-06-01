package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestBatchUpdateCombinesPerKey 验证批量 flush 用单条 CASE WHEN 语句把多个 id 的
// 增量正确落库：累加正确、不同 id 各取其值、token 双列（remain +/used -）同时更新。
func TestBatchUpdateCombinesPerKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Token{}, &Channel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := DB
	DB = db
	t.Cleanup(func() { DB = origDB })

	// 初始数据
	db.Create(&User{Id: 1, Username: "u1", AccessToken: "at1", AffCode: "af1", Quota: 1000, UsedQuota: 0, RequestCount: 0})
	db.Create(&User{Id: 2, Username: "u2", AccessToken: "at2", AffCode: "af2", Quota: 2000, UsedQuota: 0, RequestCount: 0})
	db.Create(&Token{Id: 10, UserId: 1, Key: "k10", Name: "t10", RemainQuota: 500, UsedQuota: 100})
	db.Create(&Token{Id: 11, UserId: 2, Key: "k11", Name: "t11", RemainQuota: 800, UsedQuota: 50})
	db.Create(&Channel{Id: 100, UsedQuota: 0})

	// 累积增量：同一 id 多次 addNewRecord 应累加。
	addNewRecord(BatchUpdateTypeUserQuota, 1, -30)
	addNewRecord(BatchUpdateTypeUserQuota, 1, -20) // user1 quota: -50
	addNewRecord(BatchUpdateTypeUserQuota, 2, -100)

	addNewRecord(BatchUpdateTypeUsedQuota, 1, 50)
	addNewRecord(BatchUpdateTypeRequestCount, 1, 3)

	addNewRecord(BatchUpdateTypeTokenQuota, 10, -40) // remain -40, used +40
	addNewRecord(BatchUpdateTypeTokenQuota, 11, 25)  // remain +25, used -25

	addNewRecord(BatchUpdateTypeChannelUsedQuota, 100, 777)

	batchUpdate()

	var u1, u2 User
	db.First(&u1, 1)
	db.First(&u2, 2)
	if u1.Quota != 950 {
		t.Errorf("user1 quota: want 950, got %d", u1.Quota)
	}
	if u2.Quota != 1900 {
		t.Errorf("user2 quota: want 1900, got %d", u2.Quota)
	}
	if u1.UsedQuota != 50 {
		t.Errorf("user1 used_quota: want 50, got %d", u1.UsedQuota)
	}
	if u1.RequestCount != 3 {
		t.Errorf("user1 request_count: want 3, got %d", u1.RequestCount)
	}

	var t10, t11 Token
	db.First(&t10, 10)
	db.First(&t11, 11)
	if t10.RemainQuota != 460 || t10.UsedQuota != 140 {
		t.Errorf("token10: want remain=460 used=140, got remain=%d used=%d", t10.RemainQuota, t10.UsedQuota)
	}
	if t11.RemainQuota != 825 || t11.UsedQuota != 25 {
		t.Errorf("token11: want remain=825 used=25, got remain=%d used=%d", t11.RemainQuota, t11.UsedQuota)
	}

	var ch Channel
	db.First(&ch, 100)
	if ch.UsedQuota != 777 {
		t.Errorf("channel100 used_quota: want 777, got %d", ch.UsedQuota)
	}
}
