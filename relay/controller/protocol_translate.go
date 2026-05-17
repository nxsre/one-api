package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/billing"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/protocolbridge"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type nativeClientProtocol int

const (
	nativeClientAnthropic nativeClientProtocol = iota
	nativeClientGemini
)

func nativeClientKind(p nativeClientProtocol) string {
	switch p {
	case nativeClientAnthropic:
		return "anthropic"
	case nativeClientGemini:
		return "gemini"
	default:
		return ""
	}
}

func writeTranslatedOpenAIErrorOrFallback(c *gin.Context, raw []byte, httpStatus int, clientProto nativeClientProtocol) bool {
	eb, ct, ok := protocolbridge.TranslateOpenAIErrorForNativeClient(raw, httpStatus, nativeClientKind(clientProto))
	if !ok {
		return false
	}
	c.Writer.Header().Set("Content-Type", ct)
	code := httpStatus
	if code == http.StatusOK || code <= 0 {
		code = http.StatusBadGateway
	}
	c.Writer.WriteHeader(code)
	_, _ = c.Writer.Write(eb)
	return true
}

// relayTranslatedChatCompletion 以 OpenAI 通用语义调用当前渠道的 Adaptor，再把 OpenAI 形响应转换回原生客户端协议。
func relayTranslatedChatCompletion(c *gin.Context, base *meta.Meta, g *model.GeneralOpenAIRequest, preConsumed int64, modelRatio, groupRatio, ratio float64, originModel string, clientProto nativeClientProtocol) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	m2 := *base
	m2.Mode = relaymode.ChatCompletions
	m2.RequestURLPath = "/v1/chat/completions"
	m2.OverrideRequestURL = ""
	m2.IsStream = g.Stream
	m2.PromptTokens = getPromptTokens(g, relaymode.ChatCompletions)

	adaptor := relay.GetAdaptor(m2.APIType)
	if adaptor == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("invalid api type"), "invalid_api_type", http.StatusInternalServerError)
	}
	adaptor.Init(&m2)

	converted, convErr := adaptor.ConvertRequest(c, relaymode.ChatCompletions, g)
	if convErr != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		return openai.ErrorWrapper(convErr, "convert_request_failed", http.StatusBadRequest)
	}
	reqBody, mErr := json.Marshal(converted)
	if mErr != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		return openai.ErrorWrapper(mErr, "marshal_request_failed", http.StatusInternalServerError)
	}

	resp, dErr := adaptor.DoRequest(c, &m2, bytes.NewReader(reqBody))
	if dErr != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		return openai.ErrorWrapper(dErr, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(&m2, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		e := RelayErrorHandler(resp)
		ApplyRelayStatusCodeMapping(c, e)
		return e
	}

	capturedUsage, raw, code, hdr, bizErr := protocolbridge.CaptureAdaptorResponse(c, adaptor, resp, &m2)
	if bizErr != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		return bizErr
	}

	if code != http.StatusOK {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		if writeTranslatedOpenAIErrorOrFallback(c, raw, code, clientProto) {
			return &model.ErrorWithStatusCode{StatusCode: -1, Error: model.Error{Message: "upstream error (translated)", Type: "upstream_error"}}
		}
		for k, vv := range hdr {
			if len(vv) > 0 && k != "Content-Length" {
				c.Writer.Header().Set(k, vv[0])
			}
		}
		c.Writer.WriteHeader(code)
		_, _ = c.Writer.Write(raw)
		return &model.ErrorWithStatusCode{StatusCode: -1, Error: model.Error{Message: "upstream response forwarded", Type: "upstream_error"}}
	}

	var outUsage *model.Usage
	switch clientProto {
	case nativeClientAnthropic:
		if g.Stream {
			u, err := protocolbridge.WriteAnthropicSSEFromOpenAISSE(c, raw, originModel)
			if err != nil {
				billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
				return openai.ErrorWrapper(err, "translate_stream_failed", http.StatusBadGateway)
			}
			outUsage = u
		} else {
			out, u, err := protocolbridge.AnthropicJSONFromOpenAIChatCompletion(raw, originModel)
			if err != nil {
				billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
				if writeTranslatedOpenAIErrorOrFallback(c, raw, code, clientProto) {
					return &model.ErrorWithStatusCode{StatusCode: -1, Error: model.Error{Message: "upstream error body translated", Type: "upstream_error"}}
				}
				c.Writer.Header().Set("Content-Type", "application/json")
				c.Writer.WriteHeader(http.StatusOK)
				_, _ = c.Writer.Write(raw)
				return &model.ErrorWithStatusCode{StatusCode: -1, Error: model.Error{Message: "protocol translation returned raw openai body", Type: "translate_fallback"}}
			}
			outUsage = u
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.WriteHeader(http.StatusOK)
			_, _ = c.Writer.Write(out)
		}
	case nativeClientGemini:
		if g.Stream {
			u, err := protocolbridge.WriteGeminiSSEFromOpenAISSE(c, raw, m2.ActualModelName)
			if err != nil {
				billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
				return openai.ErrorWrapper(err, "translate_stream_failed", http.StatusBadGateway)
			}
			outUsage = u
		} else {
			out, u, err := protocolbridge.GeminiJSONFromOpenAIChatCompletion(raw, originModel)
			if err != nil {
				billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
				if writeTranslatedOpenAIErrorOrFallback(c, raw, code, clientProto) {
					return &model.ErrorWithStatusCode{StatusCode: -1, Error: model.Error{Message: "upstream error body translated", Type: "upstream_error"}}
				}
				c.Writer.Header().Set("Content-Type", "application/json")
				c.Writer.WriteHeader(http.StatusOK)
				_, _ = c.Writer.Write(raw)
				return &model.ErrorWithStatusCode{StatusCode: -1, Error: model.Error{Message: "protocol translation returned raw openai body", Type: "translate_fallback"}}
			}
			outUsage = u
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.WriteHeader(http.StatusOK)
			_, _ = c.Writer.Write(out)
		}
	}

	if outUsage == nil {
		outUsage = capturedUsage
	}
	if outUsage == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m2.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("empty usage after translation"), "empty_usage", http.StatusInternalServerError)
	}
	go postConsumeQuota(c, outUsage, &m2, g, ratio, preConsumed, modelRatio, groupRatio, false)
	return nil
}
