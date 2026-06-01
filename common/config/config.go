package config

import (
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/env"

	"github.com/google/uuid"
)

var SystemName = "One API"
var ServerAddress = "http://localhost:3000"
var Footer = ""
var Logo = ""
var TopUpLink = ""
var ChatLink = ""
var QuotaPerUnit = 500 * 1000.0 // $0.002 / 1K tokens
// DefaultBillingTzOffsetMinutes 账单聚合/导出的默认时区偏移（分钟），东八区=480。
var DefaultBillingTzOffsetMinutes = 480

// BillingCycleDayOfMonth 账期起始日（每月第几天，1-28）；按此切出"当前账期"区间。默认 1=每月第一天。
var BillingCycleDayOfMonth = 1

// BillingSettlementDays 结算账期天数（如月结 40 天），仅作展示/对账说明，不影响实时扣费。
var BillingSettlementDays = 40

var DisplayInCurrencyEnabled = true
var DisplayTokenStatEnabled = true

// Any options with "Secret", "Token" in its key won't be return by GetOptions

var SessionSecret = uuid.New().String()

var OptionMap map[string]string
var OptionMapRWMutex sync.RWMutex

// IsNacosEnabled 是否启用 Nacos 注册表与管理端集成。
func IsNacosEnabled() bool {
	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()
	if OptionMap == nil {
		return NacosEnabled
	}
	v, ok := OptionMap["NacosEnabled"]
	if !ok {
		return NacosEnabled
	}
	return v == "true"
}

// IsRelayProtocolBridgeEnabled 是否允许 Anthropic / Gemini 原生路由在渠道协议不一致时经 OpenAI 语义跨厂商转发；无库内选项时用 LoadRuntime 的进程默认。
func IsRelayProtocolBridgeEnabled() bool {
	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()
	if OptionMap == nil {
		return RelayProtocolBridgeEnabled
	}
	v, ok := OptionMap["RelayProtocolBridgeEnabled"]
	if !ok {
		return RelayProtocolBridgeEnabled
	}
	return v == "true"
}

var ItemsPerPage = 10
var MaxRecentItems = 100

var PasswordLoginEnabled = true

// SecurePasswordLoginEnabled 为 true 时登录须先取 proof 并用一次性 AES 密钥加密密码；默认 false 为明文（依赖 HTTPS）。
var SecurePasswordLoginEnabled = false
var PasswordRegisterEnabled = true
var EmailVerificationEnabled = false
var GitHubOAuthEnabled = false
var LarkOAuthEnabled = false
var OidcEnabled = false
var WeChatAuthEnabled = false
var TurnstileCheckEnabled = false

// LoginCaptchaEnabled 为 false 时，关闭 Turnstile 场景下不再要求登录点击验证码（见环境变量 login_math_captcha_enabled）。默认开启。
var LoginCaptchaEnabled = true
var RegisterEnabled = true

// NacosEnabled 系统设置开关：关闭后 Nacos API 返回 404，管理端隐藏 Nacos 菜单。
var NacosEnabled = false

var EmailDomainRestrictionEnabled = false
var EmailDomainWhitelist = []string{
	"gmail.com",
	"163.com",
	"126.com",
	"qq.com",
	"outlook.com",
	"hotmail.com",
	"icloud.com",
	"yahoo.com",
	"foxmail.com",
}

var DebugEnabled bool
var DebugSQLEnabled bool
var MemoryCacheEnabled bool

var LogConsumeEnabled = true

// ErrorLogEnabled 为 true 时，Relay 等访问失败会写入操作日志（type=错误）。
var ErrorLogEnabled = true

// LogConsumeIncludeUserInput 为 true 时，消费日志 other 字段包含从请求体提取的用户输入摘要（含提示词等）。
var LogConsumeIncludeUserInput = true
var LogConsumeUserInputMaxBytes = 8192

// LogShardByDay 为 true 时日志写入按 UTC 日历日分物理表 logs_YYYYMMDD（需配合日志库迁移）；默认 false 保持单表 logs。
var LogShardByDay bool

