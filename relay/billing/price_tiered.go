package billing

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/pkg/billingexpr"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/setting/billing_setting"
)

// EstimateTieredPreConsume 为 tiered_expr 模型估算预扣额度并冻结计费快照。
func EstimateTieredPreConsume(c *gin.Context, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int) (preConsumed int64, err error) {
	if meta == nil {
		return 0, fmt.Errorf("meta is nil")
	}
	exprStr, ok := billing_setting.GetBillingExpr(meta.OriginModelName)
	if !ok || exprStr == "" {
		return 0, fmt.Errorf("model %s is tiered_expr but has no billing expression", meta.OriginModelName)
	}
	estimatedCompletion := 0
	if textRequest != nil && textRequest.MaxTokens != 0 {
		estimatedCompletion = textRequest.MaxTokens
	}
	requestInput, err := ResolveIncomingBillingExprRequestInput(c)
	if err != nil {
		return 0, err
	}
	userGroup := meta.UserGroup
	if userGroup == "" {
		userGroup = meta.Group
	}
	usingGroup := meta.UsingGroup
	if usingGroup == "" {
		usingGroup = meta.Group
	}
	groupRatio := billingratio.GetEffectiveGroupRatio(userGroup, usingGroup)

	rawCost, trace, err := billingexpr.RunExprWithRequest(exprStr, billingexpr.TokenParams{
		P:   float64(promptTokens),
		C:   float64(estimatedCompletion),
		Len: float64(promptTokens),
	}, requestInput)
	if err != nil {
		return 0, fmt.Errorf("model %s tiered expr run failed: %w", meta.OriginModelName, err)
	}
	quotaBeforeGroup := rawCost / 1_000_000 * config.QuotaPerUnit
	preConsumedQuota := int64(billingexpr.QuotaRound(quotaBeforeGroup * groupRatio))

	snapshot := &billingexpr.BillingSnapshot{
		BillingMode:               billing_setting.BillingModeTieredExpr,
		ModelName:                 meta.OriginModelName,
		ExprString:                exprStr,
		ExprHash:                  billingexpr.ExprHashString(exprStr),
		GroupRatio:                groupRatio,
		EstimatedPromptTokens:     promptTokens,
		EstimatedCompletionTokens: estimatedCompletion,
		EstimatedQuotaBeforeGroup: quotaBeforeGroup,
		EstimatedQuotaAfterGroup:  int(preConsumedQuota),
		EstimatedTier:             trace.MatchedTier,
		QuotaPerUnit:              config.QuotaPerUnit,
		ExprVersion:               billingexpr.ExprVersion(exprStr),
	}
	c.Set(ctxkey.TieredBillingSnapshot, snapshot)
	c.Set(ctxkey.BillingRequestInput, &requestInput)
	c.Set(ctxkey.FinalPreConsumedQuota, preConsumedQuota)
	return preConsumedQuota, nil
}
