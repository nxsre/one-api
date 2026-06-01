package controller

import (
	"errors"
	"testing"

	"github.com/songquanpeng/one-api/relay/relaymode"
)

func TestClassifyModelTestSpec(t *testing.T) {
	cases := []struct {
		model    string
		kind     modelTestKind
		relay    int
		skip     bool
	}{
		{"gpt-4o", modelTestKindChat, relaymode.ChatCompletions, false},
		{"claude-opus-4-6-thinking", modelTestKindChat, relaymode.ChatCompletions, false},
		{"text-embedding-3-small", modelTestKindEmbedding, relaymode.Embeddings, false},
		{"text-embedding-004", modelTestKindEmbedding, relaymode.Embeddings, false},
		{"gemini-embedding-001", modelTestKindEmbedding, relaymode.Embeddings, false},
		{"gemini-2.5-flash-image", modelTestKindChat, relaymode.ChatCompletions, false},
		{"dall-e-3", modelTestKindImage, relaymode.ImagesGenerations, false},
		{"gpt-image-1", modelTestKindImage, relaymode.ImagesGenerations, false},
		{"tts-1", modelTestKindTTS, relaymode.AudioSpeech, false},
		{"text-moderation-stable", modelTestKindModeration, relaymode.Moderations, false},
		{"whisper-1", modelTestKindSkip, 0, true},
		{"gpt-4o-realtime-preview", modelTestKindRealtime, relaymode.OpenAIRealtimeSessions, false},
		{"gpt-realtime-2", modelTestKindRealtime, relaymode.OpenAIRealtimeSessions, false},
		{"lyria-002", modelTestKindSkip, 0, true},
		{"gpt-5.4-pro", modelTestKindResponses, relaymode.OpenAIResponses, false},
		{"gpt-5-pro", modelTestKindResponses, relaymode.OpenAIResponses, false},
		{"gpt-5.4-pro-2026-03-05", modelTestKindResponses, relaymode.OpenAIResponses, false},
		{"gpt-5.4-mini", modelTestKindChat, relaymode.ChatCompletions, false},
	}
	for _, tc := range cases {
		spec := classifyModelTestSpecByName(tc.model)
		if spec.Kind != tc.kind {
			t.Fatalf("%q kind=%d want=%d", tc.model, spec.Kind, tc.kind)
		}
		if tc.skip {
			if spec.SkipReason == "" {
				t.Fatalf("%q expected skip reason", tc.model)
			}
			continue
		}
		if spec.RelayMode != tc.relay {
			t.Fatalf("%q relay=%d want=%d", tc.model, spec.RelayMode, tc.relay)
		}
	}
}

func TestShouldRecordModelTestBodyDetail(t *testing.T) {
	if !shouldRecordModelTestBodyDetail(modelTestSpec{Kind: modelTestKindChat}, nil) {
		t.Fatal("chat success should keep body")
	}
	if shouldRecordModelTestBodyDetail(modelTestSpec{Kind: modelTestKindEmbedding}, nil) {
		t.Fatal("embedding success should omit body")
	}
	if !shouldRecordModelTestBodyDetail(modelTestSpec{Kind: modelTestKindEmbedding}, errors.New("upstream failed")) {
		t.Fatal("embedding error should keep body")
	}
}

// TestSummarizeModelTestSuccessNonText 锁定：非文本类成功不得把原始响应体写进「说明」字段，
// 只返回固定摘要；文本类仍透传解析后的消息。
func TestSummarizeModelTestSuccessNonText(t *testing.T) {
	rawBody := `{"data":[{"embedding":[0.1,0.2,0.3]}],"usage":{"prompt_tokens":1}}`
	nonText := []struct {
		kind modelTestKind
		want string
	}{
		{modelTestKindEmbedding, "Embedding 接口响应正常"},
		{modelTestKindImage, "图像生成接口响应正常"},
		{modelTestKindTTS, "语音合成接口响应正常"},
		{modelTestKindModeration, "内容审核接口响应正常"},
		{modelTestKindRealtime, "Realtime 会话创建响应正常"},
	}
	for _, tc := range nonText {
		// 即便传入完整原始体，也必须被忽略、回落到固定摘要。
		if got := summarizeModelTestSuccess(modelTestSpec{Kind: tc.kind}, rawBody); got != tc.want {
			t.Fatalf("kind=%d summary=%q want=%q (原始体不应泄漏到说明字段)", tc.kind, got, tc.want)
		}
	}
	// 文本类（responses）仍透传解析后的消息。
	if got := summarizeModelTestSuccess(modelTestSpec{Kind: modelTestKindResponses}, "hello"); got != "hello" {
		t.Fatalf("responses should pass through message, got %q", got)
	}
}
