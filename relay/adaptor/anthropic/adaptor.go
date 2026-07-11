package anthropic

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

type Adaptor struct {
}

func (a *Adaptor) Init(meta *meta.Meta) {

}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	return fmt.Sprintf("%s/v1/messages", meta.BaseURL), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	apiKey := strings.TrimSpace(meta.APIKey)
	req.Header.Set("x-api-key", apiKey)
	if incoming := strings.TrimSpace(c.Request.Header.Get(claudeSessionHeader)); incoming != "" {
		req.Header.Set(claudeSessionHeader, incoming)
	} else if meta.ChannelType == channeltype.AnthropicCompatible {
		// 对 AnthropicCompatible 上游透明注入稳定会话键，减少多层网关下会话头丢失导致的会话断裂。
		if sid := deriveStableClaudeSessionID(c); sid != "" {
			req.Header.Set(claudeSessionHeader, sid)
		}
	}
	if incomingTenant := strings.TrimSpace(c.Request.Header.Get(claudeTenantHeader)); incomingTenant != "" {
		req.Header.Set(claudeTenantHeader, incomingTenant)
	} else if meta.ChannelType == channeltype.AnthropicCompatible {
		req.Header.Set(claudeTenantHeader, deriveStableClaudeTenantKey(c))
	}
	// 第三方 Anthropic 兼容代理（new-api 等）通常同时支持 Bearer；仅 x-api-key 时部分网关会报「未提供令牌」。
	if meta.ChannelType == channeltype.AnthropicCompatible && apiKey != "" {
		if strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
			req.Header.Set("Authorization", apiKey)
		} else {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}
	anthropicVersion := c.Request.Header.Get("anthropic-version")
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	anthropicBeta := strings.TrimSpace(c.Request.Header.Get("anthropic-beta"))
	if anthropicBeta == "" {
		anthropicBeta = "messages-2023-12-15"
	}
	// https://x.com/alexalbert__/status/1812921642143900036
	// claude-3-5-sonnet can support 8k context
	if strings.HasPrefix(meta.ActualModelName, "claude-3-5-sonnet") {
		beta := "max-tokens-3-5-sonnet-2024-07-15"
		if !strings.Contains(anthropicBeta, beta) {
			anthropicBeta = beta + "," + anthropicBeta
		}
	}
	if v, ok := c.Get(ctxkey.AnthropicModelVariant); ok {
		if beta := BetaForClientVariant(v.(string)); beta != "" && !strings.Contains(anthropicBeta, beta) {
			anthropicBeta = anthropicBeta + "," + beta
		}
	}
	req.Header.Set("anthropic-beta", anthropicBeta)

	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return ConvertRequest(*request), nil
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	if meta.IsStream {
		err, usage = StreamHandler(c, resp)
	} else {
		err, usage = Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "anthropic"
}
