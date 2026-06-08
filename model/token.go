package model

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/message"
)

const (
	TokenStatusEnabled   = 1 // don't use 0, 0 is the default value!
	TokenStatusDisabled  = 2 // also don't use 0
	TokenStatusExpired   = 3
	TokenStatusExhausted = 4
)

type Token struct {
	Id             int     `json:"id"`
	UserId         int     `json:"user_id"`
	Key            string  `json:"key" gorm:"type:char(48);uniqueIndex"`
	Status         int     `json:"status" gorm:"default:1"`
	Name           string  `json:"name" gorm:"index" `
	CreatedTime    int64   `json:"created_time" gorm:"bigint"`
	AccessedTime   int64   `json:"accessed_time" gorm:"bigint"`
	ExpiredTime    int64   `json:"expired_time" gorm:"bigint;default:-1"` // -1 means never expired
	RemainQuota    int64   `json:"remain_quota" gorm:"bigint;default:0"`
	UnlimitedQuota bool    `json:"unlimited_quota" gorm:"default:false"`
	UsedQuota      int64   `json:"used_quota" gorm:"bigint;default:0"` // used quota
	Models         *string `json:"models" gorm:"type:text"`            // allowed models
	// Group 非空时中继按该分组选路，覆盖用户默认分组（仍须与渠道 abilities 中的分组匹配）。
	Group  *string `json:"group" gorm:"size:64;column:token_group"`
	Subnet *string `json:"subnet" gorm:"default:''"` // allowed subnet
	// AgentClientPolicy 令牌级 agent 客户端策略（允许的客户端类型白名单 + 可选限流）；空表示放开所有类型。
	AgentClientPolicy *agentpolicy.Policy `json:"agent_client_policy,omitempty" gorm:"column:agent_client_policy;type:text;serializer:json"`
	// Username 为创建人（令牌所属用户）的用户名，非数据库字段，仅在列表接口中按需回填供前端展示。
	Username string `json:"username" gorm:"-"`
}

func GetAllUserTokens(userId int, startIdx int, num int, order string) ([]*Token, error) {
	var tokens []*Token
	var err error
	query := DB.Where("user_id = ?", userId)

	switch order {
	case "remain_quota":
		query = query.Order("unlimited_quota desc, remain_quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	default:
		query = query.Order("id desc")
	}

	err = query.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, err
}

func SearchUserTokens(userId int, keyword string) (tokens []*Token, err error) {
	err = DB.Where("user_id = ?", userId).Where("name LIKE ?", keyword+"%").Find(&tokens).Error
	return tokens, err
}

// GetAllTokensForAdmin 返回全部用户的令牌（不按 user_id 过滤），供管理员查看与管理。
func GetAllTokensForAdmin(startIdx int, num int, order string) ([]*Token, error) {
	var tokens []*Token
	query := DB

	switch order {
	case "remain_quota":
		query = query.Order("unlimited_quota desc, remain_quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	default:
		query = query.Order("id desc")
	}

	err := query.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, err
}

// SearchAllTokens 在全部用户的令牌范围内按名称搜索，供管理员使用。
func SearchAllTokens(keyword string) (tokens []*Token, err error) {
	err = DB.Where("name LIKE ?", keyword+"%").Find(&tokens).Error
	return tokens, err
}

// GetTenantTokens 返回某租户下全部成员的令牌（按 user_id 属于该租户过滤），供租户管理员查看与管理。
func GetTenantTokens(tenantID int, startIdx int, num int, order string) ([]*Token, error) {
	var tokens []*Token
	memberIDs := DB.Model(&User{}).Select("id").Where("tenant_id = ?", tenantID)
	query := DB.Where("user_id IN (?)", memberIDs)

	switch order {
	case "remain_quota":
		query = query.Order("unlimited_quota desc, remain_quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	default:
		query = query.Order("id desc")
	}

	err := query.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, err
}

// SearchTenantTokens 在某租户全部成员的令牌范围内按名称搜索，供租户管理员使用。
func SearchTenantTokens(tenantID int, keyword string) (tokens []*Token, err error) {
	memberIDs := DB.Model(&User{}).Select("id").Where("tenant_id = ?", tenantID)
	err = DB.Where("user_id IN (?)", memberIDs).Where("name LIKE ?", keyword+"%").Find(&tokens).Error
	return tokens, err
}

func ValidateUserToken(key string) (token *Token, err error) {
	if key == "" {
		return nil, errors.New("未提供令牌")
	}
	token, err = CacheGetTokenByKey(key)
	if err != nil {
		logger.SysError("CacheGetTokenByKey failed: " + err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的令牌")
		}
		return nil, errors.New("令牌验证失败")
	}
	if token.Status == TokenStatusExhausted {
		return nil, fmt.Errorf("令牌 %s（#%d）额度已用尽", token.Name, token.Id)
	} else if token.Status == TokenStatusExpired {
		return nil, errors.New("该令牌已过期")
	}
	if token.Status != TokenStatusEnabled {
		return nil, errors.New("该令牌状态不可用")
	}
	if token.ExpiredTime != -1 && token.ExpiredTime < helper.GetTimestamp() {
		if !common.RedisEnabled {
			token.Status = TokenStatusExpired
			err := token.SelectUpdate()
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, errors.New("该令牌已过期")
	}
	if !token.UnlimitedQuota && token.RemainQuota <= 0 {
		if !common.RedisEnabled {
			// in this case, we can make sure the token is exhausted
			token.Status = TokenStatusExhausted
			err := token.SelectUpdate()
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, errors.New("该令牌额度已用尽")
	}
	return token, nil
}

func GetTokenByIds(id int, userId int) (*Token, error) {
	if id == 0 || userId == 0 {
		return nil, errors.New("id 或 userId 为空！")
	}
	token := Token{Id: id, UserId: userId}
	var err error = nil
	err = DB.First(&token, "id = ? and user_id = ?", id, userId).Error
	return &token, err
}

func GetTokenById(id int) (*Token, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	token := Token{Id: id}
	var err error = nil
	err = DB.First(&token, "id = ?", id).Error
	return &token, err
}

func (t *Token) Insert() error {
	var err error
	err = DB.Create(t).Error
	return err
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (t *Token) Update() error {
	var err error
	err = DB.Model(t).Select("name", "status", "expired_time", "remain_quota", "unlimited_quota", "models", "subnet", "token_group", "agent_client_policy").Updates(t).Error
	if err == nil {
		CacheInvalidateTokenByKey(t.Key)
	}
	return err
}

func (t *Token) SelectUpdate() error {
	// This can update zero values
	err := DB.Model(t).Select("accessed_time", "status").Updates(t).Error
	if err == nil {
		CacheInvalidateTokenByKey(t.Key)
	}
	return err
}

func (t *Token) Delete() error {
	var err error
	err = DB.Delete(t).Error
	if err == nil {
		CacheInvalidateTokenByKey(t.Key)
	}
	return err
}

func (t *Token) GetModels() string {
	if t == nil {
		return ""
	}
	if t.Models == nil {
		return ""
	}
	return *t.Models
}

func DeleteTokenById(id int, userId int) (err error) {
	// Why we need userId here? In case user want to delete other's token.
	if id == 0 || userId == 0 {
		return errors.New("id 或 userId 为空！")
	}
	token := Token{Id: id, UserId: userId}
	err = DB.Where(token).First(&token).Error
	if err != nil {
		return err
	}
	return token.Delete()
}

// DeleteTokenByIdOnly 仅按 id 删除令牌，不校验所属用户，供管理员删除任意用户的令牌。
func DeleteTokenByIdOnly(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	token := Token{Id: id}
	err = DB.Where("id = ?", id).First(&token).Error
	if err != nil {
		return err
	}
	return token.Delete()
}

func IncreaseTokenQuota(id int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeTokenQuota, id, quota)
		return nil
	}
	return increaseTokenQuota(id, quota)
}

func increaseTokenQuota(id int, quota int64) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota + ?", quota),
			"used_quota":    gorm.Expr("used_quota - ?", quota),
			"accessed_time": helper.GetTimestamp(),
		},
	).Error
	if err == nil {
		CacheInvalidateTokenById(id)
	}
	return err
}

func DecreaseTokenQuota(id int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeTokenQuota, id, -quota)
		return nil
	}
	return decreaseTokenQuota(id, quota)
}

