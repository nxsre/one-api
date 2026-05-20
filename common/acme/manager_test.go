package acme

import "testing"

func TestParseIdentifiers(t *testing.T) {
	got := parseIdentifiers(" api.example.com,api.example.com ", "203.0.113.10, 203.0.113.11 ")
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %v", len(got), got)
	}
	if got[0] != "api.example.com" || got[1] != "203.0.113.10" || got[2] != "203.0.113.11" {
		t.Fatalf("unexpected order/content: %v", got)
	}
}

func TestAnyIP(t *testing.T) {
	if !anyIP([]string{"api.example.com", "203.0.113.10"}) {
		t.Fatal("expected ip detection")
	}
	if anyIP([]string{"api.example.com"}) {
		t.Fatal("expected no ip")
	}
}

func TestManagedIPDefaults(t *testing.T) {
	p, f := managedIPDefaults([]string{"api.example.com", "203.0.113.10", "203.0.113.11"})
	if p != "203.0.113.10" || f != "203.0.113.11" {
		t.Fatalf("got %q %q", p, f)
	}
	p, f = managedIPDefaults([]string{"api.example.com"})
	if p != "" || f != "" {
		t.Fatalf("expected empty, got %q %q", p, f)
	}
}
