package common

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

// NewGinSessionStore 返回会话存储。当前 one-api 使用 Cookie 存储（与原版一致）；
// Redis 仍用于缓存、登录防重放等；若需 Redis Session 可后续引入兼容 redigo 的存储实现。
func NewGinSessionStore() (sessions.Store, error) {
	logger.SysLog("session store: cookie")
	return cookie.NewStore([]byte(config.SessionSecret)), nil
}
