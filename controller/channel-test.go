package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/message"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/amap"
	"github.com/songquanpeng/one-api/relay/adaptor/aippt"
	"github.com/songquanpeng/one-api/relay/adaptor/deepresearch"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	relayctl "github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func buildTestRequest(model string) *relaymodel.GeneralOpenAIRequest {
	return buildChatTestRequest(model)
}

func parseTestResponse(resp string) (*openai.TextResponse, string, error) {
	var response openai.TextResponse
	err := json.Unmarshal([]byte(resp), &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Choices) == 0 {
		return nil, "", errors.New("response has no choices")
	}
	stringContent, ok := response.Choices[0].Content.(string)
	if !ok {
		return nil, "", errors.New("response content is not string")
	}
	return &response, stringContent, nil
}

func truncateForTestLog(s string) string {
	max := config.RequestAuditMaxBodyBytes
	if max <= 0 {
		max = 262144
	}
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "…(truncated)"
}

func mergeTestLogOther(c *gin.Context, extra map[string]interface{}) string {
	m := map[string]interface{}{}
	if hdrs := requestaudit.UpstreamHeadersJSONForLog(c); hdrs != "" {
		_ = json.Unmarshal([]byte(hdrs), &m)
	}
	for k, v := range extra {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		m[k] = v
	}
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func recordChannelTestLog(ctx context.Context, channel *model.Channel, modelName, logContent string, startAt time.Time, other string) {
	lg := &model.Log{
		ChannelId:   channel.Id,
		ModelName:   modelName,
		Content:     logContent,
		Other:       other,
		ElapsedTime: helper.CalcElapsedTime(startAt),
	}
	go model.RecordTestLog(ctx, lg)
}

func testChannelByModel(ctx context.Context, channel *model.Channel, logicalModel string) (responseMessage string, detail channelModelTestHTTPDetail, err error, openaiErr *relaymodel.Error) {
	if channel != nil && channel.Type == channeltype.AiPPT {
		t0 := time.Now()
		app, sec, uid, err := aippt.ParseChannelKey(channel.Key)
		if err != nil {
			other := mergeTestLogOther(nil, map[string]interface{}{
				"test_mode": "aippt",
				"error":     err.Error(),
			})
			recordChannelTestLog(ctx, channel, "aippt", fmt.Sprintf("渠道 %s AiPPT 测试失败：%v", channel.Name, err), t0, other)
			return "", channelModelTestHTTPDetail{}, fmt.Errorf("AiPPT 密钥: %w", err), nil
		}
		cl := &aippt.Client{
			BaseURL:   channel.GetBaseURL(),
			AppKey:    app,
			SecretKey: sec,
			UID:       uid,
		}
		if err := cl.TestPresetList(); err != nil {
			other := mergeTestLogOther(nil, map[string]interface{}{
				"test_mode":          "aippt",
				"test_upstream_base": channel.GetBaseURL(),
				"error":              err.Error(),
			})
			recordChannelTestLog(ctx, channel, "aippt", fmt.Sprintf("渠道 %s AiPPT 测试失败：%v", channel.Name, err), t0, other)
			return "", channelModelTestHTTPDetail{RequestURL: channel.GetBaseURL()}, err, nil
		}
		other := mergeTestLogOther(nil, map[string]interface{}{
			"test_mode":          "aippt",
			"test_upstream_base": channel.GetBaseURL(),
			"detail":             "TestPresetList ok",
		})
		recordChannelTestLog(ctx, channel, "aippt", fmt.Sprintf("渠道 %s AiPPT 测试成功：预置词接口检测通过", channel.Name), t0, other)
		return "AiPPT 预置词接口检测通过", channelModelTestHTTPDetail{RequestURL: channel.GetBaseURL()}, nil, nil
	}
	if channel != nil && channel.Type == channeltype.AmapPOI {
		t0 := time.Now()
		if err := amap.TestAround(channel.GetBaseURL(), channel.Key); err != nil {
			other := mergeTestLogOther(nil, map[string]interface{}{
				"test_mode":          "amap",
				"test_upstream_base": channel.GetBaseURL(),
				"error":              err.Error(),
			})
			recordChannelTestLog(ctx, channel, "amap", fmt.Sprintf("渠道 %s 高德测试失败：%v", channel.Name, err), t0, other)
			return "", channelModelTestHTTPDetail{RequestURL: channel.GetBaseURL()}, err, nil
		}
		other := mergeTestLogOther(nil, map[string]interface{}{
			"test_mode":          "amap",
			"test_upstream_base": channel.GetBaseURL(),
			"detail":             "TestAround ok",
		})
		recordChannelTestLog(ctx, channel, "amap", fmt.Sprintf("渠道 %s 高德周边搜索检测通过", channel.Name), t0, other)
		return "高德周边搜索接口检测通过", channelModelTestHTTPDetail{RequestURL: channel.GetBaseURL()}, nil, nil
	}
	if channel != nil && channel.Type == channeltype.DeepResearch {
		t0 := time.Now()
		base := strings.TrimSpace(channel.GetBaseURL())
		key := strings.TrimSpace(channel.Key)
		if key == "" {
			recordChannelTestLog(ctx, channel, deepresearch.ModelDeepResearch, fmt.Sprintf("渠道 %s Deep Research 测试失败：密钥为空", channel.Name), t0, "")
			return "", channelModelTestHTTPDetail{}, errors.New("深知 Deep Research：密钥不能为空"), nil
		}
		if base == "" {
			recordChannelTestLog(ctx, channel, deepresearch.ModelDeepResearch, fmt.Sprintf("渠道 %s Deep Research 测试失败：Base URL 为空", channel.Name), t0, "")
			return "", channelModelTestHTTPDetail{}, errors.New("深知 Deep Research：请填写上游 Base URL（完整前缀至 …/deepresearch）"), nil
		}
		other := mergeTestLogOther(nil, map[string]interface{}{
			"test_mode":          "deep_research",
			"test_upstream_base": base,
			"detail":             "skip sse probe",
		})
		recordChannelTestLog(ctx, channel, deepresearch.ModelDeepResearch, fmt.Sprintf("渠道 %s Deep Research：密钥与 Base URL 已配置（上游为 SSE，未自动拉流探测）", channel.Name), t0, other)
		return "深知 Deep Research：密钥与 Base URL 已配置（上游 SSE，未自动探测）", channelModelTestHTTPDetail{RequestURL: base}, nil, nil
	}

	spec := resolveModelTestSpec(logicalModel, channel.Type)
	if spec.Kind == modelTestKindSkip {
		t0 := time.Now()
		msg := spec.SkipReason
		other := mergeTestLogOther(nil, map[string]interface{}{
			"test_mode":     "skip",
			"test_kind":     modelTestKindLabel(spec.Kind),
			"logical_model": logicalModel,
			"detail":        msg,
		})
		recordChannelTestLog(ctx, channel, logicalModel, fmt.Sprintf("渠道 %s 模型 %s：%s", channel.Name, logicalModel, msg), t0, other)
		return msg, channelModelTestHTTPDetail{RequestMethod: "POST", RequestPath: spec.Path, WireProtocol: spec.Protocol}, nil, nil
	}

	if vendorProbeSupported(channel.Type, spec) {
		startTime := time.Now()
		mappedModel := resolveChannelTestModelName(channel, logicalModel)
		msg, detail, probeErr := runVendorModelProbe(ctx, channel, logicalModel, spec)
		recordVendorModelProbeLog(ctx, channel, logicalModel, spec, mappedModel, msg, probeErr, startTime)
		return msg, detail, probeErr, nil
	}

	startTime := time.Now()
	var usage *relaymodel.Usage
	var httpStatus int
	var upstreamURL string
	var reqBodyForLog string
	var respBodyForLog string
	var doReqErr string
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: spec.Path},
		Body:   nil,
		Header: make(http.Header),
	}
	if ctx != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	c.Request.Header.Set("Authorization", "Bearer "+channel.Key)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(ctxkey.Channel, channel.Type)
	c.Set(ctxkey.ChannelKey, channel.Key)
	c.Set(ctxkey.BaseURL, channel.GetBaseURL())
	cfg, _ := channel.LoadConfig()
	c.Set(ctxkey.Config, cfg)
	middleware.SetupContextForSelectedChannel(c, channel, logicalModel)
	meta := meta.GetByContext(c)
	apiType := channeltype.ToAPIType(channel.Type)
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		other := mergeTestLogOther(nil, map[string]interface{}{
			"test_mode":     "http_relay",
			"test_kind":     modelTestKindLabel(spec.Kind),
			"error":         fmt.Sprintf("invalid api type: %d, adaptor is nil", apiType),
			"channel_type":  channel.Type,
			"logical_model": logicalModel,
		})
		recordChannelTestLog(ctx, channel, logicalModel, fmt.Sprintf("渠道 %s 测试失败：无效的 API 类型 %d", channel.Name, apiType), startTime, other)
		return "", channelModelTestHTTPDetail{RequestMethod: "POST", RequestPath: spec.Path}, fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), nil
	}
	adaptor.Init(meta)
	modelName := logicalModel
	modelMap := channel.GetModelMapping()
	if modelName == "" || !strings.Contains(channel.Models, modelName) {
		modelNames := strings.Split(channel.Models, ",")
		if len(modelNames) > 0 {
			modelName = modelNames[0]
		}
	}
	if modelMap != nil && modelMap[modelName] != "" {
		modelName = modelMap[modelName]
	}
	meta.OriginModelName, meta.ActualModelName = logicalModel, modelName
	captureDetail := func() channelModelTestHTTPDetail {
		return captureChannelModelTestHTTPDetail(c, spec.Path, reqBodyForLog, respBodyForLog, spec.Protocol, httpStatus, upstreamURL)
	}
	defer func() {
		logContent := fmt.Sprintf("渠道 %s 测试成功，响应：%s", channel.Name, responseMessage)
		if err != nil || openaiErr != nil {
			errorMessage := ""
			if err != nil {
				errorMessage = err.Error()
			} else {
				errorMessage = openaiErr.Message
			}
			logContent = fmt.Sprintf("渠道 %s 测试失败，错误：%s", channel.Name, errorMessage)
		}
		extras := map[string]interface{}{
			"test_mode":          "http_relay",
			"test_kind":          modelTestKindLabel(spec.Kind),
			"logical_model":      meta.OriginModelName,
			"mapped_model":       modelName,
			"test_http_status":   httpStatus,
			"test_upstream_url":  upstreamURL,
			"test_request_body":  reqBodyForLog,
			"test_response_body": respBodyForLog,
		}
		if doReqErr != "" {
			extras["test_do_request_error"] = doReqErr
		}
		if usage != nil {
			extras["usage_prompt_tokens"] = usage.PromptTokens
			extras["usage_completion_tokens"] = usage.CompletionTokens
			if usage.TotalTokens > 0 {
				extras["usage_total_tokens"] = usage.TotalTokens
			}
		}
		other := mergeTestLogOther(c, extras)
		lg := &model.Log{
			ChannelId:   channel.Id,
			ModelName:   modelName,
			Content:     logContent,
			Other:       other,
			ElapsedTime: helper.CalcElapsedTime(startTime),
		}
		if usage != nil {
			lg.PromptTokens = usage.PromptTokens
			lg.CompletionTokens = usage.CompletionTokens
		}
		go model.RecordTestLog(ctx, lg)
	}()

	jsonData, err := marshalModelTestPayload(adaptor, c, spec, logicalModel, modelName)
	if err != nil {
		return "", channelModelTestHTTPDetail{RequestMethod: "POST", RequestPath: spec.Path}, err, nil
	}
	reqBodyForLog = truncateForTestLog(string(jsonData))
	logger.SysLog(string(jsonData))
	requestBody := bytes.NewBuffer(jsonData)
	c.Request.Body = io.NopCloser(requestBody)
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		doReqErr = err.Error()
		return "", captureDetail(), err, nil
	}
	if resp != nil {
		httpStatus = resp.StatusCode
		if resp.Request != nil && resp.Request.URL != nil {
			upstreamURL = resp.Request.URL.String()
		}
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		peek, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		respBodyForLog = truncateForTestLog(string(peek))
		resp.Body = io.NopCloser(bytes.NewReader(peek))
		errWith := relayctl.RelayErrorHandler(resp)
		relayctl.ApplyRelayStatusCodeMapping(c, errWith)
		errorMessage := errWith.Error.Message
		if errorMessage != "" {
			errorMessage = ", error message: " + errorMessage
		}
		return "", captureDetail(), fmt.Errorf("http status code: %d%s", errWith.StatusCode, errorMessage), &errWith.Error
	}

	if spec.Kind == modelTestKindTTS {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return "", captureDetail(), readErr, nil
		}
		respBodyForLog = truncateForTestLog(fmt.Sprintf("<binary %d bytes>", len(body)))
		if len(body) == 0 {
			return "", captureDetail(), errors.New("tts response body is empty"), nil
		}
		responseMessage = summarizeModelTestSuccess(spec, "")
		if !shouldRecordModelTestBodyDetail(spec, nil) {
			reqBodyForLog = ""
			respBodyForLog = ""
		}
		return responseMessage, captureDetail(), nil, nil
	}

	var respErr *relaymodel.ErrorWithStatusCode
	usage, respErr = adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		return "", captureDetail(), fmt.Errorf("%s", respErr.Error.Message), &respErr.Error
	}
	rawResponse := w.Body.String()
	if rawResponse != "" {
		respBodyForLog = truncateForTestLog(rawResponse)
	}
	if spec.Kind == modelTestKindChat {
		if usage == nil {
			return "", captureDetail(), errors.New("usage is nil"), nil
		}
		_, responseMessage, err = parseTestResponse(rawResponse)
		if err != nil {
			logger.SysError(fmt.Sprintf("failed to parse error: %s, \nresponse: %s", err.Error(), rawResponse))
			return "", captureDetail(), err, nil
		}
	} else {
		responseMessage = summarizeModelTestSuccess(spec, rawResponse)
	}
	if !shouldRecordModelTestBodyDetail(spec, nil) {
		reqBodyForLog = ""
		respBodyForLog = ""
	}
	result := w.Result()
	respBody, err := io.ReadAll(result.Body)
	if err != nil {
		return "", captureDetail(), err, nil
	}
	logger.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))
	return responseMessage, captureDetail(), nil, nil
}

