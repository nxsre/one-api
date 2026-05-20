package anthropic

import (
	"github.com/songquanpeng/one-api/relay/model"
)

// claudeUsageToRelayUsage 将 Anthropic usage 转为统一 Relay Usage（含缓存明细）。
func claudeUsageToRelayUsage(u Usage) *model.Usage {
	out := &model.Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.InputTokens + u.OutputTokens,
		UsageSemantic:    "anthropic",
		PromptTokensDetails: &model.PromptTokensDetails{
			CachedTokens:         u.CacheReadInputTokens,
			CachedCreationTokens: u.CacheCreationInputTokens,
		},
	}
	if u.CacheCreation != nil {
		out.ClaudeCacheCreation5mTokens = u.CacheCreation.Ephemeral5mInputTokens
		out.ClaudeCacheCreation1hTokens = u.CacheCreation.Ephemeral1hInputTokens
		if out.PromptTokensDetails.CachedCreationTokens == 0 {
			out.PromptTokensDetails.CachedCreationTokens = u.CacheCreation.Ephemeral5mInputTokens + u.CacheCreation.Ephemeral1hInputTokens
		}
	}
	return out
}
