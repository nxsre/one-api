package service

import "testing"

func TestNormalizeAmapRelayPath(t *testing.T) {
	if NormalizeAmapRelayPath("") != "" {
		t.Fatal()
	}
	if NormalizeAmapRelayPath("v5/place/text") != "/v5/place/text" {
		t.Fatal(NormalizeAmapRelayPath("v5/place/text"))
	}
	if NormalizeAmapRelayPath("/v5/place/../direction/walking") != "/v5/direction/walking" {
		t.Fatal(NormalizeAmapRelayPath("/v5/place/../direction/walking"))
	}
}

func TestIsAllowedAmapRelayPath(t *testing.T) {
	ok := []string{
		"/v3/direction/walking",
		"/v3/direction/driving",
		"/v3/direction/transit/integrated",
		"/v4/direction/bicycling",
		"/v5/direction/driving",
		"/v5/direction/walking",
		"/v5/direction/bicycling",
		"/v5/direction/electrobike",
		"/v5/direction/transit/integrated",
		"/v5/place/text",
	}
	for _, p := range ok {
		if !IsAllowedAmapRelayPath(p) {
			t.Fatalf("expected allowed: %s", p)
		}
	}
	bad := []string{
		"",
		"/",
		"/v3/geocode/geo",
		"/v5/inputtips/inputtips",
		"https://evil.com/v5/place/text",
	}
	for _, p := range bad {
		if IsAllowedAmapRelayPath(p) {
			t.Fatalf("expected denied: %s", p)
		}
	}
}
