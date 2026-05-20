package channeltype

// IsGeminiNativeAPIChannel 是否走 Google Generative Language 原生协议（generateContent / ListModels 等）。
func IsGeminiNativeAPIChannel(channelType int) bool {
	return channelType == Gemini || channelType == GeminiNativeCompatible
}
