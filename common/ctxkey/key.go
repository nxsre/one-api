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
)
