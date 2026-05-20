package billing

import (
	"fmt"
	"math"

	"github.com/songquanpeng/one-api/common/config"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// Claude 1h 缓存创建相对 5m 倍率（与 new-api 一致）。
const claudeCacheCreation1hMultiplier = 6.0 / 3.75

// TokenQuotaParams 文本类 Relay 扣费参数。
type TokenQuotaParams struct {
	OriginModel     string
	MappedModel     string
	ChannelType     int
	UserGroup       string
	UsingGroup      string
	TenantDiscount  float64
	AdditionalQuota int64
	OtherRatios     []float64
}

// TokenQuotaResult 扣费结果。
type TokenQuotaResult struct {
	Quota                 int64
	ModelRatio            float64
	GroupRatio            float64
	CompletionRatio       float64
	CacheRatio            float64
	CreateCacheRatio      float64
	CreateCacheRatio5m    float64
	CreateCacheRatio1h    float64
	ImageRatio            float64
	AudioRatio            float64
	AudioCompletionRatio  float64
	UsePrice              bool
	ModelPrice            float64
}

// ComputeTokenQuota 按 new-api 风格计算 token 额度（含缓存/图片/音频；租户规则由调用方叠加）。
func ComputeTokenQuota(usage *relaymodel.Usage, params TokenQuotaParams) TokenQuotaResult {
	out := TokenQuotaResult{
		GroupRatio: billingratio.GetEffectiveGroupRatio(params.UserGroup, params.UsingGroup),
	}
	tenantDiscount := params.TenantDiscount
	if tenantDiscount <= 0 {
		tenantDiscount = 1
	}
	if usage == nil {
		return out
	}

	if price, ok := billingratio.GetModelPrice(params.OriginModel, params.MappedModel, params.ChannelType); ok && price >= 0 {
		out.UsePrice = true
		out.ModelPrice = price
		quota := price * config.QuotaPerUnit * out.GroupRatio * tenantDiscount
		out.Quota = int64(math.Ceil(quota))
		if out.Quota <= 0 && quota > 0 {
			out.Quota = 1
		}
		out.Quota = applyOtherRatios(out.Quota, params.OtherRatios)
		out.Quota += params.AdditionalQuota
		return out
	}

	out.ModelRatio = billingratio.GetModelRatio(params.OriginModel, params.MappedModel, params.ChannelType)
	out.CompletionRatio = billingratio.GetCompletionRatio(params.OriginModel, params.MappedModel, params.ChannelType)
	out.CacheRatio, _ = billingratio.GetCacheRatio(params.OriginModel, params.MappedModel, params.ChannelType)
	out.CreateCacheRatio, _ = billingratio.GetCreateCacheRatio(params.OriginModel, params.MappedModel, params.ChannelType)
	out.CreateCacheRatio5m = out.CreateCacheRatio
	out.CreateCacheRatio1h = out.CreateCacheRatio * claudeCacheCreation1hMultiplier
	out.ImageRatio, _ = billingratio.GetImageRatio(params.OriginModel, params.MappedModel, params.ChannelType)
	out.AudioRatio = billingratio.GetAudioRatio(params.OriginModel, params.MappedModel, params.ChannelType)
	out.AudioCompletionRatio = billingratio.GetAudioCompletionRatio(params.OriginModel, params.MappedModel, params.ChannelType)

	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens
	cacheTokens := 0
	cacheCreationTokens := 0
	cacheCreation5m := 0
	cacheCreation1h := 0
	imageTokens := 0
	audioTokens := 0
	audioOutTokens := 0
	imageOutTokens := 0
	if usage.PromptTokensDetails != nil {
		cacheTokens = usage.PromptTokensDetails.CachedTokens
		cacheCreationTokens = usage.PromptTokensDetails.CachedCreationTokens
		imageTokens = usage.PromptTokensDetails.ImageTokens
		audioTokens = usage.PromptTokensDetails.AudioTokens
	}
	if usage.UsageSemantic == "anthropic" {
		cacheCreation5m = usage.ClaudeCacheCreation5mTokens
		cacheCreation1h = usage.ClaudeCacheCreation1hTokens
	}
	if usage.CompletionTokensDetails != nil {
		audioOutTokens = usage.CompletionTokensDetails.AudioTokens
		imageOutTokens = usage.CompletionTokensDetails.ImageTokens
	}

	isClaude := usage.UsageSemantic == "anthropic"
	basePrompt := float64(promptTokens)
	baseCompletion := float64(completionTokens)

	if !isClaude {
		if cacheTokens > 0 {
			basePrompt -= float64(cacheTokens)
		}
		if cacheCreationTokens > 0 {
			basePrompt -= float64(cacheCreationTokens)
		}
		if imageTokens > 0 {
			basePrompt -= float64(imageTokens)
		}
		if audioTokens > 0 {
			basePrompt -= float64(audioTokens)
		}
		if imageOutTokens > 0 {
			baseCompletion -= float64(imageOutTokens)
		}
		if audioOutTokens > 0 {
			baseCompletion -= float64(audioOutTokens)
		}
	}
	if basePrompt < 0 {
		basePrompt = 0
	}
	if baseCompletion < 0 {
		baseCompletion = 0
	}

	ratio := out.ModelRatio * out.GroupRatio * tenantDiscount
	promptPart := basePrompt
	if cacheTokens > 0 {
		promptPart += float64(cacheTokens) * out.CacheRatio
	}
	if isClaude && (cacheCreation5m > 0 || cacheCreation1h > 0) {
		remaining := cacheCreationTokens - cacheCreation5m - cacheCreation1h
		if remaining < 0 {
			remaining = 0
		}
		promptPart += float64(remaining) * out.CreateCacheRatio
		promptPart += float64(cacheCreation5m) * out.CreateCacheRatio5m
		promptPart += float64(cacheCreation1h) * out.CreateCacheRatio1h
	} else if cacheCreationTokens > 0 {
		promptPart += float64(cacheCreationTokens) * out.CreateCacheRatio
	}
	if imageTokens > 0 {
		promptPart += float64(imageTokens) * out.ImageRatio
	}
	if audioTokens > 0 {
		promptPart += float64(audioTokens) * out.AudioRatio
	}
	completionPart := baseCompletion * out.CompletionRatio
	if audioOutTokens > 0 {
		completionPart += float64(audioOutTokens) * out.AudioRatio * out.AudioCompletionRatio
	}
	if imageOutTokens > 0 {
		completionPart += float64(imageOutTokens) * out.ImageRatio
	}

	quota := (promptPart + completionPart) * ratio
	out.Quota = int64(math.Ceil(quota))
	if ratio != 0 && out.Quota <= 0 && (promptTokens+completionTokens) > 0 {
		out.Quota = 1
	}
	out.Quota = applyOtherRatios(out.Quota, params.OtherRatios)
	out.Quota += params.AdditionalQuota
	if promptTokens+completionTokens == 0 {
		out.Quota = params.AdditionalQuota
	}
	return out
}

func applyOtherRatios(quota int64, others []float64) int64 {
	if len(others) == 0 {
		return quota
	}
	f := float64(quota)
	for _, r := range others {
		if r <= 0 {
			continue
		}
		f *= r
	}
	if f <= 0 && quota > 0 {
		return quota
	}
	return int64(math.Ceil(f))
}

// FormatConsumeLogRatioLine 生成消费日志中的倍率行。
func FormatConsumeLogRatioLine(r TokenQuotaResult) string {
	if r.UsePrice {
		return fmt.Sprintf("价格：$%.4f × %.2f", r.ModelPrice, r.GroupRatio)
	}
	return fmt.Sprintf("倍率：%.2f × %.2f × %.2f", r.ModelRatio, r.GroupRatio, r.CompletionRatio)
}
