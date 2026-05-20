package relaymode

import "strings"

// NormalizeAPIPath strips service prefix /openai, /anthropic, or /gemini so routing logic matches legacy /v1… paths.
func NormalizeAPIPath(path string) string {
	if strings.HasPrefix(path, "/openai/") {
		return strings.TrimPrefix(path, "/openai")
	}
	if strings.HasPrefix(path, "/anthropic/") {
		return strings.TrimPrefix(path, "/anthropic")
	}
	if strings.HasPrefix(path, "/gemini/") {
		return strings.TrimPrefix(path, "/gemini")
	}
	return path
}

// IsPassthroughMode 是否为请求/响应原样透传的平台 API（Files、图像编辑、Assistants 等）。
func IsPassthroughMode(mode int) bool {
	switch mode {
	case Files, ImagesEdits, ImagesVariations, FineTuning, Assistants, Threads, ModelsDelete:
		return true
	default:
		return false
	}
}

func GetByPath(path string) int {
	path = NormalizeAPIPath(path)
	relayMode := Unknown
	if strings.HasPrefix(path, "/v1/chat/completions") {
		relayMode = ChatCompletions
	} else if strings.HasPrefix(path, "/v1/completions") {
		relayMode = Completions
	} else if strings.HasPrefix(path, "/v1/embeddings") {
		relayMode = Embeddings
	} else if strings.HasSuffix(path, "embeddings") {
		relayMode = Embeddings
	} else if strings.HasPrefix(path, "/v1/moderations") {
		relayMode = Moderations
	} else if strings.HasPrefix(path, "/v1/images/generations") {
		relayMode = ImagesGenerations
	} else if strings.HasPrefix(path, "/v1/images/edits") {
		relayMode = ImagesEdits
	} else if strings.HasPrefix(path, "/v1/images/variations") {
		relayMode = ImagesVariations
	} else if strings.HasPrefix(path, "/v1/edits") {
		relayMode = Edits
	} else if strings.HasPrefix(path, "/v1/files") {
		relayMode = Files
	} else if strings.HasPrefix(path, "/v1/fine_tuning") {
		relayMode = FineTuning
	} else if strings.HasPrefix(path, "/v1/assistants") {
		relayMode = Assistants
	} else if strings.HasPrefix(path, "/v1/threads") {
		relayMode = Threads
	} else if strings.HasPrefix(path, "/v1/models/") {
		relayMode = ModelsDelete
	} else if strings.HasPrefix(path, "/v1/audio/speech") {
		relayMode = AudioSpeech
	} else if strings.HasPrefix(path, "/v1/audio/transcriptions") {
		relayMode = AudioTranscription
	} else if strings.HasPrefix(path, "/v1/audio/translations") {
		relayMode = AudioTranslation
	} else if strings.HasPrefix(path, "/v1/oneapi/proxy") {
		relayMode = Proxy
	} else if strings.HasPrefix(path, "/v1/messages") {
		relayMode = AnthropicMessages
	} else if strings.HasPrefix(path, "/v1/realtime/sessions") {
		relayMode = OpenAIRealtimeSessions
	} else if strings.HasPrefix(path, "/v1/responses") {
		relayMode = OpenAIResponses
	} else if strings.HasPrefix(path, "/v1beta/models/") || (strings.HasPrefix(path, "/models/") && strings.Contains(path, ":")) {
		relayMode = GeminiGenerate
	}
	return relayMode
}
