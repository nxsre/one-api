package router

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
)

// spaRequestPath 用于 SPA 回退判断：优先 URL.Path（与代理、Request-Line 形式无关）。
func spaRequestPath(c *gin.Context) string {
	if c.Request.URL != nil {
		if p := c.Request.URL.Path; p != "" {
			return p
		}
	}
	u := c.Request.RequestURI
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	return u
}

func SetWebRouter(router *gin.Engine, buildFS embed.FS) {
	indexPageData, _ := buildFS.ReadFile(fmt.Sprintf("web/build/%s/index.html", config.Theme))
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, fmt.Sprintf("web/build/%s", config.Theme))))
	router.NoRoute(func(c *gin.Context) {
		path := spaRequestPath(c)
		if strings.HasPrefix(path, "/nacos-ui") {
			if !NacosConsoleStaticServing() {
				c.Status(http.StatusNotFound)
				c.Header("Content-Type", "text/plain; charset=utf-8")
				_, _ = c.Writer.WriteString(
					"Nacos console UI is not embedded (missing web/nacos-console/dist from Docker build). See Dockerfile third_party/nacos/console-ui-next.\n",
				)
				return
			}
			c.Status(http.StatusNotFound)
			c.Header("Content-Type", "text/plain; charset=utf-8")
			_, _ = c.Writer.WriteString("Not found under /nacos-ui\n")
			return
		}
		if middleware.IsNacosConsoleSpaPath(path) {
			if !config.IsNacosEnabled() {
				c.Status(http.StatusNotFound)
				return
			}
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexPageData)
			return
		}
		if strings.HasPrefix(path, "/v1") || strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/v1beta") ||
			strings.HasPrefix(path, "/openai/") || strings.HasPrefix(path, "/anthropic/") || strings.HasPrefix(path, "/gemini/") {
			controller.RelayNotFound(c)
			return
		}
		if middleware.IsNacosRegistryAPIPath(path) {
			controller.NacosV3NoRoute(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPageData)
	})
}
