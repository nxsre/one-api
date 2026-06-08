package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicopt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
	"google.golang.org/genai"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
	geminiadaptor "github.com/songquanpeng/one-api/relay/adaptor/gemini"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

const (
	vendorSDKAnthropic = "github.com/anthropics/anthropic-sdk-go"
	vendorSDKGemini    = "google.golang.org/genai"
	vendorSDKOpenAI    = "github.com/openai/openai-go"
)

var httpStatusCodePattern = regexp.MustCompile(`\b[1-5][0-9]{2}\b`)

func vendorProbeSupported(channelType int, spec modelTestSpec) bool {
	if spec.Kind == modelTestKindSkip || spec.Kind == modelTestKindTTS || spec.Kind == modelTestKindRealtime {
		return false
	}
	if channelType == channeltype.AwsClaude {
		return false
	}
	family := channelWireProtocolFamily(channelType)
	switch family {
	case wireProtoAnthropic:
		return spec.Kind == modelTestKindChat
	case wireProtoGeminiNative:
		return spec.Kind == modelTestKindChat || spec.Kind == modelTestKindEmbedding
	case wireProtoGeminiOpenAI, wireProtoOpenAI:
		switch spec.Kind {
		case modelTestKindChat, modelTestKindEmbedding, modelTestKindImage, modelTestKindModeration, modelTestKindResponses:
			return true
		}
	}
	return false
}

func resolveChannelTestModelName(channel *model.Channel, logicalModel string) string {
	if channel == nil {
		return logicalModel
	}
	modelName := logicalModel
	if modelName == "" || !strings.Contains(channel.Models, modelName) {
		modelNames := strings.Split(channel.Models, ",")
		if len(modelNames) > 0 {
			modelName = strings.TrimSpace(modelNames[0])
		}
	}
	if modelMap := channel.GetModelMapping(); modelMap != nil && modelMap[modelName] != "" {
		modelName = modelMap[modelName]
	}
	return modelName
}

func channelTestBaseURL(channel *model.Channel) string {
	if channel == nil {
		return ""
	}
	if u := strings.TrimSpace(channel.GetBaseURL()); u != "" {
		return strings.TrimRight(u, "/")
	}
	return strings.TrimRight(channeltype.DefaultBaseURL(channel.Type), "/")
}

func vendorSDKWireLabel(family channelWireProtocol, kind modelTestKind, sdkModule string) string {
	return modelTestWireProtocolLabel(family, kind) + " · " + sdkModule
}

