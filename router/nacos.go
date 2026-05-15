package router

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/service"
)

// registerNacosV3AIAdminRoutes 注册 /nacos/v3/{prefix}/ai 下与 Nacos console-ui-next 对齐的管理端 AI 路由（不含 Group 前缀本身）。
// omitLabelPostForConsole：为 true 时不注册 POST */labels，由 registerNacosV3ConsoleAIExtraRoutes 注册控制台表单版 PUT+POST，避免与 JSON 版路由重复。
func registerNacosV3AIAdminRoutes(ai *gin.RouterGroup, omitLabelPostForConsole bool) {
	sk := ai.Group("/skills")
	sk.GET("/list", middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true), controller.NacosSkillList)
	sk.GET("", middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true), controller.NacosSkillDescribe)
	sk.GET("/download", middleware.NacosRegistryAccess(service.PermAdminAISkillsRead, true), controller.NacosSkillDownloadAdmin)
	if !omitLabelPostForConsole {
		sk.POST("/labels", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillLabelsUpdate)
	}
	sk.POST("/upload", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillUpload)
	sk.POST("/submit", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillSubmit)
	sk.POST("/publish", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillPublish)
	sk.DELETE("", middleware.NacosRegistryAccess(service.PermAdminAISkillsWrite, false), controller.NacosSkillDelete)

	as := ai.Group("/agentspecs")
	as.GET("/list", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsRead, true), controller.NacosAgentSpecList)
	as.GET("", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsRead, true), controller.NacosAgentSpecDescribe)
	as.GET("/download", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsRead, true), controller.NacosAgentSpecDownloadAdmin)
	if !omitLabelPostForConsole {
		as.POST("/labels", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecLabelsUpdate)
	}
	as.POST("/upload", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecUpload)
	as.POST("/submit", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecSubmit)
	as.POST("/publish", middleware.NacosRegistryAccess(service.PermAdminAIAgentSpecsWrite, false), controller.NacosAgentSpecPublish)

	mcp := ai.Group("/mcp")
	mcp.GET("/list", middleware.NacosRegistryAccess(service.PermAdminAIMcpRead, true), controller.NacosMcpList)
	mcp.GET("", middleware.NacosRegistryAccess(service.PermAdminAIMcpRead, true), controller.NacosMcpDescribe)
	mcp.POST("", middleware.NacosRegistryAccess(service.PermAdminAIMcpWrite, false), controller.NacosMcpUpsert)
	mcp.DELETE("", middleware.NacosRegistryAccess(service.PermAdminAIMcpWrite, false), controller.NacosMcpDelete)

	a2 := ai.Group("/a2a")
	a2.GET("/list", middleware.NacosRegistryAccess(service.PermAdminAIA2ARead, true), controller.NacosA2AList)
	a2.GET("", middleware.NacosRegistryAccess(service.PermAdminAIA2ARead, true), controller.NacosA2ADescribe)
	a2.POST("", middleware.NacosRegistryAccess(service.PermAdminAIA2AWrite, false), controller.NacosA2AUpsert)
	a2.DELETE("", middleware.NacosRegistryAccess(service.PermAdminAIA2AWrite, false), controller.NacosA2ADelete)

	pm := ai.Group("/prompt")
	pm.GET("/list", middleware.NacosRegistryAccess(service.PermAdminAIPromptRead, true), controller.NacosPromptList)
	pm.GET("", middleware.NacosRegistryAccess(service.PermAdminAIPromptRead, true), controller.NacosPromptDescribe)
	if !omitLabelPostForConsole {
		pm.POST("/labels", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptLabelsUpdate)
	}
	pm.POST("/header", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptUpsertHeader)
	pm.POST("/version", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptAddVersion)
	pm.POST("/submit", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptSubmit)
	pm.POST("/publish", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptPublish)
	pm.DELETE("", middleware.NacosRegistryAccess(service.PermAdminAIPromptWrite, false), controller.NacosPromptDelete)

	pl := ai.Group("/pipelines")
	pl.GET("/list", middleware.NacosRegistryAccess(service.PermAdminAIPipelineRead, true), controller.NacosPipelineList)
	pl.GET("/detail", middleware.NacosRegistryAccess(service.PermAdminAIPipelineRead, true), controller.NacosPipelineDescribe)
	pl.POST("/run/scan", middleware.NacosRegistryAccess(service.PermAdminAIPipelineWrite, false), controller.NacosPipelineRunScan)
}

