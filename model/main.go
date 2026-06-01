package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"strings"
	"time"
)

var DB *gorm.DB
var LOG_DB *gorm.DB

func CreateRootAccountIfNeed() error {
	var user User
	//if user.Status != util.UserStatusEnabled {
	if err := DB.First(&user).Error; err != nil {
		logger.SysLog("no user exists, creating a root user for you: username is root, password is 123456")
		hashedPassword, err := common.Password2Hash("123456")
		if err != nil {
			return err
		}
		accessToken := random.GetUUID()
		if config.InitialRootAccessToken != "" {
			accessToken = config.InitialRootAccessToken
		}
		rootUser := User{
			Username:    "root",
			Password:    hashedPassword,
			Role:        RoleRootUser,
			Status:      UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: accessToken,
			Quota:       500000000000000,
		}
		DB.Create(&rootUser)
		if config.InitialRootToken != "" {
			logger.SysLog("creating initial root token as requested")
			token := Token{
				Id:             1,
				UserId:         rootUser.Id,
				Key:            config.InitialRootToken,
				Status:         TokenStatusEnabled,
				Name:           "Initial Root Token",
				CreatedTime:    helper.GetTimestamp(),
				AccessedTime:   helper.GetTimestamp(),
				ExpiredTime:    -1,
				RemainQuota:    500000000000000,
				UnlimitedQuota: true,
			}
			DB.Create(&token)
		}
	}
	return nil
}

func chooseDB(dsn string) (*gorm.DB, error) {
	switch {
	case strings.HasPrefix(dsn, "postgres://"):
		// Use PostgreSQL
		return openPostgreSQL(dsn)
	case dsn != "":
		// Use MySQL
		return openMySQL(dsn)
	default:
		// Use SQLite
		return openSQLite()
	}
}

func openPostgreSQL(dsn string) (*gorm.DB, error) {
	logger.SysLog("using PostgreSQL as database")
	common.UsingPostgreSQL = true
	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		PrepareStmt: true, // precompile SQL
	})
}

func openMySQL(dsn string) (*gorm.DB, error) {
	logger.SysLog("using MySQL as database")
	common.UsingMySQL = true
	dsn = ensureMySQLTimeParams(dsn)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true, // precompile SQL
	})
}

func ensureMySQLTimeParams(dsn string) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	if !strings.Contains(dsn, "parseTime=") {
		dsn += sep + "parseTime=true"
		sep = "&"
	}
	if !strings.Contains(dsn, "loc=") {
		dsn += sep + "loc=Local"
	}
	return dsn
}

func openSQLite() (*gorm.DB, error) {
	logger.SysLog("SQL_DSN not set, using SQLite as database")
	common.UsingSQLite = true
	dsn := fmt.Sprintf("%s?_busy_timeout=%d", common.SQLitePath, common.SQLiteBusyTimeout)
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		PrepareStmt: true, // precompile SQL
	})
}

func InitDB() {
	var err error
	DB, err = chooseDB(env.StringAlways("sql_dsn"))
	if err != nil {
		logger.FatalLog("failed to initialize database: " + err.Error())
		return
	}

	sqlDB := setDBConns(DB)

	if !config.IsMasterNode {
		return
	}

	if common.UsingMySQL {
		_, _ = sqlDB.Exec("DROP INDEX idx_channels_key ON channels;") // TODO: delete this line when most users have upgraded
	}

	logger.SysLog("database migration started")
	if err = migrateDB(); err != nil {
		logger.FatalLog("failed to migrate database: " + err.Error())
		return
	}
	logger.SysLog("database migrated")
}

