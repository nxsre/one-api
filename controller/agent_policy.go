package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/agentdetect"
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
)

// GetAgentPolicyMeta 返回可识别的客户端类型清单与当前全局策略，供后台白名单/限流配置页使用。
func GetAgentPolicyMeta(c *gin.Context) {
	config.OptionMapRWMutex.RLock()
	globalRaw := config.OptionMap[agentpolicy.OptionKey]
	config.OptionMapRWMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			// known_clients：已知 agent 类型；other_client：普通 API/SDK 直连的兜底类别。
			"known_clients": agentdetect.KnownClients(),
			"other_client":  agentpolicy.OtherClient,
			"global":        globalRaw,
		},
	})
}

// PlatformUpdateTenantAgentPolicy 更新某租户的 agent 客户端策略（白名单 + 限流）。
func PlatformUpdateTenantAgentPolicy(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "租户 ID 无效"})
		return
	}
	var body struct {
		AgentClientPolicy *agentpolicy.Policy `json:"agent_client_policy"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.SetTenantAgentPolicy(id, body.AgentClientPolicy); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
