package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

// RelayAnthropicCountTokensOnce handles POST /v1/messages/count_tokens (Claude Code 用其校验模型可用性).
func RelayAnthropicCountTokensOnce(c *gin.Context) *model.ErrorWithStatusCode {
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
	if strings.TrimSpace(req.Model) == "" {
		return openai.ErrorWrapper(fmt.Errorf("model is required"), "invalid_request", http.StatusBadRequest)
	}
	if len(req.Messages) == 0 {
		return openai.ErrorWrapper(fmt.Errorf("messages is required"), "invalid_request", http.StatusBadRequest)
	}

	routeModel := req.Model
	originModel := req.Model
	if base, variant := anthropic.SplitClientModelVariant(req.Model); variant != "" {
		c.Set(ctxkey.AnthropicModelVariant, variant)
		routeModel = base
	}
	mapped, _ := getMappedModelName(routeModel, m.ModelMapping)
	req.Model = mapped

	if channelUsesAnthropicNativePassthrough(m) {
		outBody, err := buildAnthropicNativeOutBody(body, originModel, mapped, "")
		if err != nil {
			return openai.ErrorWrapper(err, "marshal_request_failed", http.StatusInternalServerError)
		}
		base := strings.TrimSuffix(m.BaseURL, "/")
		m.OverrideRequestURL = base + "/v1/messages/count_tokens"
		adaptor := relay.GetAdaptor(anthropicNativeAPIType(m))
		if adaptor == nil {
			return openai.ErrorWrapper(fmt.Errorf("invalid api type"), "invalid_api_type", http.StatusInternalServerError)
		}
		adaptor.Init(m)
		resp, err := adaptor.DoRequest(c, m, bytes.NewReader(outBody))
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			defer func() { _ = resp.Body.Close() }()
			respBody, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				ct := resp.Header.Get("Content-Type")
				if ct == "" {
					ct = "application/json"
				}
				c.Writer.Header().Set("Content-Type", ct)
				c.Writer.WriteHeader(http.StatusOK)
				_, _ = c.Writer.Write(respBody)
				return nil
			}
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		// Novita 等网关常无 count_tokens，回退本地估算，避免 Claude Code 误判模型不可用。
	}

	tokens := estimateAnthropicTokens(&req)
	if tokens < 1 {
		tokens = 1
	}
	c.JSON(http.StatusOK, gin.H{"input_tokens": tokens})
	return nil
}