func vendorProbeRequestURL(baseURL, path string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path = strings.TrimSpace(path)
	if path == "" {
		return baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

func marshalVendorProbeBody(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return truncateForTestLog(string(b))
}

func vendorProbeDetail(method, baseURL, path, wireProtocol, reqBody, respBody string, status int) channelModelTestHTTPDetail {
	return channelModelTestHTTPDetail{
		RequestMethod:  method,
		RequestURL:     vendorProbeRequestURL(baseURL, path),
		RequestPath:    path,
		RequestBody:    reqBody,
		ResponseBody:   respBody,
		ResponseStatus: status,
		WireProtocol:   wireProtocol,
	}
}

func vendorProbeHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var anthropicErr *anthropic.Error
	if errors.As(err, &anthropicErr) && anthropicErr.StatusCode > 0 {
		return anthropicErr.StatusCode
	}
	var openaiErr *openai.Error
	if errors.As(err, &openaiErr) && openaiErr.StatusCode > 0 {
		return openaiErr.StatusCode
	}
	return 0
}

func vendorProbeFallbackStatusFromErr(err error) int {
	if err == nil {
		return 0
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "status") && !strings.Contains(msg, "http") {
		return 0
	}
	codes := httpStatusCodePattern.FindAllString(msg, -1)
	for _, codeText := range codes {
		code, convErr := strconv.Atoi(codeText)
		if convErr == nil && code >= 100 && code <= 599 {
			return code
		}
	}
	return 0
}

func cloneHeaderMap(h http.Header) map[string][]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string][]string, len(h))
	for k, v := range h {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func vendorProbeHeadersFromErr(err error) (map[string][]string, map[string][]string) {
	if err == nil {
		return nil, nil
	}
	var openaiErr *openai.Error
	if errors.As(err, &openaiErr) && openaiErr != nil {
		var reqHdrs map[string][]string
		var respHdrs map[string][]string
		if openaiErr.Request != nil {
			reqHdrs = cloneHeaderMap(openaiErr.Request.Header)
		}
		if openaiErr.Response != nil {
			respHdrs = cloneHeaderMap(openaiErr.Response.Header)
		}
		return reqHdrs, respHdrs
	}
	return nil, nil
}

func runVendorModelProbe(ctx context.Context, channel *model.Channel, logicalModel string, spec modelTestSpec) (message string, detail channelModelTestHTTPDetail, err error) {
	if channel == nil {
		return "", channelModelTestHTTPDetail{}, errors.New("channel is nil")
	}
	mappedModel := resolveChannelTestModelName(channel, logicalModel)
	family := channelWireProtocolFamily(channel.Type)
	switch family {
	case wireProtoAnthropic:
		return probeAnthropicVendor(ctx, channel, mappedModel, spec)
	case wireProtoGeminiNative:
		return probeGeminiVendor(ctx, channel, mappedModel, spec)
	default:
		return probeOpenAIVendor(ctx, channel, mappedModel, spec, family)
	}
}

func probeAnthropicVendor(ctx context.Context, channel *model.Channel, modelName string, spec modelTestSpec) (string, channelModelTestHTTPDetail, error) {
	baseURL := channelTestBaseURL(channel)
	wire := vendorSDKWireLabel(wireProtoAnthropic, spec.Kind, vendorSDKAnthropic)
	path := "/v1/messages"
	reqSummary := map[string]any{
		"model":      modelName,
		"max_tokens": 64,
		"messages": []map[string]string{
			{"role": "user", "content": config.TestPrompt},
		},
	}
	client := anthropic.NewClient(
		anthropicopt.WithBaseURL(baseURL),
		anthropicopt.WithAPIKey(channel.Key),
	)
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(modelName),
		MaxTokens: 64,
		Messages: []anthropic.MessageParam{{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: config.TestPrompt},
			}},
		}},
	})
	respBody := ""
	status := vendorProbeHTTPStatus(err)
	if err == nil {
		respBody = truncateForTestLog(msg.RawJSON())
	}
	detail := vendorProbeDetail("POST", baseURL, path, wire, marshalVendorProbeBody(reqSummary), respBody, status)
	if err != nil {
		return "", detail, err
	}
	text := anthropicVendorResponseText(msg)
	if strings.TrimSpace(text) == "" {
		return "", detail, errors.New("anthropic response has no text content")
	}
	return text, detail, nil
}

