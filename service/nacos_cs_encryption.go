package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
)

// 与 Nacos 配置加密文档一致：dataId 以 cipher-{算法}- 为前缀时启用加密，此处实现 cipher-aes-。
// 参见 https://nacos.io/docs/latest/plugin/config-encryption-plugin/

const nacosCsEncEnvelope = "aes-gcm-v1"

// NacosCsEncryptionAlg 从 dataId 解析出的算法标识；空字符串表示不加密。
func NacosCsEncryptionAlg(dataID string) (string, error) {
	dl := strings.ToLower(strings.TrimSpace(dataID))
	switch {
	case strings.HasPrefix(dl, "cipher-aes-"):
		return "aes", nil
	case strings.HasPrefix(dl, "cipher-"):
		return "", fmt.Errorf("暂不支持的 dataId 加密前缀，仅支持 cipher-aes-（参考 Nacos 文档）")
	default:
		return "", nil
	}
}

// NacosCsEncryptionKeyConfigured 是否已配置 CS 主密钥（用于新发布 cipher-aes-*）。
func NacosCsEncryptionKeyConfigured() bool {
	return strings.TrimSpace(config.NacosCsEncryptionKey) != ""
}

// NacosCsEncryptionRotationConfigured 是否配置了轮换/历史密钥材料（仅参与解密）。
func NacosCsEncryptionRotationConfigured() bool {
	return strings.TrimSpace(config.NacosCsEncryptionKeyPrevious) != ""
}

func nacosCsKeyMaterialListDecryptOrder() []string {
	var out []string
	seen := make(map[string]struct{})
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(config.NacosCsEncryptionKey)
	prev := strings.TrimSpace(config.NacosCsEncryptionKeyPrevious)
	if prev != "" {
		for _, line := range strings.Split(prev, "\n") {
			add(strings.TrimSpace(line))
		}
	}
	return out
}

func nacosCsAES256KeyFromMaterial(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("密钥材料为空")
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

// nacosCsDecryptWithKey 使用给定 AES-256 密钥尝试解密 content（GCM）。
func nacosCsDecryptWithKey(key []byte, content, encKeyField string) (string, error) {
	if strings.TrimSpace(encKeyField) == "" {
		return content, nil
	}
	if strings.TrimSpace(encKeyField) != nacosCsEncEnvelope {
		return "", fmt.Errorf("未知的 encrypted_data_key 标记 %q", encKeyField)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content))
	if err != nil {
		return "", fmt.Errorf("解密失败: 非法密文 base64: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("解密失败: 密文过短")
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// NacosCsDecryptStored 将库中 content 与 encrypted_data_key 还原为明文（控制台/管理端展示）。
// 支持密钥轮换：依次尝试主密钥与 nacos_cs_encryption_key_previous（可多行）。
func NacosCsDecryptStored(content, encKeyField string) (string, error) {
	if strings.TrimSpace(encKeyField) == "" {
		return content, nil
	}
	materials := nacosCsKeyMaterialListDecryptOrder()
	if len(materials) == 0 {
		return "", errors.New("未设置 NACOS_CS_ENCRYPTION_KEY / nacos_cs_encryption_key，无法解密")
	}
	var lastErr error
	for _, mat := range materials {
		key, err := nacosCsAES256KeyFromMaterial(mat)
		if err != nil {
			lastErr = err
			continue
		}
		plain, err := nacosCsDecryptWithKey(key, content, encKeyField)
		if err == nil {
			return plain, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", fmt.Errorf("解密失败（已尝试主密钥与轮换密钥）: %w", lastErr)
	}
	return "", errors.New("解密失败")
}

// nacosCsEncryptPlaintext 将明文加密为落库 content 与 encrypted_data_key 字段值（仅使用主密钥）。
func nacosCsEncryptPlaintext(plain string) (content string, encKeyField string, err error) {
	key, err := nacosCsAES256KeyFromMaterial(config.NacosCsEncryptionKey)
	if err != nil {
		return "", "", errors.New("未设置 NACOS_CS_ENCRYPTION_KEY，无法读写 cipher-aes- 配置")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nacosCsEncEnvelope, nil
}

// NacosCsPreparePublishPayload 根据 dataId（cipher-aes-*）决定落库明文或 AES-GCM 密文。
func NacosCsPreparePublishPayload(dataID, plaintext string) (storedContent, encryptedDataKey string, err error) {
	alg, err := NacosCsEncryptionAlg(dataID)
	if err != nil {
		return "", "", err
	}
	if alg == "aes" {
		if !NacosCsEncryptionKeyConfigured() {
			return "", "", errors.New("未设置 NACOS_CS_ENCRYPTION_KEY，无法发布 cipher-aes- 配置")
		}
		return nacosCsEncryptPlaintext(plaintext)
	}
	return plaintext, "", nil
}
