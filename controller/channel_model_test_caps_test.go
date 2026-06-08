package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func TestResolveModelTestSpecChannelCaps(t *testing.T) {
	spec := resolveModelTestSpec("text-embedding-3-small", channeltype.Anthropic)
	if spec.Kind != modelTestKindSkip {
		t.Fatalf("anthropic+embedding should skip, got kind=%d", spec.Kind)
	}
	if spec.SkipReason == "" {
		t.Fatal("expected skip reason")
	}

	spec = resolveModelTestSpec("text-embedding-3-small", channeltype.OpenAI)
	if spec.Kind != modelTestKindEmbedding || spec.RelayMode != relaymode.Embeddings {
		t.Fatalf("openai+embedding want embedding, got %+v", spec)
	}

	spec = resolveModelTestSpec("gpt-4o", channeltype.Anthropic)
	if spec.Kind != modelTestKindChat {
		t.Fatalf("anthropic+gpt want chat, got kind=%d", spec.Kind)
	}

	spec = resolveModelTestSpec("dall-e-3", channeltype.Anthropic)
	if spec.Kind != modelTestKindSkip {
		t.Fatalf("anthropic+image should skip")
	}

	spec = resolveModelTestSpec("text-embedding-004", channeltype.Gemini)
	if spec.Kind != modelTestKindEmbedding {
		t.Fatalf("gemini+embedding want embedding, got kind=%d", spec.Kind)
	}
}
