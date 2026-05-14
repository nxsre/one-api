package model

import (
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
)

const (
	GlobalAccessModeNone      = "none"
	GlobalAccessModeWhitelist = "whitelist"
	GlobalAccessModeBlacklist = "blacklist"
	GlobalAccessEntryTypeIP   = "ip"
	GlobalAccessEntryTypeAPIKey = "api_key"
)

type GlobalAccessWhitelist struct {
	ID        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Type      string `json:"type" gorm:"size:16;index;not null"`
	Value     string `json:"value" gorm:"type:text;not null"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	Remark    string `json:"remark" gorm:"size:512"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (GlobalAccessWhitelist) TableName() string { return "global_access_whitelists" }

type GlobalAccessBlacklist struct {
	ID        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Type      string `json:"type" gorm:"size:16;index;not null"`
	Value     string `json:"value" gorm:"type:text;not null"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	Remark    string `json:"remark" gorm:"size:512"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (GlobalAccessBlacklist) TableName() string { return "global_access_blacklists" }

func GetGlobalAccessMode() string {
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return GlobalAccessModeNone
	}
	m := config.OptionMap["GlobalAccessListMode"]
	if m == "" {
		return GlobalAccessModeNone
	}
	return m
}

var (
	globalAccessListsMu     sync.RWMutex
	globalAccessWhitelist   []*GlobalAccessWhitelist
	globalAccessBlacklist   []*GlobalAccessBlacklist
	globalAccessListsLoaded time.Time
)

const globalAccessListCacheTTL = 30 * time.Second

func LoadGlobalAccessListsCached() (w []*GlobalAccessWhitelist, b []*GlobalAccessBlacklist) {
	globalAccessListsMu.RLock()
	if time.Since(globalAccessListsLoaded) < globalAccessListCacheTTL && globalAccessListsLoaded.Unix() > 0 {
		w, b = globalAccessWhitelist, globalAccessBlacklist
		globalAccessListsMu.RUnlock()
		return
	}
	globalAccessListsMu.RUnlock()

	globalAccessListsMu.Lock()
	defer globalAccessListsMu.Unlock()
	if time.Since(globalAccessListsLoaded) < globalAccessListCacheTTL && globalAccessListsLoaded.Unix() > 0 {
		return globalAccessWhitelist, globalAccessBlacklist
	}
	var wl []*GlobalAccessWhitelist
	var bl []*GlobalAccessBlacklist
	_ = DB.Where("enabled = ?", true).Order("id").Find(&wl).Error
	_ = DB.Where("enabled = ?", true).Order("id").Find(&bl).Error
	globalAccessWhitelist = wl
	globalAccessBlacklist = bl
	globalAccessListsLoaded = time.Now()
	return globalAccessWhitelist, globalAccessBlacklist
}

func InvalidateGlobalAccessListCache() {
	globalAccessListsMu.Lock()
	defer globalAccessListsMu.Unlock()
	globalAccessListsLoaded = time.Time{}
	globalAccessWhitelist = nil
	globalAccessBlacklist = nil
}

func ListGlobalWhitelistsAll() ([]*GlobalAccessWhitelist, error) {
	var list []*GlobalAccessWhitelist
	err := DB.Order("id").Find(&list).Error
	return list, err
}

func ListGlobalBlacklistsAll() ([]*GlobalAccessBlacklist, error) {
	var list []*GlobalAccessBlacklist
	err := DB.Order("id").Find(&list).Error
	return list, err
}

func CreateGlobalWhitelist(e *GlobalAccessWhitelist) error {
	now := time.Now().Unix()
	e.CreatedAt = now
	e.UpdatedAt = now
	err := DB.Create(e).Error
	if err == nil {
		InvalidateGlobalAccessListCache()
	}
	return err
}

func CreateGlobalBlacklist(e *GlobalAccessBlacklist) error {
	now := time.Now().Unix()
	e.CreatedAt = now
	e.UpdatedAt = now
	err := DB.Create(e).Error
	if err == nil {
		InvalidateGlobalAccessListCache()
	}
	return err
}

func DeleteGlobalWhitelist(id int) error {
	err := DB.Delete(&GlobalAccessWhitelist{}, id).Error
	if err == nil {
		InvalidateGlobalAccessListCache()
	}
	return err
}

func DeleteGlobalBlacklist(id int) error {
	err := DB.Delete(&GlobalAccessBlacklist{}, id).Error
	if err == nil {
		InvalidateGlobalAccessListCache()
	}
	return err
}

func UpdateGlobalAccessMode(mode string) error {
	err := UpdateOption("GlobalAccessListMode", mode)
	if err == nil {
		InvalidateGlobalAccessListCache()
	}
	return err
}
