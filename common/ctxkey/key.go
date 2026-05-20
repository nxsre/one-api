package ctxkey

const (
	Config            = "config"
	Id                = "id"
	// UserTenantID 中继选路用：令牌所属用户的租户 ID，平台用户为 0。
	UserTenantID = "user_tenant_id"
	// UserAllowedChannelIDs 租户子账号允许使用的渠道 ID 集合（map[int]struct{}）；未设置表示不按用户白名单限制渠道。
	UserAllowedChannelIDs = "user_allowed_channel_ids"
	// UserAllowedModels 租户子账号允许使用的模型名集合（map[string]struct{}）；未设置表示不按用户白名单限制模型。
	UserAllowedModels = "user_allowed_models"
	// TokenBoundGroup 令牌绑定的分组覆盖（非空则覆盖用户默认分组参与别名解析与选路）。
	TokenBoundGroup = "token_bound_group"
	Username          = "username"
	Role              = "role"
	Status            = "status"
	Channel           = "channel"
	ChannelId         = "channel_id"
	SpecificChannelId = "specific_channel_id"
	RequestModel      = "request_model"
	ConvertedRequest  = "converted_request"
	OriginalModel     = "original_model"
	Group             = "group"
	// UserGroup 用户账号所属分组（选路前），用于 GroupGroupRatio。
	UserGroup = "user_group"
	// UsingGroup 本次请求实际使用渠道的分组标签（首个），用于 GroupGroupRatio。
	UsingGroup = "using_group"
	ModelMapping      = "model_mapping"
	ChannelName       = "channel_name"
	TokenId           = "token_id"
	TokenName         = "token_name"
	BaseURL           = "base_url"
	// ChannelKey 为选中渠道的密钥（上游凭据），与客户端 Authorization 中的 sk- 令牌无关。
	ChannelKey                 = "channel_key"
	AvailableModels            = "available_models"
	KeyRequestBody             = "key_request_body"
	SystemPrompt               = "system_prompt"
	RoutingStickyKey           = "routing_sticky_key"
	LogicalModel               = "logical_model"
	// AnthropicModelVariant Claude Code 档位后缀，如 [1m]（对应上游 anthropic-beta）。
	AnthropicModelVariant = "anthropic_model_variant"
	ChannelRoutingProvider     = "channel_routing_provider"
	// ChannelAutoBan 为 true 时允许在满足全局策略时自动禁用渠道（与渠道 auto_ban 字段对应，默认 true）。
	ChannelAutoBan = "channel_auto_ban"
	// OpenAIOrganization 非空时写入 OpenAI-Organization 请求头。
	OpenAIOrganization = "openai_organization"
	// ChannelStatusCodeMapping 渠道级 HTTP 状态码映射 JSON 字符串。
	ChannelStatusCodeMapping = "status_code_mapping"
	// ChannelParamOverride 解析后的 map，用于合并进 JSON 请求体。
	ChannelParamOverride = "channel_param_override"
	// ChannelHeaderOverride 解析后的 map[string]string，用于额外请求头。
	ChannelHeaderOverride = "channel_header_override"
	// ParamOverrideRuntimeHeaders param_override operations 生成的完整 header_override 状态（小写键）；与 ParamOverrideHeadersExclusive 联用。
	ParamOverrideRuntimeHeaders  = "param_override_runtime_headers"
	ParamOverrideHeadersExclusive  = "param_override_headers_exclusive"

	// RequestAuditRecorder 挂载 *requestaudit.Recorder（Relay 审计，可选）。
	RequestAuditRecorder = "request_audit_recorder"

	// UpstreamRequestHeadersLog / UpstreamResponseHeadersLog 发往上游的 HTTP 头与上游响应头快照（脱敏后），写入消费日志 other。
	UpstreamRequestHeadersLog  = "upstream_request_headers_log"
	UpstreamResponseHeadersLog = "upstream_response_headers_log"
	// ClientResponseHeadersLog / ResponseBodyBytesLog 由 ConsumeLogCapture 中间件写入。
	ClientResponseHeadersLog = "client_response_headers_log"
	ResponseBodyBytesLog     = "response_body_bytes_log"
	// UpstreamRequestMetaLog / UpstreamResponseMetaLog 上游 HTTP 元信息（method/url/status）。
	UpstreamRequestMetaLog  = "upstream_request_meta_log"
	UpstreamResponseMetaLog = "upstream_response_meta_log"

	// TieredBillingSnapshot 预扣时冻结的 tiered_expr 计费快照（*billingexpr.BillingSnapshot）。
	TieredBillingSnapshot = "tiered_billing_snapshot"
	// BillingRequestInput 预扣时冻结的请求探测输入（*billingexpr.RequestInput）。
	BillingRequestInput = "billing_request_input"
	// FinalPreConsumedQuota tiered 预扣额度（结算失败时回退）。
	FinalPreConsumedQuota = "final_pre_consumed_quota"
)
