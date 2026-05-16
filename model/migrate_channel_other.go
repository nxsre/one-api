package model

import (
	"encoding/json"
	"fmt"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

// migrateChannelMergeOtherIntoConfig 将旧列 other 合并进 config JSON（与 middleware 中原兼容逻辑一致），便于随后删除 other 列。
func migrateChannelMergeOtherIntoConfig() error {
	ok, err := channelsTableHasOtherColumn()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	type row struct {
		ID     int
		Type   int
		Other  *string
		Config string
	}
	q := `SELECT id, type, other, COALESCE(config, '') AS config FROM channels WHERE other IS NOT NULL AND other != ''`
	if !common.UsingPostgreSQL {
		q = `SELECT id, type, other, IFNULL(config, '') AS config FROM channels WHERE other IS NOT NULL AND other != ''`
	}
	var rows []row
	if err := DB.Raw(q).Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		if r.Other == nil || *r.Other == "" {
			continue
		}
		var cfg ChannelConfig
		if r.Config != "" {
			if err := json.Unmarshal([]byte(r.Config), &cfg); err != nil {
				logger.SysError(fmt.Sprintf("migrate channel %d: reset invalid config: %v", r.ID, err))
				cfg = ChannelConfig{}
			}
		}
		changed := false
		switch r.Type {
		case channeltype.Azure, channeltype.Xunfei, channeltype.Gemini:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *r.Other
				changed = true
			}
		case channeltype.AIProxyLibrary:
			if cfg.LibraryID == "" {
				cfg.LibraryID = *r.Other
				changed = true
			}
		case channeltype.Ali:
			if cfg.Plugin == "" {
				cfg.Plugin = *r.Other
				changed = true
			}
		}
		if !changed {
			continue
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := DB.Model(&Channel{}).Where("id = ?", r.ID).Update("config", string(b)).Error; err != nil {
			return err
		}
		logger.SysLog(fmt.Sprintf("channels: migrated legacy other -> config for id=%d type=%d", r.ID, r.Type))
	}
	return nil
}

func channelsTableHasOtherColumn() (bool, error) {
	var n int64
	switch {
	case common.UsingSQLite:
		if err := DB.Raw(`SELECT COUNT(*) FROM pragma_table_info('channels') WHERE name = 'other'`).Scan(&n).Error; err != nil {
			return false, err
		}
	case common.UsingPostgreSQL:
		if err := DB.Raw(`
                            SELECT COUNT(*) FROM information_schema.columns
                            WHERE table_schema = current_schema() AND table_name = 'channels' AND column_name = 'other'`).Scan(&n).Error; err != nil {
			return false, err
		}
	case common.UsingMySQL:
		if err := DB.Raw(`
                            SELECT COUNT(*) FROM information_schema.COLUMNS
                            WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'channels' AND COLUMN_NAME = 'other'`).Scan(&n).Error; err != nil {
			return false, err
		}
	default:
		return false, nil
	}
	return n > 0, nil
}

// migrateChannelDropOtherColumn 删除 channels.other（模型已不再声明该字段）。
func migrateChannelDropOtherColumn() error {
	ok, err := channelsTableHasOtherColumn()
	if err != nil || !ok {
		return err
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	switch {
	case common.UsingSQLite:
		if _, err := sqlDB.Exec(`ALTER TABLE channels DROP COLUMN other`); err != nil {
			return fmt.Errorf("drop channels.other (sqlite): %w", err)
		}
	case common.UsingPostgreSQL:
		if _, err := sqlDB.Exec(`ALTER TABLE channels DROP COLUMN IF EXISTS other`); err != nil {
			return fmt.Errorf("drop channels.other (postgres): %w", err)
		}
	case common.UsingMySQL:
		if _, err := sqlDB.Exec(`ALTER TABLE channels DROP COLUMN other`); err != nil {
			return fmt.Errorf("drop channels.other (mysql): %w", err)
		}
	default:
		return nil
	}
	logger.SysLog("channels: dropped legacy column other")
	return nil
}
