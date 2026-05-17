package channeltype

// 渠道类型枚举：Unknown=0 表示无效；有效类型自 1 起连续编号（与 channels.type 一致）。
// 类型列表由后端 BuiltinEditorTypes 写死提供，不再依赖 channel_editor_types 表。
const Unknown = 0

const (
	OpenAI = iota + 1
	Azure
	Custom
	PaLM
	Anthropic
	Baidu
	Zhipu
	Ali
	Xunfei
	AI360
	OpenRouter
	AIProxyLibrary
	Tencent
	Gemini
	// GeminiOpenAICompatible：Gemini API 官方 OpenAI 兼容端点（generativelanguage …/v1beta/openai）
	GeminiOpenAICompatible
	Moonshot
	Baichuan
	Minimax
	Mistral
	Groq
	Ollama
	LingYiWanWu
	StepFun
	AwsClaude
	Coze
	Cohere
	DeepSeek
	Cloudflare
	DeepL
	TogetherAI
	Doubao
	Novita
	VertextAI
	Proxy
	SiliconFlow
	XAI
	Replicate
	BaiduV2
	XunfeiV2
	AliBailian
	OpenAICompatible
	// GeminiCompatible：gemini 兼容（非官方前缀须填 Base URL）
	GeminiCompatible
	AiPPT
	AmapPOI
	// DeepResearch：深知 Deep Research（上游 POST /chat 为 SSE；客户端须 stream=true）
	DeepResearch
	AnthropicCompatible
	Dummy // 哨兵：等于最后一个有效类型 + 1
)
