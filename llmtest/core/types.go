// Package core 实现一套协议无关的大模型测试框架：把「场景」翻译成各家协议
// （OpenAI / Anthropic / Gemini）的线格式，发送请求，归一化响应，再做严格断言与语义检查。
package core

import "encoding/json"

// EndpointKind 区分被测接口形态。
type EndpointKind string

const (
	KindChat       EndpointKind = "chat"       // 对话补全（OpenAI chat / Anthropic messages / Gemini generateContent）
	KindCompletion EndpointKind = "completion" // 传统文本补全（OpenAI /v1/completions）
	KindEmbedding  EndpointKind = "embedding"  // 向量嵌入（OpenAI /v1/embeddings）
)

// Protocol 区分上游协议，决定请求/响应的线格式。
type Protocol string

const (
	ProtocolOpenAI    Protocol = "openai"
	ProtocolAnthropic Protocol = "anthropic"
	ProtocolGemini    Protocol = "gemini"
)

// Message 协议无关的对话消息。
type Message struct {
	Role     string // system | user | assistant
	Text     string
	ImageB64 string // 多模态：base64 编码的图片（不含 data: 前缀）
	ImageTyp string // 图片 MIME，如 image/png
}

// Tool 协议无关的工具/函数定义（参数为 JSON Schema）。
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// ToolCall 归一化后的模型工具调用。
type ToolCall struct {
	Name string
	Args json.RawMessage
}

// Usage token 统计（尽力解析，缺失记 0）。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ChatRequest 协议无关的对话/补全请求。
type ChatRequest struct {
	Model       string
	System      string
	Messages    []Message
	Tools       []Tool
	Stream      bool
	JSONMode    bool
	MaxTokens   int
	Temperature float64
	Prompt      string // 仅 KindCompletion 使用
}

// Kind 依据请求内容推断接口形态：设置了 Prompt 视为传统补全，否则为对话。
func (r ChatRequest) Kind() EndpointKind {
	if r.Prompt != "" {
		return KindCompletion
	}
	return KindChat
}

// ChatResponse 归一化的对话响应。
type ChatResponse struct {
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
	Chunks       int    // 流式分块计数（非流式为 0）
	Raw          []byte // 原始响应；流式时为拼接的事件日志，便于排查
}

// EmbeddingRequest 向量嵌入请求。
type EmbeddingRequest struct {
	Model string
	Input []string
}

// EmbeddingResponse 归一化的向量响应。
type EmbeddingResponse struct {
	Vectors [][]float64
	Usage   Usage
	Raw     []byte
}
