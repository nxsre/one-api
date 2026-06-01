package main

import (
	"fmt"
	"slices"
	"sort"
	"time"
)

// result 是单次请求的观测结果。
type result struct {
	latency time.Duration // 完整请求耗时（含读完响应体）
	ttfb    time.Duration // time-to-first-byte（流式场景下约等于首 token 延迟）
	status  int           // HTTP 状态码；0 表示传输层失败
	bytes   int64         // 读取的响应字节数
	err     error
}

// stats 聚合所有 result。
type stats struct {
	latencies []time.Duration
	ttfbs     []time.Duration
	statusCnt map[int]int64
	errCnt    map[string]int64
	totalReq  int64
	totalErr  int64
	totalByte int64
	saturated int64 // 开环模式下，负载工具因 in-flight 上限无法按目标速率发出的次数
	wall      time.Duration
}

func newStats() *stats {
	return &stats{
		statusCnt: map[int]int64{},
		errCnt:    map[string]int64{},
	}
}

func (s *stats) add(r result) {
	s.totalReq++
	s.totalByte += r.bytes
	if r.err != nil {
		s.totalErr++
		s.errCnt[classifyErr(r.err)]++
		return
	}
	s.statusCnt[r.status]++
	if r.status < 200 || r.status >= 300 {
		s.totalErr++
	}
	s.latencies = append(s.latencies, r.latency)
	if r.ttfb > 0 {
		s.ttfbs = append(s.ttfbs, r.ttfb)
	}
}

func classifyErr(err error) string {
	msg := err.Error()
	switch {
	case contains(msg, "context deadline exceeded"), contains(msg, "Timeout"), contains(msg, "timeout"):
		return "timeout"
	case contains(msg, "connection refused"):
		return "connection refused"
	case contains(msg, "connection reset"):
		return "connection reset"
	case contains(msg, "EOF"):
		return "EOF"
	case contains(msg, "no such host"):
		return "dns"
	default:
		return "other"
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[n-1]
	}
	idx := int(p/100*float64(n-1) + 0.5)
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

func mean(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return sum / time.Duration(len(ds))
}

func (s *stats) report() {
	slices.Sort(s.latencies)
	slices.Sort(s.ttfbs)

	ok := s.totalReq - s.totalErr
	secs := s.wall.Seconds()
	rps := 0.0
	if secs > 0 {
		rps = float64(s.totalReq) / secs
	}

	fmt.Println()
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println(" Load test results")
	fmt.Println("──────────────────────────────────────────────")
	fmt.Printf("  duration        %s\n", s.wall.Round(time.Millisecond))
	fmt.Printf("  requests        %d  (ok %d, failed %d)\n", s.totalReq, ok, s.totalErr)
	fmt.Printf("  throughput      %.1f req/s\n", rps)
	fmt.Printf("  data received   %s\n", humanBytes(s.totalByte))
	if s.saturated > 0 {
		fmt.Printf("  saturated       %d  (load tool hit in-flight cap; server slower than target rate)\n", s.saturated)
	}

	fmt.Println("\n  latency (full request)")
	printDurStats(s.latencies)

	if len(s.ttfbs) > 0 {
		fmt.Println("\n  TTFB / first token (streaming)")
		printDurStats(s.ttfbs)
	}

	if len(s.statusCnt) > 0 {
		fmt.Println("\n  status codes")
		codes := make([]int, 0, len(s.statusCnt))
		for c := range s.statusCnt {
			codes = append(codes, c)
		}
		sort.Ints(codes)
		for _, c := range codes {
			fmt.Printf("    %d  %d\n", c, s.statusCnt[c])
		}
	}
	if len(s.errCnt) > 0 {
		fmt.Println("\n  transport errors")
		for k, v := range s.errCnt {
			fmt.Printf("    %-20s %d\n", k, v)
		}
	}
	fmt.Println("──────────────────────────────────────────────")
}

func printDurStats(sorted []time.Duration) {
	if len(sorted) == 0 {
		fmt.Println("    (no successful samples)")
		return
	}
	fmt.Printf("    min %-10s mean %-10s max %s\n",
		round(sorted[0]), round(mean(sorted)), round(sorted[len(sorted)-1]))
	fmt.Printf("    p50 %-10s p90 %-10s p95 %-10s p99 %s\n",
		round(percentile(sorted, 50)), round(percentile(sorted, 90)),
		round(percentile(sorted, 95)), round(percentile(sorted, 99)))
}

func round(d time.Duration) time.Duration {
	if d >= time.Second {
		return d.Round(time.Millisecond)
	}
	return d.Round(time.Microsecond)
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGT"[exp])
}
