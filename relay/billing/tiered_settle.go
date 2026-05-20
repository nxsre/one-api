package billing

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/pkg/billingexpr"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// BuildTieredTokenParams 从 Usage 构造分层表达式计费参数（与 new-api 一致）。
func BuildTieredTokenParams(usage *relaymodel.Usage, isClaudeUsageSemantic bool, usedVars map[string]bool) billingexpr.TokenParams {
	if usage == nil {
		return billingexpr.TokenParams{}
	}
	p := float64(usage.PromptTokens)
	c := float64(usage.CompletionTokens)
	cr, cc5m, cc1h, img, ai, imgO, ao := float64(0), float64(0), float64(0), float64(0), float64(0), float64(0), float64(0)
	if usage.PromptTokensDetails != nil {
		cr = float64(usage.PromptTokensDetails.CachedTokens)
		cc5m = float64(usage.PromptTokensDetails.CachedCreationTokens)
		img = float64(usage.PromptTokensDetails.ImageTokens)
		ai = float64(usage.PromptTokensDetails.AudioTokens)
	}
	if usage.UsageSemantic == "anthropic" {
		cc1h = float64(usage.ClaudeCacheCreation1hTokens)
		cc5m = float64(usage.ClaudeCacheCreation5mTokens)
	}
	if usage.CompletionTokensDetails != nil {
		imgO = float64(usage.CompletionTokensDetails.ImageTokens)
		ao = float64(usage.CompletionTokensDetails.AudioTokens)
	}
	inputLen := p
	if isClaudeUsageSemantic {
		inputLen = p + cr + cc5m + cc1h
	}
	if !isClaudeUsageSemantic && usedVars != nil {
		if usedVars["cr"] {
			p -= cr
		}
		if usedVars["cc"] {
			p -= cc5m
		}
		if usedVars["cc1h"] {
			p -= cc1h
		}
		if usedVars["img"] {
			p -= img
		}
		if usedVars["ai"] {
			p -= ai
		}
		if usedVars["img_o"] {
			c -= imgO
		}
		if usedVars["ao"] {
			c -= ao
		}
	}
	if p < 0 {
		p = 0
	}
	if c < 0 {
		c = 0
	}
	return billingexpr.TokenParams{
		P: p, C: c, Len: inputLen,
		CR: cr, CC: cc5m, CC1h: cc1h,
		Img: img, ImgO: imgO, AI: ai, AO: ao,
	}
}

// TryTieredSettle 若请求使用 tiered_expr 计费则结算额度。
func TryTieredSettle(c *gin.Context, usage *relaymodel.Usage) (ok bool, quota int64, result *billingexpr.TieredResult) {
	if c == nil {
		return false, 0, nil
	}
	raw, exists := c.Get(ctxkey.TieredBillingSnapshot)
	if !exists || raw == nil {
		return false, 0, nil
	}
	snap, okSnap := raw.(*billingexpr.BillingSnapshot)
	if !okSnap || snap == nil || snap.BillingMode != "tiered_expr" {
		return false, 0, nil
	}
	usedVars := billingexpr.UsedVars(snap.ExprString)
	isClaude := usage != nil && usage.UsageSemantic == "anthropic"
	params := BuildTieredTokenParams(usage, isClaude, usedVars)

	requestInput := billingexpr.RequestInput{}
	if rawIn, ok := c.Get(ctxkey.BillingRequestInput); ok && rawIn != nil {
		if in, ok := rawIn.(*billingexpr.RequestInput); ok && in != nil {
			requestInput = *in
		}
	}

	tr, err := billingexpr.ComputeTieredQuotaWithRequest(snap, params, requestInput)
	if err != nil {
		fallback := int64(0)
		if v, ok := c.Get(ctxkey.FinalPreConsumedQuota); ok {
			if q, ok := v.(int64); ok {
				fallback = q
			}
		}
		if fallback <= 0 {
			fallback = int64(snap.EstimatedQuotaAfterGroup)
		}
		return true, fallback, nil
	}
	return true, int64(tr.ActualQuotaAfterGroup), &tr
}
