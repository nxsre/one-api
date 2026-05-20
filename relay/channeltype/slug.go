package channeltype

// 渠道 slug：用于把 channelType 数字渲染成人类可读的稳定字符串。
//
// 主要使用场景：在 ModelRatio / CompletionRatio 等 "按渠道覆盖" 的 JSON 配置中，
// 使用 "<modelName>@<channelSlug>" 作为键，避免阅读时去 channeltype.define 里数 iota，
// 也避免常量顺序调整后倍率静默漂移。
//
// 设计约定：
//   - slug 全小写、连字符分隔；
//   - 一旦发布不可随意改名（DB JSON 里会引用），必要时只追加；
//   - 新加渠道类型请同步追加 channelSlugByID / channelIDBySlug；缺失时单测会失败。
var channelSlugByID = map[int]string{
	OpenAI:                 "openai",
	Azure:                  "azure",
	Custom:                 "custom",
	PaLM:                   "palm",
	Anthropic:              "anthropic",
	Baidu:                  "baidu",
	Zhipu:                  "zhipu",
	Ali:                    "ali",
	Xunfei:                 "xunfei",
	AI360:                  "ai360",
	OpenRouter:             "openrouter",
	AIProxyLibrary:         "aiproxy-library",
	Tencent:                "tencent",
	Gemini:                 "gemini",
	GeminiOpenAICompatible: "gemini-openai-compatible",
	Moonshot:               "moonshot",
	Baichuan:               "baichuan",
	Minimax:                "minimax",
	Mistral:                "mistral",
	Groq:                   "groq",
	Ollama:                 "ollama",
	LingYiWanWu:            "lingyiwanwu",
	StepFun:                "stepfun",
	AwsClaude:              "aws-claude",
	Coze:                   "coze",
	Cohere:                 "cohere",
	DeepSeek:               "deepseek",
	Cloudflare:             "cloudflare",
	DeepL:                  "deepl",
	TogetherAI:             "together-ai",
	Doubao:                 "doubao",
	Novita:                 "novita",
	VertextAI:              "vertex-ai",
	Proxy:                  "proxy",
	SiliconFlow:            "siliconflow",
	XAI:                    "xai",
	Replicate:              "replicate",
	BaiduV2:                "baidu-v2",
	XunfeiV2:               "xunfei-v2",
	AliBailian:             "ali-bailian",
	OpenAICompatible:       "openai-compatible",
	GeminiCompatible:       "gemini-compatible",
	AiPPT:                  "aippt",
	AmapPOI:                "amap-poi",
	DeepResearch:           "deep-research",
	AnthropicCompatible:    "anthropic-compatible",
	GeminiNativeCompatible: "gemini-native-compatible",
}

var channelIDBySlug = func() map[string]int {
	m := make(map[string]int, len(channelSlugByID))
	for id, slug := range channelSlugByID {
		m[slug] = id
	}
	return m
}()

// SlugByID 返回渠道类型对应的 slug；未注册时返回空串。
func SlugByID(id int) string {
	return channelSlugByID[id]
}

// IDBySlug 反查 slug 对应的渠道类型 id；未注册时返回 Unknown(0)。
func IDBySlug(slug string) int {
	if id, ok := channelIDBySlug[slug]; ok {
		return id
	}
	return Unknown
}
