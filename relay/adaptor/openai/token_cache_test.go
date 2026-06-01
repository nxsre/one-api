package openai

import (
	"strings"
	"testing"

	"github.com/songquanpeng/one-api/relay/model"
)

// TestCachedTokenNumMatchesUncached 保证缓存路径与未缓存路径结果完全一致
// （包括短串命中、二次命中、以及超长文本绕过缓存的情况）。
func TestCachedTokenNumMatchesUncached(t *testing.T) {
	InitTokenEncoders()
	const m = "gpt-3.5-turbo"
	enc := getTokenEncoder(m)
	cases := []string{
		"", "user", "assistant", "system",
		"Hello, world!",
		"You are a helpful assistant.",
		strings.Repeat("token ", 50),
		strings.Repeat("x", 600), // > tokenCountCacheMaxKeyLen，应绕过缓存
	}
	for _, s := range cases {
		want := getTokenNum(enc, s)
		got1 := cachedTokenNum(m, enc, s) // 首次：填充缓存
		got2 := cachedTokenNum(m, enc, s) // 二次：命中缓存
		if got1 != want || got2 != want {
			t.Errorf("text %q (len %d): want %d, got1 %d got2 %d", truncate(s), len(s), want, got1, got2)
		}
	}
}

// TestCountTokenMessagesDeterministic 保证 CountTokenMessages 缓存前后稳定且非零。
func TestCountTokenMessagesDeterministic(t *testing.T) {
	InitTokenEncoders()
	msgs := sampleConversation()
	a := CountTokenMessages(msgs, "gpt-3.5-turbo")
	b := CountTokenMessages(msgs, "gpt-3.5-turbo")
	if a != b || a <= 0 {
		t.Fatalf("expected stable positive count, got a=%d b=%d", a, b)
	}
}

func sampleConversation() []model.Message {
	return []model.Message{
		{Role: "system", Content: "You are a helpful assistant that answers concisely."},
		{Role: "user", Content: "What is the capital of France?"},
		{Role: "assistant", Content: "The capital of France is Paris."},
		{Role: "user", Content: "And the capital of Japan?"},
		{Role: "assistant", Content: "The capital of Japan is Tokyo."},
		{Role: "user", Content: "Thanks!"},
	}
}

func truncate(s string) string {
	if len(s) > 32 {
		return s[:32] + "..."
	}
	return s
}

// countMessagesUncached 复刻 CountTokenMessages 的循环，但全部走未缓存的 getTokenNum，
// 用于在基准测试中量化缓存收益（仅覆盖纯字符串消息，足够对比）。
func countMessagesUncached(messages []model.Message, m string) int {
	enc := getTokenEncoder(m)
	tokenNum := 0
	for _, message := range messages {
		tokenNum += 3
		if s, ok := message.Content.(string); ok {
			tokenNum += getTokenNum(enc, s)
		}
		tokenNum += getTokenNum(enc, message.Role)
	}
	return tokenNum + 3
}

// BenchmarkCountTokenMessages_Cached：内容重复时的稳态（role/系统提示命中缓存）。
func BenchmarkCountTokenMessages_Cached(b *testing.B) {
	InitTokenEncoders()
	msgs := sampleConversation()
	_ = CountTokenMessages(msgs, "gpt-3.5-turbo") // 预热缓存
	for b.Loop() {
		_ = CountTokenMessages(msgs, "gpt-3.5-turbo")
	}
}

// BenchmarkCountTokenMessages_Uncached：同样的消息全部重新编码（无缓存）。
func BenchmarkCountTokenMessages_Uncached(b *testing.B) {
	InitTokenEncoders()
	msgs := sampleConversation()
	for b.Loop() {
		_ = countMessagesUncached(msgs, "gpt-3.5-turbo")
	}
}

// BenchmarkRolesOnly：最贴近真实流量——每请求 role 都重复，用户内容各异。
// 这里只编码 role，体现缓存对高频短串的收益。
func BenchmarkRolesOnly_Cached(b *testing.B) {
	InitTokenEncoders()
	enc := getTokenEncoder("gpt-3.5-turbo")
	roles := []string{"system", "user", "assistant", "user", "assistant", "user"}
	for b.Loop() {
		for _, r := range roles {
			_ = cachedTokenNum("gpt-3.5-turbo", enc, r)
		}
	}
}

func BenchmarkRolesOnly_Uncached(b *testing.B) {
	InitTokenEncoders()
	enc := getTokenEncoder("gpt-3.5-turbo")
	roles := []string{"system", "user", "assistant", "user", "assistant", "user"}
	for b.Loop() {
		for _, r := range roles {
			_ = getTokenNum(enc, r)
		}
	}
}