// mountNacosRegistryV1V3 在 base 下注册与 /nacos 相同的 v1|v3 注册表与控制台嵌入路由。
// base 为 router.Group("/nacos") 时路径为 /nacos/v1、/nacos/v3；为 router.Group("/nacos").Group("/legacy") 时为 /nacos/legacy/v1、/nacos/legacy/v3。
func mountNacosRegistryV1V3(base *gin.RouterGroup) {
	v1 := base.Group("/v1")
	v1.POST("/auth/user/login", controller.NacosUserLogin)

	v3 := base.Group("/v3")
	v3.POST("/auth/user/login", controller.NacosUserLoginDisabledV3)

	auth := v3.Group("/auth")
	auth.GET("/role/list", middleware.NacosRegistryAuthOnly(), controller.NacosAuthRoleList)
	auth.GET("/permission/list", middleware.NacosRegistryAuthOnly(), controller.NacosAuthPermissionList)
	auth.GET("/user/list", middleware.NacosRegistryAuthOnly(), controller.NacosAuthUserList)

	consoleSrv := v3.Group("/console/server")
	{
		consoleSrv.GET("/state", controller.NacosConsoleServerState)
		consoleSrv.GET("/announcement", controller.NacosConsoleServerAnnouncement)
		consoleSrv.GET("/guide", controller.NacosConsoleServerGuide)
	}

	registerNacosV3ConsoleEmbeds(base)

	v1.GET("/cs/configs", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigListV1)

	csAdmin := v3.Group("/admin/cs")
	{
		csAdmin.GET("/config/list", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigListV3)
		csAdmin.GET("/config/history", middleware.NacosRegistryAccess(service.PermAdminCsConfigRead, true), controller.NacosCsConfigHistoryListV3)
		csAdmin.POST("/config", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigPublish)
		csAdmin.POST("/config/rollback", middleware.NacosRegistryAccess(service.PermAdminCsConfigWrite, false), controller.NacosCsConfigRollbackV3)
	}
	v3.GET("/client/cs/config", middleware.NacosRegistryAccess(service.PermClientCsConfigRead, true), controller.NacosCsConfigGetClient)

	// console-ui-next 使用 …/v3/console/ai/*；nacos-cli 使用 …/v3/admin/ai/*
	registerNacosV3AIAdminRoutes(v3.Group("/admin/ai"), false)

	client := v3.Group("/client/ai")
	{
		client.GET("/skills", middleware.NacosRegistryAccess(service.PermClientAISkillsRead, true), controller.NacosSkillClientGet)
		client.GET("/agentspecs", middleware.NacosRegistryAccess(service.PermClientAIAgentSpecsRead, true), controller.NacosAgentSpecClientGet)
		client.GET("/mcp", middleware.NacosRegistryAccess(service.PermClientAIMcpRead, true), controller.NacosMcpClientGet)
		client.GET("/a2a", middleware.NacosRegistryAccess(service.PermClientAIA2ARead, true), controller.NacosA2AClientGet)
		client.GET("/prompt", middleware.NacosRegistryAccess(service.PermClientAIPromptRead, true), controller.NacosPromptClientGet)
	}
}

// SetNacosRegistryRouter 注册 nacos-cli 兼容的 /nacos/v1|v3，并在 /nacos/legacy/v1|v3 挂载相同能力（旧版 URL 前缀）。
// 另在 /nacos-ui/v1|v3 挂载相同 API：旧版 console-ui 嵌入 /nacos-ui/legacy/ 时 axios baseURL 为 /nacos-ui/。
func SetNacosRegistryRouter(router *gin.Engine) {
	nacos := router.Group("/nacos")
	mountNacosRegistryV1V3(nacos)
	mountNacosRegistryV1V3(nacos.Group("/legacy"))

	nacosUI := router.Group("/nacos-ui")
	mountNacosRegistryV1V3(nacosUI)
}
