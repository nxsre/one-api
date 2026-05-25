package controller

import (
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type channelWireProtocol int

const (
	wireProtoOpenAI channelWireProtocol = iota
	wireProtoAnthropic
	wireProtoGeminiNative
	wireProtoGeminiOpenAI
)

func channelWireProtocolFamily(channelType int) channelWireProtocol {
	switch channelType {
	case channeltype.Anthropic, channeltype.AnthropicCompatible, channeltype.AwsClaude:
		return wireProtoAnthropic
	case channeltype.Gemini, channeltype.GeminiNativeCompatible:
		return wireProtoGeminiNative
	case channeltype.GeminiOpenAICompatible, channeltype.GeminiCompatible:
		return wireProtoGeminiOpenAI
	default:
		return wireProtoOpenAI
	}
}

func modelTestWireProtocolLabel(family channelWireProtocol, kind modelTestKind) string {
	switch family {
	case wireProtoAnthropic:
		switch kind {
		case modelTestKindChat:
			return "anthropic/messages"
		}
	case wireProtoGeminiNative:
		switch kind {
		case modelTestKindChat:
			return "gemini/generateContent"
		case modelTestKindEmbedding:
			return "gemini/batchEmbedContents"
		}
	case wireProtoGeminiOpenAI:
		switch kind {
		case modelTestKindChat:
			return "openai/chat-completions (gemini-compat)"
		case modelTestKindEmbedding:
			return "openai/embeddings (gemini-compat)"
		case modelTestKindImage:
			return "openai/images (gemini-compat)"
		case modelTestKindTTS:
			return "openai/tts (gemini-compat)"
		case modelTestKindModeration:
			return "openai/moderations (gemini-compat)"
		case modelTestKindResponses:
			return "openai/responses (gemini-compat)"
		case modelTestKindRealtime:
			return "openai/realtime/sessions (gemini-compat)"
		}
	default:
		switch kind {
		case modelTestKindChat:
			return "openai/chat-completions"
		case modelTestKindEmbedding:
			return "openai/embeddings"
		case modelTestKindImage:
			return "openai/images/generations"
		case modelTestKindTTS:
			return "openai/audio/speech"
		case modelTestKindModeration:
			return "openai/moderations"
		case modelTestKindResponses:
			return "openai/responses"
		case modelTestKindRealtime:
			return "openai/realtime/sessions"
		}
	}
	return modelTestKindLabel(kind)
}

// applyChannelModelTestRoute 按渠道类型设置探测用的路由路径与 relay mode，避免 Anthropic/Gemini 原生渠道误用 OpenAI 路径语义。
func applyChannelModelTestRoute(channelType int, spec *modelTestSpec) {
	if spec == nil || spec.Kind == modelTestKindSkip {
		return
	}
	family := channelWireProtocolFamily(channelType)
	spec.Protocol = modelTestWireProtocolLabel(family, spec.Kind)

	switch family {
	case wireProtoAnthropic:
		switch spec.Kind {
		case modelTestKindChat:
			spec.Path = "/v1/messages"
			spec.RelayMode = relaymode.AnthropicMessages
		}
	case wireProtoGeminiNative:
		switch spec.Kind {
		case modelTestKindChat:
			spec.Path = "/v1beta/models/_:generateContent"
			spec.RelayMode = relaymode.GeminiGenerate
		case modelTestKindEmbedding:
			spec.Path = "/v1beta/models/_:batchEmbedContents"
			spec.RelayMode = relaymode.Embeddings
		}
	default:
		switch spec.Kind {
		case modelTestKindResponses:
			spec.Path = "/v1/responses"
			spec.RelayMode = relaymode.OpenAIResponses
		case modelTestKindRealtime:
			spec.Path = "/v1/realtime/sessions"
			spec.RelayMode = relaymode.OpenAIRealtimeSessions
		}
	}
}
