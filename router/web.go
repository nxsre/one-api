package router

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
)

// webThemeCookie lets users switch between the embedded UI builds (e.g. the new
// Vue UI and the classic React UI) at runtime without redeploying.
const webThemeCookie = "web_theme"

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

// themeAssets bundles the per-theme static file server and its index.html so a
// single request can be routed to whichever build the user selected.
type themeAssets struct {
	fileServer http.Handler
	fs         interface {
		Exists(prefix string, path string) bool
	}
	index []byte
}

func SetWebRouter(router *gin.Engine, buildFS embed.FS) {
	// Build one asset bundle per valid theme that actually shipped a build.
	themes := make(map[string]themeAssets)
	for theme := range config.ValidThemes {
		index, err := buildFS.ReadFile(fmt.Sprintf("web/build/%s/index.html", theme))
		if err != nil {
			// Theme registered but not embedded (e.g. not built); skip it.
			continue
		}
		sfs := common.EmbedFolder(buildFS, fmt.Sprintf("web/build/%s", theme))
		themes[theme] = themeAssets{
			fileServer: http.FileServer(sfs),
			fs:         sfs,
			index:      index,
		}
	}

	// defaultTheme is the server-configured THEME when present, otherwise any
	// embedded theme (so a misconfigured THEME still serves something).
	defaultTheme := config.Theme
	if _, ok := themes[defaultTheme]; !ok {
		for theme := range themes {
			defaultTheme = theme
			break
		}
	}

	resolveTheme := func(c *gin.Context) themeAssets {
		if cookie, err := c.Cookie(webThemeCookie); err == nil {
			cookie = strings.TrimSpace(cookie)
			if config.ValidThemes[cookie] {
				if a, ok := themes[cookie]; ok {
					return a
				}
			}
		}
		return themes[defaultTheme]
	}

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())

	// Serve static assets from the resolved theme; fall through to NoRoute for
	// SPA paths and API handlers.
	router.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			assets := resolveTheme(c)
			if assets.fs != nil && assets.fs.Exists("/", c.Request.URL.Path) {
				assets.fileServer.ServeHTTP(c.Writer, c.Request)
				c.Abort()
				return
			}
		}
		c.Next()
	})

	router.NoRoute(func(c *gin.Context) {
		path := spaRequestPath(c)
		indexPageData := resolveTheme(c).index
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
