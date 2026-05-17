package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/monitor"
	relayctl "github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"github.com/songquanpeng/one-api/routing"
)

// https://platform.openai.com/docs/api-reference/chat

func relayHelper(c *gin.Context, relayMode int) *model.ErrorWithStatusCode {
	var err *model.ErrorWithStatusCode
	switch relayMode {
	case relaymode.ImagesGenerations:
		err = relayctl.RelayImageHelper(c, relayMode)
	case relaymode.AudioSpeech:
		fallthrough
	case relaymode.AudioTranslation:
		fallthrough
	case relaymode.AudioTranscription:
		err = relayctl.RelayAudioHelper(c, relayMode)
	case relaymode.Proxy:
		err = relayctl.RelayProxyHelper(c, relayMode)
	default:
		err = relayctl.RelayTextHelper(c)
	}
	return err
}

func Relay(c *gin.Context) {
	relayMode := relaymode.GetByPath(c.Request.URL.Path)
	relayWithRetry(c, func(c *gin.Context) *model.ErrorWithStatusCode {
		return relayHelper(c, relayMode)
	})
}

func relayWithRetry(c *gin.Context, exec func(*gin.Context) *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	if config.DebugEnabled {
		requestBody, _ := common.GetRequestBody(c)
		logger.Debugf(ctx, "request body: %s", string(requestBody))
	}
	startAt := time.Now()
	requestaudit.Attach(c, startAt)
	var bizErr *model.ErrorWithStatusCode
	defer func() {
		requestaudit.FinalizeRelayErrorIfUnhandled(c, bizErr)
	}()

	channelId := c.GetInt(ctxkey.ChannelId)
	userId := c.GetInt(ctxkey.Id)
	group := c.GetString(ctxkey.Group)
	originalModel := c.GetString(ctxkey.OriginalModel)
	retryPol := routing.CurrentRelayRetryPolicy()
	maxRetries := routing.EffectiveMaxRetries()

	tried := map[int]struct{}{channelId: {}}

	bizErr = exec(c)
	recordRelayOutcome(c, originalModel, bizErr, startAt)

	if bizErr != nil && bizErr.StatusCode == -1 {
		return
	}
	if bizErr == nil {
		monitor.Emit(channelId, true)
		routing.RecordCircuitSuccess(channelId)
		return
	}
	lastFailedChannelId := channelId
	channelName := c.GetString(ctxkey.ChannelName)
	perChAutoBan := true
	if v, ok := c.Get(ctxkey.ChannelAutoBan); ok {
		if b, ok2 := v.(bool); ok2 {
			perChAutoBan = b
		}
	}
	go processChannelRelayError(ctx, userId, channelId, channelName, perChAutoBan, *bizErr)
	requestId := c.GetString(helper.RequestIdKey)
	if !shouldRelayRetry(c, bizErr.StatusCode, retryPol) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		maxRetries = 0
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryBackoffDuration(attempt-1, retryPol))
		}

		excluded := relayExcludeIDs(tried, lastFailedChannelId, retryPol)
		opts := routing.SelectOpts{
			StickyKey:           c.GetString(ctxkey.RoutingStickyKey),
			RequestModel:        originalModel,
			ExcludeChannelIDs:   excluded,
			SkipCircuitDisabled: routing.CurrentRoutingPolicy().CircuitFailThreshold > 0,
		}
		channel, err := routing.SelectChannel(group, originalModel, attempt > 0, opts)
		if err != nil {
			logger.Errorf(ctx, "SelectChannel failed: %+v", err)
			break
		}
		if _, dup := tried[channel.Id]; dup {
			continue
		}
		logger.Infof(ctx, "using channel #%d to retry (attempt %d/%d)", channel.Id, attempt+1, maxRetries)
		tried[channel.Id] = struct{}{}

		middleware.SetupContextForSelectedChannel(c, channel, originalModel)
		requestBody, err := common.GetRequestBody(c)
		if err != nil {
			logger.Errorf(ctx, "GetRequestBody failed: %v", err)
			break
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

		startAt = time.Now()
		bizErr = exec(c)
		recordRelayOutcome(c, originalModel, bizErr, startAt)

		if bizErr != nil && bizErr.StatusCode == -1 {
			return
		}
		if bizErr == nil {
			okCh := c.GetInt(ctxkey.ChannelId)
			monitor.Emit(okCh, true)
			routing.RecordCircuitSuccess(okCh)
			return
		}
		lastFailedChannelId = c.GetInt(ctxkey.ChannelId)
		channelName = c.GetString(ctxkey.ChannelName)
		perChAutoBan := true
		if v, ok := c.Get(ctxkey.ChannelAutoBan); ok {
			if b, ok2 := v.(bool); ok2 {
				perChAutoBan = b
			}
		}
		go processChannelRelayError(ctx, userId, lastFailedChannelId, channelName, perChAutoBan, *bizErr)
	}
	if bizErr != nil {
		if bizErr.StatusCode == http.StatusTooManyRequests {
			bizErr.Error.Message = "当前分组上游负载已饱和，请稍后再试"
		}

		// BUG: bizErr is in race condition
		bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, requestId)
		c.JSON(bizErr.StatusCode, gin.H{
			"error": bizErr.Error,
		})
	}
}

