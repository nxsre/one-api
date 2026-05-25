package controller

import (
	"errors"
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
)

func TestVendorProbeSupported(t *testing.T) {
	cases := []struct {
		channelType int
		kind        modelTestKind
		want        bool
	}{
		{channeltype.Anthropic, modelTestKindChat, true},
		{channeltype.Anthropic, modelTestKindEmbedding, false},
		{channeltype.AwsClaude, modelTestKindChat, false},
		{channeltype.Gemini, modelTestKindChat, true},
		{channeltype.Gemini, modelTestKindEmbedding, true},
		{channeltype.GeminiOpenAICompatible, modelTestKindChat, true},
		{channeltype.OpenAI, modelTestKindResponses, true},
		{channeltype.OpenAI, modelTestKindRealtime, false},
		{channeltype.OpenAI, modelTestKindTTS, false},
		{channeltype.DeepSeek, modelTestKindChat, true},
	}
	for _, tc := range cases {
		spec := modelTestSpec{Kind: tc.kind}
		got := vendorProbeSupported(tc.channelType, spec)
		if got != tc.want {
			t.Errorf("channelType=%d kind=%v: got %v want %v", tc.channelType, tc.kind, got, tc.want)
		}
	}
}

func TestVendorProbeRequestURL(t *testing.T) {
	got := vendorProbeRequestURL("https://generativelanguage.googleapis.com", "/v1beta/models/gemini-2.5-flash:generateContent")
	want := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestOpenAIVendorSDKBaseURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "append v1 for host only",
			in:   "https://www.anyfast.ai",
			want: "https://www.anyfast.ai/v1",
		},
		{
			name: "keep existing v1",
			in:   "https://api.openai.com/v1",
			want: "https://api.openai.com/v1",
		},
		{
			name: "keep v1beta style",
			in:   "https://generativelanguage.googleapis.com/v1beta/openai",
			want: "https://generativelanguage.googleapis.com/v1beta/openai",
		},
		{
			name: "trim trailing slash then append",
			in:   "https://api.deepseek.com/",
			want: "https://api.deepseek.com/v1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := openAIVendorSDKBaseURL(tc.in)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestVendorProbeFallbackStatusFromErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "extract status code",
			err:  errors.New("upstream http status code: 403 forbidden"),
			want: 403,
		},
		{
			name: "no status context",
			err:  errors.New("network timeout"),
			want: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := vendorProbeFallbackStatusFromErr(tc.err)
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestFinishOpenAIVendorProbeUsesErrorAsResponseBody(t *testing.T) {
	errText := "expected destination type of 'string' or '[]byte' for responses with content-type 'text/html; charset=utf-8'"
	_, detail, err := finishOpenAIVendorProbe(
		"https://www.anyfast.ai",
		"/v1/chat/completions",
		"openai/chat-completions",
		map[string]any{"model": "gpt-5.5"},
		"",
		errors.New(errText),
		modelTestSpec{Kind: modelTestKindChat},
		"",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if detail.ResponseBody == "" {
		t.Fatal("expected non-empty response body on error")
	}
	if detail.ResponseBody != errText {
		t.Fatalf("response body mismatch: got %q want %q", detail.ResponseBody, errText)
	}
}

func TestFinishOpenAIVendorProbeOmitsBodyOnNonTextSuccess(t *testing.T) {
	_, detail, err := finishOpenAIVendorProbe(
		"https://api.example.com",
		"/v1/embeddings",
		"openai/embeddings",
		map[string]any{"model": "text-embedding-3-small", "input": "hello"},
		`{"data":[{"embedding":[0.1,0.2]}]}`,
		nil,
		modelTestSpec{Kind: modelTestKindEmbedding},
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.RequestBody != "" || detail.ResponseBody != "" {
		t.Fatalf("non-text success should omit bodies, got request=%q response=%q", detail.RequestBody, detail.ResponseBody)
	}
}
