package channeltype

import (
	"fmt"
	"sort"
	"strings"
)

// channelDefaultBaseURL 各渠道类型对应的默认 API 根地址（硬编码）。value 为空表示无内置默认，须在渠道中填写 Base URL。
// 未写入本表的类型（如 Azure、OpenRouter、Ollama、AWS Claude 等）视同无内置默认，由前端与校验逻辑要求用户填写 Base URL。
// 条目顺序与 BuiltinEditorTypes 中出现在本表的类型一致，便于对照维护。
// editor_options 中 channel_types 顺序与 BuiltinEditorTypes 一致；default_base_urls 遍历顺序见 SortedTypesInDefaultBaseURLRegistry。
var channelDefaultBaseURL = map[int]string{
	OpenAI:              "https://api.openai.com",
	Anthropic:           "https://api.anthropic.com",
	Gemini:              "https://generativelanguage.googleapis.com",
	GeminiOpenAICompatible: "https://generativelanguage.googleapis.com/v1beta/openai",
	OpenAICompatible:    "",
	AnthropicCompatible: "",
	GeminiCompatible:       "",
	GeminiNativeCompatible: "",
	PaLM:                "https://generativelanguage.googleapis.com",
	Baidu:               "https://aip.baidubce.com",
	Zhipu:               "https://open.bigmodel.cn",
	Ali:                 "https://dashscope.aliyuncs.com",
	Xunfei:              "https://spark-api-open.xf-yun.com",
	AI360:               "https://ai.360.cn",
	Tencent:             "https://hunyuan.tencentcloudapi.com",
	Moonshot:            "https://api.moonshot.cn",
	Baichuan:            "https://api.baichuan-ai.com",
	Minimax:             "https://api.minimax.chat",
	Mistral:             "https://api.mistral.ai",
	Groq:                "https://api.groq.com/openai",
	LingYiWanWu:         "https://api.lingyiwanwu.com",
	StepFun:             "https://api.stepfun.com",
	Coze:                "https://api.coze.com",
	Cohere:              "https://api.cohere.ai",
	DeepSeek:            "https://api.deepseek.com",
	Cloudflare:          "https://api.cloudflare.com",
	DeepL:               "https://api-free.deepl.com",
	TogetherAI:          "https://api.together.xyz",
	Doubao:              "https://ark.cn-beijing.volces.com",
	XAI:                 "https://api.x.ai",
	Replicate:           "https://api.replicate.com/v1/models/",
	BaiduV2:             "https://qianfan.baidubce.com",
	XunfeiV2:            "https://spark-api-open.xf-yun.com",
	AliBailian:          "https://dashscope.aliyuncs.com",
	AiPPT:               "https://co.aippt.cn",
	AmapPOI:             "https://restapi.amap.com",
	DeepResearch:        "",
}

// SortedTypesInDefaultBaseURLRegistry 先按 BuiltinEditorTypes 顺序列出 registry 中的 id，再追加其余仅存在于 map、未出现在前述列表中的键（按数值升序），供 default_base_urls 遍历顺序。
func SortedTypesInDefaultBaseURLRegistry() []int {
	seen := make(map[int]struct{}, len(channelDefaultBaseURL))
	out := make([]int, 0, len(channelDefaultBaseURL))
	for _, b := range BuiltinEditorTypes() {
		id := b.TypeValue
		if _, ok := channelDefaultBaseURL[id]; ok {
			out = append(out, id)
			seen[id] = struct{}{}
		}
	}
	rest := make([]int, 0)
	for id := range channelDefaultBaseURL {
		if _, ok := seen[id]; !ok {
			rest = append(rest, id)
		}
	}
	sort.Ints(rest)
	return append(out, rest...)
}

// DefaultBaseURL 返回渠道类型的官方默认根 URL（未配置则空串）。
func DefaultBaseURL(channelType int) string {
	return channelDefaultBaseURL[channelType]
}

// ChannelDefaultBaseURLs 返回全部内置默认根地址（map 拷贝），供 API 下发前端。
func ChannelDefaultBaseURLs() map[int]string {
	out := make(map[int]string, len(channelDefaultBaseURL))
	for k, v := range channelDefaultBaseURL {
		out[k] = v
	}
	return out
}

// HasBuiltinBaseURL 为 true 时表示存在内置默认根地址，Base URL 可留空直连官方，亦可填写代理。
func HasBuiltinBaseURL(channelType int) bool {
	return strings.TrimSpace(DefaultBaseURL(channelType)) != ""
}

// ValidateChannelBaseURL 校验渠道 Base URL：无内置默认时必须填写非空 baseURL。
func ValidateChannelBaseURL(channelType int, baseURL string) error {
	if HasBuiltinBaseURL(channelType) {
		return nil
	}
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("该渠道类型未配置默认 API 根地址，必须填写 Base URL")
	}
	return nil
}
