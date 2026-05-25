package gemini

import (
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

// DefaultAPIVersion 解析 Gemini 原生 API 版本（v1 / v1beta），与拉取上游模型列表逻辑对齐。
func DefaultAPIVersion(channelType int, modelName string) string {
	if channelType == channeltype.GeminiNativeCompatible {
		return "v1beta"
	}
	if modelPrefersV1Beta(modelName) {
		return "v1beta"
	}
	if v := strings.TrimSpace(config.GeminiVersion); v != "" {
		return v
	}
	return "v1"
}

// DefaultAPIVersionForChannel 无具体模型时（如 ListModels）的默认版本。
func DefaultAPIVersionForChannel(channelType int) string {
	if channelType == channeltype.GeminiNativeCompatible {
		return "v1beta"
	}
	if v := strings.TrimSpace(config.GeminiVersion); v != "" {
		return v
	}
	return "v1beta"
}

func modelPrefersV1Beta(modelName string) bool {
	m := strings.ToLower(strings.TrimSpace(modelName))
	if m == "" {
		return false
	}
	for _, prefix := range []string{
		"gemini-1.5", "gemini-2.0", "gemini-2.5", "gemini-3.", "gemini-3-",
	} {
		if strings.Contains(m, prefix) {
			return true
		}
	}
	if strings.Contains(m, "preview") || strings.Contains(m, "-exp") || strings.HasSuffix(m, "-exp") {
		return true
	}
	return false
}
