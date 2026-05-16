package routing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	dbmodel "github.com/songquanpeng/one-api/model"
)

func sanitizeModelSegment(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	if len(s) > 120 {
		return s[:120]
	}
	return s
}

func lookupRule(model string, tab map[string]RateLimitRule) (RateLimitRule, bool) {
	if tab == nil {
		return RateLimitRule{}, false
	}
	if r, ok := tab[model]; ok {
		return r, true
	}
	if r, ok := tab["*"]; ok {
		return r, true
	}
	return RateLimitRule{}, false
}

// AcquireModelLimits 在进入上游前获取令牌/用户/分组三层限速；返回必须在请求结束时调用的释放函数。
func AcquireModelLimits(c *gin.Context) (release func(), ok bool) {
	pol := CurrentModelRateLimitPolicy()
	if !pol.Enabled || !common.RedisEnabled || common.RDB == nil {
		return func() {}, true
	}
	model := strings.TrimSpace(c.GetString(ctxkey.RequestModel))
	if model == "" {
		return func() {}, true
	}
	sm := sanitizeModelSegment(model)
	tokenID := c.GetInt(ctxkey.TokenId)
	userID := c.GetInt(ctxkey.Id)
	group, _ := dbmodel.CacheGetUserGroup(userID)

	var cleaners []func()

	try := func(rule RateLimitRule, scopeTag string) bool {
		if scopeTag == "" {
			return true
		}
		ctx := context.Background()
		rdb := common.RDB
		now := time.Now().UTC()
		day := now.Format("20060102")
		sec := now.Unix()

		if rule.QPS > 0 {
			k := fmt.Sprintf("oa:rl:qps:%s:%s:%d", scopeTag, sm, sec)
			n, err := rdb.Incr(ctx, k).Result()
			if err != nil {
				return true
			}
			_ = rdb.Expire(ctx, k, 5*time.Second).Err()
			limit := rule.QPS
			if rule.Burst > limit {
				limit = rule.Burst
			}
			if n > int64(limit) {
				return false
			}
		}
		if rule.DailyQuota > 0 {
			k := fmt.Sprintf("oa:rl:day:%s:%s:%s", scopeTag, sm, day)
			n, err := rdb.Incr(ctx, k).Result()
			if err != nil {
				return true
			}
			_ = rdb.Expire(ctx, k, 80*time.Hour).Err()
			if n > rule.DailyQuota {
				return false
			}
		}
		if rule.Concurrency > 0 {
			k := fmt.Sprintf("oa:rl:conc:%s:%s", scopeTag, sm)
			n, err := rdb.Incr(ctx, k).Result()
			if err != nil {
				return true
			}
			_ = rdb.Expire(ctx, k, 120*time.Second).Err()
			if n > int64(rule.Concurrency) {
				_, _ = rdb.Decr(ctx, k).Result()
				return false
			}
			cleaners = append(cleaners, func() {
				_, _ = rdb.Decr(context.Background(), k).Result()
			})
		}
		return true
	}

	rt, ok := lookupRule(model, pol.ByToken)
	if ok && tokenID > 0 {
		if !try(rt, fmt.Sprintf("tok:%d", tokenID)) {
			return nil, false
		}
	}
	ru, ok := lookupRule(model, pol.ByUser)
	if ok && userID > 0 {
		if !try(ru, fmt.Sprintf("usr:%d", userID)) {
			for _, cl := range cleaners {
				cl()
			}
			return nil, false
		}
	}
	rgRule, ok := lookupRule(model, pol.ByGroup)
	if ok {
		g := strings.TrimSpace(group)
		if g == "" {
			g = "default"
		}
		gtag := fmt.Sprintf("grp:%s", sanitizeModelSegment(g))
		if !try(rgRule, gtag) {
			for _, cl := range cleaners {
				cl()
			}
			return nil, false
		}
	}

	return func() {
		for _, cl := range cleaners {
			cl()
		}
	}, true
}

// AbortModelRateLimited JSON 429。
func AbortModelRateLimited(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"type":    "one_api_error",
			"message": "模型维度请求速率或配额已达上限",
			"code":    "model_rate_limited",
		},
	})
	c.Abort()
}
