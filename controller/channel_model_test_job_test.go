package controller

import "testing"

func TestNormalizeModelTestConcurrency(t *testing.T) {
	if got := normalizeModelTestConcurrency(0); got != 3 {
		t.Fatalf("default=%d", got)
	}
	if got := normalizeModelTestConcurrency(5); got != 5 {
		t.Fatalf("five=%d", got)
	}
	if got := normalizeModelTestConcurrency(99); got != 10 {
		t.Fatalf("cap=%d", got)
	}
}
