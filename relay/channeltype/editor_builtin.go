package channeltype

// BuiltinEditorMeta 内置渠道类型下拉项（与 define 中常量数值一致），供 API 与库表覆盖合并。
type BuiltinEditorMeta struct {
	TypeValue int
	Text      string
	Protocol  string // openai | anthropic | gemini | aippt | amap | deepresearch（与模型目录聚合分支一致）
	TypeKind  string // basic | custom
	Tip       string
}

// BuiltinEditorTypes 返回新建/编辑页类型下拉项（后端写死，不入库）。不含 Custom：请使用「OpenAI 兼容」等类型对接自建网关。
func BuiltinEditorTypes() []BuiltinEditorMeta {
	return []BuiltinEditorMeta{
		{OpenAI, "OpenAI", "openai", "basic", ""},
		{Anthropic, "Anthropic", "anthropic", "basic", "Anthropic Messages 官方 API；Base URL 可留空使用默认 https://api.anthropic.com，或填代理。"},
		{Gemini, "Gemini", "gemini", "basic", "Google Gemini 原生 API；可在配置 JSON 中填写 api_version（默认 v1）。"},
		{GeminiOpenAICompatible, "Gemini OpenAI 兼容（Google 官方）", "openai", "basic", "Gemini API OpenAI 兼容协议（文档：https://ai.google.dev/gemini-api/docs/openai）；Base URL 可留空使用官方 …/v1beta/openai；密钥使用 Gemini API Key（AI Studio）。"},
		{OpenAICompatible, "OpenAI 兼容", "openai", "basic", "对接任意 OpenAI 形态网关，必须填写 Base URL。"},
		{AnthropicCompatible, "Anthropic 兼容", "anthropic", "basic", "Anthropic Messages 兼容上游，必须填写 Base URL。"},
		{GeminiCompatible, "Gemini 兼容（OpenAI 形态）", "openai", "basic", "gemini 兼容（非官方前缀须填 Base URL）"},
		{Azure, "Azure", "openai", "basic", "填写 AZURE_OPENAI_ENDPOINT；可在配置 JSON 中设置 api_version。"},
		{PaLM, "Google PaLM", "gemini", "basic", ""},
		{Baidu, "百度文心", "openai", "basic", ""},
		{Zhipu, "智谱 GLM", "openai", "basic", ""},
		{Ali, "阿里通义", "openai", "basic", ""},
		{Xunfei, "讯飞星火", "openai", "basic", ""},
		{AI360, "360 智脑", "openai", "basic", ""},
		{OpenRouter, "OpenRouter", "openai", "basic", "必须填写 Base URL（通常为 https://openrouter.ai/api/v1）。"},
		{AIProxyLibrary, "AIProxy 知识库", "openai", "basic", ""},
		{Tencent, "腾讯混元", "openai", "basic", ""},
		{Moonshot, "Moonshot", "openai", "basic", ""},
		{Baichuan, "百川", "openai", "basic", ""},
		{Minimax, "MiniMax", "openai", "basic", ""},
		{Mistral, "Mistral", "openai", "basic", ""},
		{Groq, "Groq", "openai", "basic", ""},
		{Ollama, "Ollama", "openai", "basic", "本地或自建 Ollama，必须填写 Base URL。"},
		{LingYiWanWu, "零一万物", "openai", "basic", ""},
		{StepFun, "阶跃星辰", "openai", "basic", ""},
		{AwsClaude, "AWS Claude", "anthropic", "basic", "Bedrock / Claude；配置 JSON 中填写 region、ak、sk。"},
		{Coze, "Coze", "openai", "basic", ""},
		{Cohere, "Cohere", "openai", "basic", ""},
		{DeepSeek, "DeepSeek", "openai", "basic", ""},
		{Cloudflare, "Cloudflare", "openai", "basic", ""},
		{DeepL, "DeepL", "openai", "basic", ""},
		{TogetherAI, "Together AI", "openai", "basic", ""},
		{Doubao, "火山方舟", "openai", "basic", ""},
		{Novita, "Novita", "openai", "basic", "必须填写 Base URL。"},
		{VertextAI, "Vertex AI", "gemini", "basic", "必须填写区域与 Vertex 相关配置。"},
		{Proxy, "通用代理", "openai", "basic", "必须填写 Base URL。"},
		{SiliconFlow, "SiliconFlow", "openai", "basic", ""},
		{XAI, "xAI", "openai", "basic", ""},
		{Replicate, "Replicate", "openai", "basic", ""},
		{BaiduV2, "百度千帆 V2", "openai", "basic", ""},
		{XunfeiV2, "讯飞星火 V2", "openai", "basic", ""},
		{AliBailian, "阿里百炼", "openai", "basic", ""},
		{AiPPT, "AiPPT", "openai", "custom", "模型名 aippt-ppt；密钥格式见说明。"},
		{AmapPOI, "高德", "amap", "custom", "模型名 amap。"},
		{DeepResearch, "深知 Deep Research", "openai", "custom", "模型名 deep-research；上游 SSE，客户端须 stream: true。"},
	}
}
