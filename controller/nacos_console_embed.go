package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NacosV3NoRoute 未匹配的 Nacos 注册表 API（落入 SPA NoRoute 时），返回 Nacos 风格 JSON，避免被误判为 OpenAI relay。
func NacosV3NoRoute(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    404,
		"message": fmt.Sprintf("未实现的 Nacos API: %s %s", c.Request.Method, c.Request.URL.Path),
		"data":    nil,
	})
}

// NacosConsoleServerState 供嵌入的 console-ui-next 拉取 /v3/console/server/state（与 one-api 会话统一前仍须存在该接口）。
func NacosConsoleServerState(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":               "3.0.0-one-api",
		"standalone_mode":       "standalone",
		// 空串表示「全能力」控制台：侧栏同时展示配置管理、服务管理与 AI（由 ai_enabled 控制）。
		"function_mode":         "",
		"login_page_enabled":    "true",
		"auth_enabled":          "true",
		"console_ui_enabled":    "true",
		"startup_mode":          "standalone",
		"config_retention_days": "30",
		"auth_admin_request":    "false",
		"auth_system_type":      "one-api",
		"copilot_enabled":       "true",
		"ai_enabled":            "true",
	})
}

// NacosConsoleServerAnnouncement 占位，避免控制台启动时报错。
func NacosConsoleServerAnnouncement(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}

// NacosConsoleServerGuide 占位。
func NacosConsoleServerGuide(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}
