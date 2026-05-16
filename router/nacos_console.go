package router

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/service"
)

// registerNacosV3ConsoleEmbeds 在 parent 下注册 …/v3/console/*（parent 为 /nacos 或 /nacos/legacy 等前缀）。
func registerNacosV3ConsoleEmbeds(parent *gin.RouterGroup) {
	v3 := parent.Group("/v3")

	// --- AI：与 /admin/ai 相同能力 + console 专有路径（omitLabelPost=true 避免与 extra 中表单版 labels 重复注册）---
	registerNacosV3AIAdminRoutes(v3.Group("/console/ai"), true)
	registerNacosV3ConsoleAIExtraRoutes(v3.Group("/console/ai"))

	// --- 配置中心：console 路径与 admin 对齐（history 在 /console/cs/history 下）---
	registerNacosV3ConsoleCSRoutes(v3)

	// --- 命名空间 ---
	registerNacosV3ConsoleNamespaceRoutes(v3)

	// --- 其它控制台页面可能请求的 API（占位，避免落入 SPA NoRoute）---
	registerNacosV3ConsoleMiscStubs(v3)
}

func registerNacosV3ConsoleCSRoutes(v3 *gin.RouterGroup) {
	cs := v3.Group("/console/cs")
	cfg := cs.Group("/config")
	{
		cfg.GET("/list", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigListV3)
		cfg.GET("/searchDetail", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigListV3)
		cfg.GET("", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigGet)
		cfg.POST("", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigPublish)
		cfg.DELETE("", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigDeleteConsole)
		cfg.DELETE("/batchDelete", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigBatchDeleteConsole)
		cfg.GET("/listener", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigListenerConsole)
		cfg.GET("/listener/ip", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigListenerConsole)
		cfg.GET("/beta", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigBetaGetConsole)
		cfg.DELETE("/beta", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigBetaDeleteConsole)
		cfg.POST("/import", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigImportConsole)
		cfg.POST("/clone", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigCloneConsole)
		cfg.GET("/export2", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigExport2Console)
	}
	hist := cs.Group("/history")
	{
		hist.GET("/list", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigHistoryListV3)
		hist.GET("", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigHistoryDetailConsole)
		hist.GET("/previous", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigHistoryPreviousConsole)
	}
}

func registerNacosV3ConsoleNamespaceRoutes(v3 *gin.RouterGroup) {
	nsReadPerms := []string{service.PermAdminCsConfigRead, service.PermAdminAISkillsRead}
	nsWritePerms := []string{service.PermAdminCsConfigWrite, service.PermAdminAISkillsWrite}
	ns := v3.Group("/console/core/namespace")
	{
		ns.GET("/list", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleNamespaceList)
		ns.GET("", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleNamespaceGet)
		ns.POST("", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleNamespaceMutateStub)
		ns.PUT("", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleNamespaceMutateStub)
		ns.DELETE("", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleNamespaceDeleteStub)
	}
}

func registerNacosV3ConsoleMiscStubs(v3 *gin.RouterGroup) {
	nsReadPerms := []string{service.PermAdminCsConfigRead, service.PermAdminAISkillsRead}
	nsWritePerms := []string{service.PermAdminCsConfigWrite, service.PermAdminAISkillsWrite}
	ns := v3.Group("/console/ns")
	{
		ns.GET("/service/list", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleDiscoveryServiceList)
		ns.GET("/service", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleDiscoveryServiceGet)
		ns.POST("/service", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleDiscoveryServicePost)
		ns.PUT("/service", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleDiscoveryServicePut)
		ns.DELETE("/service", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleDiscoveryServiceDelete)
		ns.GET("/service/selector/types", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleDiscoverySelectorTypes)
		ns.PUT("/service/cluster", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleDiscoveryClusterPut)
		ns.GET("/instance/list", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleDiscoveryInstanceList)
		ns.PUT("/instance", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleDiscoveryInstancePut)
		ns.DELETE("/instance", middleware.NacosRegistryAccessAny(nsWritePerms, false), controller.NacosConsoleDiscoveryInstanceDelete)
		ns.GET("/service/subscribers", middleware.NacosRegistryAccessAny(nsReadPerms, true), controller.NacosConsoleDiscoverySubscribersList)
	}

	pl := v3.Group("/console/plugin")
	pl.Use(middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true))
	{
		pl.GET("/list", controller.NacosConsolePluginListConsole)
		pl.PUT("/status", controller.NacosConsolePluginStatusPut)
	}

	cl := v3.Group("/console/core/cluster")
	cl.Use(middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true))
	{
		cl.GET("/nodes", controller.NacosConsoleClusterNodesList)
		cl.POST("/server/leave", controller.NacosConsoleClusterServerLeave)
	}

	cp := v3.Group("/console/copilot")
	cp.Use(middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true))
	{
		cp.GET("/config", controller.NacosConsoleCopilotConfigGet)
		cp.POST("/config", controller.NacosConsoleCopilotConfigPost)
		cp.POST("/skill/generate", controller.NacosCopilotSkillGenerate)
		cp.POST("/skill/optimize", controller.NacosCopilotSkillOptimize)
		cp.POST("/prompt/optimize", controller.NacosCopilotPromptOptimize)
		cp.POST("/prompt/debug", controller.NacosCopilotPromptDebug)
	}
}

