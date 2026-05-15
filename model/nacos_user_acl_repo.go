package model

import (
	"errors"

	"gorm.io/gorm"
)

func GetNacosUserACL(userId int) (*NacosUserACL, error) {
	var a NacosUserACL
	err := DB.Where("user_id = ?", userId).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func UpsertNacosUserACL(userId int, rulesJSON string, updatedBy int) error {
	var a NacosUserACL
	err := DB.Where("user_id = ?", userId).First(&a).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		a = NacosUserACL{UserId: userId, RulesJSON: rulesJSON, UpdatedBy: updatedBy}
		return DB.Create(&a).Error
	}
	a.RulesJSON = rulesJSON
	a.UpdatedBy = updatedBy
	return DB.Save(&a).Error
}