func anthropicVendorResponseText(msg *anthropic.Message) string {
	if msg == nil {
		return ""
	}
	var parts []string
	for _, block := range msg.Content {
		if t := block.AsText(); strings.TrimSpace(t.Text) != "" {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, "")
}

func probeGeminiVendor(ctx context.Context, channel *model.Channel, modelName string, spec modelTestSpec) (string, channelModelTestHTTPDetail, error) {
	baseURL := channelTestBaseURL(channel)
	apiVersion := geminiadaptor.DefaultAPIVersion(channel.Type, modelName)
	wire := vendorSDKWireLabel(wireProtoGeminiNative, spec.Kind, vendorSDKGemini)
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  channel.Key,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL:    baseURL,
			APIVersion: apiVersion,
		},
	})
	if err != nil {
		return "", channelModelTestHTTPDetail{}, err
	}

	switch spec.Kind {
	case modelTestKindEmbedding:
		path := fmt.Sprintf("/%s/models/%s:batchEmbedContents", apiVersion, modelName)
		reqSummary := map[string]any{
			"model": modelName,
			"contents": []map[string]string{
				{"text": "hello"},
			},
		}
		resp, err := client.Models.EmbedContent(ctx, modelName, genai.Text("hello"), nil)
		respBody := ""
		status := http.StatusOK
		if err != nil {
			status = vendorProbeHTTPStatus(err)
		} else if resp != nil {
			if resp.SDKHTTPResponse != nil && resp.SDKHTTPResponse.Body != "" {
				respBody = truncateForTestLog(resp.SDKHTTPResponse.Body)
			} else if b, mErr := json.Marshal(resp); mErr == nil {
				respBody = truncateForTestLog(string(b))
			}
		}
		detail := vendorProbeDetail("POST", baseURL, path, wire, marshalVendorProbeBody(reqSummary), respBody, status)
		if !shouldRecordModelTestBodyDetail(spec, err) {
			detail.RequestBody = ""
			detail.ResponseBody = ""
		}
		if err != nil {
			return "", detail, err
		}
		return summarizeModelTestSuccess(spec, ""), detail, nil
	default:
		path := fmt.Sprintf("/%s/models/%s:generateContent", apiVersion, modelName)
		reqSummary := map[string]any{
			"model": modelName,
			"contents": []map[string]any{
				{"role": "user", "parts": []map[string]string{{"text": config.TestPrompt}}},
			},
		}
		resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(config.TestPrompt), nil)
		respBody := ""
		status := http.StatusOK
		if err != nil {
			status = vendorProbeHTTPStatus(err)
		} else if resp != nil {
			if resp.SDKHTTPResponse != nil && resp.SDKHTTPResponse.Body != "" {
				respBody = truncateForTestLog(resp.SDKHTTPResponse.Body)
			} else if b, mErr := json.Marshal(resp); mErr == nil {
				respBody = truncateForTestLog(string(b))
			}
		}
		detail := vendorProbeDetail("POST", baseURL, path, wire, marshalVendorProbeBody(reqSummary), respBody, status)
		if err != nil {
			return "", detail, err
		}
		text := geminiVendorResponseText(resp)
		if strings.TrimSpace(text) == "" {
			return "", detail, errors.New("gemini response has no text content")
		}
		return text, detail, nil
	}
}

func geminiVendorResponseText(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}
	var parts []string
	for _, c := range resp.Candidates {
		if c == nil || c.Content == nil {
			continue
		}
		for _, p := range c.Content.Parts {
			if p != nil && strings.TrimSpace(p.Text) != "" {
				parts = append(parts, p.Text)
			}
		}
	}
	return strings.Join(parts, "")
}

func probeOpenAIVendor(ctx context.Context, channel *model.Channel, modelName string, spec modelTestSpec, family channelWireProtocol) (string, channelModelTestHTTPDetail, error) {
	baseURL := channelTestBaseURL(channel)
	sdkBaseURL := openAIVendorSDKBaseURL(baseURL)
	wire := vendorSDKWireLabel(family, spec.Kind, vendorSDKOpenAI)
	client := openai.NewClient(
		openaiopt.WithBaseURL(sdkBaseURL),
		openaiopt.WithAPIKey(channel.Key),
	)

	switch spec.Kind {
	case modelTestKindEmbedding:
		path := "/v1/embeddings"
		reqSummary := map[string]any{"model": modelName, "input": "hello"}
		resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Model: openai.EmbeddingModel(modelName),
			Input: openai.EmbeddingNewParamsInputUnion{OfString: openai.String("hello")},
		})
		respJSON := ""
		if resp != nil {
			respJSON = resp.RawJSON()
		}
		return finishOpenAIVendorProbe(baseURL, path, wire, reqSummary, respJSON, err, spec, "")
	case modelTestKindImage:
		path := "/v1/images/generations"
		size := "1024x1024"
		if strings.Contains(strings.ToLower(modelName), "dall-e-2") {
			size = "256x256"
		}
		reqSummary := map[string]any{"model": modelName, "prompt": "a red circle", "n": 1, "size": size}
		resp, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
			Model:  openai.ImageModel(modelName),
			Prompt: "a red circle",
			N:      openai.Int(1),
			Size:   openai.ImageGenerateParamsSize(size),
		})
		respJSON := ""
		if resp != nil {
			respJSON = resp.RawJSON()
		}
		return finishOpenAIVendorProbe(baseURL, path, wire, reqSummary, respJSON, err, spec, "")
	case modelTestKindModeration:
		path := "/v1/moderations"
		reqSummary := map[string]any{"input": config.TestPrompt}
		params := openai.ModerationNewParams{Input: openai.ModerationNewParamsInputUnion{OfString: openai.String(config.TestPrompt)}}
		if strings.TrimSpace(modelName) != "" && !strings.Contains(strings.ToLower(modelName), "stable") {
			params.Model = openai.ModerationModel(modelName)
			reqSummary["model"] = modelName
		}
		resp, err := client.Moderations.New(ctx, params)
		respJSON := ""
		if resp != nil {
			respJSON = resp.RawJSON()
		}
		return finishOpenAIVendorProbe(baseURL, path, wire, reqSummary, respJSON, err, spec, "")
	case modelTestKindResponses:
		path := "/v1/responses"
		reqSummary := map[string]any{"model": modelName, "input": config.TestPrompt}
		resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
			Model: shared.ResponsesModel(modelName),
			Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(config.TestPrompt)},
		})
		respJSON := ""
		if resp != nil {
			respJSON = resp.RawJSON()
		}
		return finishOpenAIVendorProbe(baseURL, path, wire, reqSummary, respJSON, err, spec, "")
	default:
		path := "/v1/chat/completions"
		reqSummary := map[string]any{
			"model": modelName,
			"messages": []map[string]string{
				{"role": "user", "content": config.TestPrompt},
			},
		}
		resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: shared.ChatModel(modelName),
			Messages: []openai.ChatCompletionMessageParamUnion{{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(config.TestPrompt),
					},
				},
			}},
		})
		msg := ""
		if err == nil && len(resp.Choices) > 0 {
			msg = resp.Choices[0].Message.Content
		}
		respJSON := ""
		if resp != nil {
			respJSON = resp.RawJSON()
		}
		return finishOpenAIVendorProbe(baseURL, path, wire, reqSummary, respJSON, err, spec, msg)
	}
}

func openAIVendorSDKBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return baseURL
	}
	// openai-go client typically expects API root with version segment (e.g. /v1).
	// Many OpenAI-compatible channels store host-only Base URL; append /v1 to avoid HTML landing pages.
	if strings.Contains(baseURL, "/v1") {
		return baseURL
	}
	return baseURL + "/v1"
}

func finishOpenAIVendorProbe(baseURL, path, wire string, reqSummary map[string]any, respJSON string, err error, spec modelTestSpec, message string) (string, channelModelTestHTTPDetail, error) {
	status := vendorProbeHTTPStatus(err)
	if status == 0 {
		status = vendorProbeFallbackStatusFromErr(err)
	}
	if status == 0 && err == nil {
		status = http.StatusOK
	}
	responseBody := truncateForTestLog(respJSON)
	if responseBody == "" && err != nil {
		responseBody = truncateForTestLog(err.Error())
	}
	detail := vendorProbeDetail("POST", baseURL, path, wire, marshalVendorProbeBody(reqSummary), responseBody, status)
	reqHdrs, respHdrs := vendorProbeHeadersFromErr(err)
	if len(reqHdrs) > 0 {
		detail.RequestHeaders = reqHdrs
	}
	if len(respHdrs) > 0 {
		detail.ResponseHeaders = respHdrs
	}
	if !shouldRecordModelTestBodyDetail(spec, err) {
		detail.RequestBody = ""
		detail.ResponseBody = ""
	}
	if err != nil {
		return "", detail, err
	}
	if strings.TrimSpace(message) == "" {
		message = summarizeModelTestSuccess(spec, respJSON)
	}
	return message, detail, nil
}

func recordVendorModelProbeLog(ctx context.Context, channel *model.Channel, logicalModel string, spec modelTestSpec, mappedModel, message string, probeErr error, startAt time.Time) {
	logContent := fmt.Sprintf("渠道 %s 测试成功，响应：%s", channel.Name, message)
	if probeErr != nil {
		logContent = fmt.Sprintf("渠道 %s 测试失败，错误：%s", channel.Name, probeErr.Error())
	}
	extras := map[string]interface{}{
		"test_mode":     "vendor_sdk",
		"test_kind":     modelTestKindLabel(spec.Kind),
		"logical_model": logicalModel,
		"mapped_model":  mappedModel,
		"test_protocol": spec.Protocol,
	}
	if probeErr != nil {
		extras["error"] = probeErr.Error()
	}
	other := mergeTestLogOther(nil, extras)
	recordChannelTestLog(ctx, channel, mappedModel, logContent, startAt, other)
}