func decreaseTokenQuota(id int, quota int64) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota - ?", quota),
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"accessed_time": helper.GetTimestamp(),
		},
	).Error
	if err == nil {
		CacheInvalidateTokenById(id)
	}
	return err
}

func PreConsumeTokenQuota(tokenId int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	if !token.UnlimitedQuota && token.RemainQuota < quota {
		return errors.New("令牌额度不足")
	}
	// 用户额度走 Redis 缓存读（relay 调用方此前已 CacheGetUserQuota 预热），
	// 省掉这里的冗余 DB SELECT；Redis 未启用时 CacheGetUserQuota 自动回退到 DB，行为不变。
	userQuota, err := CacheGetUserQuota(context.Background(), token.UserId)
	if err != nil {
		return err
	}
	if userQuota < quota {
		return errors.New("用户额度不足")
	}
	quotaTooLow := userQuota >= config.QuotaRemindThreshold && userQuota-quota < config.QuotaRemindThreshold
	noMoreQuota := userQuota-quota <= 0
	if quotaTooLow || noMoreQuota {
		go func() {
			email, err := GetUserEmail(token.UserId)
			if err != nil {
				logger.SysError("failed to fetch user email: " + err.Error())
			}
			prompt := "额度提醒"
			var contentText string
			if noMoreQuota {
				contentText = "您的额度已用尽"
			} else {
				contentText = "您的额度即将用尽"
			}
			if email != "" {
				topUpLink := fmt.Sprintf("%s/topup", config.ServerAddress)
				content := message.EmailTemplate(
					prompt,
					fmt.Sprintf(`
						<p>您好！</p>
						<p>%s，当前剩余额度为 <strong>%d</strong>。</p>
						<p>为了不影响您的使用，请及时充值。</p>
						<p style="text-align: center; margin: 30px 0;">
							<a href="%s" style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">立即充值</a>
						</p>
						<p style="color: #666;">如果按钮无法点击，请复制以下链接到浏览器中打开：</p>
						<p style="background-color: #f8f8f8; padding: 10px; border-radius: 4px; word-break: break-all;">%s</p>
					`, contentText, userQuota, topUpLink, topUpLink),
				)
				err = message.SendEmail(prompt, email, content)
				if err != nil {
					logger.SysError("failed to send email: " + err.Error())
				}
			}
		}()
	}
	if !token.UnlimitedQuota {
		err = DecreaseTokenQuota(tokenId, quota)
		if err != nil {
			return err
		}
	}
	err = DecreaseUserQuota(token.UserId, quota)
	return err
}

func PostConsumeTokenQuota(tokenId int, quota int64) (err error) {
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	if quota > 0 {
		err = DecreaseUserQuota(token.UserId, quota)
	} else {
		err = IncreaseUserQuota(token.UserId, -quota)
	}
	if !token.UnlimitedQuota {
		if quota > 0 {
			err = DecreaseTokenQuota(tokenId, quota)
		} else {
			err = IncreaseTokenQuota(tokenId, -quota)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