// EnableHighCardinalityMetrics 为 true 时开启高基数打点（如消耗额度的 user_id 维度）
var EnableHighCardinalityMetrics = false

var SMTPServer = ""
var SMTPPort = 587
var SMTPAccount = ""
var SMTPFrom = ""
var SMTPToken = ""

var GitHubClientId = ""
var GitHubClientSecret = ""

var LarkClientId = ""
var LarkClientSecret = ""

var OidcClientId = ""
var OidcClientSecret = ""
var OidcWellKnown = ""
var OidcAuthorizationEndpoint = ""
var OidcTokenEndpoint = ""
var OidcUserinfoEndpoint = ""

var WeChatServerAddress = ""
var WeChatServerToken = ""
var WeChatAccountQRCodeImageURL = ""

var MessagePusherAddress = ""
var MessagePusherToken = ""

var TurnstileSiteKey = ""
var TurnstileSecretKey = ""

var QuotaForNewUser int64 = 0
var QuotaForInviter int64 = 0
var QuotaForInvitee int64 = 0
var ChannelDisableThreshold = 5.0
var AutomaticDisableChannelEnabled = false
var AutomaticEnableChannelEnabled = false
var QuotaRemindThreshold int64 = 1000
var PreConsumedQuota int64 = 500
var ApproximateTokenEnabled = false
var RetryTimes = 0

var RootUserEmail = ""

var IsMasterNode = true

var RequestInterval time.Duration

var SyncFrequency int

var BatchUpdateEnabled bool
var BatchUpdateInterval int

// 异步消费日志队列：将高频的 Log 写入从请求热路径剥离，由后台 worker 批量落库。
// 队列满时回退为同步写入，保证不丢日志。
var LogAsyncEnabled bool
var LogAsyncBufferSize int
var LogAsyncBatchSize int
var LogAsyncFlushIntervalMs int

var RelayTimeout int

// relay 上游 HTTP 客户端连接池。Go 默认 MaxIdleConnsPerHost=2，对网关来说过小：
// 上游主机只保留 2 条 keepalive 连接，高并发下会反复新建/丢弃连接（TCP+TLS 握手），
// 严重限制单上游吞吐。这里提供更合理的默认值并允许调优。
var RelayMaxIdleConns int
var RelayMaxIdleConnsPerHost int
var RelayMaxConnsPerHost int

// 优雅关闭：收到 SIGINT/SIGTERM 后等待在途请求处理完成的最长秒数。
var ServerShutdownTimeout int

var GeminiSafetySetting string

var Theme string
var ValidThemes = map[string]bool{
	"vue": true,
}

// Login captcha modes backed by github.com/wenlng/go-captcha. "random" rotates
// among the concrete modes per challenge. Configurable via LOGIN_CAPTCHA_MODE
// (TOML key login_captcha_mode). The classic React theme only renders click, so
// the controller forces click for it regardless of this setting.
const (
	LoginCaptchaModeRandom = "random"
	LoginCaptchaModeClick  = "click"
	LoginCaptchaModeSlide  = "slide"
	LoginCaptchaModeRotate = "rotate"
)

var LoginCaptchaMode = LoginCaptchaModeRandom

func normalizeLoginCaptchaMode(v string) string {
	switch v {
	case LoginCaptchaModeRandom, LoginCaptchaModeClick, LoginCaptchaModeSlide, LoginCaptchaModeRotate:
		return v
	default:
		return LoginCaptchaModeRandom
	}
}

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitNum      int
	GlobalApiRateLimitDuration int64 = 3 * 60
	GlobalWebRateLimitNum      int
	GlobalWebRateLimitDuration int64 = 3 * 60
	UploadRateLimitNum               = 10
	UploadRateLimitDuration    int64 = 60
	DownloadRateLimitNum             = 10
	DownloadRateLimitDuration  int64 = 60
	CriticalRateLimitNum             = 20
	CriticalRateLimitDuration  int64 = 20 * 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute

