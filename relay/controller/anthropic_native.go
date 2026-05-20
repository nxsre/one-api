package controller

import (
	"strings"

	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
)

// channelUsesAnthropicNativePassthrough 判断是否应对渠道走 Anthropic Messages 原生透传。
// 兼容「类型填错但 Base URL 为 …/anthropic」的 Novita 等网关（如 type=Gemini + novita anthropic URL）。
func channelUsesAnthropicNativePassthrough(m *meta.Meta) bool {
	if m == nil {
		return false
	}
	switch m.ChannelType {
	case channeltype.Anthropic, channeltype.AnthropicCompatible:
		return true
	}
	base := strings.ToLower(strings.TrimSpace(m.BaseURL))
	return strings.Contains(base, "/anthropic") || strings.HasSuffix(base, "anthropic")
}

func anthropicNativeAPIType(m *meta.Meta) int {
	if channelUsesAnthropicNativePassthrough(m) {
		return apitype.Anthropic
	}
	return m.APIType
}
