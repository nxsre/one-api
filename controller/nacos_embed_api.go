package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/model"
)

// GetNacosConsoleEmbedToken 已登录 one-api Web 会话（或 UserAuth 认可的 access token）时，返回嵌套 Nacos 控制台所需的 accessToken，避免在 iframe 内再次输入密码。
func GetNacosConsoleEmbedToken(c *gin.Context) {
	id := c.GetInt(ctxkey.Id)
	user, err := model.GetUserById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	if user.Status != model.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户已被封禁"})
		return
	}
	if strings.TrimSpace(user.AccessToken) == "" {
		user.AccessToken = random.GetUUID()
		if err := user.Update(false); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"accessToken": user.AccessToken,
			"username":    user.Username,
			"globalAdmin": user.Role >= model.RoleAdminUser && user.TenantID == nil,
		},
	})
}
