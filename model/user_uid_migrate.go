package model

import (
	"fmt"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

// migrateUserPublicUID 为存量用户补齐 uid，并创建唯一索引（uid 列由 AutoMigrate 先创建，此处不加 unique 以免空串冲突）。
func migrateUserPublicUID() error {
	if !DB.Migrator().HasColumn(&User{}, "Uid") {
		return nil
	}
	const batch = 500
	for {
		var ids []int
		err := DB.Model(&User{}).
			Where("uid IS NULL OR TRIM(uid) = ?", "").
			Order("id").
			Limit(batch).
			Pluck("id", &ids).Error
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			uid := NewUserPublicID()
			if err := DB.Model(&User{}).Where("id = ?", id).Update("uid", uid).Error; err != nil {
				return fmt.Errorf("backfill user uid id=%d: %w", id, err)
			}
		}
		logger.SysLog(fmt.Sprintf("users: backfilled uid for %d row(s)", len(ids)))
	}
	if err := ensureUserUIDUniqueIndex(); err != nil {
		return err
	}
	return nil
}

func ensureUserUIDUniqueIndex() error {
	switch {
	case common.UsingSQLite, common.UsingPostgreSQL:
		return DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uid ON users(uid)`).Error
	case common.UsingMySQL:
		var cnt int64
		if err := DB.Raw(`
SELECT COUNT(1) FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = 'users' AND index_name = 'idx_users_uid'`).Scan(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return nil
		}
		return DB.Exec(`CREATE UNIQUE INDEX idx_users_uid ON users(uid)`).Error
	default:
		return nil
	}
}
