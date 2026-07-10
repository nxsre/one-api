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
		name    string
		h       http.Header
		want    string
		wantVia string
	}{
		{"claude-code by UA", hdr(map[string]string{"User-Agent": "claude-cli/1.2.3 (external, cli)"}), "claude-code", "header"},
		{"claude-code by x-app", hdr(map[string]string{"User-Agent": "axios/1.0", "x-app": "cli"}), "claude-code", "header"},
		{"claude-code by beta", hdr(map[string]string{"anthropic-beta": "claude-code-20250219,oauth-2025-04-20"}), "claude-code", "header"},
		{"codex tui", hdr(map[string]string{"User-Agent": "codex_cli_rs/0.20.0 (Mac OS 15.0; arm64) Terminal"}), "codex", "header"},
		{"codex exec", hdr(map[string]string{"User-Agent": "codex_exec/0.137.0 (Debian 12.0.0; aarch64) unknown"}), "codex", "header"},
		{"codex originator", hdr(map[string]string{"User-Agent": "openai-python/1.0", "originator": "codex_vscode"}), "codex", "header"},
		{"codex proprietary header", hdr(map[string]string{"User-Agent": "x", "x-codex-turn-metadata": "{\"turn_id\":\"t1\"}"}), "codex", "header"},
		{"gemini-cli UA", hdr(map[string]string{"User-Agent": "GeminiCLI/1.2.3/gemini-2.5-pro (darwin; arm64)"}), "gemini-cli", "header"},
		{"gemini-cli vscode proxy", hdr(map[string]string{"User-Agent": "CloudCodeVSCode/1.0 (aidev_client; proxy_client=geminicli)"}), "gemini-cli", "header"},
		{"gemini-cli privileged header", hdr(map[string]string{"User-Agent": "x", "x-gemini-api-privileged-user-id": "u1"}), "gemini-cli", "header"},
		{"openclaw UA", hdr(map[string]string{"User-Agent": "OpenClaw/0.5"}), "openclaw", "header"},
		{"hermes UA", hdr(map[string]string{"User-Agent": "hermes-agent/2.0"}), "hermes", "header"},
		{"copilot header", hdr(map[string]string{"User-Agent": "x", "copilot-integration-id": "vscode-chat"}), "copilot", "header"},
		{"opencode UA", hdr(map[string]string{"User-Agent": "opencode/0.3.5"}), "opencode", "header"},
		{"qwen-code UA", hdr(map[string]string{"User-Agent": "QwenCode/1.0.0 (linux; x64)"}), "qwen-code", "header"},
		{"generic sdk UA unknown", hdr(map[string]string{"User-Agent": "OpenAI/JS 6.39.1"}), "", ""},
		{"postman unknown", hdr(map[string]string{"User-Agent": "PostmanRuntime/7.0"}), "", ""},
		{"android UA is not droid", hdr(map[string]string{"User-Agent": "Dalvik/2.1.0 (Linux; U; Android 12; Pixel 6)"}), "", ""},
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

func TestVersionFollowsMatchedToken(t *testing.T) {
	cases := []struct {
		name string
		h    http.Header
		want string
	}{
		{"claude-cli slash version", hdr(map[string]string{"User-Agent": "claude-cli/1.0.44 (external, cli)"}), "1.0.44"},
		{"codex tui version", hdr(map[string]string{"User-Agent": "codex_cli_rs/0.20.0 (Mac OS 15.0; arm64)"}), "0.20.0"},
		{"gemini tui variant", hdr(map[string]string{"User-Agent": "GeminiCLI-tui/0.45.0/gemini-2.5-pro (linux; arm64)"}), "0.45.0"},
		// x-app 命中但 UA 无 claude 产品名：不得把 SDK 版本误报为 agent 版本。
		{"no token no version", hdr(map[string]string{"User-Agent": "axios/1.8.1", "x-app": "cli"}), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectHeader(tc.h)
			if got.Version != tc.want {
				t.Fatalf("version = %q, want %q", got.Version, tc.want)
			}
		})
	}
}

