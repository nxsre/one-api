package router

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
)

// registerOpenAIV1RelayRoutes registers OpenAI-compatible relay paths under a group (e.g. /v1 or /openai/v1).
func registerOpenAIV1RelayRoutes(g *gin.RouterGroup) {
	g.Any("/oneapi/proxy/:channelid/*target", controller.Relay)
	g.POST("/completions", controller.Relay)
	g.POST("/chat/completions", controller.Relay)
	g.POST("/responses", controller.Relay)
	g.GET("/responses/:id", controller.Relay)
	g.POST("/responses/:id/cancel", controller.Relay)
	g.POST("/realtime/sessions", controller.Relay)
	g.GET("/realtime", controller.RelayRealtimeWebSocket)
	g.POST("/edits", controller.Relay)
	g.POST("/images/generations", controller.Relay)
	g.POST("/images/edits", controller.Relay)
	g.POST("/images/variations", controller.Relay)
	g.POST("/embeddings", controller.Relay)
	g.POST("/engines/:model/embeddings", controller.Relay)
	g.POST("/audio/transcriptions", controller.Relay)
	g.POST("/audio/translations", controller.Relay)
	g.POST("/audio/speech", controller.Relay)
	g.GET("/files", controller.Relay)
	g.POST("/files", controller.Relay)
	g.DELETE("/files/:id", controller.Relay)
	g.GET("/files/:id", controller.Relay)
	g.GET("/files/:id/content", controller.Relay)
	g.POST("/fine_tuning/jobs", controller.RelayNotImplemented)
	g.GET("/fine_tuning/jobs", controller.RelayNotImplemented)
	g.GET("/fine_tuning/jobs/:id", controller.RelayNotImplemented)
	g.POST("/fine_tuning/jobs/:id/cancel", controller.RelayNotImplemented)
	g.GET("/fine_tuning/jobs/:id/events", controller.RelayNotImplemented)
	g.DELETE("/models/:model", controller.RelayNotImplemented)
	g.POST("/moderations", controller.Relay)
	g.POST("/assistants", controller.RelayNotImplemented)
	g.GET("/assistants/:id", controller.RelayNotImplemented)
	g.POST("/assistants/:id", controller.RelayNotImplemented)
	g.DELETE("/assistants/:id", controller.RelayNotImplemented)
	g.GET("/assistants", controller.RelayNotImplemented)
	g.POST("/assistants/:id/files", controller.RelayNotImplemented)
	g.GET("/assistants/:id/files/:fileId", controller.RelayNotImplemented)
	g.DELETE("/assistants/:id/files/:fileId", controller.RelayNotImplemented)
	g.GET("/assistants/:id/files", controller.RelayNotImplemented)
	g.POST("/threads", controller.RelayNotImplemented)
	g.GET("/threads/:id", controller.RelayNotImplemented)
	g.POST("/threads/:id", controller.RelayNotImplemented)
	g.DELETE("/threads/:id", controller.RelayNotImplemented)
	g.POST("/threads/:id/messages", controller.RelayNotImplemented)
	g.GET("/threads/:id/messages/:messageId", controller.RelayNotImplemented)
	g.POST("/threads/:id/messages/:messageId", controller.RelayNotImplemented)
	g.GET("/threads/:id/messages/:messageId/files/:filesId", controller.RelayNotImplemented)
	g.GET("/threads/:id/messages/:messageId/files", controller.RelayNotImplemented)
	g.POST("/threads/:id/runs", controller.RelayNotImplemented)
	g.GET("/threads/:id/runs/:runsId", controller.RelayNotImplemented)
	g.POST("/threads/:id/runs/:runsId", controller.RelayNotImplemented)
	g.GET("/threads/:id/runs", controller.RelayNotImplemented)
	g.POST("/threads/:id/runs/:runsId/submit_tool_outputs", controller.RelayNotImplemented)
	g.POST("/threads/:id/runs/:runsId/cancel", controller.RelayNotImplemented)
	g.GET("/threads/:id/runs/:runsId/steps/:stepId", controller.RelayNotImplemented)
	g.GET("/threads/:id/runs/:runsId/steps", controller.RelayNotImplemented)
}

