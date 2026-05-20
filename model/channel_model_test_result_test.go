package model

import "testing"

func TestChannelTestBaseURLHashStable(t *testing.T) {
	a := channelTestBaseURLHash("https://api.example.com/v1")
	b := channelTestBaseURLHash("https://api.example.com/v1")
	if a != b || len(a) != 64 {
		t.Fatalf("hash=%q len=%d", a, len(a))
	}
	if channelTestBaseURLHash("https://other.example.com") == a {
		t.Fatal("expected different hash for different url")
	}
}