func recordRelayOutcome(c *gin.Context, originalModel string, bizErr *model.ErrorWithStatusCode, startAt time.Time) {
	channelId := c.GetInt(ctxkey.ChannelId)
	provider := c.GetString(ctxkey.ChannelRoutingProvider)
	tokenID := c.GetInt(ctxkey.TokenId)
	ok := bizErr == nil
	if bizErr != nil && bizErr.StatusCode == -1 {
		ok = true
	}
	ms := time.Since(startAt).Milliseconds()
	routing.RecordRelayMetric(channelId, originalModel, tokenID, provider, ok, ms)
}

func relayExcludeIDs(tried map[int]struct{}, lastFailed int, pol routing.RelayRetryPolicy) map[int]struct{} {
	if pol.ForceDifferentChannelEachAttempt {
		ex := make(map[int]struct{}, len(tried))
		for id := range tried {
			ex[id] = struct{}{}
		}
		return ex
	}
	return map[int]struct{}{lastFailed: {}}
}

func retryBackoffDuration(zeroBasedAttempt int, pol routing.RelayRetryPolicy) time.Duration {
	ms := pol.BaseBackoffMs
	for i := 0; i < zeroBasedAttempt; i++ {
		ms *= 2
		if ms > pol.MaxBackoffMs {
			ms = pol.MaxBackoffMs
			break
		}
	}
	if ms > pol.MaxBackoffMs {
		ms = pol.MaxBackoffMs
	}
	return time.Duration(ms) * time.Millisecond
}

func shouldRelayRetry(c *gin.Context, statusCode int, rp routing.RelayRetryPolicy) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	for _, code := range rp.RetryHTTPStatusDenylist {
		if statusCode == code {
			return false
		}
	}
	if len(rp.RetryHTTPStatusAllowlist) > 0 {
		for _, code := range rp.RetryHTTPStatusAllowlist {
			if statusCode == code {
				return true
			}
		}
		return false
	}
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode/100 == 5 {
		return true
	}
	if statusCode == http.StatusBadRequest {
		return false
	}
	if statusCode/100 == 2 {
		return false
	}
	return true
}

func processChannelRelayError(ctx context.Context, userId int, channelId int, channelName string, perChannelAutoBan bool, err model.ErrorWithStatusCode) {
	logger.Errorf(ctx, "relay error (channel id %d, user id: %d): %s", channelId, userId, err.Message)
	// https://platform.openai.com/docs/guides/error-codes/api-errors
	if monitor.ShouldDisableChannel(&err.Error, err.StatusCode, perChannelAutoBan) {
		monitor.DisableChannel(channelId, channelName, err.Message)
	} else {
		monitor.Emit(channelId, false)
		routing.RecordCircuitFailure(channelId)
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := model.Error{
		Message: "API not implemented",
		Type:    "one_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := model.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
