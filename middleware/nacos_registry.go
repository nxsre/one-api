package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
)

func nacosV3Abort(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(http.StatusOK, gin.H{
		"code":    code,
		"message": msg,
		"data":    nil,
	})
}

// NacosRegistryAuthOnly 仅校验 one-api access token（不校验 Nacos ACL），用于控制台占位接口（如角色/权限列表）。
func NacosRegistryAuthOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		if raw == "" {
			nacosV3Abort(c, 401, "未登录且未提供 access token")
			return
		}
		user := model.ValidateAccessToken(raw)
		if user == nil || user.Username == "" {
			nacosV3Abort(c, 401, "access token 无效")
			return
		}
		if user.Status != model.UserStatusEnabled {
			nacosV3Abort(c, 403, "用户已被封禁")
			return
		}
		c.Set(service.NacosCtxUserKey(), user)
		c.Next()
	}
}

// NacosRegistryAccess 鉴权：allowAnonymousNoToken 且开启匿名读时，可无 token 访问；
// 否则必须带有效 access_token，并按 Nacos 风格 ACL（service.NacosCheckPerm）校验 perm。
func NacosRegistryAccess(perm string, allowAnonymousNoToken bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		if raw == "" {
			if allowAnonymousNoToken && config.NacosRegistryAnonymousRead {
				c.Next()
				return
			}
			nacosV3Abort(c, 401, "未登录且未提供 access token")
			return
		}
		user := model.ValidateAccessToken(raw)
		if user == nil || user.Username == "" {
			nacosV3Abort(c, 401, "access token 无效")
			return
		}
		if user.Status != model.UserStatusEnabled {
			nacosV3Abort(c, 403, "用户已被封禁")
			return
		}
		if !service.NacosCheckPerm(user, perm) {
			nacosV3Abort(c, 403, "无权限执行此操作")
			return
		}
		c.Set(service.NacosCtxUserKey(), user)
		c.Next()
	}
}

// NacosRegistryAccessAny 满足 perms 中任一权限即可（用于命名空间/服务发现：配置中心权限与 AI Skills 权限均可）。
func NacosRegistryAccessAny(perms []string, allowAnonymousNoToken bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		if raw == "" {
			if allowAnonymousNoToken && config.NacosRegistryAnonymousRead {
				c.Next()
				return
			}
			nacosV3Abort(c, 401, "未登录且未提供 access token")
			return
		}
		user := model.ValidateAccessToken(raw)
		if user == nil || user.Username == "" {
			nacosV3Abort(c, 401, "access token 无效")
			return
		}
		if user.Status != model.UserStatusEnabled {
			nacosV3Abort(c, 403, "用户已被封禁")
			return
		}
		ok := false
		for _, p := range perms {
			if service.NacosCheckPerm(user, p) {
				ok = true
				break
			}
		}
		if !ok {
			nacosV3Abort(c, 403, "无权限执行此操作")
			return
		}
		c.Set(service.NacosCtxUserKey(), user)
		c.Next()
	}
}
