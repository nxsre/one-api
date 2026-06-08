package agentdetect

import (
	"net/http"
	"testing"
)

func hdr(kv map[string]string) http.Header {
	h := http.Header{}
	for k, v := range kv {
		h.Set(k, v)
	}
	return h
}

func TestDetectHeader(t *testing.T) {
	cases := []struct {
		name   string
		h      http.Header
		want   string
		wantVia string
	}{
		{"claude-code by UA", hdr(map[string]string{"User-Agent": "claude-cli/1.2.3 (external, cli)"}), "claude-code", "header"},
		{"claude-code by x-app", hdr(map[string]string{"User-Agent": "axios/1.0", "x-app": "cli"}), "claude-code", "header"},
		{"claude-code by beta", hdr(map[string]string{"anthropic-beta": "claude-code-20250219,oauth-2025-04-20"}), "claude-code", "header"},
		{"openclaw", hdr(map[string]string{"User-Agent": "OpenClaw/0.5"}), "openclaw", "header"},
		{"hermes", hdr(map[string]string{"User-Agent": "hermes-agent/2.0"}), "hermes", "header"},
		{"unknown", hdr(map[string]string{"User-Agent": "PostmanRuntime/7.0"}), "", ""},
		{"empty", http.Header{}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectHeader(tc.h)
			if got.Client != tc.want {
				t.Fatalf("client = %q, want %q", got.Client, tc.want)
			}
			if tc.want != "" && got.Via != tc.wantVia {
				t.Fatalf("via = %q, want %q", got.Via, tc.wantVia)
			}
		})
	}
}

func TestDetectBodyAnthropicSystem(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4","system":"You are Claude Code, Anthropic's official CLI for Claude.","messages":[]}`)
	got := Detect(hdr(map[string]string{"User-Agent": "axios/1.0"}), body)
	if got.Client != "claude-code" || got.Via != "body" {
		t.Fatalf("got %+v, want claude-code/body", got)
	}
}

func TestDetectBodySystemBlocks(t *testing.T) {
	body := []byte(`{"system":[{"type":"text","text":"You are Claude Code, the CLI."}],"messages":[]}`)
	got := Detect(http.Header{}, body)
	if got.Client != "claude-code" {
		t.Fatalf("got %+v, want claude-code", got)
	}
}

func TestDetectBodyTools(t *testing.T) {
	body := []byte(`{"system":"generic assistant","tools":[{"name":"Bash"},{"name":"Read"},{"name":"Edit"},{"name":"Glob"}]}`)
	got := Detect(http.Header{}, body)
	if got.Client != "claude-code" || got.Via != "body" {
		t.Fatalf("got %+v, want claude-code/body", got)
	}
}

func TestDetectBodyOpenAISystemMessage(t *testing.T) {
	body := []byte(`{"messages":[{"role":"system","content":"You are Claude Code helper"}]}`)
	got := Detect(http.Header{}, body)
	if got.Client != "claude-code" {
		t.Fatalf("got %+v, want claude-code", got)
	}
}

// TestDetectOpenClawRealFingerprint 用真实抓取的 openclaw 出站特征验证：
// UA 是通用 OpenAI SDK（不可据此识别），但 system prompt 暴露身份 → 应识别为 openclaw。
func TestDetectOpenClawRealFingerprint(t *testing.T) {
	h := hdr(map[string]string{
		"User-Agent":                  "OpenAI/JS 6.39.1",
		"x-stainless-lang":            "js",
		"x-stainless-package-version": "6.39.1",
	})
	// 仅看头：识别不出（与真实情况一致）。
	if got := DetectHeader(h); got.Client != "" {
		t.Fatalf("header-only should not identify openclaw, got %+v", got)
	}
	// 看体：system prompt 命中。
	body := []byte(`{"model":"probe-model","messages":[{"role":"system","content":"You are a personal assistant running inside OpenClaw.\n## Tooling\n- read: Read file contents\n- exec: Run shell commands"}]}`)
	if got := Detect(h, body); got.Client != "openclaw" || got.Via != "body" {
		t.Fatalf("expected openclaw/body, got %+v", got)
	}
}

// TestDetectOpenClawByTools 验证工具集指纹（system 不含名字时的兜底路径）。
func TestDetectOpenClawByTools(t *testing.T) {
	body := []byte(`{"messages":[{"role":"system","content":"generic"}],"tools":[{"type":"function","function":{"name":"exec"}},{"type":"function","function":{"name":"web_fetch"}},{"type":"function","function":{"name":"sessions_spawn"}},{"type":"function","function":{"name":"read"}}]}`)
	if got := Detect(http.Header{}, body); got.Client != "openclaw" {
		t.Fatalf("expected openclaw via tools, got %+v", got)
	}
}

