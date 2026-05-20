package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/billing"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/gemini"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/protocolbridge"
	"github.com/songquanpeng/one-api/relay/relaymode"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
)

// 客户端可能用 ?key= 传 one-api 令牌（仿 Google）；转发上游时必须去掉，由渠道 x-goog-api-key 鉴权。
func sanitizeGeminiUpstreamQuery(raw string) string {
	if raw == "" {
		return ""
	}
	v, err := url.ParseQuery(raw)
	if err != nil {
		return ""
	}
	v.Del("key")
	if len(v) == 0 {
		return ""
	}
	return v.Encode()
}

func estimateAnthropicTokens(req *anthropic.Request) int {
	var b strings.Builder
	b.WriteString(req.System.String())
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

func anthropicResponseClientModel(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.OriginModelName)
}

func rewriteAnthropicResponseModelBody(body []byte, clientModel string) []byte {
	clientModel = strings.TrimSpace(clientModel)
	if clientModel == "" {
		return body
	}
	var ar anthropic.Response
	if err := json.Unmarshal(body, &ar); err != nil {
		return body
	}
	if ar.Model == clientModel {
		return body
	}
	ar.Model = clientModel
	rewritten, err := json.Marshal(ar)
	if err != nil {
		return body
	}
	return rewritten
}

