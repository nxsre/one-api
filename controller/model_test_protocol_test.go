package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func TestApplyChannelModelTestRoute(t *testing.T) {
	cases := []struct {
		name      string
		channel   int
		model     string
		wantPath  string
		wantMode  int
		wantProto string
	}{
		{
			name:      "openai chat",
			channel:   channeltype.OpenAI,
			model:     "gpt-4o",
			wantPath:  "/v1/chat/completions",
			wantMode:  relaymode.ChatCompletions,
			wantProto: "openai/chat-completions",
		},
		{
			name:      "anthropic chat",
			channel:   channeltype.Anthropic,
			model:     "claude-opus-4-6",
			wantPath:  "/v1/messages",
			wantMode:  relaymode.AnthropicMessages,
			wantProto: "anthropic/messages",
		},
		{
			name:      "anthropic compatible",
			channel:   channeltype.AnthropicCompatible,
			model:     "claude-sonnet-4",
			wantPath:  "/v1/messages",
			wantMode:  relaymode.AnthropicMessages,
			wantProto: "anthropic/messages",
		},
		{
			name:      "gemini native chat",
			channel:   channeltype.GeminiNativeCompatible,
			model:     "gemini-2.5-flash",
			wantPath:  "/v1beta/models/_:generateContent",
			wantMode:  relaymode.GeminiGenerate,
			wantProto: "gemini/generateContent",
		},
		{
			name:      "gemini official native",
			channel:   channeltype.Gemini,
			model:     "gemini-2.5-flash",
			wantPath:  "/v1beta/models/_:generateContent",
			wantMode:  relaymode.GeminiGenerate,
			wantProto: "gemini/generateContent",
		},
		{
			name:      "gemini native embedding",
			channel:   channeltype.Gemini,
			model:     "text-embedding-004",
			wantPath:  "/v1beta/models/_:batchEmbedContents",
			wantMode:  relaymode.Embeddings,
			wantProto: "gemini/batchEmbedContents",
		},
		{
			name:      "gemini openai compatible chat",
			channel:   channeltype.GeminiOpenAICompatible,
			model:     "gemini-2.5-flash",
			wantPath:  "/v1/chat/completions",
			wantMode:  relaymode.ChatCompletions,
			wantProto: "openai/chat-completions (gemini-compat)",
		},
		{
			name:      "gemini openai shape proxy",
			channel:   channeltype.GeminiCompatible,
			model:     "gemini-2.5-flash",
			wantPath:  "/v1/chat/completions",
			wantMode:  relaymode.ChatCompletions,
			wantProto: "openai/chat-completions (gemini-compat)",
		},
		{
			name:      "openai responses only model",
			channel:   channeltype.OpenAI,
			model:     "gpt-5.4-pro",
			wantPath:  "/v1/responses",
			wantMode:  relaymode.OpenAIResponses,
			wantProto: "openai/responses",
		},
		{
			name:      "openai realtime model",
			channel:   channeltype.OpenAI,
			model:     "gpt-realtime-2",
			wantPath:  "/v1/realtime/sessions",
			wantMode:  relaymode.OpenAIRealtimeSessions,
			wantProto: "openai/realtime/sessions",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := resolveModelTestSpec(tc.model, tc.channel)
			if spec.Kind == modelTestKindSkip {
				t.Fatalf("unexpected skip: %s", spec.SkipReason)
			}
			if spec.Path != tc.wantPath {
				t.Fatalf("path want %q got %q", tc.wantPath, spec.Path)
			}
			if spec.RelayMode != tc.wantMode {
				t.Fatalf("relay mode want %d got %d", tc.wantMode, spec.RelayMode)
			}
			if spec.Protocol != tc.wantProto {
				t.Fatalf("protocol want %q got %q", tc.wantProto, spec.Protocol)
			}
		})
	}
}

func TestResolveModelTestSpecAnthropicSkipsEmbedding(t *testing.T) {
	spec := resolveModelTestSpec("text-embedding-3-small", channeltype.Anthropic)
	if spec.Kind != modelTestKindSkip {
		t.Fatalf("expected skip")
	}
}

func TestResolveModelTestSpecResponsesOnlyOnGeminiSkips(t *testing.T) {
	spec := resolveModelTestSpec("gpt-5.4-pro", channeltype.GeminiNativeCompatible)
	if spec.Kind != modelTestKindSkip {
		t.Fatalf("gemini channel should skip responses-only model, got %+v", spec)
	}
}
