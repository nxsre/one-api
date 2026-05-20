package model

import "github.com/songquanpeng/one-api/common/helper"

const (
	TenantUpgradeStatusPending  = 0
	TenantUpgradeStatusApproved = 1
	TenantUpgradeStatusRejected = 2
)

// TenantUpgradeRequest 普通用户申请成为租户
type TenantUpgradeRequest struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	UserId      int    `json:"user_id" gorm:"index"`
	Name        string `json:"name" gorm:"size:128;not null"`
	Slug        string `json:"slug" gorm:"size:64;uniqueIndex;not null"`
	Status      int    `json:"status" gorm:"type:int;default:0"` // 0: Pending, 1: Approved, 2: Rejected
	Remark      string `json:"remark" gorm:"type:varchar(512);default:''"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func InsertTenantUpgradeRequest(req *TenantUpgradeRequest) error {
	req.CreatedTime = helper.GetTimestamp()
	req.UpdatedTime = req.CreatedTime
	return DB.Create(req).Error
}

func GetTenantUpgradeRequestById(id int) (*TenantUpgradeRequest, error) {
	var req TenantUpgradeRequest
	err := DB.First(&req, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func GetTenantUpgradeRequests(offset int, limit int) ([]*TenantUpgradeRequest, error) {
	var requests []*TenantUpgradeRequest
	err := DB.Order("id desc").Offset(offset).Limit(limit).Find(&requests).Error
	return requests, err
}

func UpdateTenantUpgradeRequestStatus(id int, status int) error {
	return DB.Model(&TenantUpgradeRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       status,
		"updated_time": helper.GetTimestamp(),
	}).Error
}