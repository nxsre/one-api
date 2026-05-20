package ratio

import "strings"

// upstreamModelPrefixes 渠道 model_mapping 常见上游别名前缀；计费与运营设置应使用映射前的官方模型名。
var upstreamModelPrefixes = []string{
	"pa/",
	"anthropic/",
}

// BillingModelName 返回用于 ModelRatio / CompletionRatio 查表的模型名。
// 优先使用用户请求的 originModel；若仅有映射后的 mappedModel，则剥掉上游别名前缀（如 pa/claude-sonnet-4-6 → claude-sonnet-4-6）。
func BillingModelName(originModel, mappedModel string) string {
	if s := strings.TrimSpace(originModel); s != "" {
		return s
	}
	return stripUpstreamModelPrefix(strings.TrimSpace(mappedModel))
}

func stripUpstreamModelPrefix(name string) string {
	for name != "" {
		stripped := false
		lower := strings.ToLower(name)
		for _, p := range upstreamModelPrefixes {
			if strings.HasPrefix(lower, p) {
				name = name[len(p):]
				stripped = true
				break
			}
		}
		if !stripped {
			return name
		}
	}
	return name
}
