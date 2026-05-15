package router

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
)

func SetRouter(router *gin.Engine, buildFS embed.FS, nacosConsoleFS embed.FS) {
	router.Use(middleware.NacosFeatureGate())
	SetS3CompatRouter(router)
	SetApiRouter(router)
	SetDashboardRouter(router)
	SetRelayRouter(router)
	SetNacosRegistryRouter(router)
	SetS3PathStyleRootRouter(router)
	MountNacosConsoleDist(router, nacosConsoleFS)
	frontendBaseUrl := strings.TrimSpace(env.StringAlways("frontend_base_url"))
	if config.IsMasterNode && frontendBaseUrl != "" {
		frontendBaseUrl = ""
		logger.SysLog("frontend_base_url is ignored on master node")
	}
	if frontendBaseUrl == "" {
		SetWebRouter(router, buildFS)
	} else {
		frontendBaseUrl = strings.TrimSuffix(frontendBaseUrl, "/")
		router.NoRoute(func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s%s", frontendBaseUrl, c.Request.RequestURI))
		})
	}
}
