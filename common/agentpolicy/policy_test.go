package agentpolicy

import "testing"

func allow(clients ...string) *Policy { return &Policy{AllowedClients: clients} }

func TestEvaluateEmptyAllowsAll(t *testing.T) {
	d := Evaluate(nil, nil, nil, nil, "claude-code")
	if d.Blocked {
		t.Fatalf("expected allowed, got %+v", d)
	}
	if d.Client != "claude-code" {
		t.Fatalf("client = %q", d.Client)
	}
}

func TestEvaluateUndetectedBecomesOther(t *testing.T) {
	d := Evaluate(nil, nil, nil, nil, "")
	if d.Client != OtherClient {
		t.Fatalf("client = %q, want %q", d.Client, OtherClient)
	}
}

func TestEvaluateTokenOverridesUserAndTenant(t *testing.T) {
	// 令牌允许 claude-code；用户/租户更严格也应被令牌覆盖（就近覆盖）。
	token := allow("claude-code")
	user := allow("openclaw")
	tenant := allow("hermes")
	if d := Evaluate(nil, token, user, tenant, "claude-code"); d.Blocked {
		t.Fatalf("token should allow claude-code, got %+v", d)
	}
	// 令牌白名单不含 openclaw → 拦截（即使用户允许）。
	if d := Evaluate(nil, token, user, tenant, "openclaw"); !d.Blocked {
		t.Fatalf("token should block openclaw")
	}
}

func TestEvaluateFallsThroughToUserThenTenant(t *testing.T) {
	user := allow("claude-code")
	tenant := allow("hermes")
	// 令牌未设白名单 → 用 用户层。
	if d := Evaluate(nil, nil, user, tenant, "claude-code"); d.Blocked {
		t.Fatalf("user should allow claude-code, got %+v", d)
	}
	if d := Evaluate(nil, nil, user, tenant, "hermes"); !d.Blocked {
		t.Fatalf("user layer should block hermes (only claude-code allowed)")
	}
	// 令牌、用户都未设 → 用 租户层。
	if d := Evaluate(nil, nil, nil, tenant, "hermes"); d.Blocked {
		t.Fatalf("tenant should allow hermes, got %+v", d)
	}
	if d := Evaluate(nil, nil, nil, tenant, "claude-code"); !d.Blocked {
		t.Fatalf("tenant layer should block claude-code")
	}
}

func TestEvaluateGlobalDisableWins(t *testing.T) {
	global := &Policy{Rules: map[string]ClientRule{"openclaw": {Disabled: true}}}
	token := allow("openclaw") // 令牌放行，但全局禁用优先。
	if d := Evaluate(global, token, nil, nil, "openclaw"); !d.Blocked || d.Reason != "globally disabled" {
		t.Fatalf("global disable should win, got %+v", d)
	}
}

func TestEvaluateGlobalAllowList(t *testing.T) {
	global := allow("claude-code")
	if d := Evaluate(global, nil, nil, nil, "hermes"); !d.Blocked {
		t.Fatalf("global allow list should block hermes")
	}
}

func TestEffectiveRuleDefaultFallback(t *testing.T) {
	p := &Policy{
		Default: &ClientRule{MaxRequests: 10, WindowSec: 60},
		Rules:   map[string]ClientRule{"claude-code": {MaxRequests: 5, WindowSec: 1}},
	}
	if r := p.EffectiveRule("claude-code"); r.MaxRequests != 5 {
		t.Fatalf("explicit rule expected, got %+v", r)
	}
	if r := p.EffectiveRule("openclaw"); r.MaxRequests != 10 {
		t.Fatalf("default rule expected, got %+v", r)
	}
}

func TestParse(t *testing.T) {
	if p := Parse(""); !p.IsZero() {
		t.Fatalf("empty should be zero")
	}
	if p := Parse("{}"); !p.IsZero() {
		t.Fatalf("{} should be zero")
	}
	p := Parse(`{"allowed_clients":["claude-code"],"default":{"max_requests":3,"window_sec":1}}`)
	if !p.Allows("claude-code") || p.Allows("hermes") {
		t.Fatalf("allow list parse wrong: %+v", p)
	}
	if p.EffectiveRule("anything").MaxRequests != 3 {
		t.Fatalf("default parse wrong: %+v", p)
	}
}
