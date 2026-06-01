// Command bench is a dependency-free load-testing tool for the one-api gateway.
//
// 两个子命令：
//
//	bench load   —— HTTP 负载生成器：可对任意端点（/healthz、/v1/chat/completions 等）
//	               施压，输出吞吐、状态码分布、延迟分位（p50/p90/p99）以及流式 TTFB。
//	bench mock   —— OpenAI 兼容的 mock 上流：把 one-api 的一个渠道 base_url 指向它，
//	               即可在不依赖真实厂商的情况下压测 relay 全链路（鉴权→选路→配额→计费→日志）。
//
// 典型用法见各子命令的 -h。
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "load":
		runLoadCmd(os.Args[2:])
	case "mock":
		runMockCmd(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `bench — one-api load-testing tool

Usage:
  bench load  [flags]   run an HTTP load generator against a target URL
  bench mock  [flags]   run an OpenAI-compatible mock upstream

Examples:
  # 1) baseline: hammer the health endpoint, 200 conns for 30s
  bench load -url http://127.0.0.1:3000/healthz -c 200 -d 30s

  # 2) start a mock upstream that simulates 80ms think-time + streaming
  bench mock -listen :9000 -latency 80ms -chunks 20 -chunk-delay 10ms

  # 3) point a one-api channel at the mock, then drive the relay path
  bench load -url http://127.0.0.1:3000/v1/chat/completions \
             -H "Authorization: Bearer sk-xxx" \
             -body @prompt.json -c 100 -d 60s -stream

Run 'bench load -h' or 'bench mock -h' for the full flag list.
`)
}
