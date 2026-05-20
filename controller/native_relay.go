package controller

import (
	"github.com/gin-gonic/gin"

	relayctl "github.com/songquanpeng/one-api/relay/controller"
)

// RelayAnthropicNative handles Anthropic Messages API: POST /v1/messages
func RelayAnthropicNative(c *gin.Context) {
	relayWithRetry(c, relayctl.RelayAnthropicNativeOnce)
}

// RelayAnthropicCountTokens handles POST /v1/messages/count_tokens
func RelayAnthropicCountTokens(c *gin.Context) {
	relayWithRetry(c, relayctl.RelayAnthropicCountTokensOnce)
}

// RelayGeminiNative handles Gemini generateContent API: POST /v1beta/models/*:generateContent
func RelayGeminiNative(c *gin.Context) {
	relayWithRetry(c, relayctl.RelayGeminiNativeOnce)
}
