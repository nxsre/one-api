package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/relay/constant/role"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/metrics"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller/validator"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"gorm.io/gorm"
)

func getAndValidateTextRequest(c *gin.Context, relayMode int) (*relaymodel.GeneralOpenAIRequest, error) {
	textRequest := &relaymodel.GeneralOpenAIRequest{}
	err := common.UnmarshalBodyReusable(c, textRequest)
	if err != nil {
		if relayMode == relaymode.OpenAIResponses || relayMode == relaymode.OpenAIRealtimeSessions {
			if rm := strings.TrimSpace(c.GetString(ctxkey.RequestModel)); rm != "" {
				textRequest = &relaymodel.GeneralOpenAIRequest{Model: rm}
				err = nil
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if relayMode == relaymode.Moderations && textRequest.Model == "" {
		textRequest.Model = "text-moderation-latest"
	}
	if relayMode == relaymode.Embeddings && textRequest.Model == "" {
		textRequest.Model = c.Param("model")
	}
	err = validator.ValidateTextRequest(textRequest, relayMode)
	if err != nil {
		return nil, err
	}
	return textRequest, nil
}

func getPromptTokens(textRequest *relaymodel.GeneralOpenAIRequest, relayMode int) int {
	switch relayMode {
	case relaymode.OpenAIResponses:
		if len(textRequest.Messages) > 0 {
			return openai.CountTokenMessages(textRequest.Messages, textRequest.Model)
		}
		raw, _ := json.Marshal(textRequest)
		n := len(raw) / 4
		if n < 128 {
			n = 128
		}
		return n
	case relaymode.OpenAIRealtimeSessions:
		raw, _ := json.Marshal(textRequest)
		n := len(raw)/4 + 64
		if n < 96 {
			n = 96
		}
		return n
	case relaymode.ChatCompletions:
		return openai.CountTokenMessages(textRequest.Messages, textRequest.Model)
	case relaymode.Completions:
		return openai.CountTokenInput(textRequest.Prompt, textRequest.Model)
	case relaymode.Moderations:
		return openai.CountTokenInput(textRequest.Input, textRequest.Model)
	case relaymode.Embeddings:
		return openai.CountTokenInput(textRequest.Input, textRequest.Model)
	}
	return 0
}

func getPreConsumedQuota(textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64) int64 {
	preConsumedTokens := config.PreConsumedQuota + int64(promptTokens)
	if textRequest.MaxTokens != 0 {
		preConsumedTokens += int64(textRequest.MaxTokens)
	}
	return int64(float64(preConsumedTokens) * ratio)
}

func preConsumeQuotaAmount(ctx context.Context, preConsumedQuota int64, meta *meta.Meta) (int64, *relaymodel.ErrorWithStatusCode) {
	if preConsumedQuota <= 0 {
		return 0, nil
	}
	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-preConsumedQuota < 0 {
		return preConsumedQuota, openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	err = model.CacheDecreaseUserQuota(meta.UserId, preConsumedQuota)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota > 100*preConsumedQuota {
		preConsumedQuota = 0
		logger.Info(ctx, fmt.Sprintf("user %d has enough quota %d, trusted and no need to pre-consume", meta.UserId, userQuota))
	}
	if preConsumedQuota > 0 {
		err := model.PreConsumeTokenQuota(meta.TokenId, preConsumedQuota)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
	}
	return preConsumedQuota, nil
}

func preConsumeQuota(ctx context.Context, textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64, meta *meta.Meta) (int64, *relaymodel.ErrorWithStatusCode) {
	preConsumedQuota := getPreConsumedQuota(textRequest, promptTokens, ratio)

	return preConsumeQuotaAmount(ctx, preConsumedQuota, meta)
}

func relayTenantQuotaAdjustments(meta *meta.Meta) (tenantDiscount float64, additionalQuota int64) {
	tenantDiscount = 1.0
	if meta == nil || meta.UserTenantId <= 0 {
		return tenantDiscount, 0
	}
	if meta.UserTenantId > 0 {
		ch, err := model.GetChannelById(meta.ChannelId, true)
		if err == nil && ch != nil {
			now := helper.GetTimestamp()
			if ch.TenantID == nil {
				// Platform channel, apply discount
				activeRules, _ := model.GetActiveTenantBillingRules(meta.UserTenantId, model.BillingRuleTypeDiscountRatio, now)
				
				var minDiscount float64 = -1.0
				var minGlobalDiscount float64 = -1.0
				
				for _, r := range activeRules {
					if r.ChannelId == meta.ChannelId {
						if minDiscount < 0 || r.Value < minDiscount {
							minDiscount = r.Value
						}
					} else if r.ChannelId == 0 {
						if minGlobalDiscount < 0 || r.Value < minGlobalDiscount {
							minGlobalDiscount = r.Value
						}
					}
				}

				if minDiscount >= 0 {
					tenantDiscount = minDiscount
				} else if minGlobalDiscount >= 0 {
					tenantDiscount = minGlobalDiscount
				} else {
					tenant, err := model.GetTenantByID(meta.UserTenantId)
					if err == nil && tenant != nil && tenant.DiscountRatio > 0 {
						tenantDiscount = tenant.DiscountRatio
					}
				}
			} else if *ch.TenantID == meta.UserTenantId {
				// Tenant private channel, apply price per 1k calls
				activeRules, _ := model.GetActiveTenantBillingRules(meta.UserTenantId, model.BillingRuleTypePricePer1k, now)
				
				var minPrice float64 = -1.0
				var minGlobalPrice float64 = -1.0
				
				for _, r := range activeRules {
					if r.ChannelId == meta.ChannelId {
						if minPrice < 0 || r.Value < minPrice {
							minPrice = r.Value
						}
					} else if r.ChannelId == 0 {
						if minGlobalPrice < 0 || r.Value < minGlobalPrice {
							minGlobalPrice = r.Value
						}
					}
				}

				var price float64 = -1.0
				if minPrice >= 0 {
					price = minPrice
				} else if minGlobalPrice >= 0 {
					price = minGlobalPrice
				}

				if price < 0 {
					// Fallback to legacy fields
					tenant, err := model.GetTenantByID(meta.UserTenantId)
					price = config.DefaultTenantChannelPricePer1k
					if err == nil && tenant != nil && tenant.PricePer1kApiCall >= 0 {
						price = tenant.PricePer1kApiCall
					}
				}

				if price > 0 {
					// Convert price per 1k calls to quota for 1 call
					// amount = price / 1000
					// quota = amount * QuotaPerUnit
					additionalQuota = int64(math.Ceil((price / 1000.0) * config.QuotaPerUnit))
				}
			}
		}
	}
	return tenantDiscount, additionalQuota
}

// ComputeRelayConsumedQuota 与 postConsumeQuota 中扣费额度计算一致（未减去预扣）。
func ComputeRelayConsumedQuota(c *gin.Context, usage *relaymodel.Usage, textRequest *relaymodel.GeneralOpenAIRequest, meta *meta.Meta, _ float64) int64 {
	if usage == nil || textRequest == nil || meta == nil {
		return 0
	}
	if c != nil {
		if ok, quota, _ := billing.TryTieredSettle(c, usage); ok {
			return quota
		}
	}
	return computeRelayQuotaResult(usage, textRequest, meta).Quota
}

func computeRelayQuotaResult(usage *relaymodel.Usage, textRequest *relaymodel.GeneralOpenAIRequest, meta *meta.Meta) billing.TokenQuotaResult {
	tenantDiscount, additionalQuota := relayTenantQuotaAdjustments(meta)
	userGroup := meta.UserGroup
	if userGroup == "" {
		userGroup = meta.Group
	}
	usingGroup := meta.UsingGroup
	if usingGroup == "" {
		usingGroup = meta.Group
	}
	return billing.ComputeTokenQuota(usage, billing.TokenQuotaParams{
		OriginModel:     meta.OriginModelName,
		MappedModel:     textRequest.Model,
		ChannelType:     meta.ChannelType,
		UserGroup:       userGroup,
		UsingGroup:      usingGroup,
		TenantDiscount:  tenantDiscount,
		AdditionalQuota: additionalQuota,
	})
}

func postConsumeQuota(c *gin.Context, usage *relaymodel.Usage, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, ratio float64, preConsumedQuota int64, modelRatio float64, groupRatio float64, systemPromptReset bool) {
	ctx := c.Request.Context()
	if usage == nil {
		logger.Error(ctx, "usage is nil, which is unexpected")
		return
	}
	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens
	var quota int64
	var logContent string
	if ok, tieredQuota, tieredResult := billing.TryTieredSettle(c, usage); ok {
		quota = tieredQuota
		if tieredResult != nil && tieredResult.MatchedTier != "" {
			logContent = fmt.Sprintf("分层计费：%s", tieredResult.MatchedTier)
		} else {
			logContent = "分层计费"
		}
	} else {
		quotaResult := computeRelayQuotaResult(usage, textRequest, meta)
		quota = quotaResult.Quota
		logContent = billing.FormatConsumeLogRatioLine(quotaResult)
	}
	quotaDelta := quota - preConsumedQuota
	if meta.UserTenantId > 0 {
		ch, err := model.GetChannelById(meta.ChannelId, false)
		if err == nil && ch != nil && ch.TenantID != nil && *ch.TenantID == meta.UserTenantId {
			// Increment ApiCallCount
			model.DB.Model(&model.Channel{}).Where("id = ?", meta.ChannelId).UpdateColumn("api_call_count", gorm.Expr("api_call_count + 1"))
		}
	}

	err := model.PostConsumeTokenQuota(meta.TokenId, quotaDelta)
	if err != nil {
		logger.Error(ctx, "error consuming token remain quota: "+err.Error())
	}
	err = model.CacheUpdateUserQuota(ctx, meta.UserId)
	if err != nil {
		logger.Error(ctx, "error update user quota cache: "+err.Error())
	}
	other := requestaudit.ConsumeLogOtherJSON(c)
	model.RecordConsumeLog(ctx, &model.Log{
		UserId:            meta.UserId,
		ChannelId:         meta.ChannelId,
		TokenId:           meta.TokenId,
		Group:             meta.Group,
		PromptTokens:      promptTokens,
		CompletionTokens:  completionTokens,
		ModelName:         textRequest.Model,
		TokenName:         meta.TokenName,
		Quota:             int(quota),
		Content:           logContent,
		Other:             other,
		IsStream:          meta.IsStream,
		ElapsedTime:       helper.CalcElapsedTime(meta.StartTime),
		SystemPromptReset: systemPromptReset,
	})
	model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
	model.UpdateChannelUsedQuota(meta.ChannelId, quota)
	
	if config.EnableHighCardinalityMetrics {
		metrics.QuotaConsumedTotal.WithLabelValues(fmt.Sprintf("%d", meta.UserId)).Add(float64(quotaDelta))
	}
}

func getMappedModelName(modelName string, mapping map[string]string) (string, bool) {
	if mapping == nil {
		return modelName, false
	}
	mappedModelName := mapping[modelName]
	if mappedModelName != "" {
		return mappedModelName, true
	}
	return modelName, false
}

func isErrorHappened(meta *meta.Meta, resp *http.Response) bool {
	if resp == nil {
		if meta.ChannelType == channeltype.AwsClaude {
			return false
		}
		return true
	}
	if resp.StatusCode != http.StatusOK &&
		// replicate return 201 to create a task
		resp.StatusCode != http.StatusCreated {
		return true
	}
	if meta.ChannelType == channeltype.DeepL {
		// skip stream check for deepl
		return false
	}

	if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") &&
		// Even if stream mode is enabled, replicate will first return a task info in JSON format,
		// requiring the client to request the stream endpoint in the task info
		meta.ChannelType != channeltype.Replicate {
		return true
	}
	return false
}

func setSystemPrompt(ctx context.Context, request *relaymodel.GeneralOpenAIRequest, prompt string) (reset bool) {
	if prompt == "" {
		return false
	}
	if len(request.Messages) == 0 {
		return false
	}
	if request.Messages[0].Role == role.System {
		request.Messages[0].Content = prompt
		logger.Infof(ctx, "rewrite system prompt")
		return true
	}
	request.Messages = append([]relaymodel.Message{{
		Role:    role.System,
		Content: prompt,
	}}, request.Messages...)
	logger.Infof(ctx, "add system prompt")
	return true
}
