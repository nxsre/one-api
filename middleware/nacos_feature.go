package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
)

// IsNacosRegistryAPIPath 匹配 Nacos 注册表 / 控制台兼容 API（不含 One API 管理端 SPA 路径如 /nacos/skills）。
func IsNacosRegistryAPIPath(path string) bool {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	prefixes := []string{
		"/nacos/v1", "/nacos/v3",
		"/nacos/legacy/v1", "/nacos/legacy/v3",
		"/nacos-ui/v1", "/nacos-ui/v3",
		"/nacos-ui/legacy/v1", "/nacos-ui/legacy/v3",
	}
	for _, p := range prefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// IsNacosBlockedPath 在 Nacos 功能关闭时需返回 404 的路径（API、嵌入控制台及 /nacos-ui 资源）。
func IsNacosBlockedPath(path string) bool {
	if strings.HasPrefix(path, "/api/nacos") {
		return true
	}
	if path == "/api/user/nacos-console-token" {
		return true
	}
	if IsNacosRegistryAPIPath(path) {
		return true
	}
	if path == "/nacos-ui" || strings.HasPrefix(path, "/nacos-ui/") {
		return true
	}
	return false
}

// IsNacosConsoleSpaPath One API 管理端内嵌的 Nacos 页面路由（React SPA）。
func IsNacosConsoleSpaPath(path string) bool {
	switch path {
	case "/nacos/skills", "/nacos/permissions", "/nacos/namespaces", "/nacos/cs", "/nacos/console",
		"/nacos/agentspecs", "/nacos/mcp", "/nacos/a2a", "/nacos/prompts", "/nacos/pipelines":
		return true
	default:
		return strings.HasPrefix(path, "/nacos/skills/") ||
			strings.HasPrefix(path, "/nacos/permissions/") ||
			strings.HasPrefix(path, "/nacos/namespaces/") ||
			strings.HasPrefix(path, "/nacos/cs/") ||
			strings.HasPrefix(path, "/nacos/console/") ||
			strings.HasPrefix(path, "/nacos/agentspecs/") ||
			strings.HasPrefix(path, "/nacos/mcp/") ||
			strings.HasPrefix(path, "/nacos/a2a/") ||
			strings.HasPrefix(path, "/nacos/prompts/") ||
			strings.HasPrefix(path, "/nacos/pipelines/")
	}
}

// NacosFeatureGate 关闭 Nacos 时对所有 Nacos 相关 API 与 /nacos-ui 返回 404。
func NacosFeatureGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.IsNacosEnabled() {
			c.Next()
			return
		}
		if IsNacosBlockedPath(c.Request.URL.Path) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
