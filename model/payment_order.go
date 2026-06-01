package model

import (
	"errors"
	"fmt"

	"github.com/songquanpeng/one-api/common/helper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PaymentOrder 在线充值订单。金额以「分」存储；Quota 为支付成功后应入账的额度。
// 状态机：pending -> paid（成功，已入账）/ closed（关闭/超时，不入账）。
type PaymentOrder struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderNo       string `json:"order_no" gorm:"uniqueIndex;size:64;not null"`
	UserId        int    `json:"user_id" gorm:"index;not null"`
	Channel       string `json:"channel" gorm:"size:16;not null"` // wxpay 等
	TradeType     string `json:"trade_type" gorm:"size:16"`       // NATIVE 等
	AmountCents   int    `json:"amount_cents" gorm:"not null"`    // 支付金额（分）
	Quota         int64  `json:"quota" gorm:"not null"`           // 到账额度
	Status        string `json:"status" gorm:"size:16;index;not null;default:'pending'"`
	TransactionId string `json:"transaction_id" gorm:"size:64"`
	CodeUrl       string `json:"code_url" gorm:"size:512"`
	CreatedTime   int64  `json:"created_time" gorm:"bigint"`
	PaidTime      int64  `json:"paid_time" gorm:"bigint"`
	UpdatedTime   int64  `json:"updated_time" gorm:"bigint"`
}

func CreatePaymentOrder(o *PaymentOrder) error {
	now := helper.GetTimestamp()
	o.CreatedTime = now
	o.UpdatedTime = now
	if o.Status == "" {
		o.Status = "pending"
	}
	return DB.Create(o).Error
}

func GetPaymentOrderByNo(orderNo string) (*PaymentOrder, error) {
	var o PaymentOrder
	err := DB.Where("order_no = ?", orderNo).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func UpdatePaymentOrderCodeURL(orderNo, codeURL string) error {
	return DB.Model(&PaymentOrder{}).Where("order_no = ?", orderNo).
		Updates(map[string]any{"code_url": codeURL, "updated_time": helper.GetTimestamp()}).Error
}

// SettlePaymentOrderPaid 幂等地结算一笔已支付订单：在单个事务里把 pending 置为 paid 并给用户加额度。
//   - 已是 paid（重复回调/重复查单）直接返回 credited=false，不重复入账；
//   - paidCents>0 时校验金额一致，不一致拒绝入账（防篡改/错单）。
//
// 返回 credited 表示本次是否真正入账（供调用方记录充值日志）。
func SettlePaymentOrderPaid(orderNo, transactionId string, paidCents int) (credited bool, userId int, quota int64, err error) {
	err = DB.Transaction(func(tx *gorm.DB) error {
		var o PaymentOrder
		e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&o).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return fmt.Errorf("payment order not found: %s", orderNo)
		}
		if e != nil {
			return e
		}
		if o.Status != "pending" {
			// 已结算（paid）或已关闭：幂等返回，不重复入账。
			return nil
		}
		if paidCents > 0 && paidCents != o.AmountCents {
			return fmt.Errorf("payment amount mismatch for %s: paid %d, expected %d", orderNo, paidCents, o.AmountCents)
		}
		now := helper.GetTimestamp()
		res := tx.Model(&PaymentOrder{}).
			Where("id = ? AND status = ?", o.Id, "pending").
			Updates(map[string]any{
				"status":         "paid",
				"transaction_id": transactionId,
				"paid_time":      now,
				"updated_time":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 并发下被其它事务抢先结算，视为幂等成功。
			return nil
		}
		if e := tx.Model(&User{}).Where("id = ?", o.UserId).
			Update("quota", gorm.Expr("quota + ?", o.Quota)).Error; e != nil {
			return e
		}
		credited = true
		userId = o.UserId
		quota = o.Quota
		return nil
	})
	return
}

// ClosePaymentOrder 把仍处于 pending 的订单标记为关闭（超时/取消），不入账。
func ClosePaymentOrder(orderNo string) error {
	return DB.Model(&PaymentOrder{}).
		Where("order_no = ? AND status = ?", orderNo, "pending").
		Updates(map[string]any{"status": "closed", "updated_time": helper.GetTimestamp()}).Error
}
