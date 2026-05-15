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

var ItemsPerPage = 10
var MaxRecentItems = 100

var PasswordLoginEnabled = true
// SecurePasswordLoginEnabled 为 true 时登录须先取 proof 并用一次性 AES 密钥加密密码；默认 false 为明文（依赖 HTTPS）。
var SecurePasswordLoginEnabled = false
var PasswordRegisterEnabled = true
var EmailVerificationEnabled = false
var GitHubOAuthEnabled = false
var OidcEnabled = false
var WeChatAuthEnabled = false
var TurnstileCheckEnabled = false
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

var RelayTimeout int

var GeminiSafetySetting string

var Theme string
var ValidThemes = map[string]bool{
	"default": true,
	"berry":   true,
	"air":     true,
}

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitNum         int
	GlobalApiRateLimitDuration    int64 = 3 * 60
	GlobalWebRateLimitNum         int
	GlobalWebRateLimitDuration    int64 = 3 * 60
	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60
	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60
	CriticalRateLimitNum            = 20
	CriticalRateLimitDuration int64 = 20 * 60
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

	RelayTimeout = env.Int("RELAY_TIMEOUT", 0)
	GeminiSafetySetting = env.String("GEMINI_SAFETY_SETTING", "BLOCK_NONE")
	Theme = env.String("THEME", "default")

	GlobalApiRateLimitNum = env.Int("GLOBAL_API_RATE_LIMIT", 480)
	GlobalWebRateLimitNum = env.Int("GLOBAL_WEB_RATE_LIMIT", 240)

	EnableMetric = env.Bool("ENABLE_METRIC", false)
	MetricQueueSize = env.Int("METRIC_QUEUE_SIZE", 10)
	MetricSuccessRateThreshold = env.Float64("METRIC_SUCCESS_RATE_THRESHOLD", 0.8)
	MetricSuccessChanSize = env.Int("METRIC_SUCCESS_CHAN_SIZE", 1024)
	MetricFailChanSize = env.Int("METRIC_FAIL_CHAN_SIZE", 128)

	InitialRootToken = env.String("INITIAL_ROOT_TOKEN", "")
	InitialRootAccessToken = env.String("INITIAL_ROOT_ACCESS_TOKEN", "")

	GeminiVersion = env.String("GEMINI_VERSION", "v1")
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
}
