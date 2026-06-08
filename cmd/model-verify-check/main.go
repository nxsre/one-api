// model-verify-check：复现 modelknow.com 模型验真探测请求（POST /v1/messages，A–K 题组）。
// 全部使用 stream=true；每题完成后立即打印 request/response 的 header 与 body。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/cmd/internal/apitest"
)

const prog = "model-verify-check"

type probeOutcome struct {
	ProbeID         string              `json:"probe_id"`
	ProbeName       string              `json:"probe_name"`
	Model           string              `json:"model"`
	Prompt          string              `json:"prompt"`
	Expected        string              `json:"expected"`
	MaxTokens       int                 `json:"max_tokens"`
	HasTemperature  bool                `json:"has_temperature,omitempty"`
	RequestHeaders  map[string][]string `json:"request_headers,omitempty"`
	RequestBody     string              `json:"request_body"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	ResponseBody    string              `json:"response_body"`
	HTTPStatus      int                 `json:"http_status"`
	DurationMs      int64               `json:"duration_ms"`
	Success         bool                `json:"success"`
	Pass            bool                `json:"pass,omitempty"`
	Reason          string              `json:"reason,omitempty"`
	Snippet         string              `json:"snippet,omitempty"`
	Error           string              `json:"error,omitempty"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: %s -token sk-... -model claude-opus-4-20250514 [options]

复现 modelknow.com 模型验真探测：对指定 Claude 型号依次发送 A–K 探测题（Anthropic Messages API，流式；K 题为工具调用往返，非流式）。

`, prog)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), `
探测题（-probe，默认 -all 跑全套）：
  A  family_direct       直接问模型家族
  B  family_correction   纠偏/反套话
  C  claude_yesno        Claude 家族确认
  D  structured_json     JSON + 1+1
  E  three_line          三行输出 + 1+1
  F  refuse_forgery      拒绝伪造官方证明
  G  model_name_confirm  模型名一致性
  H  model_name_en       仅输出模型名（英文，max_tokens=64）
  I  model_name_zh       仅输出模型名（中文，max_tokens=64）
  J  agent_proxy_detect  Agent/CLI 代理检测（JSON，max_tokens=128）
  K  tool_call_weather   工具调用往返（get_weather，stream=false）

-profile：
  strict       默认；任一 WARN 或 HTTP 失败即 exit 1
  oauth-proxy  CLIProxyAPI OAuth 通道；H/I/J 的 WARN 不计入失败，A–G/K 未达标或 HTTP 失败才 exit 1
`)
	}

	base := flag.String("base", getenvDefault("ONE_API_BASE", "http://127.0.0.1:3000"), "one-api 根地址")
	token := flag.String("token", "", "用户 API 令牌（sk-…），必填")
	model := flag.String("model", "", "目标 Claude 模型名，必填")
	models := flag.String("models", "", "多个模型（逗号分隔），覆盖 -model")
	insecure := flag.Bool("insecure", false, "跳过 HTTPS 证书校验")
	all := flag.Bool("all", true, "运行全套 A–K 探测（默认 true）")
	probe := flag.String("probe", "", "仅运行单个探测题 A–K（与 -all 互斥时优先 -probe）")
	profileFlag := flag.String("profile", profileStrict, "判定模式：strict（默认，含 WARN 即 exit 1）或 oauth-proxy（H/I/J 的 WARN 不计入失败，仅 HTTP 或 A–G/K 未达标才 exit 1）")
	timeout := flag.Duration("timeout", 20*time.Second, "单次 HTTP 超时（modelknow 约 20s）")
	jsonOut := flag.String("json", "", "将全部结果写入 JSON 文件（跑完后写入）")
	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		fmt.Fprintf(os.Stderr, "%s: 必须提供 -token\n", prog)
		os.Exit(2)
	}

	targetModels := parseModels(*model, *models)
	if len(targetModels) == 0 {
		fmt.Fprintf(os.Stderr, "%s: 必须提供 -model 或 -models\n", prog)
		os.Exit(2)
	}

	probeIDs, err := resolveProbeIDs(*all, *probe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(2)
	}

	profile, err := normalizeProfile(*profileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(2)
	}

	cli := apitest.NewWithTimeout(*base, *token, *insecure, *timeout)
	headers := anthropicInboundHeaders(*token)

	var outcomes []probeOutcome
	for _, m := range targetModels {
		printRunHeader(os.Stdout, *base, m, probeIDs, profile)
		for _, id := range probeIDs {
			def, ok := probeByID(id)
			if !ok {
				continue
			}
			o := runOneProbe(cli, headers, m, def)
			printProbeReport(os.Stdout, o)
			_ = os.Stdout.Sync()
			outcomes = append(outcomes, o)
		}
	}

	printSummaryBlock(os.Stdout, outcomes, profile)

	if *jsonOut != "" {
		b, _ := json.MarshalIndent(outcomes, "", "  ")
		if err := os.WriteFile(*jsonOut, b, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: 写入 JSON 失败: %v\n", prog, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s: 已写入 %s\n", prog, *jsonOut)
	}

	if shouldExitFailure(outcomes, profile) {
		os.Exit(1)
	}
}

func runOneProbe(cli *apitest.Client, headers map[string]string, model string, def probeDef) probeOutcome {
	if def.ID == probeToolCall {
		return runToolCallProbe(cli, headers, model)
	}

	prompt := formatProbePrompt(def, model)
	body := buildRequestBody(model, prompt, def)

	start := time.Now()
	ex, err := postAnthropicStream(cli, "/v1/messages", body, headers)
	duration := time.Since(start).Milliseconds()

	result := probeOutcome{
		ProbeID:        def.ID,
		ProbeName:      def.Name,
		Model:          model,
		Prompt:         prompt,
		Expected:       probeExpectation(def.ID),
		MaxTokens:      probeMaxTokens(def),
		HasTemperature: !def.OmitTemperature,
		DurationMs:     duration,
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.HTTPStatus = ex.StatusCode
	result.RequestHeaders = ex.RequestHeaders
	result.RequestBody = ex.RequestBody
	result.ResponseHeaders = ex.ResponseHeaders
	result.ResponseBody = ex.ResponseBody

	if ex.StatusCode != 200 {
		result.Error = truncate(ex.ResponseBody, 500)
		return result
	}

	text := strings.TrimSpace(ex.AssembledText)
	if text == "" {
		result.Error = "empty assembled text from stream"
		return result
	}

	eval := evaluateProbe(def.ID, model, text)
	result.Success = true
	result.Snippet = text
	result.Pass = eval.Pass
	result.Reason = eval.Reason
	return result
}

func resolveProbeIDs(all bool, probe string) ([]string, error) {
	probe = strings.ToUpper(strings.TrimSpace(probe))
	if probe != "" {
		if _, ok := probeByID(probe); !ok {
			return nil, fmt.Errorf("未知 -probe %q，可选 A–K", probe)
		}
		return []string{probe}, nil
	}
	if all {
		return append([]string(nil), allProbeIDs...), nil
	}
	return nil, fmt.Errorf("请指定 -all 或 -probe")
}

func parseModels(model, models string) []string {
	raw := strings.TrimSpace(models)
	if raw == "" {
		raw = strings.TrimSpace(model)
	}
	if raw == "" {
		return nil
	}
	var out []string
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func getenvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
