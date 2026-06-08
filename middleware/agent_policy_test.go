package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/ctxkey"
)

// newCtx 构造一个最小 gin 上下文：带 UA、令牌策略在 context，userId/tenantId 为 0 以避开 DB 查询。
func newCtx(ua string, tokenPolicy *agentpolicy.Policy) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	c.Request = req
	c.Set(ctxkey.Id, 0)
	c.Set(ctxkey.UserTenantID, 0)
	c.Set(ctxkey.TokenId, 123)
	if tokenPolicy != nil {
		c.Set(ctxkey.TokenAgentPolicy, tokenPolicy)
	}
	return c, w
}

func init() { gin.SetMode(gin.TestMode) }

func TestAgentPolicyDisabledFastPath(t *testing.T) {
	agentpolicy.SetEnabled(false)
	c, w := newCtx("openclaw/1.0", &agentpolicy.Policy{AllowedClients: []string{"claude-code"}})
	AgentClientPolicy()(c)
	if c.IsAborted() || w.Code != http.StatusOK {
		t.Fatalf("disabled feature must pass through, got aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}

func TestAgentPolicyBlocksDisallowedClient(t *testing.T) {
	agentpolicy.SetEnabled(true)
	defer agentpolicy.SetEnabled(false)
	// 令牌仅允许 claude-code；UA 为 openclaw → 应 403。
	c, w := newCtx("openclaw/1.0", &agentpolicy.Policy{AllowedClients: []string{"claude-code"}})
	AgentClientPolicy()(c)
	if !c.IsAborted() || w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}

func TestAgentPolicyAllowsListedClient(t *testing.T) {
	agentpolicy.SetEnabled(true)
	defer agentpolicy.SetEnabled(false)
	c, w := newCtx("claude-cli/1.0 (external, cli)", &agentpolicy.Policy{AllowedClients: []string{"claude-code"}})
	AgentClientPolicy()(c)
	if c.IsAborted() || w.Code != http.StatusOK {
		t.Fatalf("expected pass, got aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}

func TestAgentPolicyRateLimit(t *testing.T) {
	agentpolicy.SetEnabled(true)
	defer agentpolicy.SetEnabled(false)
	// 令牌允许 claude-code 且限流 1 次/60 秒（内存限流器；非 Redis、非 Debug）。
	pol := &agentpolicy.Policy{
		AllowedClients: []string{"claude-code"},
		Default:        &agentpolicy.ClientRule{MaxRequests: 1, WindowSec: 60},
	}
	// 第一次放行。
	c1, w1 := newCtx("claude-cli/1.0 (cli)", pol)
	AgentClientPolicy()(c1)
	if c1.IsAborted() || w1.Code != http.StatusOK {
		t.Fatalf("first request should pass, code=%d", w1.Code)
	}
	// 第二次（同令牌同类型）应触发 429。
	c2, w2 := newCtx("claude-cli/1.0 (cli)", pol)
	AgentClientPolicy()(c2)
	if !c2.IsAborted() || w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be 429, got code=%d", w2.Code)
	}
}
