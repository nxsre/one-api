// kimi-search-check：通过 one-api POST /v1/chat/completions 发起一次 Kimi（Moonshot）联网搜索请求。
// 需在 one-api 已配置可用的 Moonshot 渠道，且令牌有权使用 -model 指定的模型。令牌须通过 -token 传入。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/cmd/internal/apitest"
)

func main() {
	base := flag.String("base", getenvDefault("ONE_API_BASE", "http://127.0.0.1:3000"), "one-api 根地址，勿以 / 结尾")
	token := flag.String("token", "", "用户 API 令牌（sk-…），必填")
	insecure := flag.Bool("insecure", false, "跳过 HTTPS 证书校验（调试/自签证书）")
	model := flag.String("model", getenvDefault("KIMI_SEARCH_MODEL", "moonshot-v1-8k"), "Moonshot/Kimi 模型名（须与渠道可用模型一致）")
	prompt := flag.String("prompt", "用一句话概括北京故宫开放时间（需联网）。", "用户提示内容")
	verbose := flag.Bool("verbose", false, "打印完整 JSON 响应（可能很长）")
	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		fmt.Fprintln(os.Stderr, "kimi-search-check: 必须提供 -token")
		os.Exit(2)
	}

	cli := apitest.New(*base, *token, *insecure)
	body := map[string]any{
		"model": *model,
		"messages": []map[string]string{
			{"role": "user", "content": *prompt},
		},
		"tools": []map[string]any{
			{
				"type": "builtin_function",
				"function": map[string]string{
					"name": "$web_search",
				},
			},
		},
		"stream": false,
	}

	status, resp, err := cli.PostJSON("/v1/chat/completions", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kimi-search-check: request: %v\n", err)
		os.Exit(1)
	}
	if status != 200 {
		fmt.Fprintf(os.Stderr, "kimi-search-check: HTTP %d\n%s\n", status, string(resp))
		os.Exit(1)
	}

	snippet, err := apitest.ChatCompletionSnippet(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kimi-search-check: response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("kimi-search-check: OK")
	fmt.Println(snippet)

	if *verbose {
		var raw map[string]any
		if json.Unmarshal(resp, &raw) == nil {
			b, _ := json.MarshalIndent(raw, "", "  ")
			if len(b) > 8000 {
				b = append(b[:8000], []byte("\n…(truncated)")...)
			}
			fmt.Println(string(b))
		}
	}
}

func getenvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
