package controller

import (
	"fmt"

	"github.com/songquanpeng/one-api/relay/channeltype"
)

type channelTestCaps struct {
	chat       bool
	embedding  bool
	image      bool
	tts        bool
	moderation bool
	responses  bool
	realtime   bool
}

func openAIWireChannelTestCaps() channelTestCaps {
	return channelTestCaps{
		chat: true, embedding: true, image: true, tts: true, moderation: true, responses: true, realtime: true,
	}
}

func channelTestCapabilities(channelType int) channelTestCaps {
	switch channelType {
	case channeltype.Anthropic, channeltype.AnthropicCompatible, channeltype.AwsClaude:
		return channelTestCaps{chat: true}
	case channeltype.Gemini, channeltype.GeminiCompatible, channeltype.GeminiOpenAICompatible, channeltype.GeminiNativeCompatible:
		return channelTestCaps{chat: true, embedding: true}
	case channeltype.Ollama, channeltype.Cohere, channeltype.Ali, channeltype.Tencent, channeltype.Baidu, channeltype.BaiduV2, channeltype.Zhipu:
		return channelTestCaps{chat: true, embedding: true}
	case channeltype.DeepL, channeltype.AmapPOI:
		return channelTestCaps{}
	case channeltype.AiPPT, channeltype.DeepResearch:
		return channelTestCaps{chat: true}
	case channeltype.Replicate, channeltype.Cloudflare:
		return channelTestCaps{chat: true, image: true}
	default:
		return openAIWireChannelTestCaps()
	}
}

func channelSupportsTestKind(caps channelTestCaps, kind modelTestKind) bool {
	switch kind {
	case modelTestKindChat:
		return caps.chat
	case modelTestKindEmbedding:
		return caps.embedding
	case modelTestKindImage:
		return caps.image
	case modelTestKindTTS:
		return caps.tts
	case modelTestKindModeration:
		return caps.moderation
	case modelTestKindResponses:
		return caps.responses
	case modelTestKindRealtime:
		return caps.realtime
	default:
		return false
	}
}

func channelTypeLabel(channelType int) string {
	if slug := channeltype.SlugByID(channelType); slug != "" {
		return slug
	}
	return fmt.Sprintf("type-%d", channelType)
}

func resolveModelTestSpec(modelName string, channelType int) modelTestSpec {
	spec := classifyModelTestSpecByName(modelName)
	if spec.Kind == modelTestKindSkip {
		return spec
	}
	caps := channelTestCapabilities(channelType)
	if !channelSupportsTestKind(caps, spec.Kind) {
		return modelTestSpec{
			Kind:       modelTestKindSkip,
			SkipReason: fmt.Sprintf("当前渠道（%s）不支持 %s 协议探测", channelTypeLabel(channelType), modelTestKindLabel(spec.Kind)),
		}
	}
	applyChannelModelTestRoute(channelType, &spec)
	return spec
}
