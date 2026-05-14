package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/gemini"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func estimateAnthropicTokens(req *anthropic.Request) int {
	var b strings.Builder
	b.WriteString(req.System)
	for _, msg := range req.Messages {
		for _, part := range msg.Content {
			b.WriteString(part.Text)
			b.WriteString(part.Content)
		}
	}
	s := b.String()
	if s == "" {
		return 8
	}
	return openai.CountTokenText(s, req.Model)
}

func relayAnthropicNativeResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	if meta.IsStream {
		return relayAnthropicNativeStream(c, resp, meta)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	c.Writer.Header().Set("Content-Type", ct)
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(body)

	if resp.StatusCode != http.StatusOK {
		return nil, &model.ErrorWithStatusCode{
			StatusCode: -1,
			Error: model.Error{
				Message: "upstream error body forwarded",
				Type:    "upstream_error",
			},
		}
	}
	var ar anthropic.Response
	_ = json.Unmarshal(body, &ar)
	usage := &model.Usage{
		PromptTokens:     ar.Usage.InputTokens,
		CompletionTokens: ar.Usage.OutputTokens,
		TotalTokens:      ar.Usage.InputTokens + ar.Usage.OutputTokens,
	}
	return usage, nil
}

func relayAnthropicNativeStream(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), "\n"); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	common.SetEventStreamHeaders(c)
	var usage model.Usage

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var claudeResponse anthropic.StreamResponse
			if json.Unmarshal([]byte(payload), &claudeResponse) == nil {
				if claudeResponse.Type == "message_start" && claudeResponse.Message != nil {
					usage.PromptTokens += claudeResponse.Message.Usage.InputTokens
					usage.CompletionTokens += claudeResponse.Message.Usage.OutputTokens
				}
				if claudeResponse.Usage != nil {
					usage.PromptTokens += claudeResponse.Usage.InputTokens
					usage.CompletionTokens += claudeResponse.Usage.OutputTokens
				}
			}
		}
		_, _ = c.Writer.WriteString(line)
		_, _ = c.Writer.WriteString("\n")
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		_ = resp.Body.Close()
		return nil, openai.ErrorWrapper(err, "stream_read_failed", http.StatusBadGateway)
	}
	_ = resp.Body.Close()
	if usage.PromptTokens+usage.CompletionTokens == 0 {
		usage.PromptTokens = meta.PromptTokens
	}
	return &usage, nil
}

func relayGeminiNativeResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	if meta.IsStream || strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return relayGeminiNativeStream(c, resp, meta)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	c.Writer.Header().Set("Content-Type", ct)
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(body)

	if resp.StatusCode != http.StatusOK {
		return nil, &model.ErrorWithStatusCode{
			StatusCode: -1,
			Error: model.Error{
				Message: "upstream error body forwarded",
				Type:    "upstream_error",
			},
		}
	}
	var gr gemini.ChatResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return &model.Usage{
			PromptTokens: meta.PromptTokens,
			TotalTokens:  meta.PromptTokens,
		}, nil
	}
	text := gr.GetResponseText()
	comp := openai.CountTokenText(text, meta.ActualModelName)
	usage := &model.Usage{
		PromptTokens:     meta.PromptTokens,
		CompletionTokens: comp,
		TotalTokens:      meta.PromptTokens + comp,
	}
	return usage, nil
}

func relayGeminiNativeStream(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)

	common.SetEventStreamHeaders(c)
	var responseText strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		_, _ = c.Writer.WriteString(line)
		_, _ = c.Writer.WriteString("\n")
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
		data := strings.TrimSpace(line)
		if !strings.HasPrefix(data, "data: ") {
			continue
		}
		data = strings.TrimPrefix(data, "data: ")
		data = strings.TrimSuffix(data, "\"")
		var geminiResponse gemini.ChatResponse
		if json.Unmarshal([]byte(data), &geminiResponse) == nil {
			responseText.WriteString(geminiResponse.GetResponseText())
		}
	}
	if err := scanner.Err(); err != nil {
		_ = resp.Body.Close()
		return nil, openai.ErrorWrapper(err, "stream_read_failed", http.StatusBadGateway)
	}
	_ = resp.Body.Close()
	comp := openai.CountTokenText(responseText.String(), meta.ActualModelName)
	usage := &model.Usage{
		PromptTokens:     meta.PromptTokens,
		CompletionTokens: comp,
		TotalTokens:      meta.PromptTokens + comp,
	}
	return usage, nil
}

