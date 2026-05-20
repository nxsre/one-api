package anthropic

import "testing"

func TestSplitClientModelVariant(t *testing.T) {
	base, v := SplitClientModelVariant("claude-opus-4-7[1m]")
	if base != "claude-opus-4-7" || v != "[1m]" {
		t.Fatalf("got %q %q", base, v)
	}
	base, v = SplitClientModelVariant("claude-3-5-sonnet-20241022")
	if base != "claude-3-5-sonnet-20241022" || v != "" {
		t.Fatalf("got %q %q", base, v)
	}
}

func TestClientContextVariants(t *testing.T) {
	if got := ClientContextVariants("claude-opus-4-7"); len(got) != 1 || got[0] != "[1m]" {
		t.Fatalf("opus: %v", got)
	}
	if got := ClientContextVariants("claude-haiku-4-5-20251001"); len(got) != 0 {
		t.Fatalf("haiku: %v", got)
	}
}
