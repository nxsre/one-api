package core

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

// HasFailures 是否存在 FAIL 或 ERROR（用于决定进程退出码）。
func HasFailures(results []Result) bool {
	for _, r := range results {
		if r.Status == StatusFail || r.Status == StatusError {
			return true
		}
	}
	return false
}

// WriteReport 把结果渲染成对齐表格 + 汇总，写入 w。verbose 时附带失败详情与原始响应。
func WriteReport(w io.Writer, results []Result, verbose bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TARGET\tSCENARIO\tKIND\tSTATUS\tLATENCY\tCHUNKS\tTOKENS\tNOTE")
	for _, r := range results {
		note := ""
		switch r.Status {
		case StatusSkip:
			note = r.Reason
		case StatusError:
			note = truncate(r.Err, 80)
		case StatusFail:
			if len(r.Failures) > 0 {
				note = truncate(r.Failures[0], 80)
			}
		}
		lat := "-"
		if r.Latency > 0 {
			lat = fmt.Sprintf("%dms", r.Latency.Milliseconds())
		}
		tokens := "-"
		if r.Usage.TotalTokens > 0 || r.Usage.PromptTokens > 0 || r.Usage.CompletionTokens > 0 {
			tokens = fmt.Sprintf("%d/%d", r.Usage.PromptTokens, r.Usage.CompletionTokens)
		}
		chunks := "-"
		if r.Chunks > 0 {
			chunks = fmt.Sprintf("%d", r.Chunks)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Target, r.Scenario, r.Kind, statusIcon(r.Status), lat, chunks, tokens, note)
	}
	tw.Flush()

	// 失败详情
	if verbose {
		for _, r := range results {
			if r.Status == StatusFail && len(r.Failures) > 1 {
				fmt.Fprintf(w, "\n[%s/%s] 失败明细:\n", r.Target, r.Scenario)
				for _, f := range r.Failures {
					fmt.Fprintf(w, "  - %s\n", f)
				}
			}
		}
	}

	// 汇总
	type agg struct{ pass, fail, err, skip int }
	byTarget := map[string]*agg{}
	var order []string
	total := agg{}
	for _, r := range results {
		a := byTarget[r.Target]
		if a == nil {
			a = &agg{}
			byTarget[r.Target] = a
			order = append(order, r.Target)
		}
		switch r.Status {
		case StatusPass:
			a.pass++
			total.pass++
		case StatusFail:
			a.fail++
			total.fail++
		case StatusError:
			a.err++
			total.err++
		case StatusSkip:
			a.skip++
			total.skip++
		}
	}
	fmt.Fprintln(w, "\n—— 汇总 ——")
	sumTw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(sumTw, "TARGET\tPASS\tFAIL\tERROR\tSKIP")
	for _, name := range order {
		a := byTarget[name]
		fmt.Fprintf(sumTw, "%s\t%d\t%d\t%d\t%d\n", name, a.pass, a.fail, a.err, a.skip)
	}
	fmt.Fprintf(sumTw, "总计\t%d\t%d\t%d\t%d\n", total.pass, total.fail, total.err, total.skip)
	sumTw.Flush()
	fmt.Fprintf(w, "\n生成于 %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func statusIcon(s string) string {
	switch s {
	case StatusPass:
		return "✓ PASS"
	case StatusFail:
		return "✗ FAIL"
	case StatusError:
		return "! ERROR"
	case StatusSkip:
		return "- SKIP"
	default:
		return s
	}
}

// ResultsToJSON 序列化为带缩进的 JSON。
func ResultsToJSON(results []Result) []byte {
	b, _ := json.MarshalIndent(results, "", "  ")
	return b
}
