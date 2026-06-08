// Package core 解析 one-api 的运行日志（[GIN] 访问日志 + [ERROR] 中继错误日志），
// 归一化为结构化事件，用于聚合上游/渠道健康度并发现异常。
package core

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Level 日志级别。
type Level string

const (
	LevelGin   Level = "GIN"
	LevelError Level = "ERROR"
	LevelWarn  Level = "WARN"
	LevelInfo  Level = "INFO"
	LevelFatal Level = "FATAL"
	LevelOther Level = "OTHER"
)

// Event 结构化后的单条日志事件。
type Event struct {
	Time  time.Time
	Level Level
	Raw   string

	// 访问日志（[GIN]）字段
	IsAccess  bool
	Status    int
	LatencyMs float64
	Method    string
	Path      string
	ClientIP  string

	// Canonical 标记该事件是否为「应计入统计」的规范行。one-api 对一次中继错误会打印
	// [ErrorWrapper]+[relayWithRetry]+[processChannelRelayError] 多行，仅规范行计数以避免重复。
	Canonical bool

	// 中继错误（[ERROR]）字段
	IsRelayError bool
	ReqID        string
	ChannelID    int
	UserID       int
	Upstream     string // 从错误信息里的 URL 提取的 host
	ErrClass     string // eof|timeout|client_canceled|conn_refused|dns|tls|bad_request|http_4xx|http_5xx|other
	UpstreamCode int    // relayWithRetry 报出的 "status code is N"
	Message      string
}

const logTimeLayout = "2006/01/02 - 15:04:05"

var (
	reTime     = regexp.MustCompile(`(\d{4}/\d{2}/\d{2} - \d{2}:\d{2}:\d{2})`)
	reLevel    = regexp.MustCompile(`^\[(GIN|ERROR|WARN|INFO|FATAL)\]`)
	reChannel  = regexp.MustCompile(`channel id (\d+)`)
	reUser     = regexp.MustCompile(`user id:?\s*(\d+)`)
	reURLHost  = regexp.MustCompile(`https?://([^/"\s]+)`)
	reStatusCd = regexp.MustCompile(`status code is (\d+)`)
	reReqID    = regexp.MustCompile(`\b(\d{25,})\b`)
)

// ParseLine 解析一行日志为 Event；无法识别时返回 (Event{Level:OTHER}, true) 以便计数。
func ParseLine(line string) (Event, bool) {
	line = strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(line) == "" {
		return Event{}, false
	}
	ev := Event{Raw: line, Level: LevelOther}
	if m := reLevel.FindStringSubmatch(line); m != nil {
		ev.Level = Level(m[1])
	}
	if m := reTime.FindStringSubmatch(line); m != nil {
		if t, err := time.ParseInLocation(logTimeLayout, m[1], time.Local); err == nil {
			ev.Time = t
		}
	}

	switch ev.Level {
	case LevelGin:
		parseGin(line, &ev)
	case LevelError, LevelWarn, LevelFatal:
		parseError(line, &ev)
	}
	return ev, true
}

// parseGin 解析 [GIN] 访问日志：
// [GIN] 2026/06/03 - 09:11:29 | <reqid> | 400 |   31.86788ms |  192.168.156.1 |   POST /v1/messages
func parseGin(line string, ev *Event) {
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) < 6 {
		return
	}
	ev.IsAccess = true
	ev.ReqID = parts[1]
	if code, err := strconv.Atoi(parts[2]); err == nil {
		ev.Status = code
	}
	if d, err := time.ParseDuration(strings.ReplaceAll(parts[3], " ", "")); err == nil {
		ev.LatencyMs = float64(d.Nanoseconds()) / 1e6
	}
	ev.ClientIP = parts[4]
	if mp := strings.Fields(parts[5]); len(mp) >= 2 {
		ev.Method = mp[0]
		ev.Path = mp[1]
	}
	if ev.Status >= 500 {
		ev.ErrClass = "http_5xx"
		ev.Canonical = true
	} else if ev.Status >= 400 {
		ev.ErrClass = "http_4xx"
		ev.Canonical = true
	}
}

// parseError 解析 [ERROR] 中继错误日志，抽取渠道、上游 host、错误分类等。
func parseError(line string, ev *Event) {
	if m := reChannel.FindStringSubmatch(line); m != nil {
		ev.ChannelID, _ = strconv.Atoi(m[1])
		ev.IsRelayError = true
	}
	if m := reUser.FindStringSubmatch(line); m != nil {
		ev.UserID, _ = strconv.Atoi(m[1])
	}
	if m := reURLHost.FindStringSubmatch(line); m != nil {
		ev.Upstream = m[1]
	}
	if m := reStatusCd.FindStringSubmatch(line); m != nil {
		ev.UpstreamCode, _ = strconv.Atoi(m[1])
	}
	if m := reReqID.FindStringSubmatch(line); m != nil && ev.ReqID == "" {
		ev.ReqID = m[1]
	}
	// 取 [ErrorWrapper] / [processChannelRelayError] 之后的信息作为 message
	ev.Message = line
	if idx := strings.LastIndex(line, "] "); idx != -1 {
		ev.Message = strings.TrimSpace(line[idx+2:])
	}
	if cls := classifyError(line); cls != "" {
		ev.ErrClass = cls
		if ev.ErrClass != "bad_request" && ev.ErrClass != "other" {
			ev.IsRelayError = true // 连接类错误即便没有 channel id 也计入上游异常
		}
	}
	// 规范行：processChannelRelayError（带 channel/上游/分类的汇总行），或非中继的独立错误。
	// 跳过 ErrorWrapper / relayWithRetry 这两类伴随行，避免同一次错误被重复计数。
	companion := strings.Contains(line, "[ErrorWrapper]") || strings.Contains(line, "[relayWithRetry]")
	ev.Canonical = !companion && ev.ErrClass != ""
}

// classifyError 依据错误文本归类。顺序敏感：先判更具体的。
func classifyError(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "unexpected eof") || strings.Contains(l, "eof"):
		return "eof"
	case strings.Contains(l, "context canceled"):
		return "client_canceled"
	case strings.Contains(l, "deadline exceeded") || strings.Contains(l, "timeout") || strings.Contains(l, "timed out"):
		return "timeout"
	case strings.Contains(l, "connection refused"):
		return "conn_refused"
	case strings.Contains(l, "no such host") || strings.Contains(l, "lookup "):
		return "dns"
	case strings.Contains(l, "x509") || strings.Contains(l, "certificate") || strings.Contains(l, "tls"):
		return "tls"
	case strings.Contains(l, "proto is invalid") || strings.Contains(l, "must be a string or array") || strings.Contains(l, "invalid_json") || strings.Contains(l, "invalid_request"):
		return "bad_request"
	case strings.Contains(l, "status code is 5") || strings.Contains(l, " 5xx"):
		return "http_5xx"
	case strings.Contains(l, "status code is 4") || strings.Contains(l, " 4xx"):
		return "http_4xx"
	case strings.Contains(l, "[error]"):
		return "other"
	default:
		return ""
	}
}

// connClasses 属于上游连接层异常的分类集合。
var connClasses = map[string]bool{
	"eof": true, "timeout": true, "conn_refused": true, "dns": true, "tls": true, "http_5xx": true,
}

// IsUpstreamFault 判断该错误是否为「上游/连接」问题（区别于客户端 4xx / bad_request / 客户端取消）。
func (e Event) IsUpstreamFault() bool {
	return connClasses[e.ErrClass]
}
