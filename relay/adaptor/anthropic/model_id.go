package anthropic

import "strings"

// SplitClientModelVariant 解析 Claude Code 模型名，如 claude-opus-4-7[1m] → (claude-opus-4-7, [1m])。
func SplitClientModelVariant(model string) (base, variant string) {
	model = strings.TrimSpace(model)
	i := strings.Index(model, "[")
	if i <= 0 {
		return model, ""
	}
	j := strings.LastIndex(model, "]")
	if j <= i {
		return model, ""
	}
	return strings.TrimSpace(model[:i]), model[i : j+1]
}

// ClientContextVariants 返回应在模型列表中额外暴露的 Claude Code 上下文档位后缀。
func ClientContextVariants(base string) []string {
	lower := strings.ToLower(strings.TrimSpace(base))
	if !strings.HasPrefix(lower, "claude-") || strings.Contains(lower, "haiku") {
		return nil
	}
	if strings.Contains(lower, "-4-") || strings.Contains(lower, "-4.") {
		return []string{"[1m]"}
	}
	return nil
}

// BetaForClientVariant 返回档位对应的 anthropic-beta 特性名。
func BetaForClientVariant(variant string) string {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "[1m]":
		return "context-1m-2025-08-07"
	default:
		return ""
	}
}
