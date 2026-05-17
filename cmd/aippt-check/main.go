// aippt-check：通过 one-api POST /v1/chat/completions 调用 AiPPT（模型 aippt-ppt，stream=false）。
// 需在 one-api 配置 AiPPT 渠道且令牌有权使用该模型；会真实走 AiPPT 上游，耗时可能较长。令牌须通过 -token 传入。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/cmd/internal/apitest"
	"github.com/songquanpeng/one-api/relay/adaptor/aippt"
)

func main() {
	base := flag.String("base", getenvDefault("ONE_API_BASE", "http://127.0.0.1:3000"), "one-api 根地址")
	token := flag.String("token", "", "用户 API 令牌 sk-…，必填")
	insecure := flag.Bool("insecure", false, "跳过 HTTPS 证书校验（调试/自签证书）")
	model := flag.String("model", aippt.ModelAipptPPT, "模型名，默认 aippt-ppt")
	title := flag.String("title", "one-api AiPPT 连通性测试", "PPT 标题/主题（最后一条 user 文本）")
	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		fmt.Fprintln(os.Stderr, "aippt-check: 必须提供 -token")
		os.Exit(2)
	}

	cli := apitest.New(*base, *token, *insecure)
	body := map[string]any{
		"model": strings.TrimSpace(*model),
		"messages": []map[string]string{
			{"role": "user", "content": strings.TrimSpace(*title)},
		},
		"stream": false,
	}

	status, resp, err := cli.PostJSON("/v1/chat/completions", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aippt-check: request: %v\n", err)
		os.Exit(1)
	}
	if status != 200 {
		fmt.Fprintf(os.Stderr, "aippt-check: HTTP %d\n%s\n", status, string(resp))
		os.Exit(1)
	}

	snippet, err := apitest.ChatCompletionSnippet(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aippt-check: response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("aippt-check: OK")
	fmt.Println(snippet)
}

func getenvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