var EnableMetric bool
var MetricQueueSize int
var MetricSuccessRateThreshold float64
var MetricSuccessChanSize int
var MetricFailChanSize int

var InitialRootToken string

var InitialRootAccessToken string

var GeminiVersion string

var OnlyOneLogFile bool

var RelayProxy string
var UserContentRequestProxy string
var UserContentRequestTimeout int

var EnforceIncludeUsage bool
var TestPrompt string

var NacosRegistryAnonymousRead bool
var NacosRegistryMaxUploadBytes int64
var NacosRegistryZipStorage string
var NacosRegistryZipLocalDir string
var NacosRegistryS3KeyPrefix string

// NacosCsEncryptionKey 配置中心 cipher-aes-* dataId 的 AES-256-GCM 主密钥材料（任意长度，内部 SHA256 派生 32 字节）。空则禁止创建/读取加密配置。
var NacosCsEncryptionKey string

// NacosCsEncryptionKeyPrevious 轮换用历史密钥：仅参与解密（多行则每行一条，依次尝试）；发布仍只用主密钥。
var NacosCsEncryptionKeyPrevious string

// NacosCsClientGetReturnCiphertext 为 true 时，GET /nacos/v3/client/cs/config 对已加密落库的配置返回密文 content + encryptedDataKey（与 SDK 拉取形态一致）；控制台/管理端仍返回明文。
var NacosCsClientGetReturnCiphertext bool

// RequestAuditEnabled 为 true 时，将 Token 用户的 Relay 请求/响应以 request_id 为键写入独立 JSONL 审计文件（含流式首 token、扣费等）。默认开启，注意日志内含用户提示词与模型输出；可通过配置关闭。
var RequestAuditEnabled bool
var RequestAuditLogDir string
var RequestAuditMaxBodyBytes int

// RelayProtocolBridgeEnabled 为 true 时，允许 Anthropic / Gemini 原生路由在与渠道协议不一致时自动经 OpenAI 通用语义转发（跨厂商互调）。默认关闭以防误用。
var RelayProtocolBridgeEnabled bool

var DefaultTenantChannelPricePer1k float64 = 0

