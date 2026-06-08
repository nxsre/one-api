package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestProvider() (*openAIProvider, Target) {
	return &openAIProvider{client: NewClient(5 * time.Second)}, Target{
		Name: "mock", Protocol: ProtocolOpenAI, BaseURL: "", APIKey: "sk-test",
		Models: ModelSet{Chat: "gpt-test", Embedding: "emb-test"},
	}
}

func TestOpenAIChatNonStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "gpt-test") {
			t.Errorf("请求体缺少模型名: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"The capital is Paris."},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer srv.Close()

	p, tgt := newTestProvider()
	tgt.BaseURL = srv.URL
	resp, err := p.Chat(context.Background(), tgt, ChatRequest{Model: "gpt-test", Messages: []Message{{Role: "user", Text: "capital of France?"}}})
	if err != nil {
		t.Fatalf("Chat 出错: %v", err)
	}
	if !strings.Contains(resp.Text, "Paris") {
		t.Errorf("期望文本含 Paris，实际 %q", resp.Text)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("usage 解析错误: %+v", resp.Usage)
	}
	if fails := Evaluate(Expectation{NonEmptyText: true, MustContain: []string{"paris"}, WantUsage: true}, resp, nil); len(fails) != 0 {
		t.Errorf("断言不应失败: %v", fails)
	}
}

func TestOpenAIChatStreamWithToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"get_weather","arguments":"{\"loc"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":\"Tokyo\"}"}}]}}]}`,
			`{"choices":[{"delta":{"content":"done"},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":8,"completion_tokens":3,"total_tokens":11}}`,
			`[DONE]`,
		}
		for _, c := range chunks {
			io.WriteString(w, "data: "+c+"\n\n")
		}
	}))
	defer srv.Close()

	p, tgt := newTestProvider()
	tgt.BaseURL = srv.URL
	resp, err := p.Chat(context.Background(), tgt, ChatRequest{Model: "gpt-test", Stream: true, Tools: []Tool{weatherTool},
		Messages: []Message{{Role: "user", Text: "weather in Tokyo"}}})
	if err != nil {
		t.Fatalf("Chat 出错: %v", err)
	}
	if resp.Chunks < 2 {
		t.Errorf("分块数应 >=2，实际 %d", resp.Chunks)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("工具调用解析错误: %+v", resp.ToolCalls)
	}
	var args map[string]any
	if err := json.Unmarshal(resp.ToolCalls[0].Args, &args); err != nil {
		t.Fatalf("拼接后的工具参数非法 JSON: %s", resp.ToolCalls[0].Args)
	}
	if args["location"] != "Tokyo" {
		t.Errorf("工具参数 location 错误: %v", args)
	}
	if fails := Evaluate(Expectation{ToolName: "get_weather", ToolArgKeys: []string{"location"}, MinChunks: 2}, resp, nil); len(fails) != 0 {
		t.Errorf("断言不应失败: %v", fails)
	}
}

func TestOpenAIEmbeddingSimilarity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// cat 与 kitten 接近，与 stock 远离。
		io.WriteString(w, `{"data":[
			{"index":0,"embedding":[1,0,0]},
			{"index":1,"embedding":[0.9,0.1,0]},
			{"index":2,"embedding":[0,0,1]}
		],"usage":{"prompt_tokens":6,"total_tokens":6}}`)
	}))
	defer srv.Close()

	p, tgt := newTestProvider()
	tgt.BaseURL = srv.URL
	resp, err := p.Embedding(context.Background(), tgt, EmbeddingRequest{Model: "emb-test", Input: []string{"cat", "kitten", "stock"}})
	if err != nil {
		t.Fatalf("Embedding 出错: %v", err)
	}
	if len(resp.Vectors) != 3 {
		t.Fatalf("向量条数错误: %d", len(resp.Vectors))
	}
	if fails := Evaluate(Expectation{EmbeddingCount: 3, EmbeddingMin: 1, SimilarityCheck: true}, nil, resp); len(fails) != 0 {
		t.Errorf("语义相似度断言不应失败: %v", fails)
	}
}

func TestEvaluateCatchesBadJSON(t *testing.T) {
	resp := &ChatResponse{Text: "this is not json"}
	fails := Evaluate(Expectation{JSONObject: true, JSONKeys: []string{"name"}}, resp, nil)
	if len(fails) == 0 {
		t.Errorf("非 JSON 文本应触发断言失败")
	}
}

func TestExtractJSONObjectWithFence(t *testing.T) {
	obj, ok := extractJSONObject("```json\n{\"name\":\"Alice\",\"age\":30}\n```")
	if !ok || obj["name"] != "Alice" {
		t.Errorf("围栏 JSON 解析失败: %v ok=%v", obj, ok)
	}
}
