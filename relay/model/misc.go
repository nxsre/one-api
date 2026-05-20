package model

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	UsageSemantic string `json:"usage_semantic,omitempty"`
	UsageSource   string `json:"usage_source,omitempty"`

	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`

	ClaudeCacheCreation5mTokens int `json:"claude_cache_creation_5_m_tokens,omitempty"`
	ClaudeCacheCreation1hTokens int `json:"claude_cache_creation_1_h_tokens,omitempty"`
}

// PromptTokensDetails 与 OpenAI / new-api 对齐的输入 token 明细（用于缓存/图片/音频计费）。
type PromptTokensDetails struct {
	CachedTokens         int `json:"cached_tokens"`
	CachedCreationTokens int `json:"cached_creation_tokens,omitempty"`
	TextTokens           int `json:"text_tokens"`
	AudioTokens          int `json:"audio_tokens"`
	ImageTokens          int `json:"image_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens            int `json:"reasoning_tokens"`
	AcceptedPredictionTokens   int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens   int `json:"rejected_prediction_tokens"`
	ImageTokens                int `json:"image_tokens,omitempty"`
	AudioTokens                int `json:"audio_tokens,omitempty"`
}

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    any    `json:"code"`
}

type ErrorWithStatusCode struct {
	Error
	StatusCode int `json:"status_code"`
}