func rewriteAnthropicStreamLine(line, clientModel string) string {
	clientModel = strings.TrimSpace(clientModel)
	if clientModel == "" || !strings.HasPrefix(line, "data:") {
		return line
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "" || payload == "[DONE]" {
		return line
	}
	var event anthropic.StreamResponse
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return line
	}
	if event.Type != "message_start" || event.Message == nil || event.Message.Model == clientModel {
		return line
	}
	event.Message.Model = clientModel
	b, err := json.Marshal(event)
	if err != nil {
		return line
	}
	return "data: " + string(b)
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
	if resp.StatusCode == http.StatusOK {
		if clientModel := anthropicResponseClientModel(meta); clientModel != "" {
			body = rewriteAnthropicResponseModelBody(body, clientModel)
		}
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
	clientModel := anthropicResponseClientModel(meta)

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
		if clientModel != "" {
			line = rewriteAnthropicStreamLine(line, clientModel)
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
	if meta.Mode == relaymode.Embeddings {
		return &model.Usage{
			PromptTokens:     meta.PromptTokens,
			CompletionTokens: 0,
			TotalTokens:      meta.PromptTokens,
		}, nil
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
	routeModel := req.Model
	if base, variant := anthropic.SplitClientModelVariant(req.Model); variant != "" {
		c.Set(ctxkey.AnthropicModelVariant, variant)
		routeModel = base
	}
	mapped, _ := getMappedModelName(routeModel, m.ModelMapping)
	req.Model = mapped
	if m.ForcedSystemPrompt != "" {
		if req.System == "" {
			req.System = anthropic.SystemPrompt(m.ForcedSystemPrompt)
		} else {
			req.System = anthropic.SystemPrompt(m.ForcedSystemPrompt + "\n\n" + req.System.String())
		}
	}

	m.OriginModelName = originModel
	m.ActualModelName = mapped

	modelRatio := billingratio.GetModelRatio(m.OriginModelName, mapped, m.ChannelType)
	userGroup := m.UserGroup
	if userGroup == "" {
		userGroup = m.Group
	}
	usingGroup := m.UsingGroup
	if usingGroup == "" {
		usingGroup = m.Group
	}
	groupRatio := billingratio.GetEffectiveGroupRatio(userGroup, usingGroup)
	ratio := modelRatio * groupRatio

	if !channelUsesAnthropicNativePassthrough(m) {
		if !config.IsRelayProtocolBridgeEnabled() {
			return openai.ErrorWrapper(fmt.Errorf("跨协议转发未开启：请在管理端「智能路由」策略页开启，或在配置中设置 relay_protocol_bridge_enabled；或选用 Anthropic 协议渠道走原生直连"), "protocol_bridge_disabled", http.StatusBadRequest)
		}
		g, convErr := protocolbridge.GeneralFromAnthropicRequest(&req)
		if convErr != nil {
			return openai.ErrorWrapper(convErr, "protocol_translate_failed", http.StatusBadRequest)
		}
		g.Model = mapped
		pt := getPromptTokens(g, relaymode.ChatCompletions)
		preConsumed, bizErr := preConsumeQuota(ctx, g, pt, ratio, m)
		if bizErr != nil {
			return bizErr
		}
		return relayTranslatedChatCompletion(c, m, g, preConsumed, modelRatio, groupRatio, ratio, originModel, nativeClientAnthropic)
	}

	m.Mode = relaymode.AnthropicMessages
	m.IsStream = req.Stream
	m.PromptTokens = estimateAnthropicTokens(&req)
	m.OverrideRequestURL = ""

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

	apiType := anthropicNativeAPIType(m)
	adaptor := relay.GetAdaptor(apiType)
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
		e := RelayErrorHandler(resp)
		ApplyRelayStatusCodeMapping(c, e)
		return e
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
	go postConsumeQuota(c, usage, m, fakeReq, ratio, preConsumed, modelRatio, groupRatio, false)
	return nil
}

// RelayGeminiNativeOnce handles POST /v1beta/models/{model}:{method} (Google Gemini generateContent API pass-through).
func RelayGeminiNativeOnce(c *gin.Context) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	if c.Request.Method != http.MethodPost {
		return openai.ErrorWrapper(fmt.Errorf("method not allowed"), "method_not_allowed", http.StatusMethodNotAllowed)
	}
	m := meta.GetByContext(c)

	raw := strings.TrimPrefix(c.Param("geminiAction"), "/")
	if raw == "" {
		p := relaymode.NormalizeAPIPath(c.Request.URL.Path)
		for _, prefix := range []string{"/v1beta/models/", "/models/"} {
			if strings.HasPrefix(p, prefix) {
				raw = strings.TrimPrefix(p, prefix)
				break
			}
		}
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
	if strings.Contains(strings.ToLower(action), "embed") {
		m.Mode = relaymode.Embeddings
	}
	m.IsStream = strings.Contains(action, "stream") || c.Request.URL.Query().Get("alt") == "sse"

	base := strings.TrimSuffix(m.BaseURL, "/")
	m.OverrideRequestURL = fmt.Sprintf("%s/v1beta/models/%s:%s", base, mapped, action)
	if q := sanitizeGeminiUpstreamQuery(c.Request.URL.RawQuery); q != "" {
		m.OverrideRequestURL += "?" + q
	}

	body, err := common.GetRequestBody(c)
	if err != nil {
		return openai.ErrorWrapper(err, "read_body_failed", http.StatusBadRequest)
	}

	modelRatio := billingratio.GetModelRatio(m.OriginModelName, mapped, m.ChannelType)
	userGroup := m.UserGroup
	if userGroup == "" {
		userGroup = m.Group
	}
	usingGroup := m.UsingGroup
	if usingGroup == "" {
		usingGroup = m.Group
	}
	groupRatio := billingratio.GetEffectiveGroupRatio(userGroup, usingGroup)
	ratio := modelRatio * groupRatio

	if m.ChannelType != channeltype.Gemini && m.ChannelType != channeltype.GeminiNativeCompatible {
		if !config.IsRelayProtocolBridgeEnabled() {
			return openai.ErrorWrapper(fmt.Errorf("跨协议转发未开启：请在管理端「智能路由」策略页开启，或在配置中设置 relay_protocol_bridge_enabled；或选用 Google Gemini 协议渠道走原生直连"), "protocol_bridge_disabled", http.StatusBadRequest)
		}
		if !strings.Contains(action, "generateContent") {
			return openai.ErrorWrapper(fmt.Errorf("跨协议转发当前仅支持 :generateContent / :streamGenerateContent"), "unsupported_gemini_action", http.StatusBadRequest)
		}
		var chat gemini.ChatRequest
		if err := json.Unmarshal(body, &chat); err != nil {
			return openai.ErrorWrapper(err, "invalid_json", http.StatusBadRequest)
		}
		g, convErr := protocolbridge.GeneralFromGeminiChatRequest(&chat, mapped)
		if convErr != nil {
			return openai.ErrorWrapper(convErr, "protocol_translate_failed", http.StatusBadRequest)
		}
		g.Model = mapped
		g.Stream = m.IsStream
		pt := getPromptTokens(g, relaymode.ChatCompletions)
		preConsumed, bizErr := preConsumeQuota(ctx, g, pt, ratio, m)
		if bizErr != nil {
			return bizErr
		}
		return relayTranslatedChatCompletion(c, m, g, preConsumed, modelRatio, groupRatio, ratio, originModel, nativeClientGemini)
	}

	m.PromptTokens = openai.CountTokenText(string(body), mapped)
	if m.PromptTokens < 8 {
		m.PromptTokens = 8
	}

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
		e := RelayErrorHandler(resp)
		ApplyRelayStatusCodeMapping(c, e)
		return e
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
	go postConsumeQuota(c, usage, m, fakeReq, ratio, preConsumed, modelRatio, groupRatio, false)
	return nil
}
