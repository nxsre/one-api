package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/acme"
	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/i18n"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/nacosdist"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/rbac"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/router"
	"github.com/songquanpeng/one-api/routing"
	"github.com/songquanpeng/one-api/service"
)

// all: ensures files whose names start with '_' or '.' are embedded too —
// Vite/Rolldown emits shared chunks like `_plugin-vue_export-helper.*.js`,
// which the default embed pattern would silently drop (breaking the SPA).
//
//go:embed all:web/build
var buildFS embed.FS

//go:embed web/nacos-console/dist
var nacosConsoleFS embed.FS

func main() {
	common.Init()
	logger.SetupLogger()
	logger.SysLogf("One API %s (build %s) started", common.Version, common.BuildID)

	if strings.TrimSpace(cfg.V.GetString("gin_mode")) != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	if config.DebugEnabled {
		logger.SysLog("running in debug mode")
	}

	common.InitEmbeddedTLSFromEnv()
	if cfg.V.GetBool("acme_enabled") {
		if err := acme.Init(); err != nil {
			logger.FatalLog("ACME init failed: " + err.Error())
		}
		defer acme.Shutdown()
	}

	// Initialize SQL Database
	model.InitDB()
	service.StartNacosConsoleClusterSelfHeartbeat(common.Port)
	model.InitLogDB()

	var err error
	if err = rbac.Init(); err != nil {
		logger.FatalLog("rbac init error: " + err.Error())
	}
	err = model.CreateRootAccountIfNeed()
	if err != nil {
		logger.FatalLog("database init error: " + err.Error())
	}
	if err = rbac.SyncAllUsers(); err != nil {
		logger.SysError("rbac SyncAllUsers: " + err.Error())
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
	routing.StartBackgroundJobs()

	// Initialize options
	model.InitOptionMap()
	model.InitAgentPolicyEnabled()
	if err := model.InitPricingEntryStore(); err != nil {
		logger.SysError("init pricing entry store failed: " + err.Error())
	}
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
	// 跨实例缓存失效：写入方发布事件后各实例立即重载，轮询作为兜底。Redis 未启用时为空操作。
	model.StartCacheInvalidationSubscriber()
	if fq := cfg.V.GetInt("channel_test_frequency"); fq > 0 {
		go controller.AutomaticallyTestChannels(fq)
	}
	if config.BatchUpdateEnabled {
		logger.SysLog("batch update enabled with interval " + strconv.Itoa(config.BatchUpdateInterval) + "s")
		model.InitBatchUpdater()
	}
	model.InitLogConsumer()
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

	nacosdist.Bind(nacosConsoleFS)
	router.SetRouter(server, buildFS, nacosConsoleFS)
	service.StartS3Cleaner()
	port := strconv.Itoa(common.Port)

	// 各监听路径统一注册为显式的 *http.Server，并在后台 goroutine 启动，
	// 主协程阻塞等待退出信号后对它们逐一 Shutdown，实现优雅关闭。
	var servers []*http.Server
	startHTTPS := func(addr string) {
		srv := &http.Server{
			Addr:      addr,
			Handler:   server,
			TLSConfig: acme.TLSConfig(),
		}
		servers = append(servers, srv)
		go func() {
			logger.SysLogf("server listening on https://0.0.0.0%s (ACME auto-renew)", addr)
			if e := srv.ListenAndServeTLS("", ""); e != nil && e != http.ErrServerClosed {
				logger.FatalLog("failed to start HTTPS server: " + e.Error())
			}
		}()
	}
	startTLS := func(addr string) {
		srv := &http.Server{Addr: addr, Handler: server}
		servers = append(servers, srv)
		go func() {
			logger.SysLogf("server listening on https://0.0.0.0%s", addr)
			if e := srv.ListenAndServeTLS(common.TLSCertFile, common.TLSKeyFile); e != nil && e != http.ErrServerClosed {
				logger.FatalLog("failed to start HTTPS server: " + e.Error())
			}
		}()
	}
	startHTTP := func(addr string, handler http.Handler) {
		srv := &http.Server{Addr: addr, Handler: handler}
		servers = append(servers, srv)
		go func() {
			logger.SysLogf("server listening on http://0.0.0.0%s", addr)
			if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
				logger.FatalLog("failed to start HTTP server: " + e.Error())
			}
		}()
	}

	switch {
	case acme.Enabled() && common.HTTPSOnly:
		startHTTPS(":" + port)
	case acme.Enabled() && !common.HTTPSOnly:
		httpsPort := common.TLSDualHTTPSPort
		if httpsPort == port {
			logger.FatalLog("HTTP and HTTPS cannot share the same PORT; set HTTPS_PORT to a different port")
		}
		startHTTPS(":" + httpsPort)
		startHTTP(":"+port, acme.WrapHTTPHandler(server))
	case common.TLSCertFile != "" && common.HTTPSOnly:
		startTLS(":" + port)
	case common.TLSCertFile != "" && !common.HTTPSOnly:
		httpsPort := common.TLSDualHTTPSPort
		if httpsPort == port {
			logger.FatalLog("HTTP and HTTPS cannot share the same PORT; set HTTPS_PORT to a different port")
		}
		startTLS(":" + httpsPort)
		startHTTP(":"+port, server)
	default:
		startHTTP(":"+port, server)
	}

	gracefulShutdown(servers)
}

// gracefulShutdown 阻塞等待 SIGINT/SIGTERM，随后停止接收新请求、等待在途请求处理
// 完成，最后 flush 后台缓冲（异步日志队列、批量配额更新），确保退出前不丢数据。
func gracefulShutdown(servers []*http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.SysLog("received signal " + sig.String() + ", shutting down gracefully...")

	timeout := time.Duration(config.ServerShutdownTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, srv := range servers {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			if err := s.Shutdown(ctx); err != nil {
				logger.SysError("server shutdown error: " + err.Error())
			}
		}(srv)
	}
	wg.Wait()

	// 服务器已停止接收新请求，此时再 flush 后台缓冲，保证不丢日志与配额。
	model.FlushLogQueue()
	model.FlushBatchUpdates()
	logger.SysLog("graceful shutdown complete")
}
