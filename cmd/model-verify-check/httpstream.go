package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/songquanpeng/one-api/cmd/internal/apitest"
)

type httpExchange struct {
	StatusCode      int
	RequestHeaders  map[string][]string
	RequestBody     string
	ResponseHeaders map[string][]string
	ResponseBody    string
	AssembledText   string
}

func postAnthropicStream(cli *apitest.Client, path string, body map[string]any, extra map[string]string) (*httpExchange, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, cli.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	auth := cli.Token
	if auth != "" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for k, v := range extra {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}

	res, err := cli.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	ex := &httpExchange{
		StatusCode:      res.StatusCode,
		RequestHeaders:  redactHeaders(req.Header),
		RequestBody:     string(raw),
		ResponseHeaders: cloneHeaders(res.Header),
		ResponseBody:    string(respBody),
	}

	if res.StatusCode != http.StatusOK {
		return ex, nil
	}

	ct := strings.ToLower(res.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/event-stream") {
		ex.AssembledText = parseAnthropicSSE(ex.ResponseBody)
		return ex, nil
	}

	// 少数网关流式请求仍返回整段 JSON。
	snippet, err := apitest.AnthropicMessageSnippet(respBody)
	if err == nil {
		ex.AssembledText = snippet
	}
	return ex, nil
}

func postAnthropicJSON(cli *apitest.Client, path string, body map[string]any, extra map[string]string) (*httpExchange, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, cli.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	auth := cli.Token
	if auth != "" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for k, v := range extra {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}

	res, err := cli.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	ex := &httpExchange{
		StatusCode:      res.StatusCode,
		RequestHeaders:  redactHeaders(req.Header),
		RequestBody:     string(raw),
		ResponseHeaders: cloneHeaders(res.Header),
		ResponseBody:    string(respBody),
	}

	if res.StatusCode == http.StatusOK {
		snippet, errSnippet := apitest.AnthropicMessageSnippet(respBody)
		if errSnippet == nil {
			ex.AssembledText = snippet
		}
	}
	return ex, nil
}

func parseAnthropicSSE(raw string) string {
	var textB strings.Builder
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var e struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			ContentBlock struct {
				Text string `json:"text"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			continue
		}
		switch e.Type {
		case "content_block_delta":
			if e.Delta.Type == "text_delta" {
				textB.WriteString(e.Delta.Text)
			}
		case "content_block_start":
			if e.ContentBlock.Text != "" {
				textB.WriteString(e.ContentBlock.Text)
			}
		}
	}
	return strings.TrimSpace(textB.String())
}

func redactHeaders(h http.Header) map[string][]string {
	return sanitizeHeaders(cloneHeaders(h), true)
}

func cloneHeaders(h http.Header) map[string][]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string][]string, len(h))
	for k, vs := range h {
		cp := make([]string, len(vs))
		copy(cp, vs)
		out[k] = cp
	}
	return out
}

func sanitizeHeaders(h map[string][]string, redact bool) map[string][]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string][]string, len(h))
	for k, vs := range h {
		lk := strings.ToLower(k)
		if redact && (lk == "authorization" || lk == "x-api-key") {
			masked := make([]string, len(vs))
			for i, v := range vs {
				masked[i] = redactSecret(v)
			}
			out[k] = masked
			continue
		}
		out[k] = vs
	}
	return out
}

func redactSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	plain := strings.TrimPrefix(strings.TrimPrefix(v, "Bearer "), "bearer ")
	if len(plain) <= 12 {
		return "[REDACTED]"
	}
	return plain[:6] + "…" + plain[len(plain)-4:]
}

func formatHeaders(h map[string][]string) string {
	if len(h) == 0 {
		return "(empty)"
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		for _, v := range h[k] {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
