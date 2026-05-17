package deepresearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	channelmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

const (
	// ModelDeepResearch 客户端调用 One API 时使用的模型名（可通过模型映射改名）。
	ModelDeepResearch = "deep-research"
	channelLabel      = "deepresearch"
)

type upstreamMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Adaptor 转发 OpenAI Chat Completions 至深知 POST /chat（SSE）。
type Adaptor struct{}

func (a *Adaptor) Init(_ *meta.Meta) {}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(meta.BaseURL), "/")
	if base == "" {
		return "", errors.New("deep-research: 请在渠道中填写上游 Base URL（示例：https://maas.xxx/app/deep-research/api/deepresearch）")
	}
	return base + "/chat", nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	req.Header.Set("Content-Type", "application/json")
	key := strings.TrimSpace(meta.ChannelKey)
	if key == "" {
		key = meta.APIKey
	}
	req.Header.Set("Authorization", "Bearer "+key)
	return nil
}

func messagesForUpstream(msgs []model.Message) []upstreamMsg {
	out := make([]upstreamMsg, 0, len(msgs))
	for _, m := range msgs {
		text := strings.TrimSpace(m.StringContent())
		if text == "" {
			continue
		}
		r := m.Role
		if r != "user" && r != "system" && r != "assistant" {
			r = "user"
			text = fmt.Sprintf("[%s] %s", m.Role, text)
		}
		out = append(out, upstreamMsg{Role: r, Content: text})
	}
	return out
}

func mergeDeepResearchBody(c *gin.Context, request *model.GeneralOpenAIRequest, base map[string]interface{}) {
	var cfg channelmodel.ChannelConfig
	if raw, ok := c.Get(ctxkey.Config); ok {
		if cc, ok := raw.(channelmodel.ChannelConfig); ok {
			cfg = cc
		}
	}
	if rm := strings.TrimSpace(cfg.DeepResearchMode); rm != "" {
		base["research_mode"] = rm
	}
	if request.Metadata == nil {
		return
	}
	metaMap, ok := request.Metadata.(map[string]interface{})
	if !ok {
		return
	}
	if v, ok := metaMap["deep_research_thread_id"].(string); ok && strings.TrimSpace(v) != "" {
		base["thread_id"] = strings.TrimSpace(v)
	}
	if v, ok := metaMap["deep_research_resume"].(bool); ok && v {
		base["resume"] = true
	}
	if v, ok := metaMap["deep_research_stream_mode"]; ok && v != nil {
		base["stream_mode"] = v
	}
	if v, ok := metaMap["deep_research_interrupt_after"]; ok && v != nil {
		base["interrupt_after"] = v
	}
	if v, ok := metaMap["deep_research_internal"]; ok {
		if raw, err := json.Marshal(v); err == nil {
			base["_internal"] = json.RawMessage(raw)
		}
	}
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if relayMode != relaymode.ChatCompletions {
		return nil, errors.New("deep-research: only /v1/chat/completions supported")
	}
	if request == nil {
		return nil, errors.New("deep-research: request is nil")
	}
	if !request.Stream {
		return nil, errors.New("deep-research: 上游为 SSE，请将 stream 设为 true")
	}
	msgs := messagesForUpstream(request.Messages)
	if len(msgs) == 0 {
		return nil, errors.New("deep-research: messages 不能为空")
	}
	body := map[string]interface{}{
		"messages": msgs,
	}
	mergeDeepResearchBody(c, request, body)
	return body, nil
}

func (a *Adaptor) ConvertImageRequest(*model.ImageRequest) (any, error) {
	return nil, errors.New("deep-research: image api not supported")
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	if meta.IsStream {
		return openai.PassthroughUpstreamResponse(c, resp)
	}
	defer resp.Body.Close()
	return nil, openai.ErrorWrapper(errors.New("deep-research: internal error — stream=false should be rejected earlier"), "bad_stream_mode", http.StatusInternalServerError)
}

func (a *Adaptor) GetModelList() []string {
	return []string{ModelDeepResearch}
}

func (a *Adaptor) GetChannelName() string {
	return channelLabel
}
