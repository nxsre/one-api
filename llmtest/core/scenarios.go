package core

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// Case 是一个具体可执行的测试用例（已绑定输入与期望）。
type Case struct {
	Title string
	Kind  EndpointKind
	Chat  *ChatRequest
	Emb   *EmbeddingRequest
	Exp   Expectation
}

// Scenario 是一类测试场景，可由随机种子或自定义输入实例化为 Case。
type Scenario struct {
	ID          string
	Kind        EndpointKind
	Desc        string
	Stream      bool
	NeedsTools  bool
	NeedsVision bool
	NeedsJSON   bool
	// Build 依据随机源与可选自定义提示生成一个 Case。custom 非空时使用自定义输入并放宽语义断言。
	Build func(rng *rand.Rand, custom string, m ModelSet) Case
}

// weatherTool 工具调用场景使用的天气查询工具。
var weatherTool = Tool{
	Name:        "get_weather",
	Description: "Get the current weather for a given location.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{"type": "string", "description": "City name, e.g. Tokyo"},
		},
		"required": []string{"location"},
	},
}

// AllScenarios 返回全部内置场景。
func AllScenarios() []Scenario {
	return []Scenario{
		{
			ID: "simple_chat", Kind: KindChat, Desc: "基础事实问答（语义校验答案）",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				f := pick(rng, factPrompts)
				prompt, exp := f.Q, Expectation{NonEmptyText: true, MustContain: []string{f.A}, WantUsage: true}
				if custom != "" {
					prompt, exp = custom, Expectation{NonEmptyText: true, WantUsage: true}
				}
				return Case{Title: "simple_chat", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 64, Temperature: 0,
					Messages: []Message{{Role: "user", Text: prompt}},
				}}
			},
		},
		{
			ID: "math", Kind: KindChat, Desc: "随机算术（结果可精确校验）",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				a, b := rng.Intn(80)+12, rng.Intn(80)+12
				prompt := fmt.Sprintf("Compute %d * %d. Reply with only the integer result, no commas, no words.", a, b)
				exp := Expectation{NonEmptyText: true, MustContain: []string{strconv.Itoa(a * b)}}
				if custom != "" {
					prompt, exp = custom, Expectation{NonEmptyText: true}
				}
				return Case{Title: "math", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 16, Temperature: 0,
					Messages: []Message{{Role: "user", Text: prompt}},
				}}
			},
		},
		{
			ID: "multi_turn", Kind: KindChat, Desc: "多轮上下文记忆",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				n := rng.Intn(900) + 100
				msgs := []Message{
					{Role: "user", Text: fmt.Sprintf("Please remember this number: %d. Just acknowledge briefly.", n)},
					{Role: "assistant", Text: "OK, I will remember it."},
					{Role: "user", Text: "What number did I ask you to remember? Reply with only the number."},
				}
				exp := Expectation{NonEmptyText: true, MustContain: []string{strconv.Itoa(n)}}
				if custom != "" {
					msgs = []Message{{Role: "user", Text: custom}}
					exp = Expectation{NonEmptyText: true}
				}
				return Case{Title: "multi_turn", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 16, Temperature: 0, Messages: msgs,
				}}
			},
		},
		{
			ID: "system_prompt", Kind: KindChat, Desc: "system 指令遵循",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				token := pick(rng, []string{"BANANA", "DONE", "FINISHED"})
				sys := fmt.Sprintf("You must end every response with the exact word %s.", token)
				user := "Say a short friendly greeting."
				exp := Expectation{NonEmptyText: true, MustContain: []string{token}}
				if custom != "" {
					user, exp = custom, Expectation{NonEmptyText: true}
				}
				return Case{Title: "system_prompt", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, System: sys, MaxTokens: 64, Temperature: 0,
					Messages: []Message{{Role: "user", Text: user}},
				}}
			},
		},
		{
			ID: "streaming", Kind: KindChat, Desc: "流式输出（校验分块）", Stream: true,
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				// 要求多句以确保产生多个流式分块（部分网关对极短回复会合并为单块）。
				user := "Write three short sentences about the ocean."
				if custom != "" {
					user = custom
				}
				// MinChunks:1 —— 校验 SSE 流式路径确实返回了事件且能拼出文本；
				// 分块数仅作展示：部分网关会把短回复（尤其 Gemini）合并为单个 SSE 事件。
				return Case{Title: "streaming", Kind: KindChat, Exp: Expectation{NonEmptyText: true, MinChunks: 1}, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 200, Temperature: 0, Stream: true,
					Messages: []Message{{Role: "user", Text: user}},
				}}
			},
		},
		{
			ID: "tool_call", Kind: KindChat, Desc: "函数/工具调用", NeedsTools: true,
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				city := pick(rng, []string{"Tokyo", "Paris", "Cairo", "Berlin"})
				user := fmt.Sprintf("What is the weather like in %s right now? Use the available tool.", city)
				exp := Expectation{ToolName: "get_weather", ToolArgKeys: []string{"location"}}
				if custom != "" {
					user, exp = custom, Expectation{ToolName: "get_weather"}
				}
				return Case{Title: "tool_call", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 256, Temperature: 0, Tools: []Tool{weatherTool},
					Messages: []Message{{Role: "user", Text: user}},
				}}
			},
		},
		{
			ID: "json_mode", Kind: KindChat, Desc: "结构化 JSON 输出", NeedsJSON: true,
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				user := `Return ONLY a JSON object describing a person named Alice who is 30 years old, using exactly the keys "name" and "age".`
				exp := Expectation{JSONObject: true, JSONKeys: []string{"name", "age"}}
				if custom != "" {
					user, exp = custom, Expectation{JSONObject: true}
				}
				return Case{Title: "json_mode", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 128, Temperature: 0, JSONMode: true,
					Messages: []Message{{Role: "user", Text: user}},
				}}
			},
		},
		{
			ID: "vision", Kind: KindChat, Desc: "多模态：纯色图片识别", NeedsVision: true,
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				c := pick(rng, solidColors)
				img := makeSolidImagePNG(c.RGBA, 48)
				exp := Expectation{NonEmptyText: true, MustContain: []string{c.Word}}
				text := "What is the dominant color of this image? Answer with a single English color word."
				if custom != "" {
					text, exp = custom, Expectation{NonEmptyText: true}
				}
				return Case{Title: "vision", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 32, Temperature: 0,
					Messages: []Message{{Role: "user", Text: text, ImageB64: img, ImageTyp: "image/png"}},
				}}
			},
		},
		{
			ID: "long_context", Kind: KindChat, Desc: "长上下文针检索",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				code := fmt.Sprintf("ZQ-%04d", rng.Intn(10000))
				filler := strings.Repeat("This is filler text that provides surrounding context for the document. ", 40)
				doc := fmt.Sprintf("%s\nThe secret access code is %s.\n%s", filler, code, filler)
				user := "Read the document above and reply with ONLY the secret access code."
				exp := Expectation{NonEmptyText: true, MustContain: []string{code}}
				if custom != "" {
					doc, user, exp = "", custom, Expectation{NonEmptyText: true}
				}
				return Case{Title: "long_context", Kind: KindChat, Exp: exp, Chat: &ChatRequest{
					Model: m.Chat, MaxTokens: 32, Temperature: 0,
					Messages: []Message{{Role: "user", Text: strings.TrimSpace(doc + "\n\n" + user)}},
				}}
			},
		},
		{
			ID: "completions", Kind: KindCompletion, Desc: "传统文本补全（/v1/completions）",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				prompt := "The capital city of France is"
				exp := Expectation{NonEmptyText: true, MustContain: []string{"paris"}}
				if custom != "" {
					prompt, exp = custom, Expectation{NonEmptyText: true}
				}
				return Case{Title: "completions", Kind: KindCompletion, Exp: exp, Chat: &ChatRequest{
					Model: m.Completion, MaxTokens: 16, Temperature: 0, Prompt: prompt,
				}}
			},
		},
		{
			ID: "embeddings", Kind: KindEmbedding, Desc: "向量嵌入（维度一致性 + 语义相似度）",
			Build: func(rng *rand.Rand, custom string, m ModelSet) Case {
				inputs := []string{"A cat sits on the mat.", "A kitten rests on the rug.", "The stock market crashed today."}
				exp := Expectation{EmbeddingCount: 3, EmbeddingMin: 1, SimilarityCheck: true}
				if custom != "" {
					inputs = []string{custom}
					exp = Expectation{EmbeddingCount: 1, EmbeddingMin: 1}
				}
				return Case{Title: "embeddings", Kind: KindEmbedding, Exp: exp, Emb: &EmbeddingRequest{Model: m.Embedding, Input: inputs}}
			},
		},
	}
}