func marshalModelTestPayload(adaptor adaptor.Adaptor, c *gin.Context, spec modelTestSpec, logicalModel, mappedModel string) ([]byte, error) {
	switch spec.Kind {
	case modelTestKindImage:
		imageReq := buildImageTestRequest(logicalModel)
		imageReq.Model = mappedModel
		converted, err := adaptor.ConvertImageRequest(imageReq)
		if err != nil {
			return nil, err
		}
		return json.Marshal(converted)
	case modelTestKindTTS:
		ttsReq := buildTTSTestRequest(mappedModel)
		return json.Marshal(ttsReq)
	case modelTestKindResponses:
		return json.Marshal(buildResponsesTestRequest(mappedModel))
	case modelTestKindRealtime:
		return json.Marshal(buildRealtimeSessionTestRequest(mappedModel))
	default:
		var generalReq *relaymodel.GeneralOpenAIRequest
		switch spec.Kind {
		case modelTestKindEmbedding:
			generalReq = buildEmbeddingTestRequest(logicalModel)
		case modelTestKindModeration:
			generalReq = buildModerationTestRequest(logicalModel)
		default:
			generalReq = buildChatTestRequest(logicalModel)
		}
		generalReq.Model = mappedModel
		converted, err := adaptor.ConvertRequest(c, spec.RelayMode, generalReq)
		if err != nil {
			return nil, err
		}
		return json.Marshal(converted)
	}
}

