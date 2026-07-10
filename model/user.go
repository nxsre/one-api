package model

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/blacklist"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
)

const (
	RoleGuestUser  = 0
	RoleCommonUser = 1
	// RoleTenantAdmin 租户管理员：仅能使用租户控制台 API，管理本子账号与租户渠道；不可访问平台级 /api/channel 等。
	RoleTenantAdmin = 20
	RoleAdminUser   = 10
	RoleSuperAdmin  = 50
	RoleRootUser    = 100
)

const (
	UserStatusEnabled  = 1 // don't use 0, 0 is the default value!
	UserStatusDisabled = 2 // also don't use 0
	UserStatusDeleted  = 3
)

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id               int    `json:"-" gorm:"primaryKey"`
	Uid              string `json:"user_id" gorm:"column:uid;size:26" validate:"omitempty,len=26"`
	Username         string `json:"username" gorm:"unique;index" validate:"max=12"`
	Password         string `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	DisplayName      string `json:"display_name" gorm:"index" validate:"max=20"`
	Role             int    `json:"role" gorm:"type:int;default:1"`   // admin, util
	Status           int    `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email            string `json:"email" gorm:"index" validate:"max=50"`
	GitHubId         string `json:"github_id" gorm:"column:github_id;index"`
	WeChatId         string `json:"wechat_id" gorm:"column:wechat_id;index"`
	LarkId           string `json:"lark_id" gorm:"column:lark_id;index"`
	OidcId           string `json:"oidc_id" gorm:"column:oidc_id;index"`
	VerificationCode string `json:"verification_code" gorm:"-:all"`                                    // this field is only for Email verification, don't save it to database!
	AccessToken      string `json:"access_token" gorm:"type:char(32);column:access_token;uniqueIndex"` // this token is for system management
	Quota            int64  `json:"quota" gorm:"bigint;default:0"`
	UsedQuota        int64  `json:"used_quota" gorm:"bigint;default:0;column:used_quota"` // used quota
	RequestCount     int    `json:"request_count" gorm:"type:int;default:0;"`             // request number
	Group            string `json:"group" gorm:"type:varchar(32);default:'default'"`
	AffCode          string `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	InviterId        int    `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	// TenantID 非空表示属于某租户（子账号或租户管理员）；平台管理员与 Root 应为 NULL。
	TenantID *int `json:"tenant_id,omitempty" gorm:"column:tenant_id;index"`
	// AllowedModels 租户子账号可用模型白名单；空表示不额外限制（仍受分组 abilities、令牌 models 约束）。
	AllowedModels []string `json:"allowed_models,omitempty" gorm:"column:allowed_models;type:text;serializer:json"`
	// AllowedChannelIDs 租户子账号可用渠道 ID 白名单；空表示不额外限制。
	AllowedChannelIDs []int `json:"allowed_channel_ids,omitempty" gorm:"column:allowed_channel_ids;type:text;serializer:json"`
	// TenantPermissions 租户内具体管理权限（例如：manage_users, manage_channels 等），授权普通子账号拥有租户控制台部分管理能力。
	TenantPermissions []string `json:"tenant_permissions,omitempty" gorm:"column:tenant_permissions;type:text;serializer:json"`
	// Tags 用户标签，用于运营分群（例如支付折扣按标签匹配）。
	Tags []string `json:"tags,omitempty" gorm:"column:tags;type:text;serializer:json"`
	// AgentClientPolicy 用户级 agent 客户端策略（白名单 + 可选限流）；空表示放开所有类型。
	AgentClientPolicy *agentpolicy.Policy `json:"agent_client_policy,omitempty" gorm:"column:agent_client_policy;type:text;serializer:json"`
	S3Enabled         bool                `json:"s3_enabled" gorm:"default:false;column:s3_enabled"`
	S3AccessKey       *string             `json:"s3_access_key,omitempty" gorm:"size:64;uniqueIndex;column:s3_access_key"`
	S3SecretKey       *string             `json:"-" gorm:"size:128;column:s3_secret_key"`
}

