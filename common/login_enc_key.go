package common

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginEncKeyPrefix = "login_enc:v1:"
	loginEncKeyTTL    = 10 * time.Minute
	loginEncKeySize   = 32
)

// NewLoginEncKey 生成一次性 AES-256 密钥（Base64）。
func NewLoginEncKey() (key []byte, keyB64 string, err error) {
	key = make([]byte, loginEncKeySize)
	if _, err = io.ReadFull(rand.Reader, key); err != nil {
		return nil, "", err
	}
	keyB64 = base64.StdEncoding.EncodeToString(key)
	return key, keyB64, nil
}

// StoreLoginEncKey 将密钥与 login_request_id 绑定（Redis 或 Session）。
func StoreLoginEncKey(c *gin.Context, proofID string, key []byte) error {
	if proofID == "" || len(key) != loginEncKeySize {
		return errors.New("invalid login enc key store params")
	}
	if RedisEnabled && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return RDB.Set(ctx, loginEncKeyPrefix+proofID, base64.StdEncoding.EncodeToString(key), loginEncKeyTTL).Err()
	}
	sess := sessions.Default(c)
	sess.Set("pending_login_enc_key_id", proofID)
	sess.Set("pending_login_enc_key", base64.StdEncoding.EncodeToString(key))
	return sess.Save()
}

// TakeLoginEncKey 取出并删除与 proof 绑定的一次性密钥。
func TakeLoginEncKey(c *gin.Context, proofID string) ([]byte, error) {
	if proofID == "" {
		return nil, errors.New("empty proof id")
	}
	if RedisEnabled && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		key := loginEncKeyPrefix + proofID
		b64, err := RDB.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		_ = RDB.Del(ctx, key).Err()
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil || len(raw) != loginEncKeySize {
			return nil, errors.New("invalid stored login enc key")
		}
		return raw, nil
	}
	sess := sessions.Default(c)
	sid, _ := sess.Get("pending_login_enc_key_id").(string)
	if sid != proofID {
		return nil, errors.New("login enc key not found for proof")
	}
	b64, ok := sess.Get("pending_login_enc_key").(string)
	if !ok || b64 == "" {
		return nil, errors.New("login enc key missing in session")
	}
	sess.Delete("pending_login_enc_key_id")
	sess.Delete("pending_login_enc_key")
	_ = sess.Save()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) != loginEncKeySize {
		return nil, errors.New("invalid session login enc key")
	}
	return raw, nil
}

// EncryptLoginPayloadAES 加密为 Base64(nonce||ciphertext+tag)，AES-256-GCM。
func EncryptLoginPayloadAES(key []byte, plaintext string) (string, error) {
	if len(key) != loginEncKeySize {
		return "", errors.New("invalid AES key length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	out := append(nonce, sealed...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// DecryptLoginPayloadAES 解密 Base64(nonce||ciphertext+tag)，AES-256-GCM。
func DecryptLoginPayloadAES(key []byte, b64 string) (string, error) {
	if len(key) != loginEncKeySize {
		return "", errors.New("invalid AES key length")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize+gcm.Overhead() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := raw[:nonceSize]
	ciphertext := raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
