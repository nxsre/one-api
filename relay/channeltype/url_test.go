package channeltype

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChannelDefaultBaseURLsNonEmpty(t *testing.T) {
	Convey("enumerated OEM defaults are non-empty", t, func() {
		for _, typ := range []int{
			OpenAI, PaLM, Anthropic, Baidu, Zhipu, Ali, Xunfei, AI360, Tencent, Gemini,
			Moonshot, Baichuan, Minimax, Mistral, Groq, LingYiWanWu, StepFun, Coze, Cohere,
			DeepSeek, Cloudflare, DeepL, TogetherAI, Doubao, XAI, Replicate, BaiduV2,
			XunfeiV2, AliBailian, AiPPT, AmapPOI, GeminiOpenAICompatible,
		} {
			u := DefaultBaseURL(typ)
			So(u, ShouldNotEqual, "")
			So(u, ShouldStartWith, "http")
		}
	})
}

func TestValidateChannelBaseURL(t *testing.T) {
	Convey("no builtin default requires non-empty base URL", t, func() {
		So(ValidateChannelBaseURL(OpenAICompatible, ""), ShouldNotBeNil)
		So(ValidateChannelBaseURL(OpenAICompatible, "  "), ShouldNotBeNil)
		So(ValidateChannelBaseURL(OpenAICompatible, "https://proxy.example/v1"), ShouldBeNil)
		So(ValidateChannelBaseURL(Azure, ""), ShouldNotBeNil)
		So(ValidateChannelBaseURL(Azure, "https://eastus.openai.azure.com"), ShouldBeNil)
	})
	Convey("builtin default allows empty base URL", t, func() {
		So(ValidateChannelBaseURL(OpenAI, ""), ShouldBeNil)
		So(ValidateChannelBaseURL(OpenAI, "https://cf-worker.example"), ShouldBeNil)
		So(ValidateChannelBaseURL(Anthropic, ""), ShouldBeNil)
		So(ValidateChannelBaseURL(GeminiOpenAICompatible, ""), ShouldBeNil)
		So(ValidateChannelBaseURL(AnthropicCompatible, ""), ShouldNotBeNil)
	})
}
