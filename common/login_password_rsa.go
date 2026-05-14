package common

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/logger"
)

var LoginPasswordRSAPrivateKey *rsa.PrivateKey

// LoginPasswordRSAPublicKeyPEM PKIX PEM，供 /api/status 下发给前端加密密码。
var LoginPasswordRSAPublicKeyPEM string

// LoginPasswordRSAKeyFilePath 私钥文件路径（用于排查）。
var LoginPasswordRSAKeyFilePath string

const defaultLoginPasswordRSAKeyFile = "data/login_password_rsa_private.pem"

// InitLoginPasswordRSA 初始化登录 RSA：环境变量 PEM、或文件、或自动生成并落盘。
func InitLoginPasswordRSA() error {
	LoginPasswordRSAKeyFilePath = ""

	raw := strings.TrimSpace(env.StringAlways("login_password_rsa_private_key"))
	if raw != "" {
		pemStr := strings.ReplaceAll(raw, "\\n", "\n")
		key, err := parseRSAPrivateKeyFromPEM([]byte(pemStr))
		if err != nil {
			return fmt.Errorf("login_password_rsa_private_key: %w", err)
		}
		return applyLoginPasswordRSAKey(key, "config login_password_rsa_private_key")
	}

	path := strings.TrimSpace(env.StringAlways("login_password_rsa_key_file"))
	if path == "" {
		path = defaultLoginPasswordRSAKeyFile
	}
	path = filepath.Clean(path)
	LoginPasswordRSAKeyFilePath = path

	pemData, err := os.ReadFile(path)
	if err == nil {
		key, perr := parseRSAPrivateKeyFromPEM(pemData)
		if perr != nil {
			return fmt.Errorf("login_password_rsa_key_file %q: invalid PEM: %w", path, perr)
		}
		return applyLoginPasswordRSAKey(key, fmt.Sprintf("file %s", path))
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("read login_password_rsa_key_file %q: %w", path, err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate login RSA key: %w", err)
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	if err := atomicWritePrivateFile(path, pem.EncodeToMemory(block)); err != nil {
		return fmt.Errorf("persist login RSA key to %q: %w", path, err)
	}
	logger.SysLog(fmt.Sprintf("generated login RSA private key (persisted to %s; back up for multi-node)", path))
	return applyLoginPasswordRSAKey(key, fmt.Sprintf("generated -> %s", path))
}

func applyLoginPasswordRSAKey(key *rsa.PrivateKey, source string) error {
	LoginPasswordRSAPrivateKey = key
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal RSA public key: %w", err)
	}
	LoginPasswordRSAPublicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	}))
	logger.SysLog(fmt.Sprintf("login password RSA ready (%s); public key in /api/status", source))
	return nil
}

func atomicWritePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	f, err := os.CreateTemp(dir, ".login_rsa_tmp_*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	cleanup = false
	return os.Chmod(path, 0600)
}

func parseRSAPrivateKeyFromPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsakey, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS#8 key is not RSA")
		}
		return rsakey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM type %q", block.Type)
	}
}

// DecryptLoginPasswordRSA 将 Base64(RSA-PKCS1v15(ciphertext)) 解密为 UTF-8 密码明文。
func DecryptLoginPasswordRSA(b64 string) (string, error) {
	if LoginPasswordRSAPrivateKey == nil {
		return "", errors.New("login RSA key not initialized")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "", err
	}
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, LoginPasswordRSAPrivateKey, raw)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func loginRequestProofMessageV1(id string, ts int64) []byte {
	return []byte(fmt.Sprintf("LOGIN|v1|%s|%d", id, ts))
}

// SignLoginRequestProof RSASSA-PKCS1-v1_5 + SHA-256。
func SignLoginRequestProof(id string, ts int64) (sigB64 string, err error) {
	if LoginPasswordRSAPrivateKey == nil {
		return "", errors.New("login RSA key not initialized")
	}
	h := sha256.Sum256(loginRequestProofMessageV1(id, ts))
	sig, err := rsa.SignPKCS1v15(rand.Reader, LoginPasswordRSAPrivateKey, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyLoginRequestProof 校验客户端回传的 Base64 签名。
func VerifyLoginRequestProof(id string, ts int64, sigB64 string) error {
	if LoginPasswordRSAPrivateKey == nil {
		return errors.New("login RSA key not initialized")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil {
		return err
	}
	h := sha256.Sum256(loginRequestProofMessageV1(id, ts))
	return rsa.VerifyPKCS1v15(&LoginPasswordRSAPrivateKey.PublicKey, crypto.SHA256, h[:], sig)
}
