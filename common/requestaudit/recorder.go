// Package requestaudit 将 Relay 调用按 request_id 写入 JSONL 审计文件（可选开启）。
package requestaudit

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// Entry 单条 JSONL（一行一个 JSON 对象）。
type Entry struct {
	TimeMs           int64  `json:"time_ms"`
	RequestID        string `json:"request_id"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	UserID           int    `json:"user_id"`
	TokenID          int    `json:"token_id"`
	TokenName        string `json:"token_name,omitempty"`
	ChannelID        int    `json:"channel_id"`
	ChannelName      string `json:"channel_name,omitempty"`
	Model            string `json:"model,omitempty"`
	LogicalModel     string `json:"logical_model,omitempty"`
	Group            string `json:"group,omitempty"`
	Stream           bool   `json:"stream"`
	ThinkingStream   bool   `json:"thinking_stream"`
	FirstTokenMs     int64  `json:"first_token_ms,omitempty"`
	RequestBody      string `json:"request_body,omitempty"`
	ResponseBody     string `json:"response_body,omitempty"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
	TotalTokens      int    `json:"total_tokens,omitempty"`
	Quota            int64  `json:"quota,omitempty"`
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
	HTTPStatus       int    `json:"http_status,omitempty"`
	SystemPromptRst  bool   `json:"system_prompt_reset,omitempty"`
	DurationMs       int64  `json:"duration_ms,omitempty"`
	// InboundHeaders 客户端进入本服务的 HTTP 请求头。
	InboundHeaders map[string][]string `json:"inbound_headers,omitempty"`
	// OutboundHeaders 本服务发往上游（渠道）的 HTTP 请求头。
	OutboundHeaders map[string][]string `json:"outbound_headers,omitempty"`
	// UpstreamResponseHeaders 上游 HTTP 响应头（若该次调用未走 HTTP  round-trip 则可能为空）。
	UpstreamResponseHeaders map[string][]string `json:"upstream_response_headers,omitempty"`
}

// Recorder 挂在 gin.Context 上，由 Relay 生命周期内填充。
type Recorder struct {
	mu        sync.Mutex
	startedAt time.Time

	requestID     string
	firstTokenCAS uint32
	handledCAS    uint32

	firstAt time.Time

	respBuilder strings.Builder
	usage       *relaymodel.Usage

	outboundRequestHdr   map[string][]string
	upstreamResponseHdr  map[string][]string
}

func logDir() string {
	if s := strings.TrimSpace(config.RequestAuditLogDir); s != "" {
		return s
	}
	return filepath.Join(cfg.LogDir, "request_audit")
}

func truncate(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "…(truncated)"
}

// 与明文请求体审计一致，头部 value 单值过长时截断，避免单行 JSONL 失控。
const maxHeaderValueRunes = 4096

var sensitiveHeaderNames = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"api-key":             {},
	"openai-api-key":      {},
}

// AuditHeaderSnapshot 将 http.Header 转为可序列化快照（敏感头脱敏、单值过长截断），供审计与操作日志共用。
func AuditHeaderSnapshot(h http.Header) map[string][]string {
	return auditHeaderSnapshot(h)
}

