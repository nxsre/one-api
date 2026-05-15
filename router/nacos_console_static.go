package router

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/nacosdist"
)

var nacosConsoleStaticMounted bool

// NacosConsoleStaticServing 为 true 时表示已将 Nacos 控制台 dist 以 embed 挂到 /nacos-ui/。
func NacosConsoleStaticServing() bool {
	return nacosConsoleStaticMounted
}

func nacosUiLegacyPlaceholderHTML() []byte {
	return []byte(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>旧版控制台 · Nacos</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Microsoft YaHei", sans-serif;
      margin: 0; padding: 48px 24px; background: #f5f9fb; color: #030d12; line-height: 1.6; }
    .box { max-width: 520px; margin: 0 auto; background: #fff; border: 1px solid #dce2e6; border-radius: 12px; padding: 28px 32px; box-shadow: 0 1px 3px rgba(3,13,18,.06); }
    h1 { font-size: 18px; margin: 0 0 12px; color: #0079ce; }
    p { margin: 0 0 12px; color: #333c41; font-size: 14px; }
    a { color: #0079ce; font-weight: 600; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .en { font-size: 13px; color: #5b656a; margin-top: 16px; padding-top: 16px; border-top: 1px solid #e1e5e7; }
  </style>
</head>
<body>
  <div class="box">
    <h1>旧版控制台未随本发行版提供</h1>
    <p>当前嵌入的 Nacos 控制台仅包含<strong>新版</strong>静态资源（<code>console-ui-next</code>），没有 <code>legacy/</code> 目录。请点击下方使用新版控制台。</p>
    <p><a href="/nacos-ui/">打开新版 Nacos 控制台</a></p>
    <p class="en">The embedded distribution does not include the legacy webpack console under <code>/nacos-ui/legacy/</code>. Use the <a href="/nacos-ui/">new console</a> instead, or build and copy <code>console-ui</code> output into <code>web/nacos-console/dist/legacy/</code> if you need the legacy UI.</p>
  </div>
</body>
</html>
`)
}

func nacosUILegacyFallbackMiddleware() gin.HandlerFunc {
	body := nacosUiLegacyPlaceholderHTML()
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}
		switch c.Request.URL.Path {
		case "/nacos-ui/next", "/nacos-ui/next/":
			c.Redirect(http.StatusFound, "/nacos-ui/")
			c.Abort()
			return
		case "/nacos-ui/legacy":
			c.Redirect(http.StatusMovedPermanently, "/nacos-ui/legacy/")
			c.Abort()
			return
		case "/nacos-ui/legacy/":
			if strings.EqualFold(c.Query("format"), "json") {
				c.JSON(http.StatusOK, gin.H{
					"legacyConsole": false,
					"message":       "legacy console static bundle is not embedded under /nacos-ui/legacy/",
					"newConsoleUrl": "/nacos-ui/",
				})
				c.Abort()
				return
			}
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", body)
			c.Abort()
			return
		}
		c.Next()
	}
}

func nacosUILegacyIndexMiddleware(dist fs.FS) gin.HandlerFunc {
	legacyFS, err := fs.Sub(dist, "legacy")
	if err != nil {
		return nacosUILegacyFallbackMiddleware()
	}
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}
		switch c.Request.URL.Path {
		case "/nacos-ui/next":
			c.Redirect(http.StatusFound, "/nacos-ui/")
			c.Abort()
			return
		case "/nacos-ui/next/":
			c.Redirect(http.StatusFound, "/nacos-ui/")
			c.Abort()
			return
		case "/nacos-ui/legacy":
			c.Redirect(http.StatusMovedPermanently, "/nacos-ui/legacy/")
			c.Abort()
			return
		case "/nacos-ui/legacy/":
			c.Header("Cache-Control", "no-cache")
			content, err := fs.ReadFile(legacyFS, "index.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				c.Abort()
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", content)
			c.Abort()
			return
		}
		c.Next()
	}
}

func isNacosUIAPIPath(path string) bool {
	if !strings.HasPrefix(path, "/nacos-ui/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/nacos-ui/")
	return rest == "v1" || strings.HasPrefix(rest, "v1/") ||
		rest == "v2" || strings.HasPrefix(rest, "v2/") ||
		rest == "v3" || strings.HasPrefix(rest, "v3/")
}

// nacosUILegacyCSSAssetRel 旧版 main.css 中 url(./fonts|./icons) 相对 css/ 解析，实际文件在 dist 根下 fonts/、icons/。
func nacosUILegacyCSSAssetRel(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "legacy/css/fonts/"):
		return "legacy/fonts/" + strings.TrimPrefix(rel, "legacy/css/fonts/")
	case strings.HasPrefix(rel, "legacy/css/icons/"):
		return "legacy/icons/" + strings.TrimPrefix(rel, "legacy/css/icons/")
	default:
		return rel
	}
}

// nacosUIStaticFilesMiddleware 以中间件提供 /nacos-ui 静态资源，避免 StaticFS 通配符与 /nacos-ui/v3 API 路由冲突。
func nacosUIStaticFilesMiddleware(dist fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if path != "/nacos-ui" && !strings.HasPrefix(path, "/nacos-ui/") {
			c.Next()
			return
		}
		if isNacosUIAPIPath(path) {
			c.Next()
			return
		}
		if path == "/nacos-ui/legacy" || path == "/nacos-ui/legacy/" {
			c.Next()
			return
		}
		if path == "/nacos-ui/next" || path == "/nacos-ui/next/" {
			c.Next()
			return
		}

		rel := strings.TrimPrefix(path, "/nacos-ui/")
		if rel == "" {
			rel = "index.html"
		}
		if strings.Contains(rel, "..") {
			c.Status(http.StatusNotFound)
			c.Abort()
			return
		}
		rel = nacosUILegacyCSSAssetRel(rel)
		content, err := fs.ReadFile(dist, rel)
		if err != nil {
			c.Next()
			return
		}
		ct := mime.TypeByExtension(filepath.Ext(rel))
		if ct == "" {
			ct = "application/octet-stream"
		}
		if strings.HasSuffix(rel, ".html") {
			ct = "text/html; charset=utf-8"
			c.Header("Cache-Control", "no-cache")
		}
		c.Data(http.StatusOK, ct, content)
		c.Abort()
	}
}

// MountNacosConsoleDist 将 go:embed 的 web/nacos-console/dist 挂到 /nacos-ui/（同源，无额外配置项）。
func MountNacosConsoleDist(r *gin.Engine, dist embed.FS) {
	nacosConsoleStaticMounted = false
	if !nacosdist.BundleReady(dist) {
		r.Use(nacosUILegacyFallbackMiddleware())
		return
	}
	sub, err := fs.Sub(dist, "web/nacos-console/dist")
	if err != nil {
		logger.SysLog("nacos console embed: fs.Sub: " + err.Error())
		r.Use(nacosUILegacyFallbackMiddleware())
		return
	}
	if _, err := sub.Open("index.html"); err != nil {
		r.Use(nacosUILegacyFallbackMiddleware())
		return
	}
	if _, err := sub.Open("legacy/index.html"); err != nil {
		r.Use(nacosUILegacyFallbackMiddleware())
		logger.SysLog("nacos console: legacy/ not in embed; /nacos-ui/legacy* serves placeholder (avoid one-api SPA 404)")
	} else {
		r.Use(nacosUILegacyIndexMiddleware(sub))
	}
	r.Use(nacosUIStaticFilesMiddleware(sub))
	nacosConsoleStaticMounted = true
	logger.SysLog("Nacos console UI mounted at /nacos-ui/ (embedded dist)")
}
