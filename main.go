package main

import (
	"embed"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/i18n"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/router"
	"github.com/songquanpeng/one-api/service"
)

//go:embed web/build/*
var buildFS embed.FS

func main() {
	common.Init()
	logger.SetupLogger()
	logger.SysLogf("One API %s started", common.Version)

	if strings.TrimSpace(cfg.V.GetString("gin_mode")) != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	if config.DebugEnabled {
		logger.SysLog("running in debug mode")
	}

	common.InitEmbeddedTLSFromEnv()
	if err := common.InitLoginPasswordRSA(); err != nil {
		logger.FatalLog("login RSA init error: " + err.Error())
	}

	// Initialize SQL Database
	model.InitDB()
	model.InitLogDB()

	var err error
	err = model.CreateRootAccountIfNeed()
	if err != nil {
		logger.FatalLog("database init error: " + err.Error())
	}
	defer func() {
		err := model.CloseDB()
		if err != nil {
			logger.FatalLog("failed to close database: " + err.Error())
		}
	}()

	// Initialize Redis
	err = common.InitRedisClient()
	if err != nil {
		logger.FatalLog("failed to initialize Redis: " + err.Error())
	}

	// Initialize options
	model.InitOptionMap()
	logger.SysLog(fmt.Sprintf("using theme %s", config.Theme))
	if common.RedisEnabled {
		// for compatibility with old versions
		config.MemoryCacheEnabled = true
	}
	if config.MemoryCacheEnabled {
		logger.SysLog("memory cache enabled")
		logger.SysLog(fmt.Sprintf("sync frequency: %d seconds", config.SyncFrequency))
		model.InitChannelCache()
	}
	if config.MemoryCacheEnabled {
		go model.SyncOptions(config.SyncFrequency)
		go model.SyncChannelCache(config.SyncFrequency)
	}
	if fq := cfg.V.GetInt("channel_test_frequency"); fq > 0 {
		go controller.AutomaticallyTestChannels(fq)
	}
	if config.BatchUpdateEnabled {
		logger.SysLog("batch update enabled with interval " + strconv.Itoa(config.BatchUpdateInterval) + "s")
		model.InitBatchUpdater()
	}
	if config.EnableMetric {
		logger.SysLog("metric enabled, will disable channel if too much request failed")
	}
	openai.InitTokenEncoders()
	client.Init()

	// Initialize i18n
	if err := i18n.Init(); err != nil {
		logger.FatalLog("failed to initialize i18n: " + err.Error())
	}

	// Initialize HTTP server
	server := gin.New()
	server.Use(gin.Recovery())
	// This will cause SSE not to work!!!
	//server.Use(gzip.Gzip(gzip.DefaultCompression))
	server.Use(middleware.RequestId())
	server.Use(middleware.Language())
	middleware.SetUpLogger(server)
	store, err := common.NewGinSessionStore()
	if err != nil {
		logger.FatalLog("session store: " + err.Error())
	}
	sessionSecure := common.TLSCertFile != "" && common.HTTPSOnly
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	server.Use(sessions.Sessions("session", store))

	router.SetRouter(server, buildFS)
	service.StartS3Cleaner()
	port := strconv.Itoa(common.Port)
	switch {
	case common.TLSCertFile != "" && common.HTTPSOnly:
		logger.SysLogf("server listening on https://0.0.0.0:%s", port)
		err = server.RunTLS(":"+port, common.TLSCertFile, common.TLSKeyFile)
		if err != nil {
			logger.FatalLog("failed to start HTTPS server: " + err.Error())
		}
	case common.TLSCertFile != "" && !common.HTTPSOnly:
		httpsPort := common.TLSDualHTTPSPort
		if httpsPort == port {
			logger.FatalLog("HTTP and HTTPS cannot share the same PORT; set HTTPS_PORT to a different port")
		}
		go func() {
			if e := server.RunTLS(":"+httpsPort, common.TLSCertFile, common.TLSKeyFile); e != nil {
				logger.FatalLog("failed to start HTTPS server: " + e.Error())
			}
		}()
		logger.SysLogf("server listening on http://0.0.0.0:%s and https://0.0.0.0:%s", port, httpsPort)
		err = server.Run(":" + port)
		if err != nil {
			logger.FatalLog("failed to start HTTP server: " + err.Error())
		}
	default:
		logger.SysLogf("server started on http://localhost:%s", port)
		err = server.Run(":" + port)
		if err != nil {
			logger.FatalLog("failed to start HTTP server: " + err.Error())
		}
	}
}