// UserListItem 管理端用户列表/搜索 API 返回字段（不含 access_token、OAuth ID、S3 密钥等）。
type UserListItem struct {
	Uid          string `json:"user_id" gorm:"column:uid"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         int    `json:"role"`
	Status       int    `json:"status"`
	Email        string `json:"email"`
	Quota        int64  `json:"quota"`
	UsedQuota    int64  `json:"used_quota" gorm:"column:used_quota"`
	RequestCount int    `json:"request_count" gorm:"column:request_count"`
	Group        string `json:"group"`
	TenantID     *int   `json:"tenant_id,omitempty" gorm:"column:tenant_id"`
}

// UserManageResult 用户管理操作（启用/禁用/升降级等）后的最小返回。
type UserManageResult struct {
	Role   int `json:"role"`
	Status int `json:"status"`
}

var userListSelectColumns = []string{
	"uid", "username", "display_name", "role", "status", "email",
	"quota", "used_quota", "request_count", "group", "tenant_id",
}

func GetMaxUserId() int {
	var user User
	DB.Last(&user)
	return user.Id
}

type UserQueryOptions struct {
	Page     int
	PageSize int
	Order    string
	Scope    string // "independent", "tenant", "all"
	TenantID *int
	Roles    []int
	Status   *int
	Keyword  string
}

func ListUsersPaged(opts UserQueryOptions) (items []UserListItem, total int64, err error) {
	q := DB.Model(&User{}).Where("status != ?", UserStatusDeleted)

	if opts.Scope == "independent" {
		q = q.Where("tenant_id IS NULL")
	} else if opts.Scope == "tenant" {
		if opts.TenantID != nil {
			q = q.Where("tenant_id = ?", *opts.TenantID)
		} else {
			q = q.Where("tenant_id IS NOT NULL")
		}
	}

	if len(opts.Roles) > 0 {
		q = q.Where("role IN ?", opts.Roles)
	}
	if opts.Status != nil {
		q = q.Where("status = ?", *opts.Status)
	}

	kw := strings.TrimSpace(opts.Keyword)
	if kw != "" {
		pattern := kw + "%"
		if _, perr := ulid.Parse(kw); perr == nil {
			q = q.Where("(uid = ? OR username LIKE ? OR email LIKE ? OR display_name LIKE ?)", kw, pattern, pattern, pattern)
		} else {
			q = q.Where("(username LIKE ? OR email LIKE ? OR display_name LIKE ?)", pattern, pattern, pattern)
		}
	}

	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	switch opts.Order {
	case "quota":
		q = q.Order("quota desc")
	case "used_quota":
		q = q.Order("used_quota desc")
	case "request_count":
		q = q.Order("request_count desc")
	default:
		q = q.Order("id desc")
	}

	if opts.PageSize > 0 {
		q = q.Limit(opts.PageSize).Offset(opts.Page * opts.PageSize)
	}

	err = q.Select(userListSelectColumns).Find(&items).Error
	return items, total, err
}

func GetAllUsers(startIdx int, num int, order string) (users []UserListItem, err error) {
	query := DB.Model(&User{}).
		Select(userListSelectColumns).
		Limit(num).
		Offset(startIdx).
		Where("status != ?", UserStatusDeleted)

	switch order {
	case "quota":
		query = query.Order("quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	case "request_count":
		query = query.Order("request_count desc")
	default:
		query = query.Order("id desc")
	}

	err = query.Find(&users).Error
	return users, err
}

// GetTenantSubUsers 租户控制台：本子账号列表（普通用户角色）。
func GetTenantSubUsers(tenantID int, startIdx, num int, order string) (users []UserListItem, err error) {
	if tenantID <= 0 {
		return nil, errors.New("tenant id 无效")
	}
	query := DB.Model(&User{}).
		Select(userListSelectColumns).
		Limit(num).
		Offset(startIdx).
		Where("status != ?", UserStatusDeleted).
		Where("tenant_id = ?", tenantID).
		Where("(role = ? OR role = ?)", RoleCommonUser, RoleTenantAdmin)
	switch order {
	case "quota":
		query = query.Order("quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	case "request_count":
		query = query.Order("request_count desc")
	default:
		query = query.Order("id desc")
	}
	err = query.Find(&users).Error
	return users, err
}

// SearchTenantSubUsers 在租户内搜索子账号（keyword 为空返回空列表）。
func SearchTenantSubUsers(tenantID int, keyword string) (users []UserListItem, err error) {
	if tenantID <= 0 {
		return nil, errors.New("tenant id 无效")
	}
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return []UserListItem{}, nil
	}
	pattern := kw + "%"
	q := DB.Model(&User{}).Select(userListSelectColumns).
		Where("status != ?", UserStatusDeleted).
		Where("tenant_id = ?", tenantID).
		Where("(role = ? OR role = ?)", RoleCommonUser, RoleTenantAdmin)
	if _, perr := ulid.Parse(kw); perr == nil {
		err = q.Where("uid = ? OR username LIKE ? OR email LIKE ? OR display_name LIKE ?",
			kw, pattern, pattern, pattern).Find(&users).Error
		return users, err
	}
	err = q.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ?",
		pattern, pattern, pattern).Find(&users).Error
	return users, err
}

func SearchUsers(keyword string) (users []UserListItem, err error) {
	kw := strings.TrimSpace(keyword)
	pattern := kw + "%"
	q := DB.Model(&User{}).Select(userListSelectColumns).Where("status != ?", UserStatusDeleted)
	if kw != "" {
		if _, perr := ulid.Parse(kw); perr == nil {
			err = q.Where("uid = ? OR username LIKE ? OR email LIKE ? OR display_name LIKE ?",
				kw, pattern, pattern, pattern).Find(&users).Error
			return users, err
		}
	}
	err = q.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ?",
		pattern, pattern, pattern).Find(&users).Error
	return users, err
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	user := User{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&user, "id = ?", id).Error
	} else {
		err = DB.Omit("password", "access_token", "S3SecretKey").First(&user, "id = ?", id).Error
	}
	return &user, err
}

func GetUserIdByAffCode(affCode string) (int, error) {
	if affCode == "" {
		return 0, errors.New("affCode 为空！")
	}
	var user User
	err := DB.Select("id").First(&user, "aff_code = ?", affCode).Error
	return user.Id, err
}

func DeleteUserById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	user := User{Id: id}
	return user.Delete()
}

func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	if user.Id == 0 {
		user.Id = common.GetNextId()
	}
	return nil
}

func (user *User) Insert(ctx context.Context, inviterId int) error {
	var err error
	if user.Uid == "" {
		user.Uid = NewUserPublicID()
	}
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = config.QuotaForNewUser
	user.AccessToken = random.GetUUID()
	user.AffCode = random.GetRandomString(4)
	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}
	if config.QuotaForNewUser > 0 {
		RecordLog(ctx, user.Id, LogTypeSystem, fmt.Sprintf("新用户注册赠送 %s", common.LogQuota(config.QuotaForNewUser)))
	}
	if inviterId != 0 {
		if config.QuotaForInvitee > 0 {
			_ = IncreaseUserQuota(user.Id, config.QuotaForInvitee)
			RecordLog(ctx, user.Id, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", common.LogQuota(config.QuotaForInvitee)))
		}
		if config.QuotaForInviter > 0 {
			_ = IncreaseUserQuota(inviterId, config.QuotaForInviter)
			RecordLog(ctx, inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", common.LogQuota(config.QuotaForInviter)))
		}
	}
	// create default token
	cleanToken := Token{
		UserId:         user.Id,
		Name:           "default",
		Key:            random.GenerateKey(),
		CreatedTime:    helper.GetTimestamp(),
		AccessedTime:   helper.GetTimestamp(),
		ExpiredTime:    -1,
		RemainQuota:    -1,
		UnlimitedQuota: true,
	}
	result.Error = cleanToken.Insert()
	if result.Error != nil {
		// do not block
		logger.SysError(fmt.Sprintf("create default token for user %d failed: %s", user.Id, result.Error.Error()))
	}
	return nil
}

func (user *User) Update(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	if user.Status == UserStatusDisabled {
		blacklist.BanUser(user.Id)
	} else if user.Status == UserStatusEnabled {
		blacklist.UnbanUser(user.Id)
	}
	err = DB.Model(user).Updates(user).Error
	if err == nil {
		CacheInvalidateUserGroup(user.Id)
	}
	return err
}

func (user *User) Delete() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	blacklist.BanUser(user.Id)
	user.Username = fmt.Sprintf("deleted_%s", random.GetUUID())
	user.Status = UserStatusDeleted
	err := DB.Model(user).Updates(user).Error
	return err
}

// LoginCredentialFailureReason 登录校验失败原因（用于服务端审计日志，客户端仍统一提示模糊文案）。
type LoginCredentialFailureReason string

const (
	LoginFailUserNotFound LoginCredentialFailureReason = "user_not_found"
	LoginFailBadPassword  LoginCredentialFailureReason = "bad_password"
	LoginFailDisabled     LoginCredentialFailureReason = "account_disabled"
)

// LoginCredentialError 凭证校验失败（用户名/邮箱不存在、密码错误或账号未启用）。
type LoginCredentialError struct {
	Reason LoginCredentialFailureReason
}

func (e *LoginCredentialError) Error() string {
	return "用户名或密码错误，或用户已被封禁"
}

// ValidateAndFill check password & user status
func (user *User) ValidateAndFill() (err error) {
	// When querying with struct, GORM will only query with non-zero fields,
	// that means if your field’s value is 0, '', false or other zero values,
	// it won’t be used to build query conditions
	password := user.Password
	if user.Username == "" || password == "" {
		return errors.New("用户名或密码为空")
	}
	err = DB.Where("username = ?", user.Username).First(user).Error
	if err != nil {
		// we must make sure check username firstly
		// consider this case: a malicious user set his username as other's email
		err := DB.Where("email = ?", user.Username).First(user).Error
		if err != nil {
			return &LoginCredentialError{Reason: LoginFailUserNotFound}
		}
	}
	okay := common.ValidatePasswordAndHash(password, user.Password)
	if !okay {
		return &LoginCredentialError{Reason: LoginFailBadPassword}
	}
	if user.Status != UserStatusEnabled {
		return &LoginCredentialError{Reason: LoginFailDisabled}
	}
	return nil
}

func (user *User) FillUserById() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	DB.Where(User{Id: user.Id}).First(user)
	return nil
}

func (user *User) FillUserByEmail() error {
	if user.Email == "" {
		return errors.New("email 为空！")
	}
	DB.Where(User{Email: user.Email}).First(user)
	return nil
}

func (user *User) FillUserByGitHubId() error {
	if user.GitHubId == "" {
		return errors.New("GitHub id 为空！")
	}
	DB.Where(User{GitHubId: user.GitHubId}).First(user)
	return nil
}

func (user *User) FillUserByLarkId() error {
	if user.LarkId == "" {
		return errors.New("lark id 为空！")
	}
	DB.Where(User{LarkId: user.LarkId}).First(user)
	return nil
}

func (user *User) FillUserByOidcId() error {
	if user.OidcId == "" {
		return errors.New("oidc id 为空！")
	}
	DB.Where(User{OidcId: user.OidcId}).First(user)
	return nil
}

func (user *User) FillUserByWeChatId() error {
	if user.WeChatId == "" {
		return errors.New("WeChat id 为空！")
	}
	DB.Where(User{WeChatId: user.WeChatId}).First(user)
	return nil
}

func (user *User) FillUserByUsername() error {
	if user.Username == "" {
		return errors.New("username 为空！")
	}
	DB.Where(User{Username: user.Username}).First(user)
	return nil
}

func IsEmailAlreadyTaken(email string) bool {
	return DB.Where("email = ?", email).Find(&User{}).RowsAffected == 1
}

func IsWeChatIdAlreadyTaken(wechatId string) bool {
	return DB.Where("wechat_id = ?", wechatId).Find(&User{}).RowsAffected == 1
}

func IsGitHubIdAlreadyTaken(githubId string) bool {
	return DB.Where("github_id = ?", githubId).Find(&User{}).RowsAffected == 1
}

func IsLarkIdAlreadyTaken(githubId string) bool {
	return DB.Where("lark_id = ?", githubId).Find(&User{}).RowsAffected == 1
}

func IsOidcIdAlreadyTaken(oidcId string) bool {
	return DB.Where("oidc_id = ?", oidcId).Find(&User{}).RowsAffected == 1
}

func IsUsernameAlreadyTaken(username string) bool {
	return DB.Where("username = ?", username).Find(&User{}).RowsAffected == 1
}

func ResetUserPasswordByEmail(email string, password string) error {
	if email == "" || password == "" {
		return errors.New("邮箱地址或密码为空！")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	err = DB.Model(&User{}).Where("email = ?", email).Update("password", hashedPassword).Error
	return err
}

func IsAdmin(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role", "tenant_id").Find(&user).Error
	if err != nil {
		logger.SysError("no such user " + err.Error())
		return false
	}
	if user.Role < RoleAdminUser {
		return false
	}
	// 租户管理员不走「平台管理员」能力（例如 sk-key-channelId 指定渠道）。
	if user.TenantID != nil {
		return false
	}
	return true
}

// IsTenantConsoleAdmin 是否为租户管理员（可登录租户控制台）。
func IsTenantConsoleAdmin(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role", "tenant_id").Find(&user).Error
	if err != nil {
		return false
	}
	return user.Role == RoleTenantAdmin && user.TenantID != nil
}

// IsPlatformConsoleOperator 平台控制台操作者（Root / 平台管理员且非租户账号）。
func IsPlatformConsoleOperator(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role", "tenant_id").Find(&user).Error
	if err != nil {
		return false
	}
	if user.Role >= RoleRootUser {
		return true
	}
	if user.Role >= RoleAdminUser && user.TenantID == nil {
		return true
	}
	return false
}

// UserHasTenantPermission 子账号是否具备指定租户委派权限（租户管理员请在外层单独判断）。
func UserHasTenantPermission(u *User, perm string) bool {
	if u == nil || perm == "" {
		return false
	}
	for _, p := range u.TenantPermissions {
		if p == perm {
			return true
		}
	}
	return false
}

// TenantConsoleAccessTenantID 返回有权访问租户控制台的用户（租户管理员，或具有具体管理权限的租户子账号）所属的租户 ID；否则返回 0。
func TenantConsoleAccessTenantID(userId int) int {
	if userId <= 0 {
		return 0
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role", "tenant_id", "tenant_permissions").Find(&user).Error
	if err != nil || user.TenantID == nil {
		return 0
	}
	if user.Role == RoleTenantAdmin || (user.Role == RoleCommonUser && len(user.TenantPermissions) > 0) {
		return *user.TenantID
	}
	return 0
}

// GetUserTenantIDNumeric 返回用户所属租户，平台用户为 0。
func GetUserTenantIDNumeric(userId int) int {
	if userId == 0 {
		return 0
	}
	var user User
	err := DB.Where("id = ?", userId).Select("tenant_id").Find(&user).Error
	if err != nil || user.TenantID == nil {
		return 0
	}
	return *user.TenantID
}

// LoadTenantSubRelayRestrictions 若为租户子账号则 ok 为 true，并返回该账号配置的模型/渠道白名单（切片为空表示该维度不限制）。
func LoadTenantSubRelayRestrictions(userId int) (allowedModels []string, allowedChannelIDs []int, ok bool, err error) {
	if userId == 0 {
		return nil, nil, false, nil
	}
	var u User
	err = DB.Where("id = ?", userId).Select("role", "tenant_id", "allowed_models", "allowed_channel_ids").First(&u).Error
	if err != nil {
		return nil, nil, false, err
	}
	if u.TenantID == nil || u.Role != RoleCommonUser {
		return nil, nil, false, nil
	}
	return u.AllowedModels, u.AllowedChannelIDs, true, nil
}

func IsUserEnabled(userId int) (bool, error) {
	if userId == 0 {
		return false, errors.New("user id is empty")
	}
	var user User
	err := DB.Where("id = ?", userId).Select("status").Find(&user).Error
	if err != nil {
		return false, err
	}
	return user.Status == UserStatusEnabled, nil
}

func ValidateAccessToken(token string) (user *User) {
	if token == "" {
		return nil
	}
	token = strings.Replace(token, "Bearer ", "", 1)
	user = &User{}
	if DB.Where("access_token = ?", token).First(user).RowsAffected == 1 {
		return user
	}
	return nil
}

func GetUserQuota(id int) (quota int64, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("quota").Find(&quota).Error
	return quota, err
}

func GetUserUsedQuota(id int) (quota int64, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("used_quota").Find(&quota).Error
	return quota, err
}

func GetUserEmail(id int) (email string, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("email").Find(&email).Error
	return email, err
}

func GetUserGroup(id int) (group string, err error) {
	groupCol := "`group`"
	if common.UsingPostgreSQL {
		groupCol = `"group"`
	}

	err = DB.Model(&User{}).Where("id = ?", id).Select(groupCol).Find(&group).Error
	return group, err
}

func IncreaseUserQuota(id int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, quota)
		return nil
	}
	return increaseUserQuota(id, quota)
}

func increaseUserQuota(id int, quota int64) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota + ?", quota)).Error
	return err
}

func DecreaseUserQuota(id int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, -quota)
		return nil
	}
	return decreaseUserQuota(id, quota)
}

func decreaseUserQuota(id int, quota int64) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota - ?", quota)).Error
	return err
}

func GetRootUserEmail() (email string) {
	DB.Model(&User{}).Where("role = ?", RoleRootUser).Select("email").Find(&email)
	return email
}

func UpdateUserUsedQuotaAndRequestCount(id int, quota int64) {
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUsedQuota, id, quota)
		addNewRecord(BatchUpdateTypeRequestCount, id, 1)
		return
	}
	updateUserUsedQuotaAndRequestCount(id, quota, 1)
}

func updateUserUsedQuotaAndRequestCount(id int, quota int64, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"request_count": gorm.Expr("request_count + ?", count),
		},
	).Error
	if err != nil {
		logger.SysError("failed to update user used quota and request count: " + err.Error())
	}
}

func GetUsernameById(id int) (username string) {
	// 这是日志/错误路径上的尽力而为查询；DB 未就绪时返回空串而非 panic，
	// 避免错误处理器自身崩溃。
	if DB == nil {
		return ""
	}
	DB.Model(&User{}).Where("id = ?", id).Select("username").Find(&username)
	return username
}

// GetUsernamesByIds 批量查询用户 id -> username 映射，用于列表接口回填创建人，避免 N+1 查询。
func GetUsernamesByIds(ids []int) map[int]string {
	result := make(map[int]string)
	if DB == nil || len(ids) == 0 {
		return result
	}
	var rows []struct {
		Id       int
		Username string
	}
	DB.Model(&User{}).Where("id IN ?", ids).Select("id", "username").Find(&rows)
	for _, row := range rows {
		result[row.Id] = row.Username
	}
	return result
}

// NewUserPublicID 生成 26 字符 ULID，作为对外用户标识（users.uid）。
func NewUserPublicID() string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// GetUserByPublicID 按对外用户 ID（uid / ULID）查询。
func GetUserByPublicID(uid string) (*User, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, errors.New("用户ID 为空")
	}
	user := User{}
	err := DB.Omit("password", "access_token", "S3SecretKey").Where("uid = ?", uid).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ParseUserRouteParam 将路由/API 中的对外用户 ID（ULID）解析为内部主键。
func ParseUserRouteParam(idStr string) (int, error) {
	idStr = strings.TrimSpace(idStr)
	if idStr == "" {
		return 0, errors.New("用户ID 为空")
	}
	if _, err := ulid.Parse(idStr); err != nil {
		return 0, errors.New("无效用户 ID")
	}
	u, err := GetUserByPublicID(idStr)
	if err != nil {
		return 0, err
	}
	return u.Id, nil
}
