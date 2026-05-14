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
	g.POST("/edits", controller.Relay)
	g.POST("/images/generations", controller.Relay)
	g.POST("/images/edits", controller.RelayNotImplemented)
	g.POST("/images/variations", controller.RelayNotImplemented)
	g.POST("/embeddings", controller.Relay)
	g.POST("/engines/:model/embeddings", controller.Relay)
	g.POST("/audio/transcriptions", controller.Relay)
	g.POST("/audio/translations", controller.Relay)
	g.POST("/audio/speech", controller.Relay)
	g.GET("/files", controller.RelayNotImplemented)
	g.POST("/files", controller.RelayNotImplemented)
	g.DELETE("/files/:id", controller.RelayNotImplemented)
	g.GET("/files/:id", controller.RelayNotImplemented)
	g.GET("/files/:id/content", controller.RelayNotImplemented)
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

	stack := func(g *gin.RouterGroup) {
		g.Use(middleware.RelayPanicRecover(), middleware.TokenAuth(), middleware.Distribute())
	}

	relayLegacy := router.Group("/v1")
	stack(relayLegacy)
	registerOpenAIV1RelayRoutes(relayLegacy)
	relayLegacy.POST("/messages", controller.RelayAnthropicNative)

	relayOpenAI := router.Group("/openai/v1")
	stack(relayOpenAI)
	registerOpenAIV1RelayRoutes(relayOpenAI)

	anthropicV1 := router.Group("/anthropic/v1")
	stack(anthropicV1)
	anthropicV1.POST("/messages", controller.RelayAnthropicNative)

	geminiLegacy := router.Group("/v1beta/models")
	stack(geminiLegacy)
	geminiLegacy.POST("/*geminiAction", controller.RelayGeminiNative)

	geminiPrefixed := router.Group("/gemini/v1beta/models")
	stack(geminiPrefixed)
	geminiPrefixed.POST("/*geminiAction", controller.RelayGeminiNative)
}
