package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common/config"
)

// TestAsyncLogQueueFlushes 验证异步队列把日志批量落库，并在 FlushLogQueue 后全部写入。
func TestAsyncLogQueueFlushes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origLogDB := LOG_DB
	LOG_DB = db
	t.Cleanup(func() { LOG_DB = origLogDB })

	config.LogAsyncEnabled = true
	config.LogAsyncBufferSize = 1000
	config.LogAsyncBatchSize = 50
	config.LogAsyncFlushIntervalMs = 50
	config.LogShardByDay = false

	InitLogConsumer()

	const n = 237
	for i := range n {
		if !enqueueLog(&Log{UserId: 1, Type: LogTypeConsume, CreatedAt: time.Now().Unix(), Content: "t"}) {
			t.Fatalf("enqueue %d returned false (queue should not be full)", i)
		}
	}

	FlushLogQueue()

	var count int64
	if err := db.Model(&Log{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != n {
		t.Fatalf("expected %d logs persisted, got %d", n, count)
	}
}
