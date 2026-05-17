// Mock upstream: OpenAI / Anthropic / Gemini (generateContent) minimal handlers for E2E.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

var (
	openAICalls      atomic.Uint64
	anthropicCalls   atomic.Uint64
	geminiCalls      atomic.Uint64
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/__debug/stats", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"openai_chat_completions": openAICalls.Load(),
			"anthropic_messages":      anthropicCalls.Load(),
			"gemini_generate_content": geminiCalls.Load(),
		})
	})
	mux.HandleFunc("/v1/chat/completions", handleOpenAIChat)
	mux.HandleFunc("/v1/messages", handleAnthropicMessages)
	mux.HandleFunc("/v1beta/", handleGeminiV1BetaPrefix)

	addr := ":8080"
	log.Printf("e2e mock_upstream listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleOpenAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	openAICalls.Add(1)
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	model := "gpt-4o-mini"
	if m, ok := body["model"].(string); ok && m != "" {
		model = m
	}
	resp := map[string]any{
		"id":      "chatcmpl-e2e-mock",
		"object":  "chat.completion",
		"created": 1700000000,
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": "mock-openai:" + model},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{"prompt_tokens": 3, "completion_tokens": 5, "total_tokens": 8},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	anthropicCalls.Add(1)
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	model := "claude-3-haiku-20240307"
	if m, ok := body["model"].(string); ok && m != "" {
		model = m
	}
	resp := map[string]any{
		"id":   "msg_e2e_mock",
		"type": "message",
		"role": "assistant",
		"content": []any{map[string]any{
			"type": "text", "text": "mock-anthropic:" + model,
		}},
		"model":        model,
		"stop_reason":  "end_turn",
		"stop_sequence": nil,
		"usage":        map[string]any{"input_tokens": 4, "output_tokens": 6},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleGeminiV1BetaPrefix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1beta/")
	// models/{model}:generateContent or :streamGenerateContent
	if !strings.HasPrefix(path, "models/") {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(path, "models/")
	idx := strings.LastIndex(rest, ":")
	if idx <= 0 || idx >= len(rest)-1 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	model := rest[:idx]
	action := rest[idx+1:]
	if !strings.Contains(action, "generateContent") {
		http.Error(w, "only generateContent supported in mock", http.StatusNotImplemented)
		return
	}
	geminiCalls.Add(1)
	_ = r.ParseForm()
	resp := map[string]any{
		"candidates": []any{map[string]any{
			"content": map[string]any{
				"role":  "model",
				"parts": []any{map[string]any{"text": "mock-gemini:" + model}},
			},
			"finishReason": "STOP",
			"index":        0,
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
