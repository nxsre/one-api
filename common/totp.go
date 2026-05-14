package common

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/songquanpeng/one-api/common/config"
)

const (
	BackupCodeLength = 8
	BackupCodeCount  = 4
	MaxFailAttempts  = 5
	LockoutDuration  = 300
)

func GenerateTOTPSecret(accountName string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      Get2FAIssuer(),
		AccountName: accountName,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
}

func ValidateTOTPCode(secret, code string) bool {
	cleanCode := strings.ReplaceAll(code, " ", "")
	if len(cleanCode) != 6 {
		return false
	}
	return totp.Validate(cleanCode, secret)
}

func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodeCount)
	for i := 0; i < BackupCodeCount; i++ {
		code, err := generateRandomBackupCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

func generateRandomBackupCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, BackupCodeLength)
	for i := range code {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		code[i] = charset[int(b[0])%len(charset)]
	}
	return fmt.Sprintf("%s-%s", string(code[:4]), string(code[4:])), nil
}

func ValidateBackupCode(code string) bool {
	cleanCode := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	if len(cleanCode) != BackupCodeLength {
		return false
	}
	for _, char := range cleanCode {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}

func NormalizeBackupCode(code string) string {
	cleanCode := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	if len(cleanCode) == BackupCodeLength {
		return fmt.Sprintf("%s-%s", cleanCode[:4], cleanCode[4:])
	}
	return code
}

func HashBackupCode(code string) (string, error) {
	return Password2Hash(NormalizeBackupCode(code))
}

func Get2FAIssuer() string {
	if config.SystemName != "" {
		return config.SystemName
	}
	return "One API"
}

func ValidateNumericCode(code string) (string, error) {
	code = strings.ReplaceAll(code, " ", "")
	if len(code) != 6 {
		return "", fmt.Errorf("验证码必须是6位数字")
	}
	if _, err := strconv.Atoi(code); err != nil {
		return "", fmt.Errorf("验证码只能包含数字")
	}
	return code, nil
}

func GenerateQRCodeData(secret, username string) string {
	issuer := Get2FAIssuer()
	accountName := fmt.Sprintf("%s (%s)", username, issuer)
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=6&period=30",
		issuer, accountName, secret, issuer)
}
