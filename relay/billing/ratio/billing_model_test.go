package ratio

import "testing"

func TestBillingModelName_PrefersOrigin(t *testing.T) {
	got := BillingModelName("claude-sonnet-4-6", "pa/claude-sonnet-4-6")
	if got != "claude-sonnet-4-6" {
		t.Fatalf("期望官方名，实际 %q", got)
	}
}

func TestBillingModelName_StripsPAWhenNoOrigin(t *testing.T) {
	got := BillingModelName("", "pa/claude-sonnet-4-6")
	if got != "claude-sonnet-4-6" {
		t.Fatalf("期望剥掉 pa/ 前缀，实际 %q", got)
	}
}

func TestBillingModelName_KeepsReplicateStyleNames(t *testing.T) {
	const name = "meta/meta-llama-3-70b-instruct"
	got := BillingModelName("", name)
	if got != name {
		t.Fatalf("不应误剥 replicate 风格模型名，期望 %q，实际 %q", name, got)
	}
}