// RelayAnthropicNativeOnce handles POST /v1/messages (Anthropic Messages API pass-through).
func RelayAnthropicNativeOnce(c *gin.Context) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	if c.Request.Method != http.MethodPost {
		return openai.ErrorWrapper(fmt.Errorf("method not allowed"), "method_not_allowed", http.StatusMethodNotAllowed)
	}
	m := meta.GetByContext(c)
	if m.ChannelType != channeltype.Anthropic {
		return openai.ErrorWrapper(fmt.Errorf("使用 Anthropic 原生 Messages API 须选择 Claude（Anthropic）渠道类型；路径示例 /anthropic/v1/messages 或 /v1/messages"), "channel_type_mismatch", http.StatusBadRequest)
	}
	body, err := common.GetRequestBody(c)
	if err != nil {
		return openai.ErrorWrapper(err, "read_body_failed", http.StatusBadRequest)
	}
	var req anthropic.Request
	if err := json.Unmarshal(body, &req); err != nil {
		return openai.ErrorWrapper(err, "invalid_json", http.StatusBadRequest)
	}
	if req.Model == "" {
		return openai.ErrorWrapper(fmt.Errorf("model is required"), "invalid_request", http.StatusBadRequest)
	}
	if len(req.Messages) == 0 {
		return openai.ErrorWrapper(fmt.Errorf("messages is required"), "invalid_request", http.StatusBadRequest)
	}

	originModel := req.Model
	mapped, _ := getMappedModelName(req.Model, m.ModelMapping)
	req.Model = mapped
	if m.ForcedSystemPrompt != "" {
		if req.System == "" {
			req.System = m.ForcedSystemPrompt
		} else {
			req.System = m.ForcedSystemPrompt + "\n\n" + req.System
		}
	}

	m.OriginModelName = originModel
	m.ActualModelName = mapped
	m.Mode = relaymode.AnthropicMessages
	m.IsStream = req.Stream
	m.PromptTokens = estimateAnthropicTokens(&req)
	m.OverrideRequestURL = ""

	modelRatio := billingratio.GetModelRatio(mapped, m.ChannelType)
	groupRatio := billingratio.GetGroupRatio(m.Group)
	ratio := modelRatio * groupRatio
	fakeReq := &model.GeneralOpenAIRequest{
		Model:     originModel,
		MaxTokens: req.MaxTokens,
	}
	if fakeReq.MaxTokens == 0 {
		fakeReq.MaxTokens = 4096
	}
	preConsumed, bizErr := preConsumeQuota(ctx, fakeReq, m.PromptTokens, ratio, m)
	if bizErr != nil {
		return bizErr
	}

	adaptor := relay.GetAdaptor(m.APIType)
	if adaptor == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("invalid api type"), "invalid_api_type", http.StatusInternalServerError)
	}
	adaptor.Init(m)

	outBody, err := json.Marshal(&req)
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(err, "marshal_request_failed", http.StatusInternalServerError)
	}

	resp, err := adaptor.DoRequest(c, m, bytes.NewReader(outBody))
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(m, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return RelayErrorHandler(resp)
	}

	usage, respErr := relayAnthropicNativeResponse(c, resp, m)
	if respErr != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		if respErr.StatusCode == -1 {
			return &model.ErrorWithStatusCode{StatusCode: -1, Error: respErr.Error}
		}
		return respErr
	}
	if usage == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("empty usage"), "empty_usage", http.StatusInternalServerError)
	}
	go postConsumeQuota(ctx, usage, m, fakeReq, ratio, preConsumed, modelRatio, groupRatio, false)
	return nil
}

