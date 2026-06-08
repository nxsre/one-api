package core

import (
	"context"
	"fmt"
	"strings"
)

// Provider 把协议无关请求翻译成具体协议线格式，发送并归一化响应。
type Provider interface {
	Protocol() Protocol
	Supports(kind EndpointKind) bool
	Chat(ctx context.Context, t Target, req ChatRequest) (*ChatResponse, error)
	Embedding(ctx context.Context, t Target, req EmbeddingRequest) (*EmbeddingResponse, error)
}

// NewProvider 按协议创建对应 Provider。
func NewProvider(p Protocol, client *Client) (Provider, error) {
	switch p {
	case ProtocolOpenAI:
		return &openAIProvider{client: client}, nil
	case ProtocolAnthropic:
		return &anthropicProvider{client: client}, nil
	case ProtocolGemini:
		return &geminiProvider{client: client}, nil
	default:
		return nil, fmt.Errorf("未知协议: %s", p)
	}
}

// trimBase 去掉 base_url 末尾的 "/" 与重复的版本段，便于安全拼接路径。
func trimBase(base string) string {
	base = strings.TrimRight(base, "/")
	for _, suf := range []string{"/v1", "/v1beta"} {
		if strings.HasSuffix(base, suf) {
			base = strings.TrimSuffix(base, suf)
		}
	}
	return base
}
