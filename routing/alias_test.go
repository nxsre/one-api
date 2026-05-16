package routing

import "testing"

func TestApplyChainsDetectsCycle(t *testing.T) {
	_, err := applyChains("a", map[string]string{"a": "b", "b": "a"}, 32)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestApplyChainsLinear(t *testing.T) {
	out, err := applyChains("x", map[string]string{"x": "y", "y": "z"}, 32)
	if err != nil {
		t.Fatal(err)
	}
	if out != "z" {
		t.Fatalf("got %q", out)
	}
}
