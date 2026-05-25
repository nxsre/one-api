package gemini

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
)

func TestDefaultAPIVersion(t *testing.T) {
	cases := []struct {
		channelType int
		model       string
		want        string
	}{
		{channeltype.GeminiNativeCompatible, "gemini-3.1-flash-lite-preview", "v1beta"},
		{channeltype.GeminiNativeCompatible, "gemini-pro", "v1beta"},
		{channeltype.Gemini, "gemini-3.1-flash-lite-preview", "v1beta"},
		{channeltype.Gemini, "gemini-2.0-flash", "v1beta"},
		{channeltype.Gemini, "gemini-1.5-flash", "v1beta"},
		{channeltype.Gemini, "gemini-pro", "v1"},
	}
	for _, tc := range cases {
		got := DefaultAPIVersion(tc.channelType, tc.model)
		if got != tc.want {
			t.Fatalf("channel=%d model=%q want %q got %q", tc.channelType, tc.model, tc.want, got)
		}
	}
}

func TestDefaultAPIVersionForChannel(t *testing.T) {
	if DefaultAPIVersionForChannel(channeltype.GeminiNativeCompatible) != "v1beta" {
		t.Fatal("native compatible should default v1beta")
	}
}
