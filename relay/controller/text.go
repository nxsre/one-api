package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/jsonmerge"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/setting/billing_setting"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

func auditRequestRaw(c *gin.Context) []byte {
	if v, ok := c.Get(ctxkey.KeyRequestBody); ok && v != nil {
		if b, ok := v.([]byte); ok {
			return b
		}
	}
	return nil
}

func finalizeTextAuditFailure(c *gin.Context, textRequest *model.GeneralOpenAIRequest, bizErr *model.ErrorWithStatusCode) {
	if bizErr == nil || bizErr.StatusCode == -1 {
		return
	}
	r := requestaudit.FromContext(c)
	if r == nil || r.IsFinalized() {
		return
	}
	raw := auditRequestRaw(c)
	thinking := requestaudit.InferThinkingStream(textRequest, raw)
	stream := requestaudit.InferStreamFromBody(raw)
	modelName := c.GetString(ctxkey.RequestModel)
	if textRequest != nil {
		stream = textRequest.Stream
		modelName = textRequest.Model
	}
	requestaudit.FinalizeFailure(c, r, modelName, stream, thinking, bizErr.StatusCode, bizErr.Error.Message)
}

func RelayTextHelper(c *gin.Context) (bizErr *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	var textRequest *model.GeneralOpenAIRequest
	defer func() {
		finalizeTextAuditFailure(c, textRequest, bizErr)
	}()

	textRequest, err := getAndValidateTextRequest(c, meta.Mode)
	if err != nil {
		logger.Errorf(ctx, "getAndValidateTextRequest failed: %s", err.Error())
		bizErr = openai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
		return
	}
	meta.IsStream = textRequest.Stream

	// map model name
	meta.OriginModelName = textRequest.Model
	textRequest.Model, _ = getMappedModelName(textRequest.Model, meta.ModelMapping)
	meta.ActualModelName = textRequest.Model
	// set system prompt if not empty
	systemPromptReset := setSystemPrompt(ctx, textRequest, meta.ForcedSystemPrompt)
	// get model ratio & group ratio
	userGroup := meta.UserGroup
	if userGroup == "" {
		userGroup = meta.Group
	}
	usingGroup := meta.UsingGroup
	if usingGroup == "" {
		usingGroup = meta.Group
	}
	modelRatio := billingratio.GetModelRatio(meta.OriginModelName, textRequest.Model, meta.ChannelType)
	groupRatio := billingratio.GetEffectiveGroupRatio(userGroup, usingGroup)
	ratio := modelRatio * groupRatio
	promptTokens := getPromptTokens(textRequest, meta.Mode)
	meta.PromptTokens = promptTokens
	var preConsumedQuota int64
	if billing_setting.GetBillingMode(meta.OriginModelName) == billing_setting.BillingModeTieredExpr {
		var amount int64
		amount, err = billing.EstimateTieredPreConsume(c, meta, textRequest, promptTokens)
		if err != nil {
			bizErr = openai.ErrorWrapper(err, "tiered_pre_consume_failed", http.StatusBadRequest)
			return
		}
		preConsumedQuota, bizErr = preConsumeQuotaAmount(ctx, amount, meta)
	} else {
		preConsumedQuota, bizErr = preConsumeQuota(ctx, textRequest, promptTokens, ratio, meta)
	}
	if bizErr != nil {
		logger.Warnf(ctx, "preConsumeQuota failed: %+v", *bizErr)
		return
	}

	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		bizErr = openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
		return
	}
	adaptor.Init(meta)

	// get request body
	requestBody, err := getRequestBody(c, meta, textRequest, adaptor)
	if err != nil {
		var re *jsonmerge.ReturnError
		if errors.As(err, &re) {
			code := re.Code
			bizErr = &model.ErrorWithStatusCode{
				StatusCode: re.StatusCode,
				Error: model.Error{
					Message: re.Message,
					Type:    re.Type,
					Code:    code,
				},
			}
			return
		}
		bizErr = openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
		return
	}

	// do request
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		bizErr = openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
		return
	}
	if isErrorHappened(meta, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		e := RelayErrorHandler(resp)
		ApplyRelayStatusCodeMapping(c, e)
		bizErr = e
		return
	}

	// do response
	usage, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		bizErr = respErr
		return
	}
	// post-consume quota
	go postConsumeQuota(c, usage, meta, textRequest, ratio, preConsumedQuota, modelRatio, groupRatio, systemPromptReset)

	ra := requestaudit.FromContext(c)
	if ra != nil && !ra.IsFinalized() {
		quota := ComputeRelayConsumedQuota(c, usage, textRequest, meta, ratio)
		thinking := requestaudit.InferThinkingStream(textRequest, auditRequestRaw(c))
		requestaudit.FinalizeSuccess(c, ra, textRequest.Model, textRequest.Stream, thinking, systemPromptReset, quota)
	}
	return nil
}

