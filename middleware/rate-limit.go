package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

var inMemoryRateLimiter common.InMemoryRateLimiter

// rateLimitSeq 仅用于保证同一纳秒内多个请求写入有序集合时 member 唯一。
var rateLimitSeq uint64

// rateLimitScript 在 Redis 端原子地执行「滑动窗口日志」限流：
//   - 先按时间戳裁掉窗口外的旧记录（ZREMRANGEBYSCORE）
//   - 统计窗口内请求数（ZCARD）
//   - 未超限则写入本次请求（ZADD）并返回 1，超限返回 0
//
// 整个判断-写入在单条 Lua 脚本内完成，避免了原先 LLen→LPush 多命令之间的竞态，
// 因此在多实例水平扩展下限流是全局精确的，而不是各实例各算各的。
var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local maxReq = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local member = ARGV[5]
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count < maxReq then
    redis.call('ZADD', key, now, member)
    redis.call('EXPIRE', key, ttl)
    return 1
end
redis.call('EXPIRE', key, ttl)
return 0
`)

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	ctx := context.Background()
	key := "rateLimit:" + mark + c.ClientIP()
	now := time.Now().UnixNano()
	window := duration * int64(time.Second)
	ttl := int64(config.RateLimitKeyExpirationDuration.Seconds())
	member := strconv.FormatInt(now, 10) + "-" + strconv.FormatUint(atomic.AddUint64(&rateLimitSeq, 1), 10)

	allowed, err := rateLimitScript.Run(ctx, common.RDB, []string{key},
		now, window, maxRequestNum, ttl, member).Int()
	if err != nil {
		// 故障开放（fail-open）：限流器是保护性措施而非正确性关口，Redis 异常时
		// 放行请求并记录日志，而不是用 500 拖垮全部流量。
		logger.SysError("redis rate limiter error, allowing request: " + err.Error())
		return
	}
	if allowed == 0 {
		c.Status(http.StatusTooManyRequests)
		c.Abort()
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	key := mark + c.ClientIP()
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		c.Status(http.StatusTooManyRequests)
		c.Abort()
		return
	}
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if maxRequestNum == 0 || config.DebugEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	} else {
		// It's safe to call multi times.
		inMemoryRateLimiter.Init(config.RateLimitKeyExpirationDuration)
		return func(c *gin.Context) {
			memoryRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
}

func GlobalWebRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.GlobalWebRateLimitNum, config.GlobalWebRateLimitDuration, "GW")
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.GlobalApiRateLimitNum, config.GlobalApiRateLimitDuration, "GA")
}

func CriticalRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.CriticalRateLimitNum, config.CriticalRateLimitDuration, "CT")
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.DownloadRateLimitNum, config.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.UploadRateLimitNum, config.UploadRateLimitDuration, "UP")
}
