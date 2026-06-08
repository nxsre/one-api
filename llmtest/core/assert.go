package core

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Expectation 描述一个场景的断言期望。运行器据此对归一化响应做严格校验与语义检查。
type Expectation struct {
	NonEmptyText bool     // 文本须非空
	MustContain  []string // 文本须包含这些子串（不区分大小写）
	JSONObject   bool     // 文本须为合法 JSON 对象
	JSONKeys     []string // 且须包含这些顶层键
	ToolName     string   // 须产生对该工具的调用
	ToolArgKeys  []string // 工具调用参数 JSON 须含这些键
	MinChunks    int      // 流式最少分块数
	WantUsage    bool     // 须返回非零 usage（prompt 或 completion token）

	// 向量相关
	EmbeddingCount int // 期望返回的向量条数
	EmbeddingMin   int // 每条向量最小维度（>0 时校验）
	// SimilarityCheck：要求 sim(idx[0],idx[1]) > sim(idx[0],idx[2])，用于验证嵌入语义有效。
	SimilarityCheck bool
}

// Evaluate 对归一化响应执行断言，返回失败信息列表（空切片表示通过）。
func Evaluate(exp Expectation, chat *ChatResponse, emb *EmbeddingResponse) []string {
	var fails []string

	if chat != nil {
		text := strings.TrimSpace(chat.Text)
		lower := normalizeForMatch(text)

		if exp.NonEmptyText && text == "" && len(chat.ToolCalls) == 0 {
			fails = append(fails, "响应文本为空")
		}
		for _, sub := range exp.MustContain {
			if !strings.Contains(lower, normalizeForMatch(sub)) {
				fails = append(fails, fmt.Sprintf("文本未包含期望子串 %q（实际: %q）", sub, truncate(text, 120)))
			}
		}
		if exp.JSONObject {
			obj, ok := extractJSONObject(text)
			if !ok {
				fails = append(fails, fmt.Sprintf("响应不是合法 JSON 对象（实际: %q）", truncate(text, 120)))
			} else {
				for _, k := range exp.JSONKeys {
					if _, exists := obj[k]; !exists {
						fails = append(fails, fmt.Sprintf("JSON 缺少键 %q", k))
					}
				}
			}
		}
		if exp.ToolName != "" {
			tc := findToolCall(chat.ToolCalls, exp.ToolName)
			if tc == nil {
				fails = append(fails, fmt.Sprintf("未调用期望工具 %q（实际工具: %s）", exp.ToolName, toolNames(chat.ToolCalls)))
			} else if len(exp.ToolArgKeys) > 0 {
				var args map[string]any
				if err := json.Unmarshal(tc.Args, &args); err != nil {
					fails = append(fails, fmt.Sprintf("工具参数不是合法 JSON: %v（原始: %s）", err, truncate(string(tc.Args), 120)))
				} else {
					for _, k := range exp.ToolArgKeys {
						if _, ok := args[k]; !ok {
							fails = append(fails, fmt.Sprintf("工具参数缺少键 %q", k))
						}
					}
				}
			}
		}
		if exp.MinChunks > 0 && chat.Chunks < exp.MinChunks {
			fails = append(fails, fmt.Sprintf("流式分块数 %d 少于期望 %d", chat.Chunks, exp.MinChunks))
		}
		if exp.WantUsage && chat.Usage.PromptTokens == 0 && chat.Usage.CompletionTokens == 0 && chat.Usage.TotalTokens == 0 {
			fails = append(fails, "未返回 usage token 统计")
		}
	}

	if emb != nil {
		if exp.EmbeddingCount > 0 && len(emb.Vectors) != exp.EmbeddingCount {
			fails = append(fails, fmt.Sprintf("向量条数 %d 不等于期望 %d", len(emb.Vectors), exp.EmbeddingCount))
		}
		if exp.EmbeddingMin > 0 {
			for i, v := range emb.Vectors {
				if len(v) < exp.EmbeddingMin {
					fails = append(fails, fmt.Sprintf("第 %d 条向量维度 %d 小于期望 %d", i, len(v), exp.EmbeddingMin))
					break
				}
			}
		}
		if exp.SimilarityCheck {
			if len(emb.Vectors) < 3 {
				fails = append(fails, "相似度检查需要至少 3 条向量")
			} else {
				simClose := cosine(emb.Vectors[0], emb.Vectors[1])
				simFar := cosine(emb.Vectors[0], emb.Vectors[2])
				if !(simClose > simFar) {
					fails = append(fails, fmt.Sprintf("语义相似度异常：相近文本相似度 %.4f 未大于无关文本 %.4f", simClose, simFar))
				}
			}
		}
	}

	return fails
}

// subscriptMap 把常见 Unicode 下标/上标数字映射回 ASCII，避免 H₂O 与 h2o 误判。
var subscriptMap = map[rune]rune{
	'₀': '0', '₁': '1', '₂': '2', '₃': '3', '₄': '4', '₅': '5', '₆': '6', '₇': '7', '₈': '8', '₉': '9',
	'⁰': '0', '¹': '1', '²': '2', '³': '3', '⁴': '4', '⁵': '5', '⁶': '6', '⁷': '7', '⁸': '8', '⁹': '9',
}

// normalizeForMatch 统一大小写并归一化上下标数字，用于子串匹配。
func normalizeForMatch(s string) string {
	var b strings.Builder
	for _, r := range s {
		if mapped, ok := subscriptMap[r]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func toolNames(tcs []ToolCall) string {
	if len(tcs) == 0 {
		return "无"
	}
	names := make([]string, 0, len(tcs))
	for _, t := range tcs {
		names = append(names, t.Name)
	}
	return strings.Join(names, ",")
}

func findToolCall(tcs []ToolCall, name string) *ToolCall {
	for i := range tcs {
		if tcs[i].Name == name {
			return &tcs[i]
		}
	}
	return nil
}

// extractJSONObject 尽力从文本中提取首个 JSON 对象（容忍模型在 JSON 前后附带说明或代码块围栏）。
func extractJSONObject(text string) (map[string]any, bool) {
	text = strings.TrimSpace(text)
	// 去掉 ```json ... ``` 围栏
	if strings.HasPrefix(text, "```") {
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = text[idx+1:]
		}
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
		text = strings.TrimSpace(text)
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		return nil, false
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(text[start:end+1]), &obj); err != nil {
		return nil, false
	}
	return obj, true
}

// cosine 计算余弦相似度。
func cosine(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