// LoadRuntime 须在 cfg.Init 与 env.BindViper 之后调用，从 TOML / 命令行填充运行时项。
func LoadRuntime() {
	DebugEnabled = env.Bool("DEBUG", false)
	DebugSQLEnabled = env.Bool("DEBUG_SQL", false)
	MemoryCacheEnabled = env.Bool("MEMORY_CACHE_ENABLED", false)

	IsMasterNode = strings.ToLower(strings.TrimSpace(env.String("NODE_TYPE", ""))) != "slave"
	polling := env.Int("POLLING_INTERVAL", 0)
	RequestInterval = time.Duration(polling) * time.Second

	SyncFrequency = env.Int("SYNC_FREQUENCY", 10*60)
	BatchUpdateEnabled = env.Bool("BATCH_UPDATE_ENABLED", false)
	BatchUpdateInterval = env.Int("BATCH_UPDATE_INTERVAL", 5)

	LogAsyncEnabled = env.Bool("LOG_ASYNC_ENABLED", true)
	LogAsyncBufferSize = env.Int("LOG_ASYNC_BUFFER_SIZE", 10000)
	LogAsyncBatchSize = env.Int("LOG_ASYNC_BATCH_SIZE", 100)
	LogAsyncFlushIntervalMs = env.Int("LOG_ASYNC_FLUSH_INTERVAL_MS", 1000)

	RelayTimeout = env.Int("RELAY_TIMEOUT", 0)
	RelayMaxIdleConns = env.Int("RELAY_MAX_IDLE_CONNS", 200)
	RelayMaxIdleConnsPerHost = env.Int("RELAY_MAX_IDLE_CONNS_PER_HOST", 100)
	RelayMaxConnsPerHost = env.Int("RELAY_MAX_CONNS_PER_HOST", 0)
	ServerShutdownTimeout = env.Int("SERVER_SHUTDOWN_TIMEOUT", 30)
	GeminiSafetySetting = env.String("GEMINI_SAFETY_SETTING", "BLOCK_NONE")
	Theme = env.String("THEME", "vue")
	LoginCaptchaMode = normalizeLoginCaptchaMode(env.String("LOGIN_CAPTCHA_MODE", LoginCaptchaModeRandom))

	GlobalApiRateLimitNum = env.Int("GLOBAL_API_RATE_LIMIT", 480)
	GlobalWebRateLimitNum = env.Int("GLOBAL_WEB_RATE_LIMIT", 240)

	EnableMetric = env.Bool("ENABLE_METRIC", false)
	MetricQueueSize = env.Int("METRIC_QUEUE_SIZE", 10)
	MetricSuccessRateThreshold = env.Float64("METRIC_SUCCESS_RATE_THRESHOLD", 0.8)
	MetricSuccessChanSize = env.Int("METRIC_SUCCESS_CHAN_SIZE", 1024)
	MetricFailChanSize = env.Int("METRIC_FAIL_CHAN_SIZE", 128)

	InitialRootToken = env.String("INITIAL_ROOT_TOKEN", "")
	InitialRootAccessToken = env.String("INITIAL_ROOT_ACCESS_TOKEN", "")

	GeminiVersion = env.String("GEMINI_VERSION", "v1beta")
	OnlyOneLogFile = env.Bool("ONLY_ONE_LOG_FILE", false)
	RelayProxy = env.String("RELAY_PROXY", "")
	UserContentRequestProxy = env.String("USER_CONTENT_REQUEST_PROXY", "")
	UserContentRequestTimeout = env.Int("USER_CONTENT_REQUEST_TIMEOUT", 30)
	EnforceIncludeUsage = env.Bool("ENFORCE_INCLUDE_USAGE", false)
	TestPrompt = env.String("TEST_PROMPT", "Output only your specific model name with no additional text.")

	NacosRegistryAnonymousRead = env.Bool("NACOS_REGISTRY_ANONYMOUS_READ", true)
	NacosRegistryMaxUploadBytes = env.Int64Always("nacos_registry_max_upload_bytes")
	if NacosRegistryMaxUploadBytes <= 0 {
		NacosRegistryMaxUploadBytes = 10 << 20
	}
	NacosRegistryZipStorage = env.StringAlways("nacos_registry_zip_storage")
	NacosRegistryZipLocalDir = env.StringAlways("nacos_registry_zip_local_dir")
	NacosRegistryS3KeyPrefix = env.StringAlways("nacos_registry_s3_key_prefix")
	NacosCsEncryptionKey = env.StringAlways("nacos_cs_encryption_key")
	NacosCsEncryptionKeyPrevious = env.StringAlways("nacos_cs_encryption_key_previous")
	NacosCsClientGetReturnCiphertext = env.BoolAlways("nacos_cs_client_get_return_ciphertext")

	RequestAuditEnabled = env.BoolAlways("request_audit_enabled")
	RequestAuditLogDir = env.StringAlways("request_audit_log_dir")
	RequestAuditMaxBodyBytes = env.IntAlways("request_audit_max_body_bytes")
	if RequestAuditMaxBodyBytes <= 0 {
		RequestAuditMaxBodyBytes = 262144
	}

	RelayProtocolBridgeEnabled = env.BoolAlways("relay_protocol_bridge_enabled")

	LogShardByDay = env.BoolAlways("log_shard_by_day")

	LogConsumeIncludeUserInput = env.Bool("LOG_CONSUME_INCLUDE_USER_INPUT", true)
	LogConsumeUserInputMaxBytes = env.Int("LOG_CONSUME_USER_INPUT_MAX_BYTES", 8192)
	if LogConsumeUserInputMaxBytes <= 0 {
		LogConsumeUserInputMaxBytes = 8192
	}
}
