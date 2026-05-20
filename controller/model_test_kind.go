package controller

import (
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type modelTestKind int

const (
	modelTestKindChat modelTestKind = iota
	modelTestKindEmbedding
	modelTestKindImage
	modelTestKindTTS
	modelTestKindModeration
	modelTestKindSkip
)

type modelTestSpec struct {
	Kind       modelTestKind
	RelayMode  int
	Path       string
	SkipReason string
}

func classifyModelTestSpec(modelName string) modelTestSpec {
	m := strings.ToLower(strings.TrimSpace(modelName))
	if m == "" {
		return modelTestSpec{
			Kind:      modelTestKindChat,
			RelayMode: relaymode.ChatCompletions,
			Path:      "/v1/chat/completions",
		}
	}

	for _, rule := range modelTestSkipRules {
		if rule.match(m) {
			return modelTestSpec{
				Kind:       modelTestKindSkip,
				SkipReason: rule.reason,
			}
		}
	}

	if isEmbeddingTestModel(m) {
		return modelTestSpec{
			Kind:      modelTestKindEmbedding,
			RelayMode: relaymode.Embeddings,
			Path:      "/v1/embeddings",
		}
	}
	if isModerationTestModel(m) {
		return modelTestSpec{
			Kind:      modelTestKindModeration,
			RelayMode: relaymode.Moderations,
			Path:      "/v1/moderations",
		}
	}
	if isTTSTestModel(m) {
		return modelTestSpec{
			Kind:      modelTestKindTTS,
			RelayMode: relaymode.AudioSpeech,
			Path:      "/v1/audio/speech",
		}
	}
	if isImageTestModel(m) {
		return modelTestSpec{
			Kind:      modelTestKindImage,
			RelayMode: relaymode.ImagesGenerations,
			Path:      "/v1/images/generations",
		}
	}

	return modelTestSpec{
		Kind:      modelTestKindChat,
		RelayMode: relaymode.ChatCompletions,
		Path:      "/v1/chat/completions",
	}
}

type modelTestSkipRule struct {
	match  func(string) bool
	reason string
}

var modelTestSkipRules = []modelTestSkipRule{
	{
		match:  func(m string) bool { return strings.Contains(m, "whisper") },
		reason: "语音转写模型需上传音频，跳过自动探测",
	},
	{
		match:  func(m string) bool { return strings.Contains(m, "transcribe") || strings.Contains(m, "transcription") },
		reason: "语音转写模型需上传音频，跳过自动探测",
	},
	{
		match:  func(m string) bool { return strings.Contains(m, "diarize") },
		reason: "说话人分离模型需上传音频，跳过自动探测",
	},
	{
		match:  func(m string) bool { return strings.Contains(m, "realtime") },
		reason: "Realtime 模型需 WebSocket，跳过自动探测",
	},
	{
		match:  func(m string) bool { return strings.Contains(m, "lyria") || strings.Contains(m, "musicgen") || strings.Contains(m, "suno") },
		reason: "音乐生成模型跳过自动探测",
	},
}

func isEmbeddingTestModel(m string) bool {
	if strings.Contains(m, "embedding") || strings.Contains(m, "embed") {
		// gemini-2.5-flash-image 等含 embed 子串的误判较少；image 已在后面单独匹配
		if strings.Contains(m, "-image") || strings.HasSuffix(m, "image") {
			return false
		}
		return true
	}
	return false
}

func isModerationTestModel(m string) bool {
	return strings.Contains(m, "moderation")
}

func isTTSTestModel(m string) bool {
	return m == "tts-1" || m == "tts-1-hd" || strings.HasPrefix(m, "tts-") || strings.HasSuffix(m, "-tts")
}

func isImageTestModel(m string) bool {
	if strings.Contains(m, "gemini-") && (strings.Contains(m, "-image") || strings.HasSuffix(m, "image")) {
		// Gemini 图像模型走 generateContent，不按 OpenAI images API 测
		return false
	}
	switch {
	case strings.Contains(m, "dall-e"), strings.Contains(m, "dall_e"):
		return true
	case strings.HasPrefix(m, "gpt-image"):
		return true
	case strings.Contains(m, "cogview"):
		return true
	case strings.Contains(m, "stable-diffusion"), strings.Contains(m, "stable_diffusion"):
		return true
	case strings.HasPrefix(m, "flux-"), strings.Contains(m, "flux/"):
		return true
	case strings.Contains(m, "midjourney"), strings.Contains(m, "recraft"), strings.Contains(m, "ideogram"):
		return true
	default:
		return false
	}
}

func modelTestKindLabel(kind modelTestKind) string {
	switch kind {
	case modelTestKindEmbedding:
		return "embedding"
	case modelTestKindImage:
		return "image"
	case modelTestKindTTS:
		return "tts"
	case modelTestKindModeration:
		return "moderation"
	case modelTestKindSkip:
		return "skip"
	default:
		return "chat"
	}
}

func buildChatTestRequest(model string) *relaymodel.GeneralOpenAIRequest {
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &relaymodel.GeneralOpenAIRequest{
		Model: model,
		Messages: []relaymodel.Message{
			{Role: "user", Content: config.TestPrompt},
		},
	}
}

func buildEmbeddingTestRequest(model string) *relaymodel.GeneralOpenAIRequest {
	return &relaymodel.GeneralOpenAIRequest{
		Model: model,
		Input: "hello",
	}
}

func buildModerationTestRequest(model string) *relaymodel.GeneralOpenAIRequest {
	req := &relaymodel.GeneralOpenAIRequest{Input: config.TestPrompt}
	if strings.TrimSpace(model) != "" && !strings.Contains(strings.ToLower(model), "stable") {
		req.Model = model
	}
	return req
}

func buildImageTestRequest(model string) *relaymodel.ImageRequest {
	size := "1024x1024"
	if strings.Contains(strings.ToLower(model), "dall-e-2") || model == "dall-e-2" {
		size = "256x256"
	}
	return &relaymodel.ImageRequest{
		Model:  model,
		Prompt: "a red circle",
		N:      1,
		Size:   size,
	}
}

func buildTTSTestRequest(model string) openai.TextToSpeechRequest {
	return openai.TextToSpeechRequest{
		Model:          model,
		Input:          "test",
		Voice:          "alloy",
		ResponseFormat: "mp3",
	}
}

func summarizeModelTestSuccess(spec modelTestSpec, responseMessage string) string {
	if strings.TrimSpace(responseMessage) != "" {
		return responseMessage
	}
	switch spec.Kind {
	case modelTestKindEmbedding:
		return "Embedding 接口响应正常"
	case modelTestKindImage:
		return "图像生成接口响应正常"
	case modelTestKindTTS:
		return "语音合成接口响应正常"
	case modelTestKindModeration:
		return "内容审核接口响应正常"
	default:
		return "接口响应正常"
	}
}
