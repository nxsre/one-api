package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

// MetricsAuth 保护 /metrics 端点。支持 Bearer Token (环境变量 METRICS_TOKEN) 或 Root 登录状态。
func MetricsAuth() func(c *gin.Context) {
	expectedToken := os.Getenv("METRICS_TOKEN")
	return func(c *gin.Context) {
		// 1. Check Bearer Token
		authHeader := c.GetHeader("Authorization")
		if expectedToken != "" && authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == expectedToken {
				c.Next()
				return
			}
		}

		// 2. Check session if user is Root
		session := sessions.Default(c)
		role := session.Get("role")
		if role != nil {
			if r, ok := role.(int); ok && r == model.RoleRootUser {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized access to metrics",
		})
	}
}
