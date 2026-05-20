package relaymode

const (
	Unknown = iota
	ChatCompletions
	Completions
	Embeddings
	Moderations
	ImagesGenerations
	ImagesEdits
	ImagesVariations
	Edits
	AudioSpeech
	AudioTranscription
	AudioTranslation
	// Files API（OpenAI /v1/files*）
	Files
	// FineTuning / Assistants / Threads（OpenAI Platform API 透传）
	FineTuning
	Assistants
	Threads
	ModelsDelete
	// Proxy is a special relay mode for proxying requests to custom upstream
	Proxy
	AnthropicMessages
	GeminiGenerate
	// OpenAI Responses API（POST /v1/responses 及 /v1/responses/:id 等）
	OpenAIResponses
	// OpenAI Realtime：POST /v1/realtime/sessions 创建会话
	OpenAIRealtimeSessions
)
