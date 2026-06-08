package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const sectionSep = "----------------------------------------------------------------"

func printRunHeader(w io.Writer, base, model string, probeIDs []string, profile string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "================================================================")
	fmt.Fprintln(w, "model-verify-check")
	fmt.Fprintln(w, "================================================================")
	fmt.Fprintf(w, "base:    %s\n", base)
	fmt.Fprintf(w, "model:   %s\n", model)
	fmt.Fprintf(w, "profile: %s\n", profile)
	fmt.Fprintf(w, "probes:  %s\n", strings.Join(probeIDs, " "))
	fmt.Fprintln(w, "api:    POST /v1/messages  stream=true")
	fmt.Fprintln(w, "        A-G: temperature=0 max_tokens=220")
	fmt.Fprintln(w, "        H/I: max_tokens=64 (no temperature)")
	fmt.Fprintln(w, "        J:   max_tokens=128 (Agent 代理检测)")
	fmt.Fprintln(w, "        K:   tool_call get_weather 往返 (stream=false)")
	fmt.Fprintln(w, "================================================================")
	fmt.Fprintln(w)
}

func printProbeReport(w io.Writer, o probeOutcome) {
	verdict := "FAIL"
	switch {
	case o.Success && o.Pass:
		verdict = "PASS"
	case o.Success && !o.Pass:
		verdict = "WARN"
	}

	fmt.Fprintln(w, sectionSep)
	fmt.Fprintf(w, "[%s] %s\n", o.ProbeID, probeTitle(o.ProbeID))
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintf(w, "result: %s  http=%d  %dms\n", verdict, o.HTTPStatus, o.DurationMs)
	if o.Reason != "" {
		fmt.Fprintf(w, "reason: %s\n", o.Reason)
	}
	if o.Error != "" {
		fmt.Fprintf(w, "error:  %s\n", o.Error)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "[request header]")
	printIndented(w, "  ", formatHeaders(o.RequestHeaders))
	fmt.Fprintln(w)

	fmt.Fprintln(w, "[request body]")
	printIndented(w, "  ", indentJSON(o.RequestBody))
	fmt.Fprintln(w)

	fmt.Fprintln(w, "[response header]")
	printIndented(w, "  ", formatHeaders(o.ResponseHeaders))
	fmt.Fprintln(w)

	fmt.Fprintln(w, "[response body]")
	printIndented(w, "  ", o.ResponseBody)
	fmt.Fprintln(w)

	if o.Snippet != "" {
		fmt.Fprintln(w, "[assembled text]")
		printIndented(w, "  ", o.Snippet)
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "[check]")
	fmt.Fprintf(w, "  expected: %s\n", o.Expected)
	fmt.Fprintf(w, "  actual:   %s\n", actualVerdict(o))
	fmt.Fprintln(w)
}

func printSummaryBlock(w io.Writer, outcomes []probeOutcome, profile string) {
	fmt.Fprintln(w, "================================================================")
	fmt.Fprintln(w, "summary")
	fmt.Fprintln(w, "================================================================")

	ok, pass, warn, httpFail, ignoredWarn := 0, 0, 0, 0, 0
	for _, o := range outcomes {
		if o.Success {
			ok++
		}
		if o.Pass {
			pass++
		} else if o.Success {
			warn++
			if profile == profileOAuthProxy && oauthProxyExpectedWarn(o.ProbeID) {
				ignoredWarn++
			}
		} else {
			httpFail++
		}
		mark := "FAIL"
		if o.Success && o.Pass {
			mark = "PASS"
		} else if o.Success {
			mark = "WARN"
		}
		fmt.Fprintf(w, "  [%s] %-4s  %s  %dms\n", mark, o.ProbeID, probeTitle(o.ProbeID), o.DurationMs)
		if o.Snippet != "" && (!o.Pass || !o.Success) {
			fmt.Fprintf(w, "         reply: %s\n", oneLine(o.Snippet))
		}
		if o.Error != "" {
			fmt.Fprintf(w, "         error: %s\n", oneLine(o.Error))
		}
	}
	fmt.Fprintf(w, "\ntotal=%d  http_ok=%d  pass=%d  warn=%d  http_fail=%d  profile=%s\n",
		len(outcomes), ok, pass, warn, httpFail, profile)
	if profile == profileOAuthProxy && ignoredWarn > 0 {
		fmt.Fprintf(w, "note:   oauth-proxy 模式下 %d 条 H/I/J WARN 不计入 exit 失败\n", ignoredWarn)
	} else if warn > 0 && httpFail == 0 {
		fmt.Fprintln(w, "note:   warn 多为语义未达标（HTTP 已成功）；经 CLIProxyAPI OAuth 时 H/I/J 常见属预期，可用 -profile oauth-proxy")
	}
	if shouldExitFailure(outcomes, profile) {
		fmt.Fprintln(w, "exit:   1 (存在未达标项)")
	} else {
		fmt.Fprintln(w, "exit:   0")
	}
	fmt.Fprintln(w, "================================================================")
}

func printIndented(w io.Writer, prefix, text string) {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		fmt.Fprintln(w, prefix+"(empty)")
		return
	}
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(w, "%s%s\n", prefix, line)
	}
}

func actualVerdict(o probeOutcome) string {
	if !o.Success {
		if o.Error != "" {
			return "request failed: " + o.Error
		}
		return fmt.Sprintf("request failed: http %d", o.HTTPStatus)
	}
	if o.Pass {
		return "pass: " + o.Reason
	}
	return "fail: " + o.Reason + "; reply: " + oneLine(o.Snippet)
}

func indentJSON(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return string(b)
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " / ")
	return truncate(s, 120)
}
