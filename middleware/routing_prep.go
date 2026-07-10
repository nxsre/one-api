package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/routing"
)

// RoutingPrep 全局别名解析、模型维度限速、一致性哈希钉扎键。
func RoutingPrep() func(c *gin.Context) {
	return func(c *gin.Context) {
		userId := c.GetInt(ctxkey.Id)
		userGroup, err := model.ResolveRelayUserGroup(userId, c.GetString(ctxkey.TokenBoundGroup))
		if err != nil {
			abortWithMessage(c, 500, "用户信息加载失败")
			return
		}
		c.Set(ctxkey.Group, userGroup)

		raw := strings.TrimSpace(c.GetString(ctxkey.RequestModel))
		if raw != "" {
			c.Set(ctxkey.LogicalModel, raw)
			if base, variant := anthropic.SplitClientModelVariant(raw); variant != "" {
				c.Set(ctxkey.AnthropicModelVariant, variant)
				raw = base
			}
			resolved, err := routing.ResolveModelAliasesForGroup(userGroup, raw)
			if err != nil {
				abortWithMessage(c, 400, err.Error())
				return
			}
			c.Set(ctxkey.RequestModel, resolved)
		}

		release, ok := routing.AcquireModelLimits(c)
		if !ok {
			routing.AbortModelRateLimited(c)
			return
		}
		defer release()

		pol := routing.CurrentRoutingPolicy()
		switch strings.ToLower(strings.TrimSpace(pol.StickySource)) {
		case "user_id":
			c.Set(ctxkey.RoutingStickyKey, fmt.Sprintf("u%d", c.GetInt(ctxkey.Id)))
		case "none":
			c.Set(ctxkey.RoutingStickyKey, "")
		default:
			c.Set(ctxkey.RoutingStickyKey, fmt.Sprintf("t%d", c.GetInt(ctxkey.TokenId)))
		}

		c.Next()
	}
}
