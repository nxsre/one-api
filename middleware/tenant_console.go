package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/rbac"
)

// TenantConsoleMemberGate 在 UserAuth 之后：允许平台管理员(R>=10)、租户管理员(R=20)，或具备租户委派权限的子账号。
func TenantConsoleMemberGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetInt(ctxkey.Role)
		if rid >= model.RoleAdminUser {
			c.Next()
			return
		}
		if rid == model.RoleCommonUser {
			uid := c.GetInt(ctxkey.Id)
			u, err := model.GetUserById(uid, false)
			if err == nil && u.TenantID != nil && len(u.TenantPermissions) > 0 {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问租户控制台"})
		c.Abort()
	}
}

// NacosTenantDelegatedGate 在 UserAuth 之后：允许平台管理员、租户管理员，或具备 manage_nacos 委派权限的租户子账号。
func NacosTenantDelegatedGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetInt(ctxkey.Role)
		if rid >= model.RoleAdminUser {
			c.Next()
			return
		}
		if rid == model.RoleCommonUser {
			uid := c.GetInt(ctxkey.Id)
			u, err := model.GetUserById(uid, false)
			if err == nil && u.TenantID != nil && model.UserHasTenantPermission(u, "manage_nacos") {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问 Nacos 管理"})
		c.Abort()
	}
}

// TenantAdminConsoleOnly Casbin 校验租户控制台资源（须配合 TenantConsoleMemberGate + UserAuth）。
func TenantAdminConsoleOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetInt(ctxkey.Id)
		if !rbac.EnforceHTTPMethod(id, rbac.ObjTenantConsole, c.Request.Method) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问租户控制台"})
			c.Abort()
			return
		}
		c.Next()
	}
}
