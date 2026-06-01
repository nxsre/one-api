package main

import (
	"os"
	"strings"
	"testing"
)

func requireEnv(t *testing.T) (baseURL, token, model string) {
	t.Helper()
	token = strings.TrimSpace(os.Getenv("ONE_API_TOKEN"))
	if token == "" {
		t.Skip("跳过：未设置 ONE_API_TOKEN")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(envOr("ONE_API_BASE", defaultBaseURL)), "/")
	model = strings.TrimSpace(envOr("MODEL_CHAT", "gpt-4o-mini"))
	return baseURL, token, model
}

func TestChatCompletionsURL(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"http://127.0.0.1:3000", "http://127.0.0.1:3000/v1/chat/completions"},
		{"http://127.0.0.1:3000/", "http://127.0.0.1:3000/v1/chat/completions"},
		{
			"https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1",
			"https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/chat/completions",
		},
	}
	for _, tc := range tests {
		if got := chatCompletionsURL(tc.base); got != tc.want {
			t.Fatalf("base=%q got=%q want=%q", tc.base, got, tc.want)
		}
	}
}

func TestChatCompletionUsage(t *testing.T) {
	baseURL, token, model := requireEnv(t)
	if err := runOnce(baseURL, token, model, "回复 OK 两个字母", false); err != nil {
		t.Fatal(err)
	}
}

func TestChatCompletionStreamUsage(t *testing.T) {
	baseURL, token, model := requireEnv(t)
	if err := runStream(baseURL, token, model, "数到 3", false); err != nil {
		t.Fatal(err)
	}
}