// RelayGeminiNativeOnce handles POST /v1beta/models/{model}:{method} (Google Gemini generateContent API pass-through).
func RelayGeminiNativeOnce(c *gin.Context) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	if c.Request.Method != http.MethodPost {
		return openai.ErrorWrapper(fmt.Errorf("method not allowed"), "method_not_allowed", http.StatusMethodNotAllowed)
	}
	m := meta.GetByContext(c)
	if m.ChannelType != channeltype.Gemini {
		return openai.ErrorWrapper(fmt.Errorf("使用 Gemini 原生接口须选择 Google Gemini 渠道类型；路径示例 /gemini/v1beta/models/{model}:generateContent 或 /v1beta/models/…"), "channel_type_mismatch", http.StatusBadRequest)
	}

	raw := strings.TrimPrefix(c.Param("geminiAction"), "/")
	if raw == "" {
		p := relaymode.NormalizeAPIPath(c.Request.URL.Path)
		raw = strings.TrimPrefix(p, "/v1beta/models/")
		raw = strings.TrimPrefix(raw, "/")
	}
	idx := strings.LastIndex(raw, ":")
	if idx <= 0 || idx == len(raw)-1 {
		return openai.ErrorWrapper(fmt.Errorf("invalid path, expect .../models/{model}:generateContent"), "invalid_path", http.StatusBadRequest)
	}
	modelInPath := raw[:idx]
	action := raw[idx+1:]
	if modelInPath == "" {
		return openai.ErrorWrapper(fmt.Errorf("model id required"), "invalid_path", http.StatusBadRequest)
	}

	originModel := modelInPath
	mapped, _ := getMappedModelName(modelInPath, m.ModelMapping)
	m.OriginModelName = originModel
	m.ActualModelName = mapped
	m.Mode = relaymode.GeminiGenerate
	m.IsStream = strings.Contains(action, "stream") || c.Request.URL.Query().Get("alt") == "sse"

	base := strings.TrimSuffix(m.BaseURL, "/")
	m.OverrideRequestURL = fmt.Sprintf("%s/v1beta/models/%s:%s", base, mapped, action)
	if q := c.Request.URL.RawQuery; q != "" {
		m.OverrideRequestURL += "?" + q
	}

	body, err := common.GetRequestBody(c)
	if err != nil {
		return openai.ErrorWrapper(err, "read_body_failed", http.StatusBadRequest)
	}
	m.PromptTokens = openai.CountTokenText(string(body), mapped)
	if m.PromptTokens < 8 {
		m.PromptTokens = 8
	}

	modelRatio := billingratio.GetModelRatio(mapped, m.ChannelType)
	groupRatio := billingratio.GetGroupRatio(m.Group)
	ratio := modelRatio * groupRatio
	fakeReq := &model.GeneralOpenAIRequest{Model: originModel, MaxTokens: 8192}
	preConsumed, bizErr := preConsumeQuota(ctx, fakeReq, m.PromptTokens, ratio, m)
	if bizErr != nil {
		return bizErr
	}

	adaptor := relay.GetAdaptor(m.APIType)
	if adaptor == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("invalid api type"), "invalid_api_type", http.StatusInternalServerError)
	}
	adaptor.Init(m)

	resp, err := adaptor.DoRequest(c, m, bytes.NewReader(body))
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(m, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return RelayErrorHandler(resp)
	}

	usage, respErr := relayGeminiNativeResponse(c, resp, m)
	if respErr != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		if respErr.StatusCode == -1 {
			return &model.ErrorWithStatusCode{StatusCode: -1, Error: respErr.Error}
		}
		return respErr
	}
	if usage == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumed, m.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("empty usage"), "empty_usage", http.StatusInternalServerError)
	}
	go postConsumeQuota(ctx, usage, m, fakeReq, ratio, preConsumed, modelRatio, groupRatio, false)
	return nil
}
