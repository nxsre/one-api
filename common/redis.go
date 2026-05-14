package common

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/logger"
)

const DefaultRedisConnString = "redis://127.0.0.1:6379/0"

var RDB redis.Cmdable
var RedisEnabled bool

// RedisConnString 实际连接串（默认本机 Redis）。
var RedisConnString string

func redisDisabled() bool {
	return env.BoolAlways("redis_disabled")
}

// InitRedisClient 默认连接 127.0.0.1:6379；设置 redis_disabled = true 可关闭。
func InitRedisClient() (err error) {
	if redisDisabled() {
		RedisEnabled = false
		RDB = nil
		RedisConnString = ""
		logger.SysLog("Redis is disabled (redis_disabled=true)")
		return nil
	}

	conn := strings.TrimSpace(env.StringAlways("redis_conn_string"))
	if conn == "" {
		conn = DefaultRedisConnString
		logger.SysLog("redis_conn_string not set, using default " + DefaultRedisConnString)
	}
	RedisConnString = conn

	master := strings.TrimSpace(env.StringAlways("redis_master_name"))
	if master != "" {
		logger.SysLog("Redis cluster mode enabled")
		RDB = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:      strings.Split(conn, ","),
			Password:   env.StringAlways("redis_password"),
			MasterName: master,
		})
	} else {
		logger.SysLog("Redis is enabled")
		opt, err := redis.ParseURL(conn)
		if err != nil {
			logger.FatalLog("failed to parse Redis connection string: " + err.Error())
		}
		opt.PoolSize = 10
		RDB = redis.NewClient(opt)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = RDB.Ping(ctx).Result()
	if err != nil {
		logger.FatalLog("Redis ping test failed: " + err.Error())
	}
	RedisEnabled = true
	return nil
}

func ParseRedisOption() *redis.Options {
	opt, err := redis.ParseURL(RedisConnString)
	if err != nil {
		logger.FatalLog("failed to parse Redis connection string: " + err.Error())
	}
	return opt
}

func RedisSet(key string, value string, expiration time.Duration) error {
	if RDB == nil {
		return nil
	}
	ctx := context.Background()
	return RDB.Set(ctx, key, value, expiration).Err()
}

func RedisGet(key string) (string, error) {
	ctx := context.Background()
	return RDB.Get(ctx, key).Result()
}

func RedisDel(key string) error {
	if RDB == nil {
		return nil
	}
	ctx := context.Background()
	return RDB.Del(ctx, key).Err()
}

func RedisDecrease(key string, value int64) error {
	if RDB == nil {
		return nil
	}
	ctx := context.Background()
	return RDB.DecrBy(ctx, key, value).Err()
}
