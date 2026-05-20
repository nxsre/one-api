package common

import (
	"time"

	"github.com/songquanpeng/one-api/common/env"
)

// Force2FAForAllUsers 为 true 时，未启用 TOTP 的用户无法使用控制台与 API Key 转发。
var Force2FAForAllUsers bool

// LoginMathCaptchaEnabled 为 true 且未启用 Turnstile 时，登录可要求点击验证码（还受后台选项 LoginCaptchaEnabled 约束）。
var LoginMathCaptchaEnabled bool

// LoginBruteTrustXForwardedFor 为 true 时，登录爆破计数优先取 X-Forwarded-For 最左一跳（仅可信代理后开启）。
var LoginBruteTrustXForwardedFor bool

func InitSecurityEnv() {
	Force2FAForAllUsers = env.BoolAlways("force_2fa_for_all_users")
	// 未显式关闭时默认开启（与 Turnstile 互斥，由 GetStatus 的 login_math_captcha 综合位控制展示）
	LoginMathCaptchaEnabled = env.BoolAlways("login_math_captcha_enabled")
	LoginBruteTrustXForwardedFor = env.BoolAlways("login_brute_trust_x_forwarded_for")

	if n := env.IntAlways("login_brute_ip_fail_max"); n > 0 {
		LoginBruteIPFailMax = n
	}
	if n := env.IntAlways("login_brute_pair_fail_max"); n > 0 {
		LoginBrutePairFailMax = n
	}
	if n := env.Int64Always("login_brute_fail_window_sec"); n > 0 {
		LoginBruteFailWindow = time.Duration(n) * time.Second
	}
	if n := env.Int64Always("login_brute_lock_duration_sec"); n > 0 {
		LoginBruteLockDuration = time.Duration(n) * time.Second
	}
}
