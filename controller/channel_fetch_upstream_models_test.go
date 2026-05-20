package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

func TestJoinAnthropicModelsListURL(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"", "https://api.anthropic.com/v1/models"},
		{"https://api.anthropic.com", "https://api.anthropic.com/v1/models"},
		{"https://api.novita.ai/anthropic", "https://api.novita.ai/anthropic/v1/models"},
		{"https://proxy.example/v1", "https://proxy.example/v1/models"},
		{"https://proxy.example/v1/models", "https://proxy.example/v1/models"},
	}
	for _, tc := range tests {
		if got := joinAnthropicModelsListURL(tc.base); got != tc.want {
			t.Fatalf("joinAnthropicModelsListURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}

func TestResolveChannelListBasePrefersConfiguredBaseURL(t *testing.T) {
	custom := "https://api.novita.ai/anthropic"
	ch := &model.Channel{
		Type:    channeltype.Anthropic,
		BaseURL: &custom,
	}
	if got := resolveChannelListBase(ch); got != custom {
		t.Fatalf("resolveChannelListBase() = %q, want %q", got, custom)
	}
}

func TestResolveChannelListBaseFallsBackToDefault(t *testing.T) {
	ch := &model.Channel{Type: channeltype.Anthropic}
	if got := resolveChannelListBase(ch); got != "https://api.anthropic.com" {
		t.Fatalf("resolveChannelListBase() = %q, want official default", got)
	}
}