func channelParamOverrideActive(c *gin.Context) bool {
	raw, ok := c.Get(ctxkey.ChannelParamOverride)
	if !ok || raw == nil {
		return false
	}
	m, ok := raw.(map[string]interface{})
	return ok && len(m) > 0
}

func requestHeadersFlat(c *gin.Context) map[string]string {
	h := make(map[string]string)
	for k, vv := range c.Request.Header {
		if len(vv) > 0 {
			h[k] = vv[0]
		}
	}
	return h
}

func getRequestBody(c *gin.Context, meta *meta.Meta, textRequest *model.GeneralOpenAIRequest, adaptor adaptor.Adaptor) (io.Reader, error) {
	hasPO := channelParamOverrideActive(c)
	fastOpenAI := !config.EnforceIncludeUsage &&
		meta.APIType == apitype.OpenAI &&
		meta.OriginModelName == meta.ActualModelName &&
		meta.ChannelType != channeltype.Baichuan &&
		meta.ForcedSystemPrompt == ""

	if fastOpenAI && !hasPO {
		// no need to convert request for openai
		return c.Request.Body, nil
	}

	var jsonData []byte
	var err error
	if fastOpenAI && hasPO {
		jsonData, err = io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
	} else {
		convertedRequest, err := adaptor.ConvertRequest(c, meta.Mode, textRequest)
		if err != nil {
			logger.Debugf(c.Request.Context(), "converted request failed: %s\n", err.Error())
			return nil, err
		}
		jsonData, err = json.Marshal(convertedRequest)
		if err != nil {
			logger.Debugf(c.Request.Context(), "converted request json_marshal_failed: %s\n", err.Error())
			return nil, err
		}
	}
	if hasPO {
		raw, _ := c.Get(ctxkey.ChannelParamOverride)
		patch, _ := raw.(map[string]interface{})
		c.Set(ctxkey.ParamOverrideHeadersExclusive, false)
		var channelHdr map[string]string
		if rawHO, ok := c.Get(ctxkey.ChannelHeaderOverride); ok {
			if m, ok := rawHO.(map[string]string); ok {
				channelHdr = m
			}
		}
		if channelHdr == nil {
			channelHdr = map[string]string{}
		}
		_, isOpDSL := patch["operations"]
		var poCtx map[string]interface{}
		if isOpDSL {
			poCtx = jsonmerge.BuildParamOverrideContext(meta.RequestURLPath, requestHeadersFlat(c), channelHdr, meta.OriginModelName, meta.ActualModelName)
		}
		jsonData, err = jsonmerge.ApplyParamOverride(jsonData, patch, poCtx)
		if err != nil {
			return nil, err
		}
		if isOpDSL {
			rh := jsonmerge.RuntimeHeaderOverrideFromContext(poCtx)
			if rh == nil {
				rh = map[string]string{}
			}
			c.Set(ctxkey.ParamOverrideRuntimeHeaders, rh)
			c.Set(ctxkey.ParamOverrideHeadersExclusive, true)
		}
	}
	logger.Debugf(c.Request.Context(), "converted request: \n%s", string(jsonData))
	return bytes.NewBuffer(jsonData), nil
}
