package core

import (
	"fmt"
	"sort"
	"time"
)

// Thresholds 异常判定阈值。
type Thresholds struct {
	UpstreamConnErrors int     // 单上游连接类错误数达到该值即告警
	ErrorRatePct       float64 // 整体错误率(%)阈值
	LatencyP95Ms       float64 // p95 延迟(ms)阈值
	Any5xx             bool    // 出现任意 5xx 即告警
}

// DefaultThresholds 合理默认值。
func DefaultThresholds() Thresholds {
	return Thresholds{UpstreamConnErrors: 3, ErrorRatePct: 5, LatencyP95Ms: 15000, Any5xx: true}
}

// UpstreamStat 单个上游 host 的统计。
type UpstreamStat struct {
	Host      string         `json:"host"`
	Errors    int            `json:"errors"`
	Classes   map[string]int `json:"classes"`
	Channels  map[int]int    `json:"channels"`
	LastError time.Time      `json:"last_error"`
}

// ChannelStat 单个渠道的统计。
type ChannelStat struct {
	ChannelID int            `json:"channel_id"`
	Errors    int            `json:"errors"`
	Classes   map[string]int `json:"classes"`
	Upstreams map[string]int `json:"upstreams"`
	LastError time.Time      `json:"last_error"`
}

// Report 一次分析的汇总结果。
type Report struct {
	From, To    time.Time
	TotalLines  int
	AccessLines int
	StatusHist  map[int]int
	LatP50      float64
	LatP95      float64
	LatP99      float64
	LatMax      float64
	Errors      int
	ClassHist   map[string]int
	Channels    []ChannelStat
	Upstreams   []UpstreamStat
	Anomalies   []string
}

// Analyze 聚合事件并按阈值判定异常。
func Analyze(events []Event, th Thresholds) Report {
	r := Report{StatusHist: map[int]int{}, ClassHist: map[string]int{}}
	chMap := map[int]*ChannelStat{}
	upMap := map[string]*UpstreamStat{}
	var lats []float64

	for _, e := range events {
		if e.Raw != "" {
			r.TotalLines++
		}
		if !e.Time.IsZero() {
			if r.From.IsZero() || e.Time.Before(r.From) {
				r.From = e.Time
			}
			if e.Time.After(r.To) {
				r.To = e.Time
			}
		}
		if e.IsAccess {
			r.AccessLines++
			r.StatusHist[e.Status]++
			if e.LatencyMs > 0 {
				lats = append(lats, e.LatencyMs)
			}
		}
		if e.Canonical && e.ErrClass != "" {
			// 仅统计规范行：错误类访问(4xx/5xx)与 processChannelRelayError 汇总行，避免重复计数。
			r.Errors++
			r.ClassHist[e.ErrClass]++

			if e.ChannelID > 0 {
				cs := chMap[e.ChannelID]
				if cs == nil {
					cs = &ChannelStat{ChannelID: e.ChannelID, Classes: map[string]int{}, Upstreams: map[string]int{}}
					chMap[e.ChannelID] = cs
				}
				cs.Errors++
				cs.Classes[e.ErrClass]++
				if e.Upstream != "" {
					cs.Upstreams[e.Upstream]++
				}
				if e.Time.After(cs.LastError) {
					cs.LastError = e.Time
				}
			}
			if e.Upstream != "" {
				us := upMap[e.Upstream]
				if us == nil {
					us = &UpstreamStat{Host: e.Upstream, Classes: map[string]int{}, Channels: map[int]int{}}
					upMap[e.Upstream] = us
				}
				us.Errors++
				us.Classes[e.ErrClass]++
				if e.ChannelID > 0 {
					us.Channels[e.ChannelID]++
				}
				if e.Time.After(us.LastError) {
					us.LastError = e.Time
				}
			}
		}
	}

	r.LatP50, r.LatP95, r.LatP99, r.LatMax = percentiles(lats)
	for _, cs := range chMap {
		r.Channels = append(r.Channels, *cs)
	}
	sort.Slice(r.Channels, func(i, j int) bool { return r.Channels[i].Errors > r.Channels[j].Errors })
	for _, us := range upMap {
		r.Upstreams = append(r.Upstreams, *us)
	}
	sort.Slice(r.Upstreams, func(i, j int) bool { return r.Upstreams[i].Errors > r.Upstreams[j].Errors })

	r.Anomalies = detectAnomalies(r, th)
	return r
}

func detectAnomalies(r Report, th Thresholds) []string {
	var out []string
	// 1) 上游连接类错误
	for _, us := range r.Upstreams {
		conn := 0
		for cls, n := range us.Classes {
			if connClasses[cls] {
				conn += n
			}
		}
		if conn >= th.UpstreamConnErrors {
			out = append(out, fmt.Sprintf("上游 %s 连接类异常 %d 次（%s）", us.Host, conn, fmtClasses(us.Classes)))
		}
	}
	// 2) 5xx
	if th.Any5xx {
		n5 := 0
		for code, c := range r.StatusHist {
			if code >= 500 {
				n5 += c
			}
		}
		if n5 > 0 {
			out = append(out, fmt.Sprintf("出现 %d 个 5xx 响应", n5))
		}
	}
	// 3) 错误率
	if r.AccessLines > 0 {
		rate := float64(r.Errors) / float64(r.AccessLines) * 100
		if rate > th.ErrorRatePct {
			out = append(out, fmt.Sprintf("整体错误率 %.1f%%（%d/%d）超过阈值 %.1f%%", rate, r.Errors, r.AccessLines, th.ErrorRatePct))
		}
	}
	// 4) p95 延迟
	if th.LatencyP95Ms > 0 && r.LatP95 > th.LatencyP95Ms {
		out = append(out, fmt.Sprintf("p95 延迟 %.0fms 超过阈值 %.0fms", r.LatP95, th.LatencyP95Ms))
	}
	return out
}

func percentiles(v []float64) (p50, p95, p99, max float64) {
	if len(v) == 0 {
		return
	}
	sort.Float64s(v)
	pick := func(p float64) float64 {
		idx := int(p * float64(len(v)-1))
		if idx < 0 {
			idx = 0
		}
		return v[idx]
	}
	return pick(0.50), pick(0.95), pick(0.99), v[len(v)-1]
}

func fmtClasses(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := ""
	for i, k := range keys {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%s=%d", k, m[k])
	}
	return s
}