// TestDetectHermesRealFingerprint 用 NousResearch/hermes-agent 源码默认 identity 验证。
// hermes 与 openclaw 同源、工具集相同，靠品牌 system prompt 区分（两遍匹配，品牌优先）。
func TestDetectHermesRealFingerprint(t *testing.T) {
	// UA 同样是通用 OpenAI SDK → 头识别不出。
	h := hdr(map[string]string{"User-Agent": "OpenAI/JS 6.39.1"})
	if got := DetectHeader(h); got.Client != "" {
		t.Fatalf("header-only should not identify hermes, got %+v", got)
	}
	body := []byte(`{"messages":[{"role":"system","content":"You are Hermes Agent, an intelligent AI assistant created by Nous Research. You are helpful, knowledgeable, and direct."}],"tools":[{"type":"function","function":{"name":"exec"}},{"type":"function","function":{"name":"web_fetch"}},{"type":"function","function":{"name":"sessions_spawn"}}]}`)
	got := Detect(h, body)
	if got.Client != "hermes" || got.Via != "body" {
		t.Fatalf("expected hermes/body (brand prompt must win over shared toolset), got %+v", got)
	}
}

// TestDetectCodex 验证 OpenAI Codex CLI:UA / originator 头 / 专有 x-codex-* 头 任一即可识别。
func TestDetectCodex(t *testing.T) {
	cases := []http.Header{
		hdr(map[string]string{"User-Agent": "codex_cli_rs/0.20.0 (Mac OS 15.0; arm64) Terminal"}),
		hdr(map[string]string{"User-Agent": "codex_exec/0.137.0 (Debian 12.0.0; aarch64) unknown"}), // 实测 headless exec UA
		hdr(map[string]string{"User-Agent": "openai-python/1.0", "originator": "codex_exec"}),       // 实测 exec originator
		hdr(map[string]string{"User-Agent": "openai-python/1.0", "originator": "codex_vscode"}),
		hdr(map[string]string{"User-Agent": "x", "x-codex-turn-metadata": "{\"turn_id\":\"t1\"}"}),
	}
	for i, h := range cases {
		if got := DetectHeader(h); got.Client != "codex" {
			t.Fatalf("case %d: expected codex, got %+v", i, got)
		}
	}
	// Responses API:system 提示在 instructions(实测真实文案)
	body := []byte(`{"model":"x","instructions":"You are a coding agent running in the Codex CLI, a terminal-based coding assistant."}`)
	if got := Detect(http.Header{}, body); got.Client != "codex" || got.Via != "body" {
		t.Fatalf("codex instructions body: expected codex/body, got %+v", got)
	}
}

// TestDetectGeminiCLI 验证 Google Gemini CLI:API UA 含 GeminiCLI;或体内 system prompt 命中。
func TestDetectGeminiCLI(t *testing.T) {
	if got := DetectHeader(hdr(map[string]string{"User-Agent": "GeminiCLI/1.2.3/gemini-2.5-pro (darwin; arm64)"})); got.Client != "gemini-cli" {
		t.Fatalf("UA: expected gemini-cli, got %+v", got)
	}
	// VS Code 代理路径 UA 含 proxy_client=geminicli
	if got := DetectHeader(hdr(map[string]string{"User-Agent": "CloudCodeVSCode/1.0 (aidev_client; proxy_client=geminicli)"})); got.Client != "gemini-cli" {
		t.Fatalf("vscode UA: expected gemini-cli, got %+v", got)
	}
	// 体内 system prompt
	body := []byte(`{"messages":[{"role":"system","content":"You are Gemini CLI, an interactive CLI agent specializing in software engineering tasks."}]}`)
	if got := Detect(http.Header{}, body); got.Client != "gemini-cli" || got.Via != "body" {
		t.Fatalf("body: expected gemini-cli/body, got %+v", got)
	}
}

func TestKnownClientsIncludesAll(t *testing.T) {
	want := map[string]bool{"claude-code": false, "openclaw": false, "hermes": false, "codex": false, "gemini-cli": false}
	for _, c := range KnownClients() {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for c, seen := range want {
		if !seen {
			t.Fatalf("KnownClients missing %q", c)
		}
	}
}

func TestDetectNoMatch(t *testing.T) {
	body := []byte(`{"system":"You are a helpful assistant.","messages":[]}`)
	if got := Detect(http.Header{}, body); got.Client != "" {
		t.Fatalf("got %+v, want empty", got)
	}
}

func TestHeaderTakesPrecedence(t *testing.T) {
	// 头部命中时不应再走 body 解析
	body := []byte(`{"system":"unrelated"}`)
	got := Detect(hdr(map[string]string{"x-app": "cli"}), body)
	if got.Via != "header" {
		t.Fatalf("via = %q, want header", got.Via)
	}
}
