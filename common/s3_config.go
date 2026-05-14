package common

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

// 兼容 S3 子集（SigV4 + path-style）的对象存储。
// 访问密钥由用户在控制台按账号生成，存储在 user 表，对象按用户 ID 目录隔离。
// 进程启动时会默认创建/使用本地 S3 存储目录（见 s3_storage_dir 等配置项）。
// 若无需该能力，可设置 s3_disabled = true 以跳过初始化（S3Enabled 为 false）。
//
// 若设置 s3_remote_bucket，则对象写入真实 S3 兼容服务（AWS S3 / MinIO 等），不再使用本地目录。
var (
	S3Enabled               bool
	S3RemoteEnabled         bool
	S3RemoteBucket          string
	S3RemoteEndpoint        string
	S3RemoteRegion          string
	S3RemoteAccessKeyID     string
	S3RemoteSecretAccessKey string
	S3RemotePathStyle       bool
	S3RemoteKeyPrefix       string
	S3Region                string
	S3MaxObjectBytes        int64
	S3DefaultTTL            time.Duration
	S3MaxTTL                time.Duration
	S3StorageDir            string
	S3CleanerInterval       time.Duration
	S3ClockSkew             time.Duration
	S3CORSAllowOrigin       string
	S3PathStyleAtRoot       bool
)

func s3Getenv(k string) string {
	return strings.TrimSpace(cfg.V.GetString(strings.ToLower(k)))
}

func s3EnvInt64(key string, def int64) int64 {
	v := s3Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func s3EnvDurationSeconds(key string, def time.Duration) time.Duration {
	v := s3Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}

func InitS3Config() {
	S3Enabled = false
	S3RemoteEnabled = false
	if cfg.V.GetBool("s3_disabled") {
		logger.SysLog("S3 storage skipped (s3_disabled=true)")
		return
	}

	S3Region = s3Getenv("S3_REGION")
	if S3Region == "" {
		S3Region = "us-east-1"
	}
	S3MaxObjectBytes = s3EnvInt64("S3_MAX_OBJECT_BYTES", 32<<20)
	S3DefaultTTL = s3EnvDurationSeconds("S3_DEFAULT_TTL_SECONDS", 24*time.Hour)
	S3MaxTTL = s3EnvDurationSeconds("S3_MAX_TTL_SECONDS", 7*24*time.Hour)
	if S3MaxTTL < S3DefaultTTL {
		S3MaxTTL = S3DefaultTTL
	}

	if rb := strings.TrimSpace(s3Getenv("S3_REMOTE_BUCKET")); rb != "" {
		initS3RemoteBackend(rb)
		return
	}

	dir := s3Getenv("S3_STORAGE_DIR")
	if dir == "" {
		dir = filepath.Join(SQLitePath, "..", "s3")
		if SQLitePath == "" {
			dir = "./data/s3"
		}
	}
	var err error
	S3StorageDir, err = filepath.Abs(dir)
	if err != nil {
		logger.SysError("S3_STORAGE_DIR invalid: " + err.Error())
		return
	}
	S3CleanerInterval = s3EnvDurationSeconds("S3_CLEANER_INTERVAL_SECONDS", 60*time.Second)
	if S3CleanerInterval < 10*time.Second {
		S3CleanerInterval = 10 * time.Second
	}
	if v := s3Getenv("S3_CLOCK_SKEW_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			S3ClockSkew = time.Duration(n) * time.Second
		}
	} else {
		S3ClockSkew = 15 * time.Minute
	}
	S3CORSAllowOrigin = s3Getenv("S3_CORS_ALLOW_ORIGIN")
	addr := strings.ToLower(s3Getenv("S3_ADDRESSING"))
	S3PathStyleAtRoot = addr == "path" || addr == "path-style"
	if S3PathStyleAtRoot {
		logger.SysLog("S3 path-style at site root enabled (S3_ADDRESSING=path); reserved bucket names cannot be used as the first path segment")
	}
	S3Enabled = true
	logger.SysLog("S3 storage initialized: " + S3StorageDir)
}

func initS3RemoteBackend(bucket string) {
	S3RemoteBucket = bucket
	S3RemoteEndpoint = strings.TrimSpace(s3Getenv("S3_REMOTE_ENDPOINT"))
	S3RemoteRegion = s3Getenv("S3_REMOTE_REGION")
	if S3RemoteRegion == "" {
		S3RemoteRegion = S3Region
	}
	S3RemoteAccessKeyID = s3Getenv("S3_REMOTE_ACCESS_KEY_ID")
	S3RemoteSecretAccessKey = s3Getenv("S3_REMOTE_SECRET_ACCESS_KEY")
	S3RemotePathStyle = cfg.V.GetBool("s3_remote_use_path_style")
	if S3RemoteEndpoint != "" && !S3RemotePathStyle {
		el := strings.ToLower(S3RemoteEndpoint)
		if strings.Contains(el, "localhost") || strings.Contains(el, "127.0.0.1") || strings.Contains(el, "minio") {
			S3RemotePathStyle = true
		}
	}
	p := strings.TrimSpace(s3Getenv("S3_REMOTE_KEY_PREFIX"))
	if p != "" && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	S3RemoteKeyPrefix = p

	S3CleanerInterval = s3EnvDurationSeconds("S3_CLEANER_INTERVAL_SECONDS", 60*time.Second)
	if S3CleanerInterval < 10*time.Second {
		S3CleanerInterval = 10 * time.Second
	}
	if v := s3Getenv("S3_CLOCK_SKEW_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			S3ClockSkew = time.Duration(n) * time.Second
		}
	} else {
		S3ClockSkew = 15 * time.Minute
	}
	S3CORSAllowOrigin = s3Getenv("S3_CORS_ALLOW_ORIGIN")
	addr := strings.ToLower(s3Getenv("S3_ADDRESSING"))
	S3PathStyleAtRoot = addr == "path" || addr == "path-style"
	if S3PathStyleAtRoot {
		logger.SysLog("S3 path-style at site root enabled (S3_ADDRESSING=path); reserved bucket names cannot be used as the first path segment")
	}

	S3StorageDir = ""
	S3RemoteEnabled = true
	S3Enabled = true
	if S3RemoteEndpoint != "" {
		logger.SysLog("S3 remote backend: bucket=" + S3RemoteBucket + " endpoint=" + S3RemoteEndpoint + " region=" + S3RemoteRegion)
	} else {
		logger.SysLog("S3 remote backend: bucket=" + S3RemoteBucket + " region=" + S3RemoteRegion)
	}
}

// S3SiteOpen 是否对用户开放 S3 兼容 API（存储已就绪且管理员在系统设置中开启）。
func S3SiteOpen() bool {
	if !S3Enabled {
		return false
	}
	config.OptionMapRWMutex.RLock()
	defer config.OptionMapRWMutex.RUnlock()
	if config.OptionMap == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(config.OptionMap["S3SiteEnabled"]), "true")
}
