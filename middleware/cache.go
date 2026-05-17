package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Cache 设置 Cache-Control。
// 原先对除「/」以外的所有路径统一 max-age=604800，会导致 SPA 回退的 HTML（如 /about）
// 以及 web-build.json 等也被长期缓存；镜像更新后浏览器仍用旧的入口 HTML，加载旧 JS，表现为新功能「永远不生效」。
// 仅对带 hash 的 /static/ 资源使用长期缓存。
func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		path := "/"
		if c.Request != nil && c.Request.URL != nil && c.Request.URL.Path != "" {
			path = c.Request.URL.Path
		}
		switch {
		case path == "/" || path == "/index.html":
			c.Header("Cache-Control", "no-cache")
		case strings.HasPrefix(path, "/static/"):
			c.Header("Cache-Control", "max-age=31536000, immutable")
		default:
			c.Header("Cache-Control", "no-cache")
		}
		c.Next()
	}
}
