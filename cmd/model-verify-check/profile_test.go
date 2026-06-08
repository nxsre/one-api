package main

import "testing"

func TestShouldExitFailureStrict(t *testing.T) {
	outcomes := []probeOutcome{
		{ProbeID: "A", Success: true, Pass: true},
		{ProbeID: "H", Success: true, Pass: false},
	}
	if !shouldExitFailure(outcomes, profileStrict) {
		t.Fatal("strict profile should exit on semantic warn")
	}
}

func TestShouldExitFailureOAuthProxyIgnoresHIJ(t *testing.T) {
	outcomes := []probeOutcome{
		{ProbeID: "A", Success: true, Pass: true},
		{ProbeID: "H", Success: true, Pass: false},
		{ProbeID: "I", Success: true, Pass: false},
		{ProbeID: "J", Success: true, Pass: false},
		{ProbeID: "K", Success: true, Pass: true},
	}
	if shouldExitFailure(outcomes, profileOAuthProxy) {
		t.Fatal("oauth-proxy profile should ignore H/I/J warn when core probes pass")
	}
}

func TestShouldExitFailureOAuthProxyFailsOnCoreProbe(t *testing.T) {
	outcomes := []probeOutcome{
		{ProbeID: "A", Success: true, Pass: false},
		{ProbeID: "H", Success: true, Pass: false},
	}
	if !shouldExitFailure(outcomes, profileOAuthProxy) {
		t.Fatal("oauth-proxy profile should exit when A-G/K fail")
	}
}

func TestShouldExitFailureOAuthProxyFailsOnHTTP(t *testing.T) {
	outcomes := []probeOutcome{
		{ProbeID: "A", Success: false, Pass: false, Error: "400"},
	}
	if !shouldExitFailure(outcomes, profileOAuthProxy) {
		t.Fatal("oauth-proxy profile should exit on HTTP failure")
	}
}

func TestNormalizeProfile(t *testing.T) {
	got, err := normalizeProfile("oauth-proxy")
	if err != nil || got != profileOAuthProxy {
		t.Fatalf("normalizeProfile() = %q, %v", got, err)
	}
	if _, err := normalizeProfile("unknown"); err == nil {
		t.Fatal("expected error for unknown profile")
	}
}
