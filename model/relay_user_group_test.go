package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
)

func TestResolveRelayUserGroup(t *testing.T) {
	origDB, origRedis := DB, common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		DB = origDB
		common.RedisEnabled = origRedis
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	DB = db
	user := User{Id: 1, Username: "yongyou", Group: "svip", Role: RoleCommonUser, Status: UserStatusEnabled}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := ResolveRelayUserGroup(user.Id, "default")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "svip" {
		t.Fatalf("token default should follow user group, got %q", got)
	}

	got, err = ResolveRelayUserGroup(user.Id, "vip")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "vip" {
		t.Fatalf("explicit token group should win, got %q", got)
	}
}
