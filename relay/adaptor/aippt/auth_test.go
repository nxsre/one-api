package aippt

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"testing"
)

func TestGenerateGrantSignature_opensslCompatible(t *testing.T) {
	secret := "test-secret-hex"
	ts := int64(1700000000)
	sig := GenerateGrantSignature(secret, "GET", GrantTokenPath, ts)
	signStr := "GET" + "@" + GrantTokenPath + "@" + strconv.FormatInt(ts, 10)
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(signStr))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if sig != want {
		t.Fatalf("signature mismatch: got %q want %q; signStr=%q", sig, want, signStr)
	}
}