func migrateDB() error {
	var err error
	if err = migrateChannelRenameLegacySettingsColumn(); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&Channel{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&Option{}); err != nil {
		return err
	}
	if err = migrateChannelGeminiOpenAIReorderIfNeeded(); err != nil {
		return err
	}
	if err = migrateChannelMergeOtherIntoConfig(); err != nil {
		return err
	}
	if err = migrateChannelDropOtherColumn(); err != nil {
		return err
	}
	if err = migrateChannelModelMappingToText(); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&Token{}); err != nil {
		return err
	}
	if err = migrateUserLegacyS3ColumnNames(); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&Tenant{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&User{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&Redemption{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&Ability{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&ChannelModelTestResult{}); err != nil {
		return err
	}
	logDSN := env.StringAlways("log_sql_dsn")
	if logDSN == "" {
		if config.LogShardByDay {
			if err = migrateLogShardTablesBoot(DB); err != nil {
				return err
			}
		} else {
			if err = DB.AutoMigrate(&Log{}); err != nil {
				return err
			}
		}
	} else if !config.LogShardByDay {
		if err = DB.AutoMigrate(&Log{}); err != nil {
			return err
		}
	}
	if err = DB.AutoMigrate(&GlobalAccessWhitelist{}, &GlobalAccessBlacklist{}, &TwoFA{}, &TwoFABackupCode{}); err != nil {
		return err
	}
	if err = migrateModelCatalogVersionStatus(); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&ModelCatalog{}); err != nil {
		return err
	}
	backfillModelCatalogVersionStatus()
	if err = DB.AutoMigrate(&PaymentOrder{}, &PaymentChannelAccess{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&TenantUpgradeRequest{}, &TenantBillingRule{}); err != nil {
		return err
	}
	if err = DB.AutoMigrate(&NacosRegistryNamespace{}); err != nil {
		return err
	}
	if err = migrateNacosExtensionTables(); err != nil {
		return err
	}
	return nil
}

// migrateUserLegacyS3ColumnNames 将旧列 temp_s3_* 重命名为 s3_*（若存在）。
func migrateUserLegacyS3ColumnNames() error {
	var n int64
	switch {
	case common.UsingSQLite:
		if err := DB.Raw(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'temp_s3_enabled'`).Scan(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		for _, stmt := range []string{
			`ALTER TABLE users RENAME COLUMN temp_s3_enabled TO s3_enabled`,
			`ALTER TABLE users RENAME COLUMN temp_s3_access_key TO s3_access_key`,
			`ALTER TABLE users RENAME COLUMN temp_s3_secret_key TO s3_secret_key`,
		} {
			if _, err := sqlDB.Exec(stmt); err != nil {
				return fmt.Errorf("sqlite migrate: %s: %w", stmt, err)
			}
		}
		logger.SysLog("users: renamed legacy temp_s3_* columns to s3_*")
		return nil
	case common.UsingPostgreSQL:
		if err := DB.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'temp_s3_enabled'`).Scan(&n).Error; err != nil {
			return err
		}
	case common.UsingMySQL:
		if err := DB.Raw(`SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'temp_s3_enabled'`).Scan(&n).Error; err != nil {
			return err
		}
	default:
		return nil
	}
	if n == 0 {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	for _, stmt := range []string{
		`ALTER TABLE users RENAME COLUMN temp_s3_enabled TO s3_enabled`,
		`ALTER TABLE users RENAME COLUMN temp_s3_access_key TO s3_access_key`,
		`ALTER TABLE users RENAME COLUMN temp_s3_secret_key TO s3_secret_key`,
	} {
		if _, err := sqlDB.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %s: %w", stmt, err)
		}
	}
	logger.SysLog("users: renamed legacy temp_s3_* columns to s3_*")
	return nil
}

// migrateChannelModelMappingToText widens channels.model_mapping from legacy varchar(1024)
// to TEXT so large JSON mappings do not fail (e.g. MySQL Error 1406 Data too long).
func migrateChannelModelMappingToText() error {
	if common.UsingSQLite {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	switch {
	case common.UsingMySQL:
		_, err = sqlDB.Exec(`ALTER TABLE channels MODIFY COLUMN model_mapping TEXT`)
	case common.UsingPostgreSQL:
		_, err = sqlDB.Exec(`ALTER TABLE channels ALTER COLUMN model_mapping TYPE TEXT USING model_mapping::text`)
	default:
		return nil
	}
	return err
}

func InitLogDB() {
	if env.StringAlways("log_sql_dsn") == "" {
		LOG_DB = DB
		initLogGroupColumn()
		return
	}

	logger.SysLog("using secondary database for table logs")
	var err error
	LOG_DB, err = chooseDB(env.StringAlways("log_sql_dsn"))
	if err != nil {
		logger.FatalLog("failed to initialize secondary database: " + err.Error())
		return
	}

	setDBConns(LOG_DB)

	if !config.IsMasterNode {
		initLogGroupColumn()
		return
	}

	logger.SysLog("secondary database migration started")
	err = migrateLOGDB()
	if err != nil {
		logger.FatalLog("failed to migrate secondary database: " + err.Error())
		return
	}
	logger.SysLog("secondary database migrated")
	initLogGroupColumn()
}

func migrateLOGDB() error {
	if config.LogShardByDay {
		return migrateLogShardTablesBoot(LOG_DB)
	}
	return LOG_DB.AutoMigrate(&Log{})
}

func setDBConns(db *gorm.DB) *sql.DB {
	if config.DebugSQLEnabled {
		db = db.Debug()
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.FatalLog("failed to connect database: " + err.Error())
		return nil
	}

	sqlDB.SetMaxIdleConns(env.IntAlways("sql_max_idle_conns"))
	sqlDB.SetMaxOpenConns(env.IntAlways("sql_max_open_conns"))
	sqlDB.SetConnMaxLifetime(time.Second * time.Duration(env.IntAlways("sql_max_lifetime")))
	// SQLite 是单写者：用大连接池只会触发 "database is locked" 争用，强制单连接。
	if common.UsingSQLite {
		sqlDB.SetMaxOpenConns(1)
	}
	return sqlDB
}

// PingDB 校验主库连接可用，供 /healthz 就绪探针使用。
func PingDB(ctx context.Context) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Close()
	return err
}

func CloseDB() error {
	if LOG_DB != DB {
		err := closeDB(LOG_DB)
		if err != nil {
			return err
		}
	}
	return closeDB(DB)
}

func migrateModelCatalogVersionStatus() error {
	if DB.Migrator().HasIndex(&ModelCatalog{}, "uk_model_catalog_mid") {
		err := DB.Migrator().DropIndex(&ModelCatalog{}, "uk_model_catalog_mid")
		if err != nil {
			logger.SysError("failed to drop uk_model_catalog_mid: " + err.Error())
		}
	}
	// 行身份由 (source, model_id) 扩展为 (source, provider_key, model_id)：丢弃旧复合索引，
	// 让随后的 AutoMigrate 按新名（idx_mc_src_pk_mid_*）重建含 provider_key 的索引。
	// 新键在旧键基础上多一列，只会更唯一、不会引入冲突，故重建安全。
	for _, old := range []string{"idx_mc_src_mid_ver", "idx_mc_src_mid_status"} {
		if DB.Migrator().HasIndex(&ModelCatalog{}, old) {
			if err := DB.Migrator().DropIndex(&ModelCatalog{}, old); err != nil {
				logger.SysError("failed to drop " + old + ": " + err.Error())
			}
		}
	}
	return nil
}

func backfillModelCatalogVersionStatus() {
	DB.Exec(`UPDATE model_catalogs SET version=1 WHERE version IS NULL OR version=0`)
	DB.Exec(`UPDATE model_catalogs SET status='current' WHERE status='' OR status IS NULL`)
}