func auditHeaderSnapshot(h http.Header) map[string][]string {
	if len(h) == 0 {
		return nil
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		if strings.TrimSpace(k) != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	out := make(map[string][]string, len(keys))
	for _, k := range keys {
		lk := strings.ToLower(k)
		if _, redact := sensitiveHeaderNames[lk]; redact {
			out[k] = []string{"[REDACTED]"}
			continue
		}
		vv := h.Values(k)
		cp := make([]string, len(vv))
		for i, v := range vv {
			runes := []rune(v)
			if len(runes) > maxHeaderValueRunes {
				v = string(runes[:maxHeaderValueRunes]) + "…(truncated)"
			}
			cp[i] = v
		}
		out[k] = cp
	}
	return out
}

// UpstreamHeadersJSONForLog 汇总当前请求已记录的上游请求/响应头，供写入 logs.other（JSON 字符串）。
func UpstreamHeadersJSONForLog(c *gin.Context) string {
	if c == nil {
		return ""
	}
	m := map[string]interface{}{}
	if v, ok := c.Get(ctxkey.UpstreamRequestHeadersLog); ok {
		if hdrs, ok := v.(map[string][]string); ok && len(hdrs) > 0 {
			m["upstream_request_headers"] = hdrs
		}
	}
	if v, ok := c.Get(ctxkey.UpstreamResponseHeadersLog); ok {
		if hdrs, ok := v.(map[string][]string); ok && len(hdrs) > 0 {
			m["upstream_response_headers"] = hdrs
		}
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

// SnapUpstreamHTTP 记录一次上游 HTTP 往返的出站请求头与上游响应头（在 client.Do 返回后调用）。
func SnapUpstreamHTTP(c *gin.Context, req *http.Request, resp *http.Response) {
	if c == nil {
		return
	}
	if req != nil {
		c.Set(ctxkey.UpstreamRequestHeadersLog, AuditHeaderSnapshot(req.Header))
		if req.URL != nil {
			c.Set(ctxkey.UpstreamRequestMetaLog, map[string]interface{}{
				"method": req.Method,
				"url":    req.URL.String(),
			})
		}
	}
	if resp != nil {
		c.Set(ctxkey.UpstreamResponseHeadersLog, AuditHeaderSnapshot(resp.Header))
		c.Set(ctxkey.UpstreamResponseMetaLog, map[string]interface{}{
			"status":      resp.Status,
			"status_code": resp.StatusCode,
		})
	}
	if !config.RequestAuditEnabled {
		return
	}
	r := FromContext(c)
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if req != nil {
		r.outboundRequestHdr = auditHeaderSnapshot(req.Header)
	}
	if resp != nil {
		r.upstreamResponseHdr = auditHeaderSnapshot(resp.Header)
	}
}

// Attach 若开启审计且已识别用户则创建 Recorder 并放入上下文。
func Attach(c *gin.Context, relayStarted time.Time) *Recorder {
	if !config.RequestAuditEnabled || c == nil || c.GetInt(ctxkey.Id) <= 0 {
		return nil
	}
	rec := &Recorder{
		startedAt: relayStarted,
		requestID: c.GetString(helper.RequestIdKey),
	}
	c.Set(ctxkey.RequestAuditRecorder, rec)
	return rec
}

// FromContext 取出 Recorder（可能为 nil）。
func FromContext(c *gin.Context) *Recorder {
	if c == nil {
		return nil
	}
	v, ok := c.Get(ctxkey.RequestAuditRecorder)
	if !ok || v == nil {
		return nil
	}
	r, _ := v.(*Recorder)
	return r
}

// MarkStreamFirstToken 记录流式下首包要写出的时间（仅首次生效）。
func MarkStreamFirstToken(r *Recorder) {
	if r == nil || !atomic.CompareAndSwapUint32(&r.firstTokenCAS, 0, 1) {
		return
	}
	r.mu.Lock()
	r.firstAt = time.Now()
	r.mu.Unlock()
}

// AppendStreamText 累加流式拼出的文本片段（用于审计中的 response_body）。
func AppendStreamText(r *Recorder, piece string) {
	if r == nil || piece == "" {
		return
	}
	max := config.RequestAuditMaxBodyBytes
	r.mu.Lock()
	if r.respBuilder.Len() >= max {
		r.mu.Unlock()
		return
	}
	need := max - r.respBuilder.Len()
	if len(piece) > need {
		piece = piece[:need]
	}
	_, _ = r.respBuilder.WriteString(piece)
	r.mu.Unlock()
}

// SetNonStreamResponse 非流式完整响应正文。
func SetNonStreamResponse(r *Recorder, body string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.respBuilder.Reset()
	_, _ = r.respBuilder.WriteString(truncate(body, config.RequestAuditMaxBodyBytes))
	r.mu.Unlock()
}

// SetUsage 设置用量（与 billing 一致）。
func SetUsage(r *Recorder, u *relaymodel.Usage) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.usage = u
	r.mu.Unlock()
}

// FinalizeSuccess 写入成功审计行（幂等）。
func FinalizeSuccess(c *gin.Context, r *Recorder, model string, stream, thinking, sysPrompt bool, quota int64) {
	if r == nil || c == nil {
		return
	}
	if !atomic.CompareAndSwapUint32(&r.handledCAS, 0, 1) {
		return
	}
	r.writeEntry(c, model, stream, thinking, sysPrompt, quota, true, "", 200)
}

// FinalizeFailure 写入失败审计（未在 FinalizeSuccess 写入过时调用）。
func FinalizeFailure(c *gin.Context, r *Recorder, model string, stream, thinking bool, httpStatus int, errMsg string) {
	if r == nil || c == nil {
		return
	}
	if !atomic.CompareAndSwapUint32(&r.handledCAS, 0, 1) {
		return
	}
	r.writeEntry(c, model, stream, thinking, false, 0, false, errMsg, httpStatus)
}

// FinalizeHTTPResult 用于非标准 Relay（例如 POST /amap 高德 REST 代理）：按调用方给出的 HTTP 状态码与业务是否成功写入一条审计（幂等）。
func FinalizeHTTPResult(c *gin.Context, model string, httpStatus int, success bool, errMsg string, quota int64) {
	r := FromContext(c)
	if r == nil || c == nil {
		return
	}
	if !atomic.CompareAndSwapUint32(&r.handledCAS, 0, 1) {
		return
	}
	r.writeEntry(c, model, false, false, false, quota, success, errMsg, httpStatus)
}

func (r *Recorder) writeEntry(c *gin.Context, model string, stream, thinking bool, sysPrompt bool, quota int64, success bool, errMsg string, httpStatus int) {
	max := config.RequestAuditMaxBodyBytes
	reqStr := ""
	if raw, ok := c.Get(ctxkey.KeyRequestBody); ok && raw != nil {
		if b, ok := raw.([]byte); ok && len(b) > 0 {
			reqStr = truncate(string(b), max)
		}
	}

	r.mu.Lock()
	u := r.usage
	ftMs := int64(0)
	if !r.firstAt.IsZero() {
		ftMs = r.firstAt.Sub(r.startedAt).Milliseconds()
	}
	resp := r.respBuilder.String()
	outReq := r.outboundRequestHdr
	upResp := r.upstreamResponseHdr
	r.mu.Unlock()

	e := Entry{
		TimeMs:           time.Now().UnixMilli(),
		RequestID:        strings.TrimSpace(r.requestID),
		Method:           c.Request.Method,
		Path:             c.Request.URL.Path,
		UserID:           c.GetInt(ctxkey.Id),
		TokenID:          c.GetInt(ctxkey.TokenId),
		TokenName:        c.GetString(ctxkey.TokenName),
		ChannelID:        c.GetInt(ctxkey.ChannelId),
		ChannelName:      c.GetString(ctxkey.ChannelName),
		Model:            model,
		LogicalModel:     c.GetString(ctxkey.LogicalModel),
		Group:            c.GetString(ctxkey.Group),
		Stream:           stream,
		ThinkingStream:   thinking,
		FirstTokenMs:     ftMs,
		RequestBody:      reqStr,
		ResponseBody:     truncate(resp, max),
		Quota:            quota,
		Success:          success,
		Error:            errMsg,
		HTTPStatus:       httpStatus,
		SystemPromptRst:         sysPrompt,
		DurationMs:              time.Since(r.startedAt).Milliseconds(),
		InboundHeaders:          auditHeaderSnapshot(c.Request.Header),
		OutboundHeaders:         outReq,
		UpstreamResponseHeaders: upResp,
	}
	if u != nil {
		e.PromptTokens = u.PromptTokens
		e.CompletionTokens = u.CompletionTokens
		e.TotalTokens = u.TotalTokens
	}

	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	dir := logDir()
	_ = os.MkdirAll(dir, 0o750)
	name := "request_audit_" + time.Now().Format("2006-01-02") + ".jsonl"
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	_, _ = f.Write(line)
	_, _ = f.WriteString("\n")
	_ = f.Close()
}

// InferThinkingStream 根据请求体与结构推断是否包含 thinking / reasoning 类能力。
func InferThinkingStream(req *relaymodel.GeneralOpenAIRequest, raw []byte) bool {
	if req != nil {
		if req.Stream && req.ReasoningEffort != nil && strings.TrimSpace(*req.ReasoningEffort) != "" {
			return true
		}
	}
	if len(raw) == 0 {
		return false
	}
	s := strings.ToLower(string(raw))
	keys := []string{
		`"thinking"`, `"enable_thinking"`, `"thought"`, `"reasoning"`,
		`reasoning_effort`, `"verbosity"`, `"include_reasoning"`,
	}
	for _, k := range keys {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// InferStreamFromBody 根据 JSON 粗略判断是否 stream 请求（用于早期失败或未解析 body 时）。
func InferStreamFromBody(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	s := strings.ToLower(string(raw))
	return strings.Contains(s, `"stream":true`) || strings.Contains(s, `"stream": true`)
}

// IsFinalized 是否已写入成功或失败审计（幂等收尾用）。
func (r *Recorder) IsFinalized() bool {
	if r == nil {
		return true
	}
	return atomic.LoadUint32(&r.handledCAS) != 0
}

// FinalizeRelayErrorIfUnhandled 在 Relay 重试结束后对仍未写入的审计补一条失败记录。
func FinalizeRelayErrorIfUnhandled(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode) {
	if c == nil || bizErr == nil || bizErr.StatusCode == -1 {
		return
	}
	r := FromContext(c)
	if r == nil || r.IsFinalized() {
		return
	}
	modelName := strings.TrimSpace(c.GetString(ctxkey.RequestModel))
	if modelName == "" {
		modelName = strings.TrimSpace(c.GetString(ctxkey.OriginalModel))
	}
	var raw []byte
	if v, ok := c.Get(ctxkey.KeyRequestBody); ok && v != nil {
		if b, ok := v.([]byte); ok {
			raw = b
		}
	}
	thinking := InferThinkingStream(nil, raw)
	stream := InferStreamFromBody(raw)
	FinalizeFailure(c, r, modelName, stream, thinking, bizErr.StatusCode, bizErr.Error.Message)
}
