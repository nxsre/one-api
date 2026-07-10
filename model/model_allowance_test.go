package model

import "testing"

func TestAllowanceMatchRequestModel_Exact(t *testing.T) {
	m := []map[string]string{{"claude-sonnet-4-6": "pa/claude-sonnet-4-6"}}
	if !AllowanceMatchRequestModel("claude-sonnet-4-6", "claude-sonnet-4-6", m) {
		t.Fatal("exact request name should match")
	}
}

func TestAllowanceMatchRequestModel_ViaMapping(t *testing.T) {
	m := []map[string]string{{"claude-sonnet-4-6": "pa/claude-sonnet-4-6"}}
	if !AllowanceMatchRequestModel("claude-sonnet-4-6", "pa/claude-sonnet-4-6", m) {
		t.Fatal("request name should match upstream entry when mapping exists")
	}
}

func TestAllowanceMatchRequestModel_NoMappingExactOnly(t *testing.T) {
	if AllowanceMatchRequestModel("gpt-4", "gpt-4o", nil) {
		t.Fatal("different names without mapping should not match")
	}
	if !AllowanceMatchRequestModel("gpt-4", "gpt-4", nil) {
		t.Fatal("same name without mapping should match")
	}
}

func TestClientFacingModelName(t *testing.T) {
	m := map[string]string{"claude-sonnet-4-6": "pa/claude-sonnet-4-6"}
	if got := ClientFacingModelName("pa/claude-sonnet-4-6", m); got != "claude-sonnet-4-6" {
		t.Fatalf("got %q", got)
	}
	if got := ClientFacingModelName("gpt-4", m); got != "gpt-4" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTokenModelAllowEntry(t *testing.T) {
	ent := ParseTokenModelAllowEntry("#8:claude-haiku")
	if ent.ChannelID != 8 || ent.Model != "claude-haiku" {
		t.Fatalf("got %+v", ent)
	}
	ent = ParseTokenModelAllowEntry("gpt-4")
	if ent.ChannelID != 0 || ent.Model != "gpt-4" {
		t.Fatalf("got %+v", ent)
	}
}

func TestTokenAllowlistRoutingChannelIDs_ScopedOnly(t *testing.T) {
	csv := "#8:claude-haiku,#5:claude-haiku"
	ch := TokenAllowlistRoutingChannelIDs(csv, "claude-haiku", nil)
	if len(ch) != 2 {
		t.Fatalf("expected 2 channels, got %v", ch)
	}
	if _, ok := ch[8]; !ok {
		t.Fatal("missing channel 8")
	}
	if _, ok := ch[5]; !ok {
		t.Fatal("missing channel 5")
	}
}

func TestTokenAllowlistRoutingChannelIDs_PlainEntry(t *testing.T) {
	csv := "claude-haiku,#8:claude-haiku"
	if ch := TokenAllowlistRoutingChannelIDs(csv, "claude-haiku", nil); ch != nil {
		t.Fatalf("plain entry should not restrict channels, got %v", ch)
	}
}

func TestIntersectChannelIDSets(t *testing.T) {
	a := map[int]struct{}{3: {}, 5: {}, 8: {}}
	b := map[int]struct{}{8: {}}
	out := IntersectChannelIDSets(a, b)
	if len(out) != 1 {
		t.Fatalf("got %v", out)
	}
	if _, ok := out[8]; !ok {
		t.Fatal("expected channel 8")
	}
}