func registerOpenAIV1ModelsRoutes(g *gin.RouterGroup) {
	g.GET("", controller.ListModels)
	g.GET("/:model", controller.RetrieveModel)
}

func SetRelayRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.GzipDecodeMiddleware())
	router.Use(middleware.GlobalAccessGate())

	modelsLegacy := router.Group("/v1/models")
	modelsLegacy.Use(middleware.TokenAuth())
	registerOpenAIV1ModelsRoutes(modelsLegacy)

	modelsOpenAI := router.Group("/openai/v1/models")
	modelsOpenAI.Use(middleware.TokenAuth())
	registerOpenAIV1ModelsRoutes(modelsOpenAI)

	anthropicModels := router.Group("/anthropic/v1")
	anthropicModels.Use(middleware.TokenAuth())
	anthropicModels.GET("/models", controller.ListAnthropicModels)
	anthropicModels.GET("/models/:model_id", controller.RetrieveAnthropicModel)

	router.POST("/amap",
		middleware.RelayPanicRecover(),
		middleware.TokenAuth(),
		middleware.ConsumeLogCapture(),
		controller.RelayAmapProxy,
	)

	stack := func(g *gin.RouterGroup) {
		g.Use(
			middleware.RelayPanicRecover(),
			middleware.ResponsesRealtimeFormatBridge(),
			middleware.TokenAuth(),
			middleware.ConsumeLogCapture(),
			middleware.RoutingPrep(),
			middleware.Distribute(),
		)
	}

	relayLegacy := router.Group("/v1")
	stack(relayLegacy)
	registerOpenAIV1RelayRoutes(relayLegacy)
	relayLegacy.POST("/messages", controller.RelayAnthropicNative)
	relayLegacy.POST("/messages/count_tokens", controller.RelayAnthropicCountTokens)

	relayOpenAI := router.Group("/openai/v1")
	stack(relayOpenAI)
	registerOpenAIV1RelayRoutes(relayOpenAI)

	anthropicV1 := router.Group("/anthropic/v1")
	stack(anthropicV1)
	anthropicV1.POST("/messages", controller.RelayAnthropicNative)
	anthropicV1.POST("/messages/count_tokens", controller.RelayAnthropicCountTokens)

	geminiModelsList := router.Group("/v1beta/models")
	geminiModelsList.Use(middleware.TokenAuth())
	geminiModelsList.GET("", controller.ListGeminiModels)
	geminiModelsList.GET("/:model", controller.RetrieveGeminiModel)

	geminiLegacy := router.Group("/v1beta/models")
	stack(geminiLegacy)
	geminiLegacy.POST("/*geminiAction", controller.RelayGeminiNative)

	geminiPrefixedList := router.Group("/gemini/v1beta/models")
	geminiPrefixedList.Use(middleware.TokenAuth())
	geminiPrefixedList.GET("", controller.ListGeminiModels)
	geminiPrefixedList.GET("/:model", controller.RetrieveGeminiModel)

	geminiPrefixed := router.Group("/gemini/v1beta/models")
	stack(geminiPrefixed)
	geminiPrefixed.POST("/*geminiAction", controller.RelayGeminiNative)

	// 兼容将 base 配成 …/gemini 且路径为 /models/{model}:method（与 AI Studio / 部分 SDK 习惯一致，省略 v1beta）。
	// pi-ai / @google/genai 在自定义 baseUrl 时会清空 apiVersion，GET/POST 均走 /gemini/models/…。
	geminiModelsCompatList := router.Group("/gemini/models")
	geminiModelsCompatList.Use(middleware.TokenAuth())
	geminiModelsCompatList.GET("", controller.ListGeminiModels)
	geminiModelsCompatList.GET("/:model", controller.RetrieveGeminiModel)

	geminiModelsCompat := router.Group("/gemini/models")
	stack(geminiModelsCompat)
	geminiModelsCompat.POST("/*geminiAction", controller.RelayGeminiNative)
}
