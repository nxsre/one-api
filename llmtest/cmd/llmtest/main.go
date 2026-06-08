// Command llmtest 是一套大模型接口测试 CLI：
// 支持 OpenAI（chat 全形式 / completions / embeddings）、Anthropic Messages、Gemini generateContent，
// 既可打 one-api 网关也可打各家原生端点；用例输入可随机生成或自定义，断言含结构校验与语义检查。
//
// 用法：
//
//	llmtest -init > config.json      # 生成示例配置
//	llmtest -list                    # 列出全部内置场景
//	llmtest -config config.json      # 跑全部场景
//	llmtest -config config.json -scenarios math,tool_call,embeddings -v
//	llmtest -config config.json -targets one-api-gateway -json report.json
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"llmtest/core"
)

func main() {
	var (
		configPath  = flag.String("config", "config.json", "配置文件路径 (JSON)")
		scenarios   = flag.String("scenarios", "", "仅运行这些场景（逗号分隔），默认全部")
		targets     = flag.String("targets", "", "仅运行这些 target（按 name 逗号分隔），默认全部")
		jsonOut     = flag.String("json", "", "把结果同时写入该 JSON 文件")
		verbose     = flag.Bool("v", false, "输出失败明细与原始响应")
		timeout     = flag.Int("timeout", 0, "单次请求超时秒数（覆盖配置）")
		concurrency = flag.Int("concurrency", 0, "并发数（覆盖配置）")
		seed        = flag.Int64("seed", -1, "随机种子（覆盖配置；同种子输入可复现）")
		list        = flag.Bool("list", false, "列出全部内置场景后退出")
		initCfg     = flag.Bool("init", false, "打印示例配置到标准输出后退出")
	)
	flag.Parse()

	if *initCfg {
		fmt.Println(string(core.ExampleConfig()))
		return
	}
	if *list {
		fmt.Println("内置场景：")
		for _, s := range core.AllScenarios() {
			flags := []string{}
			if s.Stream {
				flags = append(flags, "stream")
			}
			if s.NeedsTools {
				flags = append(flags, "tools")
			}
			if s.NeedsVision {
				flags = append(flags, "vision")
			}
			if s.NeedsJSON {
				flags = append(flags, "json")
			}
			extra := ""
			if len(flags) > 0 {
				extra = " [" + strings.Join(flags, ",") + "]"
			}
			fmt.Printf("  %-14s (%s) %s%s\n", s.ID, s.Kind, s.Desc, extra)
		}
		return
	}

	cfg, err := core.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(2)
	}
	if *timeout > 0 {
		cfg.TimeoutSeconds = *timeout
	}
	if *concurrency > 0 {
		cfg.Concurrency = *concurrency
	}
	if *seed >= 0 {
		cfg.Seed = *seed
	}
	if *targets != "" {
		cfg.Targets = filterTargets(cfg.Targets, splitCSV(*targets))
		if len(cfg.Targets) == 0 {
			fmt.Fprintln(os.Stderr, "错误: -targets 未匹配到任何 target")
			os.Exit(2)
		}
	}

	selected := splitCSV(*scenarios)
	results := core.Run(cfg, selected, *verbose)
	core.WriteReport(os.Stdout, results, *verbose)

	if *jsonOut != "" {
		if err := os.WriteFile(*jsonOut, core.ResultsToJSON(results), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "写入 JSON 报告失败:", err)
		} else {
			fmt.Fprintln(os.Stderr, "已写入 JSON 报告:", *jsonOut)
		}
	}

	if core.HasFailures(results) {
		os.Exit(1)
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func filterTargets(all []core.Target, names []string) []core.Target {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var out []core.Target
	for _, t := range all {
		if want[t.Name] {
			out = append(out, t)
		}
	}
	return out
}
