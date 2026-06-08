package middleware

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/blacklist"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/network"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"net/http"
	"strings"
)

func authHelper(c *gin.Context, minRole int) {
	session := sessions.Default(c)
	username := session.Get("username")
	role := session.Get("role")
	id := session.Get("id")
	status := session.Get("status")
	if username == nil {
		// Check access token
		accessToken := c.Request.Header.Get("Authorization")
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无权进行此操作，未登录且未提供 access token",
			})
			c.Abort()
			return
		}
		user := model.ValidateAccessToken(accessToken)
		if user != nil && user.Username != "" {
			// Token is valid
			username = user.Username
			role = user.Role
			id = user.Id
			status = user.Status
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无权进行此操作，access token 无效",
			})
			c.Abort()
			return
		}
	}
	if status.(int) == model.UserStatusDisabled || blacklist.IsUserBanned(id.(int)) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户已被封禁",
		})
		session := sessions.Default(c)
		session.Clear()
		_ = session.Save()
		c.Abort()
		return
	}
	if role.(int) < minRole {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权进行此操作，权限不足",
		})
		c.Abort()
		return
	}
	uid, uok := sessionInt(id)
	rid, rok := sessionInt(role)
	if !uok || !rok {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "会话数据异常"})
		c.Abort()
		return
	}
	if !CheckForce2FAWebSession(c, uid, rid) {
		return
	}
	c.Set("username", username)
	c.Set(ctxkey.Role, rid)
	c.Set(ctxkey.Id, uid)
	c.Next()
}

func UserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleCommonUser)
	}
}

func AdminAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleAdminUser)
	}
}

func SuperAdminAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleSuperAdmin)
	}
}

func RootAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleRootUser)
	}
}

func TokenAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		// OpenAI 生态多用 Authorization: Bearer；Anthropic Messages API 等使用 x-api-key。
		raw := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			raw = strings.TrimSpace(raw[7:])
		}
		raw = strings.TrimPrefix(raw, "sk-")
		if raw == "" {
			ak := strings.TrimSpace(c.Request.Header.Get("x-api-key"))
			raw = strings.TrimPrefix(ak, "sk-")
		}
		if raw == "" {
			gk := strings.TrimSpace(c.Request.Header.Get("x-goog-api-key"))
			raw = strings.TrimPrefix(gk, "sk-")
		}
		// Google Gemini 客户端常用查询参数 ?key=（与官方 generativelanguage 一致）。
		if raw == "" {
			if k := strings.TrimSpace(c.Query("key")); k != "" {
				raw = strings.TrimPrefix(k, "sk-")
			}
		}
		parts := strings.Split(raw, "-")
		key := parts[0]
		token, err := model.ValidateUserToken(key)
		if err != nil {
			abortWithMessage(c, http.StatusUnauthorized, err.Error())
			return
		}
		bindTokenIdentity(c, token)
		if token.AgentClientPolicy != nil && !token.AgentClientPolicy.IsZero() {
			c.Set(ctxkey.TokenAgentPolicy, token.AgentClientPolicy)
		}
		if token.Subnet != nil && *token.Subnet != "" {
			if !network.IsIpInSubnets(ctx, c.ClientIP(), *token.Subnet) {
				abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该令牌只能在指定网段使用：%s，当前 ip：%s", *token.Subnet, c.ClientIP()))
				return
			}
		}
		userEnabled, err := model.CacheIsUserEnabled(token.UserId)
		if err != nil {
			abortWithMessage(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !userEnabled || blacklist.IsUserBanned(token.UserId) {
			abortWithMessage(c, http.StatusForbidden, "用户已被封禁")
			return
		}
		requestModel, err := getRequestModel(c)
		if err != nil && shouldCheckModel(c) {
			abortWithMessage(c, http.StatusBadRequest, err.Error())
			return
		}
		c.Set(ctxkey.RequestModel, requestModel)
		if token.Models != nil && *token.Models != "" {
			c.Set(ctxkey.AvailableModels, *token.Models)
			if requestModel != "" && !model.IsRequestModelInAllowlist(c.Request.Context(), token.UserId, requestModel, *token.Models) {
				abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该令牌无权使用模型：%s", requestModel))
				return
			}
		}
		am, ach, subOk, err := model.LoadTenantSubRelayRestrictions(token.UserId)
		if err != nil {
			abortWithMessage(c, http.StatusInternalServerError, "用户信息加载失败")
			return
		}
		if subOk {
			if len(am) > 0 && requestModel != "" && shouldCheckModel(c) {
				if !model.IsRequestModelInAllowlistSlice(c.Request.Context(), token.UserId, requestModel, am) {
					abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该账号无权使用模型：%s", requestModel))
					return
				}
			}
			if len(am) > 0 {
				ms := make(map[string]struct{}, len(am))
				for _, m := range am {
					t := strings.TrimSpace(m)
					if t != "" {
						ms[t] = struct{}{}
					}
				}
				if len(ms) > 0 {
					c.Set(ctxkey.UserAllowedModels, ms)
				}
			}
			if len(ach) > 0 {
				chm := make(map[int]struct{}, len(ach))
				for _, id := range ach {
					if id > 0 {
						chm[id] = struct{}{}
					}
				}
				if len(chm) > 0 {
					c.Set(ctxkey.UserAllowedChannelIDs, chm)
				}
			}
		}
		if token.Group != nil {
			if g := strings.TrimSpace(*token.Group); g != "" {
				c.Set(ctxkey.TokenBoundGroup, g)
			}
		}
		if len(parts) > 1 {
			if model.IsAdmin(token.UserId) {
				c.Set(ctxkey.SpecificChannelId, parts[1])
			} else {
				abortWithMessage(c, http.StatusForbidden, "普通用户不支持指定渠道")
				return
			}
		}

		// set channel id for proxy relay
		if channelId := c.Param("channelid"); channelId != "" {
			c.Set(ctxkey.SpecificChannelId, channelId)
		}

		if !CheckForce2FATokenAuth(c, token.UserId) {
			return
		}

		c.Next()
	}
}

func shouldCheckModel(c *gin.Context) bool {
	p := relaymode.NormalizeAPIPath(c.Request.URL.Path)
	if strings.HasPrefix(p, "/v1/completions") {
		return true
	}
	if strings.HasPrefix(p, "/v1/chat/completions") {
		return true
	}
	if strings.HasPrefix(p, "/v1/images") {
		return true
	}
	if strings.HasPrefix(p, "/v1/audio") {
		return true
	}
	if strings.HasPrefix(p, "/v1/messages") {
		return true
	}
	if p == "/v1beta/models" && c.Request.Method == http.MethodGet {
		return false
	}
	if strings.HasPrefix(p, "/v1beta/models/") {
		if c.Request.Method == http.MethodGet && !strings.Contains(p, ":") {
			return false
		}
		return true
	}
	if strings.HasPrefix(p, "/models/") && strings.Contains(p, ":") {
		return true
	}
	if strings.HasPrefix(p, "/v1/responses") {
		return true
	}
	if strings.HasPrefix(p, "/v1/realtime") {
		return true
	}
	return false
}
