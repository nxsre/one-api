package billing

import (
	"testing"

	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func TestComputeTokenQuota_WithCacheTokens(t *testing.T) {
	billingratio.UpdateModelRatioByJSONString(`{"gpt-4":2}`)
	billingratio.UpdateCompletionRatioByJSONString(`{"gpt-4":3}`)
	billingratio.UpdateCacheRatioByJSONString(`{"gpt-4":0.5}`)

	usage := &relaymodel.Usage{
		PromptTokens:     1000,
		CompletionTokens: 100,
		PromptTokensDetails: &relaymodel.PromptTokensDetails{
			CachedTokens: 400,
		},
	}
	mr := billingratio.GetModelRatio("gpt-4", "gpt-4", 0)
	cr, _ := billingratio.GetCacheRatio("gpt-4", "gpt-4", 0)
	if mr != 2 {
		t.Fatalf("model ratio setup failed: %v", mr)
	}
	result := ComputeTokenQuota(usage, TokenQuotaParams{
		OriginModel:    "gpt-4",
		MappedModel:    "gpt-4",
		UserGroup:      "quota-test-user-grp",
		UsingGroup:     "quota-test-using-grp",
		TenantDiscount: 1,
	})
	// base 600 + cache 400*0.5 = 800 prompt-equiv; + completion 100*3 = 1100; * model 2 * group 1
	if result.Quota != 2200 {
		t.Fatalf("expected quota 2200, got %d (mr=%v cr=%v usePrice=%v)", result.Quota, result.ModelRatio, cr, result.UsePrice)
	}
}
