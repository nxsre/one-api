package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/rbac"
)

// PlatformConsoleOnly Casbin 校验平台资源 ObjPlatform，并对租户管理员给出明确提示。
func PlatformConsoleOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetInt(ctxkey.Id)
		if id <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
			c.Abort()
			return
		}
		if model.IsTenantConsoleAdmin(id) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "租户管理员请使用租户控制台接口 /api/tenant_console"})
			c.Abort()
			return
		}
		if !rbac.EnforceHTTPMethod(id, rbac.ObjPlatform, c.Request.Method) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问平台管理"})
			c.Abort()
			return
		}
		c.Next()
	}
}
