package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/agentdetect"
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// AgentClientPolicy 按 agent 客户端类型执行访问/限流策略。
//
// 层级合并为"就近覆盖"（令牌 > 用户 > 租户）+ 全局禁用兜底（见 agentpolicy.Evaluate）；
// 限流则"全局 + 各维度独立"叠加，任一超限即拒绝。
// 未启用本功能（无任何非空策略）时通过 agentpolicy.Enabled() 快速短路，零额外开销。
func AgentClientPolicy() gin.HandlerFunc {
	inMemoryRateLimiter.Init(config.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		if !agentpolicy.Enabled() {
			c.Next()
			return
		}

		global := agentpolicy.Global()
		tokenPol := tokenPolicyFromCtx(c)
		userPol := model.GetUserAgentPolicy(c.GetInt(ctxkey.Id))
		var tenantPol *agentpolicy.Policy
		if tid := c.GetInt(ctxkey.UserTenantID); tid > 0 {
			tenantPol = model.GetTenantAgentPolicy(tid)
		}
		if global.IsZero() && tokenPol.IsZero() && userPol.IsZero() && tenantPol.IsZero() {
			c.Next()
			return
		}

		client := resolveAgentClient(c)
		dec := agentpolicy.Evaluate(global, tokenPol, userPol, tenantPol, client)
		if dec.Blocked {
			abortWithMessage(c, http.StatusForbidden,
				fmt.Sprintf("客户端类型 %q 不被允许：%s", dec.Client, dec.Reason))
			return
		}

		if layer, ok := exceedsRateLimit(c, dec.Client, global, tokenPol, userPol, tenantPol); ok {
			abortWithMessage(c, http.StatusTooManyRequests,
				fmt.Sprintf("客户端类型 %q 触发%s限流", dec.Client, layer))
			return
		}
		c.Next()
	}
}

// resolveAgentClient 解析本次请求的客户端类型：复用已识别结果 → 头识别 → 解析请求体二次确认。
func resolveAgentClient(c *gin.Context) string {
	if v := c.GetString(ctxkey.AgentClient); v != "" {
		return v
	}
	res := agentdetect.DetectHeader(c.Request.Header)
	if res.Client == "" {
		// 头部不确定时读取并缓存请求体做 system/tools 识别；读后回填 body 供下游复用。
		if body, err := common.GetRequestBody(c); err == nil && len(body) > 0 {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			res = agentdetect.Detect(c.Request.Header, body)
		}
	}
	if res.Client != "" {
		c.Set(ctxkey.AgentClient, res.Client)
	}
	return res.Client
}

func tokenPolicyFromCtx(c *gin.Context) *agentpolicy.Policy {
	if v, ok := c.Get(ctxkey.TokenAgentPolicy); ok {
		if p, ok := v.(*agentpolicy.Policy); ok {
			return p
		}
	}
	return nil
}

// exceedsRateLimit 依次检查全局/令牌/用户/租户各层对该客户端类型的限流，任一超限返回 (层级名, true)。
func exceedsRateLimit(c *gin.Context, client string, global, token, user, tenant *agentpolicy.Policy) (string, bool) {
	tid := c.GetInt(ctxkey.UserTenantID)
	checks := []struct {
		name   string
		policy *agentpolicy.Policy
		key    string
	}{
		{"全局", global, "acp:g:" + client},
		{"令牌", token, "acp:t:" + strconv.Itoa(c.GetInt(ctxkey.TokenId)) + ":" + client},
		{"用户", user, "acp:u:" + strconv.Itoa(c.GetInt(ctxkey.Id)) + ":" + client},
		{"租户", tenant, "acp:n:" + strconv.Itoa(tid) + ":" + client},
	}
	for _, ck := range checks {
		if ck.policy == nil {
			continue
		}
		r := ck.policy.EffectiveRule(client)
		if r.MaxRequests <= 0 || r.WindowSec <= 0 {
			continue
		}
		if !agentRateAllow(ck.key, r.MaxRequests, int64(r.WindowSec)) {
			return ck.name, true
		}
	}
	return "", false
}

// agentRateAllow 复用滑动窗口限流原语（Redis 优先，否则内存），按 key 判定是否放行。
// 与按 IP 的全局限流不同，这里按"层级+实体+客户端类型"维度计数。Redis 异常时 fail-open。
func agentRateAllow(key string, maxRequests int, windowSec int64) bool {
	if maxRequests <= 0 || windowSec <= 0 || config.DebugEnabled {
		return true
	}
	if common.RedisEnabled {
		ctx := context.Background()
		now := time.Now().UnixNano()
		window := windowSec * int64(time.Second)
		ttl := int64(config.RateLimitKeyExpirationDuration.Seconds())
		member := strconv.FormatInt(now, 10) + "-" + strconv.FormatUint(atomic.AddUint64(&rateLimitSeq, 1), 10)
		allowed, err := rateLimitScript.Run(ctx, common.RDB, []string{"rateLimit:" + key},
			now, window, maxRequests, ttl, member).Int()
		if err != nil {
			logger.SysError("agent policy redis rate limiter error, allowing: " + err.Error())
			return true
		}
		return allowed == 1
	}
	return inMemoryRateLimiter.Request(key, maxRequests, windowSec)
}
