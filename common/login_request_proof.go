package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
)

func loginRequestProofMessageV1(id string, ts int64) []byte {
	return []byte(fmt.Sprintf("LOGIN|v1|%s|%d", id, ts))
}

// SignLoginRequestProof HMAC-SHA256，密钥为 SessionSecret。
func SignLoginRequestProof(id string, ts int64) (sigB64 string, err error) {
	secret := strings.TrimSpace(config.SessionSecret)
	if secret == "" {
		return "", errors.New("session secret not configured")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(loginRequestProofMessageV1(id, ts))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// VerifyLoginRequestProof 校验客户端回传的 Base64 HMAC。
func VerifyLoginRequestProof(id string, ts int64, sigB64 string) error {
	secret := strings.TrimSpace(config.SessionSecret)
	if secret == "" {
		return errors.New("session secret not configured")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(loginRequestProofMessageV1(id, ts))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return errors.New("invalid login request proof signature")
	}
	return nil
}
