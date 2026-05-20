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
