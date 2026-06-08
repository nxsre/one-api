package core

import (
	"strings"
	"testing"
)

// 取自真实 one-api 运行日志的样本。
const sampleLog = `
[GIN] 2026/06/03 - 09:11:29 | 2026060309112665255241189458900 | 200 |   26.126704ms |   192.168.156.1 |     GET /api/log/
[GIN] 2026/06/03 - 09:11:29 | 202606030911295380747189654121 | 400 |    31.86788ms |   192.168.156.1 |    POST /v1/messages
[ERROR] 2026/06/03 - 09:11:29 | relay/adaptor/openai/util.go:14 [ErrorWrapper] [invalid_json]message content must be a string or array of content blocks
[ERROR] 2026/06/03 - 09:46:41 | 202606030946154601717131388364 | controller/relay.go:262 [processChannelRelayError] relay error (channel id 1, user id: 1): context canceled
[ERROR] 2026/06/03 - 09:50:47 | 2026060309483791692328311814471 | controller/relay.go:262 [processChannelRelayError] relay error (channel id 1, user id: 1): unexpected EOF
[ERROR] 2026/06/03 - 09:53:50 | 2026060309512991083214638575295 | controller/relay.go:262 [processChannelRelayError] relay error (channel id 1, user id: 1): do request failed: Post "https://api.novita.ai/anthropic/v1/messages": unexpected EOF
[ERROR] 2026/06/03 - 08:54:24 | 2026060308542448673623707413592 | controller/relay.go:262 [processChannelRelayError] relay error (channel id 3, user id: 1): The GenerateContentRequest proto is invalid:
[GIN] 2026/06/03 - 09:11:31 | 2026060309113165214181669639272 | 500 |    1.987086ms |   192.168.156.1 |     GET /v1/chat/completions
[INFO] 2026/06/03 - 02:07:33 | model/option.go:145 [SyncOptions] syncing options from database
`

func TestParseGin(t *testing.T) {
	ev, _ := ParseLine(`[GIN] 2026/06/03 - 09:11:29 | 202606030911295380747189654121 | 400 |    31.86788ms |   192.168.156.1 |    POST /v1/messages`)
	if !ev.IsAccess {
		t.Fatal("应识别为访问日志")
	}
	if ev.Status != 400 || ev.Method != "POST" || ev.Path != "/v1/messages" {
		t.Errorf("解析错误: status=%d method=%s path=%s", ev.Status, ev.Method, ev.Path)
	}
	if ev.LatencyMs < 31 || ev.LatencyMs > 32 {
		t.Errorf("延迟解析错误: %.4fms", ev.LatencyMs)
	}
	if ev.ErrClass != "http_4xx" {
		t.Errorf("应分类为 http_4xx, 实际 %s", ev.ErrClass)
	}
}

func TestParseRelayErrorEOF(t *testing.T) {
	ev, _ := ParseLine(`[ERROR] 2026/06/03 - 09:53:50 | 20260603 | controller/relay.go:262 [processChannelRelayError] relay error (channel id 1, user id: 1): do request failed: Post "https://api.novita.ai/anthropic/v1/messages": unexpected EOF`)
	if ev.ChannelID != 1 {
		t.Errorf("channel 解析错误: %d", ev.ChannelID)
	}
	if ev.Upstream != "api.novita.ai" {
		t.Errorf("上游 host 解析错误: %q", ev.Upstream)
	}
	if ev.ErrClass != "eof" {
		t.Errorf("应分类为 eof, 实际 %s", ev.ErrClass)
	}
	if !ev.IsUpstreamFault() {
		t.Error("eof 应判为上游故障")
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"context canceled":          "client_canceled",
		"context deadline exceeded": "timeout",
		"connection refused":        "conn_refused",
		"proto is invalid":          "bad_request",
		"must be a string or array": "bad_request",
	}
	for in, want := range cases {
		if got := classifyError(in); got != want {
			t.Errorf("classify(%q)=%q, want %q", in, got, want)
		}
	}
	// 客户端取消不应算上游故障
	ev, _ := ParseLine(`[ERROR] 2026/06/03 - 09:46:41 | x | controller/relay.go:262 [processChannelRelayError] relay error (channel id 1, user id: 1): context canceled`)
	if ev.IsUpstreamFault() {
		t.Error("client_canceled 不应判为上游故障")
	}
}

func TestAnalyzeAnomalies(t *testing.T) {
	events := ParseReader(strings.NewReader(sampleLog))
	th := DefaultThresholds()
	th.UpstreamConnErrors = 2 // novita 有 eof 等
	r := Analyze(events, th)
	if r.AccessLines != 3 {
		t.Errorf("访问行数应为 3, 实际 %d", r.AccessLines)
	}
	if r.StatusHist[500] != 1 {
		t.Errorf("应有 1 个 500, 实际 %d", r.StatusHist[500])
	}
	// 应检出 5xx 异常
	found5xx := false
	for _, a := range r.Anomalies {
		if strings.Contains(a, "5xx") {
			found5xx = true
		}
	}
	if !found5xx {
		t.Errorf("应检出 5xx 异常, 实际: %v", r.Anomalies)
	}
	// novita 应出现在上游统计里
	hasNovita := false
	for _, us := range r.Upstreams {
		if us.Host == "api.novita.ai" {
			hasNovita = true
		}
	}
	if !hasNovita {
		t.Error("上游统计应包含 api.novita.ai")
	}
}
