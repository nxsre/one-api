package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common/agentpolicy"
)

func setupAgentPolicyDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Token{}, &Tenant{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	orig := DB
	DB = db
	t.Cleanup(func() { DB = orig })
}

// runWithWatchdog 执行 fn，超时即判定为卡死（证明 InitAgentPolicyEnabled 不会阻塞启动）。
func runWithWatchdog(t *testing.T, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { fn(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("function hung (>%s)", d)
	}
}

func TestInitAgentPolicyEnabled(t *testing.T) {
	setupAgentPolicyDB(t)
	agentpolicy.SetEnabled(false)

	// 无任何策略 → 不启用，且必须秒回（不阻塞启动）。
	runWithWatchdog(t, 5*time.Second, InitAgentPolicyEnabled)
	if agentpolicy.Enabled() {
		t.Fatalf("expected disabled when no policy exists")
	}

	// 写入一个带非空策略的令牌后，再探测应启用。
	pol := &agentpolicy.Policy{AllowedClients: []string{"claude-code"}}
	if err := DB.Create(&Token{Id: 1, UserId: 1, Key: "k1", Name: "t1", AgentClientPolicy: pol}).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	agentpolicy.SetEnabled(false)
	runWithWatchdog(t, 5*time.Second, InitAgentPolicyEnabled)
	if !agentpolicy.Enabled() {
		t.Fatalf("expected enabled after a token policy exists")
	}
}

func TestUserAndTenantAgentPolicyRoundTrip(t *testing.T) {
	setupAgentPolicyDB(t)

	// 用户策略读写
	upol := &agentpolicy.Policy{AllowedClients: []string{"claude-code", "other"}}
	if err := DB.Create(&User{Id: 7, Username: "u7", AccessToken: "a7", AffCode: "f7", AgentClientPolicy: upol}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	got := GetUserAgentPolicy(7)
	if got == nil || !got.Allows("claude-code") || got.Allows("hermes") {
		t.Fatalf("user policy round-trip wrong: %+v", got)
	}
	if GetUserAgentPolicy(999) != nil {
		t.Fatalf("missing user must yield nil policy")
	}

	// 租户策略写入（SetTenantAgentPolicy）→ 读回
	if err := DB.Create(&Tenant{Id: 3, Name: "T3", Slug: "t3"}).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	tpol := &agentpolicy.Policy{Default: &agentpolicy.ClientRule{MaxRequests: 5, WindowSec: 1}}
	if err := SetTenantAgentPolicy(3, tpol); err != nil {
		t.Fatalf("set tenant policy: %v", err)
	}
	tgot := GetTenantAgentPolicy(3)
	if tgot == nil || tgot.EffectiveRule("anything").MaxRequests != 5 {
		t.Fatalf("tenant policy round-trip wrong: %+v", tgot)
	}
	// 清空（{}）应使读回为 nil（IsZero）。
	if err := SetTenantAgentPolicy(3, &agentpolicy.Policy{}); err != nil {
		t.Fatalf("clear tenant policy: %v", err)
	}
	if GetTenantAgentPolicy(3) != nil {
		t.Fatalf("cleared tenant policy should read back nil")
	}
}