func TestDetectBodySystemVariants(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"anthropic string system", `{"system":"You are Claude Code, Anthropic's official CLI for Claude.","messages":[]}`, "claude-code"},
		{"anthropic block system", `{"system":[{"type":"text","text":"You are Claude Code, the CLI."}],"messages":[]}`, "claude-code"},
		{"openai system message", `{"messages":[{"role":"system","content":"You are Claude Code helper"}]}`, "claude-code"},
		{"codex instructions", `{"model":"x","instructions":"You are a coding agent running in the Codex CLI, a terminal-based coding assistant."}`, "codex"},
		{"gemini system", `{"messages":[{"role":"system","content":"You are an interactive CLI agent specializing in software engineering tasks."}]}`, "gemini-cli"},
		{"openclaw system", `{"messages":[{"role":"system","content":"You are a personal assistant running inside OpenClaw."}]}`, "openclaw"},
		{"hermes brand beats shared tools", `{"messages":[{"role":"system","content":"You are Hermes Agent, created by Nous Research."}],"tools":[{"type":"function","function":{"name":"exec"}},{"type":"function","function":{"name":"web_fetch"}},{"type":"function","function":{"name":"sessions_spawn"}}]}`, "hermes"},
		{"cursor system", `{"messages":[{"role":"system","content":"You are an AI coding assistant. You operate in Cursor."}]}`, "cursor"},
		{"cline system", `{"messages":[{"role":"system","content":"You are Cline, a highly skilled software engineer."}]}`, "cline"},
		{"roo system", `{"messages":[{"role":"system","content":"You are Roo, a software engineering agent."}]}`, "roo-code"},
		{"windsurf system", `{"messages":[{"role":"system","content":"You are Cascade, built by the Codeium engineering team."}]}`, "windsurf"},
		{"aider system", `{"messages":[{"role":"system","content":"Act as an expert software developer. Always use best practices."}]}`, "aider"},
		{"qwen beats gemini fork prompt", `{"messages":[{"role":"system","content":"You are Qwen Code, an interactive CLI agent specializing in software engineering tasks."}]}`, "qwen-code"},
		{"plain assistant unknown", `{"system":"You are a helpful assistant.","messages":[]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(http.Header{}, []byte(tc.body))
			if got.Client != tc.want {
				t.Fatalf("client = %q, want %q", got.Client, tc.want)
			}
		})
	}
}

func TestDetectBodyToolsFallback(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"claude-code tools", `{"system":"generic assistant","tools":[{"name":"Bash"},{"name":"Read"},{"name":"Edit"},{"name":"Glob"}]}`, "claude-code"},
		{"openclaw tools", `{"messages":[{"role":"system","content":"generic"}],"tools":[{"type":"function","function":{"name":"exec"}},{"type":"function","function":{"name":"web_fetch"}},{"type":"function","function":{"name":"sessions_spawn"}},{"type":"function","function":{"name":"read"}}]}`, "openclaw"},
		{"gemini tools", `{"tools":[{"name":"run_shell_command"},{"name":"replace"},{"name":"read_many_files"}]}`, "gemini-cli"},
		// 小写 bash/read/edit 不得命中 Claude Code 的大写工具集。
		{"lowercase tools unknown", `{"tools":[{"name":"bash"},{"name":"read"},{"name":"edit"}]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(http.Header{}, []byte(tc.body))
			if got.Client != tc.want {
				t.Fatalf("client = %q, want %q", got.Client, tc.want)
			}
		})
	}
}

func TestDetectClaudeCodeUserID(t *testing.T) {
	body := []byte(`{"system":"custom identity","metadata":{"user_id":"user_08636e04fdbca7dfe5e5d24bb4213284e0ff81f30cba9c4a0e63b31be7373b0d_account_d0f6bff6-70f6-46eb-9d3c-9557e0908440_session_74d5f2b0-4c07-42c9-8695-b06d543ae0b8"},"messages":[]}`)
	got := Detect(http.Header{}, body)
	if got.Client != "claude-code" || got.Via != "body" {
		t.Fatalf("got %+v, want claude-code/body", got)
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
	if got := DetectHeader(h); got.Client != "" {
		t.Fatalf("header-only should not identify openclaw, got %+v", got)
	}
	body := []byte(`{"model":"probe-model","messages":[{"role":"system","content":"You are a personal assistant running inside OpenClaw.\n## Tooling\n- read: Read file contents\n- exec: Run shell commands"}]}`)
	if got := Detect(h, body); got.Client != "openclaw" || got.Via != "body" {
		t.Fatalf("expected openclaw/body, got %+v", got)
	}
}

func TestHeaderTakesPrecedence(t *testing.T) {
	body := []byte(`{"messages":[{"role":"system","content":"running inside OpenClaw"}]}`)
	got := Detect(hdr(map[string]string{"x-app": "cli"}), body)
	if got.Client != "claude-code" || got.Via != "header" {
		t.Fatalf("got %+v, want claude-code/header", got)
	}
}

func TestKnownClientsIncludesCore(t *testing.T) {
	want := map[string]bool{
		"claude-code": false, "codex": false, "gemini-cli": false,
		"openclaw": false, "hermes": false, "cursor": false, "cline": false,
	}
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
