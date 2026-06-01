package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type mockConfig struct {
	listen          string
	latency         time.Duration
	chunks          int
	chunkDelay      time.Duration
	promptTokens    int
	completionToken int
	model           string
}

func runMockCmd(args []string) {
	fs := flag.NewFlagSet("mock", flag.ExitOnError)
	var cfg mockConfig
	fs.StringVar(&cfg.listen, "listen", ":9000", "listen address")
	fs.DurationVar(&cfg.latency, "latency", 50*time.Millisecond, "simulated upstream think-time before responding")
	fs.IntVar(&cfg.chunks, "chunks", 16, "number of SSE chunks in streaming mode")
	fs.DurationVar(&cfg.chunkDelay, "chunk-delay", 8*time.Millisecond, "delay between streaming chunks (simulated token rate)")
	fs.IntVar(&cfg.promptTokens, "prompt-tokens", 32, "prompt_tokens to report in usage")
	fs.IntVar(&cfg.completionToken, "completion-tokens", 64, "completion_tokens to report in usage")
	fs.StringVar(&cfg.model, "model", "gpt-4o-mini", "model name echoed back")
	_ = fs.Parse(args)

	var served int64
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&served, 1)
		handleChat(w, r, cfg)
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   []any{map[string]any{"id": cfg.model, "object": "model", "owned_by": "mock"}},
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	fmt.Printf("mock upstream listening on %s  (latency=%s, chunks=%d, chunk-delay=%s)\n",
		cfg.listen, cfg.latency, cfg.chunks, cfg.chunkDelay)
	fmt.Printf("point a one-api channel's base_url here, e.g. http://127.0.0.1%s\n", normalizeListen(cfg.listen))

	srv := &http.Server{Addr: cfg.listen, Handler: mux}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "mock server error: %v\n", err)
		os.Exit(1)
	}
}

func normalizeListen(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	return addr
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg mockConfig) {
	var req struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Model == "" {
		req.Model = cfg.model
	}

	if cfg.latency > 0 {
		time.Sleep(cfg.latency)
	}

	if req.Stream {
		streamChat(w, req.Model, cfg)
		return
	}

	resp := map[string]any{
		"id":      "chatcmpl-mock",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []any{
			map[string]any{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": mockContent},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     cfg.promptTokens,
			"completion_tokens": cfg.completionToken,
			"total_tokens":      cfg.promptTokens + cfg.completionToken,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func streamChat(w http.ResponseWriter, model string, cfg mockConfig) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	created := time.Now().Unix()
	for i := 0; i < cfg.chunks; i++ {
		chunk := map[string]any{
			"id":      "chatcmpl-mock",
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []any{
				map[string]any{
					"index": 0,
					"delta": map[string]any{"content": "tok "},
				},
			},
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		if cfg.chunkDelay > 0 {
			time.Sleep(cfg.chunkDelay)
		}
	}
	// 末尾 usage chunk（stream_options.include_usage 风格）+ [DONE]
	final := map[string]any{
		"id":      "chatcmpl-mock",
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}},
		"usage": map[string]any{
			"prompt_tokens":     cfg.promptTokens,
			"completion_tokens": cfg.completionToken,
			"total_tokens":      cfg.promptTokens + cfg.completionToken,
		},
	}
	b, _ := json.Marshal(final)
	fmt.Fprintf(w, "data: %s\n\n", b)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

const mockContent = "This is a mock completion from the bench upstream."
