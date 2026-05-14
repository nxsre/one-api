package cfg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// V 为进程级配置（TOML + 命令行）；在 Init 完成前勿读。
var V *viper.Viper

// PrintVersion / PrintHelp / LogDir 与命令行解析结果同步，供 common.Init 使用。
var (
	PrintVersion bool
	PrintHelp    bool
	LogDir       string
)

func setDefaults(v *viper.Viper) {
	d := func(key string, val any) { v.SetDefault(key, val) }

	d("port", 3000)
	d("log_dir", "./logs")
	d("gin_mode", "")
	d("channel_test_frequency", 0)
	d("batch_update_enabled", false)

	d("session_secret", "")
	d("sqlite_path", "")
	d("sql_dsn", "")
	d("log_sql_dsn", "")
	d("sqlite_busy_timeout", 3000)

	d("debug", false)
	d("debug_sql", false)
	d("memory_cache_enabled", false)
	d("node_type", "")
	d("polling_interval", 0)
	d("initial_root_token", "")
	d("initial_root_access_token", "")

	d("sync_frequency", 10*60)
	d("batch_update_interval", 5)
	d("relay_timeout", 0)
	d("gemini_safety_setting", "BLOCK_NONE")
	d("theme", "default")
	d("global_api_rate_limit", 480)
	d("global_web_rate_limit", 240)
	d("enable_metric", false)
	d("metric_queue_size", 10)
	d("metric_success_rate_threshold", 0.8)
	d("metric_success_chan_size", 1024)
	d("metric_fail_chan_size", 128)
	d("gemini_version", "v1")
	d("only_one_log_file", false)
	d("relay_proxy", "")
	d("user_content_request_proxy", "")
	d("user_content_request_timeout", 30)
	d("enforce_include_usage", false)
	d("test_prompt", "Output only your specific model name with no additional text.")

	d("sql_max_idle_conns", 100)
	d("sql_max_open_conns", 1000)
	d("sql_max_lifetime", 60)

	d("s3_disabled", false)
	d("s3_region", "us-east-1")
	d("s3_max_object_bytes", int64(32<<20))
	d("s3_default_ttl_seconds", int(24 * 3600))
	d("s3_max_ttl_seconds", int(7 * 24 * 3600))
	d("s3_storage_dir", "")
	d("s3_remote_bucket", "")
	d("s3_remote_endpoint", "")
	d("s3_remote_region", "")
	d("s3_remote_access_key_id", "")
	d("s3_remote_secret_access_key", "")
	d("s3_remote_use_path_style", false)
	d("s3_remote_key_prefix", "")
	d("s3_cleaner_interval_seconds", 60)
	d("s3_clock_skew_seconds", 0)
	d("s3_cors_allow_origin", "")
	d("s3_addressing", "")

	d("redis_disabled", false)
	d("redis_conn_string", "")
	d("redis_master_name", "")
	d("redis_password", "")

	d("tls_cert_file", "")
	d("tls_key_file", "")
	d("https_port", "3443")

	d("force_2fa_for_all_users", false)
	d("login_math_captcha_enabled", true)
	d("login_brute_trust_x_forwarded_for", false)
	d("login_brute_ip_fail_max", 0)
	d("login_brute_pair_fail_max", 0)
	d("login_brute_fail_window_sec", int64(0))
	d("login_brute_lock_duration_sec", int64(0))

	d("login_password_rsa_private_key", "")
	d("login_password_rsa_key_file", "")

	d("frontend_base_url", "")
	d("global_access_scope", "relay")

	d("tiktoken_cache_dir", "")
}

// Init 解析命令行、加载 TOML、写入默认值。须在整个进程最先调用之一。
func Init() error {
	v := viper.New()
	v.SetConfigType("toml")

	var configFile string
	var port int
	var logDir string

	pflag.StringVarP(&configFile, "config", "c", "", "path to TOML config file")
	pflag.IntVar(&port, "port", 3000, "listening port")
	pflag.StringVar(&logDir, "log-dir", "", "log directory (empty = use config default)")
	pflag.BoolVar(&PrintVersion, "version", false, "print version and exit")
	pflag.BoolVar(&PrintHelp, "help", false, "print help and exit")

	pflag.Parse()

	setDefaults(v)

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("read config %q: %w", configFile, err)
		}
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		_ = v.ReadInConfig()
	}

	// 仅当用户在命令行显式传入时覆盖配置文件（避免 pflag 默认值压过 TOML）
	if pf := pflag.Lookup("port"); pf != nil && pf.Changed {
		v.Set("port", port)
	}
	if lf := pflag.Lookup("log-dir"); lf != nil && lf.Changed {
		v.Set("log_dir", logDir)
	}

	V = v

	absLog, err := filepath.Abs(V.GetString("log_dir"))
	if err != nil {
		return err
	}
	LogDir = absLog
	return nil
}

// MustExitForFlags 处理 --version / --help（在 cfg.Init 之后、其余初始化之前调用）。
func MustExitForFlags(printVersionFn func(), printHelpFn func(), exit func(int)) {
	if PrintVersion {
		printVersionFn()
		exit(0)
	}
	if PrintHelp {
		printHelpFn()
		exit(0)
	}
}

// EnsureLogDir 创建日志目录（与原先 common.Init 行为一致）。
func EnsureLogDir() error {
	if LogDir == "" {
		return nil
	}
	if _, err := os.Stat(LogDir); os.IsNotExist(err) {
		if err := os.Mkdir(LogDir, 0777); err != nil {
			return err
		}
	}
	return nil
}
