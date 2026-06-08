// OpenAI 兼容 Chat Completions 联调示例（One API / 任意 OpenAI 兼容网关）
//
// 每次请求结束后打印 usage（prompt / completion / total tokens）。
//
// 用法:
//
//	export ONE_API_BASE="http://127.0.0.1:3000"                              # One API
//	export ONE_API_BASE="https://dashscope.aliyuncs.com/compatible-mode/v1" # 阿里云等（已含 /v1）
//	export ONE_API_TOKEN="sk-xxxx"
//	export MODEL_CHAT="gpt-4o-mini"
//	go run .
//
//	go run . -stream "用三句话介绍 Go 语言"
//	go run . -debug "1+1=?"
//	go run . -stream -debug "hello"
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:3000"

func main() {
	stream := flag.Bool("stream", false, "使用流式 chat/completions，并在最后一个 chunk 打印 usage")
	debug := flag.Bool("debug", false, "打印原始 response JSON（流式时为每个 SSE data chunk）")
	message := flag.String("message", "用一句话说明 token usage 是什么。", "user 消息内容")
	flag.Parse()

	baseURL := strings.TrimRight(strings.TrimSpace(envOr("ONE_API_BASE", defaultBaseURL)), "/")
	token := strings.TrimSpace(os.Getenv("ONE_API_TOKEN"))
	model := strings.TrimSpace(envOr("MODEL_CHAT", "gpt-4o-mini"))

	if token == "" {
		log.Fatal("请设置环境变量 ONE_API_TOKEN（One API 控制台创建的 API Token）")
	}

	fmt.Printf("base=%s model=%s stream=%v debug=%v\n", baseURL, model, *stream, *debug)
	if *debug {
		fmt.Printf("endpoint=%s\n", chatCompletionsURL(baseURL))
	}
	fmt.Println("---")

	if *stream {
		if err := runStream(baseURL, token, model, *message, *debug); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := runOnce(baseURL, token, model, *message, *debug); err != nil {
		log.Fatal(err)
	}
}

func runOnce(baseURL, token, model, userMessage string, debug bool) error {
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": userMessage},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, chatCompletionsURL(baseURL), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 800))
	}

	if debug {
		printRawJSON("[debug:response]", respBody)
	}

	var out chatCompletionResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("解析响应: %w", err)
	}

	text := extractAssistantText(out.Choices)
	fmt.Printf("[assistant] %s\n", text)
	if out.Usage == nil {
		fmt.Println("[usage:non-stream] 响应未包含 usage 字段")
		return nil
	}
	printUsage("non-stream", *out.Usage)
	return nil
}

func runStream(baseURL, token, model, userMessage string, debug bool) error {
	body := map[string]any{
		"model":  model,
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, chatCompletionsURL(baseURL), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(b), 800))
	}

	fmt.Print("[assistant] ")
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lastUsage *usage
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			if debug && payload == "[DONE]" {
				fmt.Println("[debug:chunk] [DONE]")
			}
			continue
		}

		if debug {
			printRawJSON("[debug:chunk]", []byte(payload))
		}

		var chunk chatCompletionResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("解析 SSE chunk: %w (payload=%s)", err, truncate(payload, 200))
		}
		if delta := extractDeltaText(chunk.Choices); delta != "" {
			fmt.Print(delta)
		}
		if chunk.Usage != nil && hasUsage(*chunk.Usage) {
			u := *chunk.Usage
			lastUsage = &u
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	fmt.Println()

	if lastUsage == nil {
		fmt.Println("[usage] stream 响应未包含 usage；请确认上游支持 stream_options.include_usage")
		return nil
	}
	printUsage("stream", *lastUsage)
	return nil
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	PromptTokensDetails map[string]any `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails map[string]any `json:"completion_tokens_details,omitempty"`
}

type chatChoice struct {
	Index        int            `json:"index"`
	Message      map[string]any `json:"message,omitempty"`
	Delta        map[string]any `json:"delta,omitempty"`
	FinishReason *string        `json:"finish_reason"`
}

type chatCompletionResponse struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   *usage       `json:"usage,omitempty"`
}

func extractAssistantText(choices []chatChoice) string {
	if len(choices) == 0 || choices[0].Message == nil {
		return ""
	}
	if s, ok := choices[0].Message["content"].(string); ok {
		return s
	}
	return fmt.Sprintf("%v", choices[0].Message["content"])
}

func extractDeltaText(choices []chatChoice) string {
	if len(choices) == 0 || choices[0].Delta == nil {
		return ""
	}
	if s, ok := choices[0].Delta["content"].(string); ok {
		return s
	}
	return ""
}

func hasUsage(u usage) bool {
	return u.PromptTokens > 0 || u.CompletionTokens > 0 || u.TotalTokens > 0
}

func printUsage(mode string, u usage) {
	fmt.Printf("[usage:%s] prompt_tokens=%d completion_tokens=%d total_tokens=%d\n",
		mode, u.PromptTokens, u.CompletionTokens, u.TotalTokens)
	if len(u.PromptTokensDetails) > 0 {
		fmt.Printf("[usage:%s] prompt_tokens_details=%s\n", mode, mustJSON(u.PromptTokensDetails))
	}
	if len(u.CompletionTokensDetails) > 0 {
		fmt.Printf("[usage:%s] completion_tokens_details=%s\n", mode, mustJSON(u.CompletionTokensDetails))
	}
}

func chatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func printRawJSON(label string, raw []byte) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		fmt.Printf("%s (raw, not JSON): %s\n", label, string(raw))
		return
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("%s %s\n", label, string(raw))
		return
	}
	fmt.Printf("%s\n%s\n", label, string(pretty))
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
