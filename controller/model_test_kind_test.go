package controller

import (
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
		{"gpt-4o-realtime-preview", modelTestKindSkip, 0, true},
		{"lyria-002", modelTestKindSkip, 0, true},
	}
	for _, tc := range cases {
		spec := classifyModelTestSpec(tc.model)
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
