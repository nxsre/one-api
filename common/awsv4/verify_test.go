package awsv4

import (
	"net/http"
	"testing"
)

func TestCanonicalQueryString_decodeThenEncode(t *testing.T) {
	// ListObjectsV2 常见 query：delimiter 在线上为 %2F，canonical 须为 delimiter=%2F 而非 %252F
	got := canonicalQueryString("list-type=2&delimiter=%2F")
	want := "delimiter=%2F&list-type=2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = canonicalQueryString("b=2&a=1")
	if got != "a=1&b=2" {
		t.Fatalf("unexpected %q", got)
	}
	got = canonicalQueryString("k=a%20b")
	if got != "k=a%20b" {
		t.Fatalf("unexpected %q", got)
	}
	got = canonicalQueryString("k=a b")
	if got != "k=a%20b" {
		t.Fatalf("space in query: got %q want %q", got, "k=a%20b")
	}
}

func TestHeaderValue_usesRequestHostWhenNotInHeaderMap(t *testing.T) {
	// 模拟 net/http Server 行为：Host 仅在 r.Host，Header 中无 host。
	r := &http.Request{
		Method: "PUT",
		Host:   "127.0.0.1:13000",
		Header: http.Header{
			"X-Amz-Date":           {"20260514T000000Z"},
			"X-Amz-Content-Sha256": {"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		},
	}
	if got := headerValue(r, "host"); got != "127.0.0.1:13000" {
		t.Fatalf("host: got %q", got)
	}
}

func TestParseAuthParams(t *testing.T) {
	s := "Credential=AK%2F20240101%2Fus-east-1%2Fs3%2Faws4_request, SignedHeaders=host%3Bx-amz-date, Signature=abc"
	m := parseAuthParams(s)
	if m["Credential"] == "" || m["SignedHeaders"] == "" || m["Signature"] != "abc" {
		t.Fatalf("parseAuthParams: %#v", m)
	}
}
