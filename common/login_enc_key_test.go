package common

import (
	"testing"

	"github.com/songquanpeng/one-api/common/config"
)

func TestLoginPayloadAESRoundTrip(t *testing.T) {
	key, _, err := NewLoginEncKey()
	if err != nil {
		t.Fatal(err)
	}
	plain := "p@ssw0rd!测试"
	enc, err := EncryptLoginPayloadAES(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecryptLoginPayloadAES(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if got != plain {
		t.Fatalf("want %q got %q", plain, got)
	}
}

func TestLoginRequestProofHMAC(t *testing.T) {
	config.SessionSecret = "unit-test-session-secret"
	id := "550e8400-e29b-41d4-a716-446655440000"
	ts := int64(1700000000)
	sig, err := SignLoginRequestProof(id, ts)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyLoginRequestProof(id, ts, sig); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLoginRequestProof(id, ts+1, sig); err == nil {
		t.Fatal("expected invalid sig for wrong ts")
	}
}
