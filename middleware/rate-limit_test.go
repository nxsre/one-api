package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
)

// TestRedisRateLimiterAtomic 验证基于 Lua 滑动窗口的限流在高并发突发下是原子精确的：
// 100 个并发请求、限额 10/窗口，恰好放行 10 个、其余 429。
// 这正是原先 LLen→LPush 多命令实现做不到的（竞态会放行超过 10 个）。
func TestRedisRateLimiterAtomic(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available, skipping: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	origRDB, origEnabled := common.RDB, common.RedisEnabled
	common.RDB, common.RedisEnabled = rdb, true
	t.Cleanup(func() { common.RDB, common.RedisEnabled = origRDB, origEnabled })
	config.RateLimitKeyExpirationDuration = 20 * time.Minute

	const clientIP = "203.0.113.77"
	const mark = "TESTATOMIC"
	// 清理可能残留的 key，保证测试独立。
	_ = rdb.Del(ctx, "rateLimit:"+mark+clientIP).Err()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	const maxReq = 10
	router.GET("/", rateLimitFactory(maxReq, 60, mark), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	const total = 100
	var ok200, got429 int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range total {
		wg.Go(func() {
			<-start // 尽量让所有请求同时发出，制造突发
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = clientIP + ":12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			switch w.Code {
			case http.StatusOK:
				atomic.AddInt64(&ok200, 1)
			case http.StatusTooManyRequests:
				atomic.AddInt64(&got429, 1)
			}
		})
	}
	close(start)
	wg.Wait()

	if ok200 != maxReq {
		t.Fatalf("expected exactly %d allowed, got %d (429=%d)", maxReq, ok200, got429)
	}
	if got429 != total-maxReq {
		t.Fatalf("expected %d rejected, got %d", total-maxReq, got429)
	}
}
