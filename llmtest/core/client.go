package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 是带超时的 HTTP 客户端封装，支持普通请求与 SSE 流式请求。
type Client struct {
	hc *http.Client
}

// NewClient 创建客户端，timeout 为单次请求的总超时。
func NewClient(timeout time.Duration) *Client {
	return &Client{hc: &http.Client{Timeout: timeout}}
}

// httpError 携带状态码与响应体，便于断言层判断。
type httpError struct {
	Status int
	Body   string
}

func (e *httpError) Error() string {
	body := e.Body
	if len(body) > 500 {
		body = body[:500] + "...(truncated)"
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, body)
}

// postJSON 发送 JSON 请求并返回响应体；非 2xx 返回 *httpError。
func (c *Client) postJSON(ctx context.Context, url string, headers map[string]string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &httpError{Status: resp.StatusCode, Body: string(respBody)}
	}
	return respBody, nil
}

// sseEvent 表示一条 SSE 事件（event 名 + data 负载）。
type sseEvent struct {
	Event string
	Data  string
}

// postSSE 发送 JSON 请求并按 SSE 解析流式响应，逐条回调事件。
// onEvent 返回 error 可提前中止读取。
func (c *Client) postSSE(ctx context.Context, url string, headers map[string]string, payload any, onEvent func(sseEvent) error) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, &httpError{Status: resp.StatusCode, Body: string(b)}
	}

	var rawLog bytes.Buffer
	reader := bufio.NewReader(resp.Body)
	var curEvent, curData string
	flush := func() error {
		if curData == "" && curEvent == "" {
			return nil
		}
		ev := sseEvent{Event: curEvent, Data: curData}
		curEvent, curData = "", ""
		if ev.Data == "" {
			return nil
		}
		return onEvent(ev)
	}
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			rawLog.WriteString(line)
			trimmed := strings.TrimRight(line, "\r\n")
			switch {
			case trimmed == "": // 事件分隔空行
				if ferr := flush(); ferr != nil {
					return rawLog.Bytes(), ferr
				}
			case strings.HasPrefix(trimmed, "event:"):
				curEvent = strings.TrimSpace(trimmed[len("event:"):])
			case strings.HasPrefix(trimmed, "data:"):
				d := strings.TrimSpace(trimmed[len("data:"):])
				if curData == "" {
					curData = d
				} else {
					curData += "\n" + d
				}
			}
		}
		if err == io.EOF {
			if ferr := flush(); ferr != nil {
				return rawLog.Bytes(), ferr
			}
			break
		}
		if err != nil {
			return rawLog.Bytes(), err
		}
	}
	return rawLog.Bytes(), nil
}

// getJSON 发送 GET 请求（Gemini 等部分接口需要），返回响应体。
func (c *Client) getJSON(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &httpError{Status: resp.StatusCode, Body: string(respBody)}
	}
	return respBody, nil
}