func testChannel(ctx context.Context, channel *model.Channel, request *relaymodel.GeneralOpenAIRequest) (responseMessage string, err error, openaiErr *relaymodel.Error) {
	modelName := ""
	if request != nil {
		modelName = request.Model
	}
	msg, _, err, openaiErr := testChannelByModel(ctx, channel, modelName)
	return msg, err, openaiErr
}

func TestChannel(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	modelName := strings.TrimSpace(c.Query("model"))
	if modelName == "" && channel.TestModel != nil {
		modelName = strings.TrimSpace(*channel.TestModel)
	}
	testRequest := buildTestRequest(modelName)
	tik := time.Now()
	responseMessage, _, err, _ := testChannelByModel(ctx, channel, testRequest.Model)
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	if err != nil {
		milliseconds = 0
	}
	go channel.UpdateResponseTime(milliseconds)
	consumedTime := float64(milliseconds) / 1000.0
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":   false,
			"message":   err.Error(),
			"time":      consumedTime,
			"modelName": modelName,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   responseMessage,
		"time":      consumedTime,
		"modelName": modelName,
	})
	return
}

var testAllChannelsLock sync.Mutex
var testAllChannelsRunning bool = false

func testChannels(ctx context.Context, notify bool, scope string) error {
	if config.RootUserEmail == "" {
		config.RootUserEmail = model.GetRootUserEmail()
	}
	testAllChannelsLock.Lock()
	if testAllChannelsRunning {
		testAllChannelsLock.Unlock()
		return errors.New("测试已在运行中")
	}
	testAllChannelsRunning = true
	testAllChannelsLock.Unlock()
	channels, err := model.GetAllChannels(0, 0, scope)
	if err != nil {
		return err
	}
	var disableThreshold = int64(config.ChannelDisableThreshold * 1000)
	if disableThreshold == 0 {
		disableThreshold = 10000000 // a impossible value
	}
	go func() {
		for _, channel := range channels {
			isChannelEnabled := channel.Status == model.ChannelStatusEnabled
			tik := time.Now()
			testRequest := buildTestRequest("")
			_, err, openaiErr := testChannel(ctx, channel, testRequest)
			tok := time.Now()
			milliseconds := tok.Sub(tik).Milliseconds()
			if isChannelEnabled && channel.AutoBanEnabled() && milliseconds > disableThreshold {
				err := fmt.Errorf("响应时间 %.2fs 超过阈值 %.2fs", float64(milliseconds)/1000.0, float64(disableThreshold)/1000.0)
				if config.AutomaticDisableChannelEnabled {
					monitor.DisableChannel(channel.Id, channel.Name, err.Error())
				} else {
					_ = message.Notify(message.ByAll, fmt.Sprintf("渠道 %s （%d）测试超时", channel.Name, channel.Id), "", err.Error())
				}
			}
			if isChannelEnabled && openaiErr != nil && monitor.ShouldDisableChannel(openaiErr, -1, channel.AutoBanEnabled()) {
				monitor.DisableChannel(channel.Id, channel.Name, openaiErr.Message)
			}
			if !isChannelEnabled && monitor.ShouldEnableChannel(err, openaiErr) {
				monitor.EnableChannel(channel.Id, channel.Name)
			}
			channel.UpdateResponseTime(milliseconds)
			time.Sleep(config.RequestInterval)
		}
		testAllChannelsLock.Lock()
		testAllChannelsRunning = false
		testAllChannelsLock.Unlock()
		if notify {
			err := message.Notify(message.ByAll, "渠道测试完成", "", "渠道测试完成，如果没有收到禁用通知，说明所有渠道都正常")
			if err != nil {
				logger.SysError(fmt.Sprintf("failed to send email: %s", err.Error()))
			}
		}
	}()
	return nil
}

func TestChannels(c *gin.Context) {
	ctx := c.Request.Context()
	scope := c.Query("scope")
	if scope == "" {
		scope = "all"
	}
	err := testChannels(ctx, true, scope)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func AutomaticallyTestChannels(frequency int) {
	ctx := context.Background()
	for {
		time.Sleep(time.Duration(frequency) * time.Minute)
		logger.SysLog("testing all channels")
		_ = testChannels(ctx, false, "all")
		logger.SysLog("channel test finished")
	}
}
