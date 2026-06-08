package core

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"
)

// HasAnomalies 是否检出异常（用于决定进程退出码）。
func (r Report) HasAnomalies() bool { return len(r.Anomalies) > 0 }

// WriteReport 渲染人类可读报告。
func WriteReport(w io.Writer, r Report) {
	span := "-"
	if !r.From.IsZero() {
		span = fmt.Sprintf("%s ~ %s", r.From.Format("15:04:05"), r.To.Format("15:04:05"))
	}
	fmt.Fprintf(w, "时间窗口: %s | 日志行: %d | 访问: %d | 错误: %d\n", span, r.TotalLines, r.AccessLines, r.Errors)
	fmt.Fprintf(w, "延迟(ms): p50=%.0f p95=%.0f p99=%.0f max=%.0f\n", r.LatP50, r.LatP95, r.LatP99, r.LatMax)

	// 状态码分布
	if len(r.StatusHist) > 0 {
		codes := make([]int, 0, len(r.StatusHist))
		for c := range r.StatusHist {
			codes = append(codes, c)
		}
		sort.Ints(codes)
		fmt.Fprint(w, "状态码: ")
		for i, c := range codes {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			fmt.Fprintf(w, "%d=%d", c, r.StatusHist[c])
		}
		fmt.Fprintln(w)
	}
	if len(r.ClassHist) > 0 {
		fmt.Fprintf(w, "错误分类: %s\n", fmtClasses(r.ClassHist))
	}

	if len(r.Upstreams) > 0 {
		fmt.Fprintln(w, "\n—— 上游 (按错误数) ——")
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "UPSTREAM\tERRORS\tCLASSES\tCHANNELS\tLAST")
		for _, us := range r.Upstreams {
			fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", us.Host, us.Errors, fmtClasses(us.Classes), fmtIntSet(us.Channels), tsOrDash(us.LastError))
		}
		tw.Flush()
	}
	if len(r.Channels) > 0 {
		fmt.Fprintln(w, "\n—— 渠道 (按错误数) ——")
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "CHANNEL\tERRORS\tCLASSES\tUPSTREAMS\tLAST")
		for _, cs := range r.Channels {
			fmt.Fprintf(tw, "%d\t%d\t%s\t%s\t%s\n", cs.ChannelID, cs.Errors, fmtClasses(cs.Classes), fmtStrSet(cs.Upstreams), tsOrDash(cs.LastError))
		}
		tw.Flush()
	}

	fmt.Fprintln(w, "\n—— 诊断 ——")
	if len(r.Anomalies) == 0 {
		fmt.Fprintln(w, "✓ 未发现上游异常")
	} else {
		for _, a := range r.Anomalies {
			fmt.Fprintf(w, "⚠ %s\n", a)
		}
	}
}

// ReportToJSON 序列化报告为 JSON。
func ReportToJSON(r Report) []byte {
	type out struct {
		From, To   string
		TotalLines int
		Access     int
		Errors     int
		Latency    map[string]float64
		StatusHist map[int]int
		ClassHist  map[string]int
		Upstreams  []UpstreamStat
		Channels   []ChannelStat
		Anomalies  []string
	}
	o := out{
		TotalLines: r.TotalLines, Access: r.AccessLines, Errors: r.Errors,
		Latency:    map[string]float64{"p50": r.LatP50, "p95": r.LatP95, "p99": r.LatP99, "max": r.LatMax},
		StatusHist: r.StatusHist, ClassHist: r.ClassHist, Upstreams: r.Upstreams, Channels: r.Channels,
		Anomalies: r.Anomalies,
	}
	if !r.From.IsZero() {
		o.From = r.From.Format(time.RFC3339)
		o.To = r.To.Format(time.RFC3339)
	}
	b, _ := json.MarshalIndent(o, "", "  ")
	return b
}

func tsOrDash(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05")
}

func fmtIntSet(m map[int]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := ""
	for i, k := range keys {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("ch%d", k)
	}
	if s == "" {
		return "-"
	}
	return s
}

func fmtStrSet(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := ""
	for i, k := range keys {
		if i > 0 {
			s += ","
		}
		s += k
	}
	if s == "" {
		return "-"
	}
	return s
}
