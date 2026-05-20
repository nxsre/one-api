package meta

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type Meta struct {
	Mode         int
	ChannelType  int
	ChannelId    int
	TokenId      int
	TokenName    string
	UserId       int
	UserTenantId int
	Group        string
	UserGroup    string
	UsingGroup   string
	ModelMapping map[string]string
	// BaseURL is the proxy url set in the channel config
	BaseURL  string
	// ChannelKey is the upstream credential from channel config (not the client's sk- token).
	ChannelKey string
	APIKey     string
	APIType  int
	Config   model.ChannelConfig
	IsStream bool
	// OriginModelName is the model name from the raw user request
	OriginModelName string
	// ActualModelName is the model name after mapping
	ActualModelName string
	RequestURLPath  string
	// OverrideRequestURL if non-empty, adaptor DoRequestHelper uses this instead of GetRequestURL (native protocol relay).
	OverrideRequestURL string
	PromptTokens       int // only for DoResponse
	ForcedSystemPrompt string
	StartTime          time.Time
}

func GetByContext(c *gin.Context) *Meta {
	meta := Meta{
		Mode:               relaymode.GetByPath(c.Request.URL.Path),
		ChannelType:        c.GetInt(ctxkey.Channel),
		ChannelId:          c.GetInt(ctxkey.ChannelId),
		TokenId:            c.GetInt(ctxkey.TokenId),
		TokenName:          c.GetString(ctxkey.TokenName),
		UserId:             c.GetInt(ctxkey.Id),
		UserTenantId:       c.GetInt(ctxkey.UserTenantID),
		Group:              c.GetString(ctxkey.Group),
		UserGroup:          c.GetString(ctxkey.UserGroup),
		UsingGroup:         c.GetString(ctxkey.UsingGroup),
		ModelMapping:       c.GetStringMapString(ctxkey.ModelMapping),
		OriginModelName:    c.GetString(ctxkey.RequestModel),
		BaseURL:            c.GetString(ctxkey.BaseURL),
		ChannelKey:         c.GetString(ctxkey.ChannelKey),
		APIKey:             bearerToken(c.Request.Header.Get("Authorization")),
		RequestURLPath:     c.Request.URL.String(),
		ForcedSystemPrompt: c.GetString(ctxkey.SystemPrompt),
		StartTime:          time.Now(),
	}
	cfg, ok := c.Get(ctxkey.Config)
	if ok {
		meta.Config = cfg.(model.ChannelConfig)
	}
	if meta.BaseURL == "" {
		meta.BaseURL = channeltype.DefaultBaseURL(meta.ChannelType)
	}
	meta.APIType = channeltype.ToAPIType(meta.ChannelType)
	return &meta
}

func bearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if len(authHeader) >= 7 && strings.EqualFold(authHeader[:7], "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return authHeader
}
