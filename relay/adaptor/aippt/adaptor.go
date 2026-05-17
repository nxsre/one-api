package aippt

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

const channelName = "aippt"

// ModelAipptPPT 默认模型名；请求需匹配该渠道绑定的模型之一。
const ModelAipptPPT = "aippt-ppt"

// Adaptor 将 OpenAI 聊天请求转为 AiPPT 标题生成并返回含下载链接的 assistant 消息。
type Adaptor struct{}

func (a *Adaptor) Init(_ *meta.Meta) {}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	if meta == nil {
		return "", errors.New("aippt: meta is nil")
	}
	base := strings.TrimSpace(meta.BaseURL)
	if base == "" && meta.ChannelType > 0 && meta.ChannelType < channeltype.Dummy {
		base = channeltype.DefaultBaseURL(meta.ChannelType)
	}
	if base == "" {
		base = "https://co.aippt.cn"
	}
	return strings.TrimRight(base, "/") + "/aippt", nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("aippt: request is nil")
	}
	if relayMode != relaymode.ChatCompletions {
		return nil, errors.New("aippt: only chat completions supported")
	}
	if request.Stream {
		return nil, errors.New("aippt: streaming is not supported; set stream to false")
	}
	return request, nil
}

func (a *Adaptor) ConvertImageRequest(*model.ImageRequest) (any, error) {
	return nil, errors.New("aippt: not supported")
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	if meta == nil {
		return nil, fmt.Errorf("aippt: meta is nil")
	}
	if meta.Mode != relaymode.ChatCompletions {
		return nil, fmt.Errorf("aippt: only /v1/chat/completions is supported")
	}
	raw, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, err
	}
	var req model.GeneralOpenAIRequest
	if err = json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("aippt: bad request: %w", err)
	}
	if req.Stream {
		return nil, fmt.Errorf("aippt: streaming is not supported")
	}
	title, err := lastUserText(&req)
	if err != nil {
		return nil, err
	}
	app, sec, uid, err := parseChannelCredentials(meta)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(meta.BaseURL)
	if base == "" && meta.ChannelType > 0 && meta.ChannelType < channeltype.Dummy {
		base = channeltype.DefaultBaseURL(meta.ChannelType)
	}
	cl := &Client{
		BaseURL:   base,
		AppKey:    app,
		SecretKey: sec,
		UID:       uid,
	}
	result, err := cl.GenerateFromTitle(title)
	if err != nil {
		return nil, err
	}
	msg := buildAssistantMessage(result)
	return okResponse(meta, msg)
}

func okResponse(meta *meta.Meta, content string) (*http.Response, error) {
	modelName := meta.OriginModelName
	if modelName == "" {
		modelName = ModelAipptPPT
	}
	out := openai.TextResponse{
		Id:      "chatcmpl-aippt-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []openai.TextResponseChoice{{
			Index: 0,
			Message: model.Message{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: "stop",
		}},
	}
	out.Usage = model.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(b))),
	}, nil
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp == nil {
		return nil, openai.ErrorWrapper(errors.New("aippt: empty response"), "bad_response", http.StatusInternalServerError)
	}
	apiErr, usage := openai.Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
	return usage, apiErr
}

func (a *Adaptor) GetModelList() []string {
	return []string{ModelAipptPPT}
}

func (a *Adaptor) GetChannelName() string {
	return channelName
}

func parseChannelCredentials(meta *meta.Meta) (appKey, secretKey, uid string, err error) {
	if meta == nil {
		return "", "", "", fmt.Errorf("aippt: meta is nil")
	}
	raw := strings.TrimSpace(meta.ChannelKey)
	if raw == "" {
		raw = strings.TrimSpace(meta.APIKey)
	}
	return ParseChannelKey(raw)
}

func lastUserText(req *model.GeneralOpenAIRequest) (string, error) {
	if req == nil || len(req.Messages) == 0 {
		return "", fmt.Errorf("aippt: messages are required; put the PPT title/topic in the last user message")
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		m := req.Messages[i]
		if m.Role != "user" {
			continue
		}
		s := strings.TrimSpace(m.StringContent())
		if s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("aippt: need a non-empty last user text message (PPT 标题/主题)")
}

func buildAssistantMessage(r *GenerateResult) string {
	var b strings.Builder
	b.WriteString("PPT 已生成（AiPPT）。\n\n")
	b.WriteString("下载（临时有效，请尽快保存）：\n")
	b.WriteString(r.DownloadURL)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "task_id: %s\n", r.TaskID)
	fmt.Fprintf(&b, "design_id: %s\ntemplate_id: %s\n", r.DesignID, r.TemplateID)
	if strings.TrimSpace(r.OutlineDraft) != "" {
		b.WriteString("\n大纲预览（节选）：\n")
		p := r.OutlineDraft
		runes := []rune(p)
		if len(runes) > 500 {
			p = string(runes[:500]) + "…"
		}
		b.WriteString(p)
	}
	return b.String()
}