// registerNacosV3ConsoleAIExtraRoutes 补全 console-ui-next 相对 /admin/ai 多出的 HTTP 路径与方法。
func registerNacosV3ConsoleAIExtraRoutes(ai *gin.RouterGroup) {
	sk := ai.Group("/skills")
	sk.GET("/version", middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true), controller.NacosSkillConsoleGetVersion)
	sk.GET("/version/download", middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true), controller.NacosSkillDownloadAdmin)
	sk.POST("/draft", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillDraftStub)
	sk.PUT("/draft", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillDraftStub)
	sk.DELETE("/draft", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillDraftStub)
	sk.POST("/force-publish", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillPublish)
	sk.POST("/online", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillOnlineOfflineStub)
	sk.POST("/offline", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillOnlineOfflineStub)
	sk.PUT("/labels", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillLabelsUpdateConsole)
	sk.POST("/labels", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillLabelsUpdateConsole)
	sk.PUT("/biz-tags", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillBizTagsUpdate)
	sk.PUT("/scope", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosConsoleSkillScopeUpdate)

	as := ai.Group("/agentspecs")
	as.GET("/version", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsRead, true), controller.NacosAgentSpecConsoleGetVersion)
	as.GET("/version/download", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsRead, true), controller.NacosAgentSpecDownloadAdmin)
	as.POST("/draft", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecDraftStub)
	as.PUT("/draft", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecDraftStub)
	as.DELETE("/draft", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecDraftStub)
	as.POST("/force-publish", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecPublish)
	as.POST("/online", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecOnlineOfflineStub)
	as.POST("/offline", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecOnlineOfflineStub)
	as.PUT("/labels", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecLabelsUpdateConsole)
	as.POST("/labels", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecLabelsUpdateConsole)
	as.PUT("/biz-tags", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecBizTagsUpdate)
	as.PUT("/scope", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosConsoleAgentSpecScopeUpdate)

	mcp := ai.Group("/mcp")
	mcp.GET("/importToolsFromMcp", middleware.NacosRegistryAccess(service.PermAdminAIMcpRead, true), controller.NacosConsoleMcpImportToolsList)
	mcp.POST("/import/validate", middleware.NacosRegistryAccess(service.PermAdminAIMcpWrite, false), controller.NacosConsoleMcpImportValidateStub)
	mcp.POST("/import/execute", middleware.NacosRegistryAccess(service.PermAdminAIMcpWrite, false), controller.NacosConsoleStubOKTrue)

	a2 := ai.Group("/a2a")
	a2.GET("/version/list", middleware.NacosRegistryAccess(service.PermAdminAIA2ARead, true), controller.NacosA2AVersionListConsole)

	pm := ai.Group("/prompt")
	pm.GET("/governance", middleware.NacosRegistryAccess(service.PermAdminAIPromptRead, true), controller.NacosPromptConsoleGovernance)
	pm.GET("/version", middleware.NacosRegistryAccess(service.PermAdminAIPromptRead, true), controller.NacosPromptConsoleVersionDetail)
	pm.GET("/versions", middleware.NacosRegistryAccess(service.PermAdminAIPromptRead, true), controller.NacosPromptConsoleVersionsPage)
	pm.GET("/version/download", middleware.NacosRegistryAccess(service.PermAdminAIPromptRead, true), controller.NacosPromptConsoleVersionDownload)
	pm.POST("/draft", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleDraftStub)
	pm.PUT("/draft", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleDraftStub)
	pm.DELETE("/draft", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleDraftStub)
	pm.POST("/force-publish", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptPublish)
	pm.POST("/online", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleOnlineOfflineStub)
	pm.POST("/offline", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleOnlineOfflineStub)
	pm.PUT("/labels", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptLabelsUpdateConsole)
	pm.POST("/labels", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptLabelsUpdateConsole)
	pm.PUT("/description", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleDescriptionUpdate)
	pm.PUT("/biz-tags", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptConsoleBizTagsUpdate)
}
