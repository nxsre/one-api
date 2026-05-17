package ctxkey

const (
	Config            = "config"
	Id                = "id"
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
)
