package model

import (
	"errors"

	"github.com/songquanpeng/one-api/common/random"
)

// GetUserByS3AccessKey 按 S3 Access Key 查询已启用 S3 且账号可用的用户（含 Secret，用于 SigV4 校验）。
func GetUserByS3AccessKey(accessKey string) (*User, error) {
	if accessKey == "" {
		return nil, errors.New("empty access key")
	}
	var user User
	err := DB.Where("s3_access_key = ? AND s3_enabled = ?", accessKey, true).
		Where("status = ?", UserStatusEnabled).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func pickS3AccessKey() (string, error) {
	for i := 0; i < 8; i++ {
		cand := random.GetRandomString(24)
		var n int64
		if err := DB.Model(&User{}).Where("s3_access_key = ?", cand).Count(&n).Error; err != nil {
			return "", err
		}
		if n == 0 {
			return cand, nil
		}
	}
	return "", errors.New("could not allocate unique access key")
}

// UserEnableS3 为用户生成 AK/SK 并启用兼容 S3；若已启用则返回错误。
func UserEnableS3(userId int) (accessKey, secretKey string, err error) {
	var u User
	if err = DB.First(&u, "id = ?", userId).Error; err != nil {
		return "", "", err
	}
	if u.S3Enabled && u.S3AccessKey != nil && *u.S3AccessKey != "" &&
		u.S3SecretKey != nil && *u.S3SecretKey != "" {
		return "", "", errors.New("S3 已启用，请先关闭或使用轮换密钥")
	}
	if u.Status != UserStatusEnabled {
		return "", "", errors.New("用户不可用")
	}
	ak, err := pickS3AccessKey()
	if err != nil {
		return "", "", err
	}
	sk := random.GenerateKey()
	err = DB.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"s3_enabled":    true,
		"s3_access_key": ak,
		"s3_secret_key": sk,
	}).Error
	if err != nil {
		return "", "", err
	}
	return ak, sk, nil
}

// UserDisableS3 关闭并清除用户的 S3 密钥。
func UserDisableS3(userId int) error {
	res := DB.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"s3_enabled":    false,
		"s3_access_key": nil,
		"s3_secret_key": nil,
	})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// UserRegenerateS3Secret 在保持 Access Key 不变的前提下换新 Secret；须已启用 S3。
func UserRegenerateS3Secret(userId int) (secretKey string, err error) {
	var u User
	if err = DB.First(&u, "id = ?", userId).Error; err != nil {
		return "", err
	}
	if !u.S3Enabled || u.S3AccessKey == nil || *u.S3AccessKey == "" {
		return "", errors.New("尚未启用 S3")
	}
	if u.Status != UserStatusEnabled {
		return "", errors.New("用户不可用")
	}
	sk := random.GenerateKey()
	err = DB.Model(&User{}).Where("id = ? AND s3_enabled = ?", userId, true).Update("s3_secret_key", sk).Error
	if err != nil {
		return "", err
	}
	return sk, nil
}

// UserS3RotateKeys 生成新的 AK 与 SK（原密钥立即失效）。
func UserS3RotateKeys(userId int) (accessKey, secretKey string, err error) {
	var u User
	if err = DB.First(&u, "id = ?", userId).Error; err != nil {
		return "", "", err
	}
	if !u.S3Enabled {
		return "", "", errors.New("尚未启用 S3")
	}
	if u.Status != UserStatusEnabled {
		return "", "", errors.New("用户不可用")
	}
	ak, err := pickS3AccessKey()
	if err != nil {
		return "", "", err
	}
	sk := random.GenerateKey()
	err = DB.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"s3_access_key": ak,
		"s3_secret_key": sk,
	}).Error
	if err != nil {
		return "", "", err
	}
	return ak, sk, nil
}
