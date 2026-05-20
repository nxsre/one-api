package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// RelayPassthroughHelper 将请求体与响应原样转发至渠道 Base URL（用于 Files、图像 edits/variations 等 multipart/二进制 API）。
func RelayPassthroughHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	tokenId := c.GetInt(ctxkey.TokenId)
	channelType := c.GetInt(ctxkey.Channel)
	channelId := c.GetInt(ctxkey.ChannelId)
	userId := c.GetInt(ctxkey.Id)
	group := c.GetString(ctxkey.Group)
	tokenName := c.GetString(ctxkey.TokenName)
	modelName := c.GetString(ctxkey.RequestModel)
	if modelName == "" {
		modelName = "passthrough"
	}

	modelRatio := billingratio.GetModelRatio(modelName, modelName, channelType)
	userGroup := c.GetString(ctxkey.UserGroup)
	if userGroup == "" {
		userGroup = group
	}
	usingGroup := c.GetString(ctxkey.UsingGroup)
	if usingGroup == "" {
		usingGroup = group
	}
	groupRatio := billingratio.GetEffectiveGroupRatio(userGroup, usingGroup)
	ratio := modelRatio * groupRatio
	preConsumedQuota := int64(float64(config.PreConsumedQuota) * ratio)
	quota := preConsumedQuota

	userQuota, err := model.CacheGetUserQuota(ctx, userId)
	if err != nil {
		return openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-preConsumedQuota < 0 {
		return openai.ErrorWrapper(fmt.Errorf("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	if err = model.CacheDecreaseUserQuota(userId, preConsumedQuota); err != nil {
		return openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota > 100*preConsumedQuota {
		preConsumedQuota = 0
	}
	if preConsumedQuota > 0 {
		if err := model.PreConsumeTokenQuota(tokenId, preConsumedQuota); err != nil {
			return openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
	}

	succeed := false
	defer func() {
		if succeed || preConsumedQuota <= 0 {
			return
		}
		go func() {
			if err := model.PostConsumeTokenQuota(tokenId, -preConsumedQuota); err != nil {
				logger.Error(ctx, "error rollback pre-consumed quota: "+err.Error())
			}
		}()
	}()

	baseURL := channeltype.DefaultBaseURL(channelType)
	if c.GetString(ctxkey.BaseURL) != "" {
		baseURL = c.GetString(ctxkey.BaseURL)
	}
	fullRequestURL := openai.GetFullRequestURL(baseURL, c.Request.URL.String(), channelType)

	requestBody := &bytes.Buffer{}
	if _, err = io.Copy(requestBody, c.Request.Body); err != nil {
		return openai.ErrorWrapper(err, "read_request_body_failed", http.StatusInternalServerError)
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody.Bytes()))

	req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for _, h := range []string{"Content-Type", "Accept", "OpenAI-Beta", "OpenAI-Organization"} {
		if v := c.Request.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	req.ContentLength = c.Request.ContentLength

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		requestaudit.SnapUpstreamHTTP(c, req, nil)
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	requestaudit.SnapUpstreamHTTP(c, req, resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := RelayErrorHandler(resp)
		ApplyRelayStatusCodeMapping(c, e)
		return e
	}

	succeed = true
	quotaDelta := quota - preConsumedQuota
	defer func(ctx context.Context) {
		go billing.PostConsumeQuota(c, ctx, tokenId, quotaDelta, quota, userId, channelId, modelRatio, groupRatio, modelName, tokenName)
	}(c.Request.Context())

	for k, v := range resp.Header {
		if len(v) > 0 {
			c.Writer.Header().Set(k, v[0])
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		return openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()
	return nil
}
