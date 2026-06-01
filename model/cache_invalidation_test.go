package model

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
)

// TestPublishChannelsChangedReloadsCache 验证完整失效链路：
// 发布渠道失效事件 → Redis → 订阅者 → debounce → InitChannelCache 重建渠道缓存。
func TestPublishChannelsChangedReloadsCache(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not available, skipping: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	origRDB, origEnabled, origDB := common.RDB, common.RedisEnabled, DB
	common.RDB, common.RedisEnabled = rdb, true
	t.Cleanup(func() { common.RDB, common.RedisEnabled, DB = origRDB, origEnabled, origDB })

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Channel{}, &Ability{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	DB = db
	db.Create(&Channel{Id: 1, Status: ChannelStatusEnabled, Group: "default", Models: "gpt-4"})
	db.Create(&Ability{Group: "default", Model: "gpt-4", ChannelId: 1, Enabled: true})

	// 起点：清空渠道缓存。
	channelSyncLock.Lock()
	group2model2channels = nil
	channelSyncLock.Unlock()

	StartCacheInvalidationSubscriber()
	time.Sleep(100 * time.Millisecond) // 等订阅就绪

	PublishChannelsChanged()

	// 轮询等待订阅者重建缓存（debounce 300ms + 网络往返）。
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		channelSyncLock.RLock()
		got := len(group2model2channels)
		channelSyncLock.RUnlock()
		if got > 0 {
			return // 成功：缓存已被订阅者重建
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("channel cache was not reloaded after PublishChannelsChanged within 3s")
}
