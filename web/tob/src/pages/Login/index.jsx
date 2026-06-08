import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getApiErrorMessage } from '@/api/client';
import LogoMark from '@/components/brand/LogoMark';
import LoginBrandIllustration from '@/components/login/LoginBrandIllustration';
import LoginCaptchaModal from '@/components/login/LoginCaptchaModal';
import { useUser } from '@/context/UserContext';
import { useLoginCaptcha } from '@/hooks/useLoginCaptcha';
import {
  fetchSystemStatus,
  login,
  login2FA,
  postLoginPath,
  saveUserSession,
} from '@/lib/auth';
import { isLoginMathCaptchaEnabled, normalizeStatusResponse } from '@/lib/systemStatus';
import '@/styles/login.css';
import '@/styles/login-captcha.css';

export default function LoginPage() {
  const navigate = useNavigate();
  const { login: setUser } = useUser();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [twoFACode, setTwoFACode] = useState('');
  const [step, setStep] = useState('credentials');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState({});

  const mathCaptchaEnabled = isLoginMathCaptchaEnabled(status);
  const captcha = useLoginCaptcha({ mathCaptchaEnabled });

  const mergeStatus = useCallback((data) => {
    if (!data) return;
    setStatus(data);
    try {
      localStorage.setItem('status', JSON.stringify(data));
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    fetchSystemStatus()
      .then(mergeStatus)
      .catch(() => {
        try {
          const cached = localStorage.getItem('status');
          if (cached) mergeStatus(normalizeStatusResponse(JSON.parse(cached)));
        } catch {
          /* ignore */
        }
      });
  }, [mergeStatus]);

  const finishLogin = (data) => {
    saveUserSession(data);
    setUser(data);
    if (data?.require_force_2fa_setup) {
      navigate('/settings', { replace: true });
      return;
    }
    navigate(postLoginPath(data), { replace: true });
  };

  const handleLogin = async (e) => {
    e.preventDefault();
    setError('');

    if (!username.trim() || !password) {
      setError('请输入用户名和密码');
      return;
    }

    const check = captcha.validateBeforeLogin();
    if (!check.ok) {
      setError(check.message);
      if (check.openModal) captcha.setModalOpen(true);
      return;
    }

    const captchaPayload = captcha.getCaptchaPayload();
    const proof = mathCaptchaEnabled ? captcha.getLoginProof() : null;

    setBusy(true);
    try {
      const data = await login(username.trim(), password, captchaPayload, proof);
      if (!data?.success) {
        setError(data?.message || '登录失败');
        if (mathCaptchaEnabled) void captcha.loadCaptcha();
        return;
      }
      if (data.data?.require_2fa) {
        setStep('mfa');
        setTwoFACode('');
        return;
      }
      finishLogin(data.data);
    } catch (err) {
      setError(getApiErrorMessage(err));
      if (mathCaptchaEnabled) void captcha.loadCaptcha();
    } finally {
      setBusy(false);
    }
  };

  const handle2FA = async (e) => {
    e.preventDefault();
    setError('');
    if (!twoFACode.trim()) {
      setError('请输入验证码或备用码');
      return;
    }
    setBusy(true);
    try {
      const data = await login2FA(twoFACode);
      if (!data?.success) {
        setError(data?.message || '验证失败');
        return;
      }
      finishLogin(data.data);
    } catch (err) {
      setError(getApiErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="tob-login-page">
      <div className="tob-login-shell">
        <aside className="tob-login-brand">
          <div className="tob-login-brand-inner">
            <div className="tob-login-brand-logo">
              <div className="tob-logo-icon">
                <LogoMark size={24} fill="#fff" />
              </div>
              <span className="tob-logo-text">TokenHub</span>
            </div>
            <h1 className="tob-login-headline">一站式接入，释放大模型无限潜能</h1>
            <p className="tob-login-slogan">更强模型 · 更低成本 · 更易落地</p>
            <p className="tob-login-desc">
              聚合 DeepSeek、Kimi、GLM 等数十款顶尖大模型，以统一 API 接口、灵活计费与全链路监控，帮助企业和开发者以最低成本快速接入
              AI 能力，专注构建下一代智能应用。
            </p>
            <LoginBrandIllustration />
          </div>
        </aside>

        <section className="tob-login-panel">
          <div className="tob-login-card">
            {step === 'credentials' ? (
              <>
                <h2 className="tob-login-card-title">用户登录</h2>
                {error && <div className="tob-error">{error}</div>}
                <form onSubmit={handleLogin}>
                  <div className="tob-field">
                    <label htmlFor="login-username">用户名</label>
                    <input
                      id="login-username"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      placeholder="请输入您的用户名"
                      autoComplete="username"
                      required
                    />
                  </div>
                  <div className="tob-field">
                    <label htmlFor="login-password">密码</label>
                    <div className="tob-field-input-wrap">
                      <input
                        id="login-password"
                        className="tob-pwd-input"
                        type={showPassword ? 'text' : 'password'}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder="请输入您的密码"
                        autoComplete="current-password"
                        required
                      />
                      <button
                        type="button"
                        className="tob-pwd-toggle"
                        onClick={() => setShowPassword((v) => !v)}
                        aria-label={showPassword ? '隐藏密码' : '显示密码'}
                        tabIndex={-1}
                      >
                        {showPassword ? (
                          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                            <line x1="1" y1="1" x2="23" y2="23" />
                          </svg>
                        ) : (
                          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                            <circle cx="12" cy="12" r="3" />
                          </svg>
                        )}
                      </button>
                    </div>
                  </div>

                  {mathCaptchaEnabled && (
                    <div className="tob-captcha-segment">
                      <div className="tob-captcha-segment-title">
                        <span>安全验证</span>
                        <span className="tob-captcha-segment-sub">
                          {captcha.mode === 'slide'
                            ? captcha.isComplete
                              ? '已完成'
                              : '滑动拼图'
                            : captcha.mode === 'rotate'
                              ? captcha.isComplete
                                ? '已完成'
                                : '旋转对齐'
                              : `${captcha.clicks.length} / ${captcha.dotNum || '—'}`}
                        </span>
                      </div>
                      <button
                        type="button"
                        className={`tob-btn-captcha${captcha.isComplete ? ' done' : ''}`}
                        onClick={() => captcha.setModalOpen(true)}
                      >
                        {captcha.isComplete ? '验证已完成' : '完成安全验证'}
                      </button>
                      {captcha.loadError && (
                        <p className="tob-error" style={{ marginTop: 8, marginBottom: 0 }}>
                          {captcha.loadError}
                        </p>
                      )}
                    </div>
                  )}

                  <button type="submit" className="tob-btn-primary" disabled={busy}>
                    {busy ? '登录中…' : '登 录'}
                  </button>
                </form>
              </>
            ) : (
              <>
                <h2 className="tob-login-card-title">两步验证</h2>
                <p className="tob-mfa-hint">请输入认证器 6 位验证码或 8 位备用码</p>
                {error && <div className="tob-error">{error}</div>}
                <form onSubmit={handle2FA}>
                  <div className="tob-field">
                    <label htmlFor="login-2fa">验证码 / 备用码</label>
                    <input
                      id="login-2fa"
                      value={twoFACode}
                      onChange={(e) => setTwoFACode(e.target.value)}
                      placeholder="请输入动态验证码"
                      autoComplete="one-time-code"
                      required
                    />
                  </div>
                  <button type="submit" className="tob-btn-primary" disabled={busy}>
                    {busy ? '验证中…' : '确认登录'}
                  </button>
                  <button
                    type="button"
                    className="tob-btn-ghost"
                    onClick={() => {
                      setStep('credentials');
                      setTwoFACode('');
                      setError('');
                    }}
                  >
                    返回上一步
                  </button>
                </form>
              </>
            )}
          </div>
        </section>
      </div>

      <LoginCaptchaModal
        open={captcha.modalOpen}
        onClose={() => captcha.setModalOpen(false)}
        mode={captcha.mode}
        thumbSrc={captcha.thumbSrc}
        masterSrc={captcha.masterSrc}
        loading={captcha.loading}
        loadError={captcha.loadError}
        thumbSize={captcha.thumbSize}
        slideMeta={captcha.slideMeta}
        masterSize={captcha.masterSize}
        onMasterLoad={(ev) =>
          captcha.setMasterSize({
            w: ev.target.naturalWidth,
            h: ev.target.naturalHeight,
          })
        }
        onRefresh={() => void captcha.refreshCaptcha()}
        onRotateConfirm={captcha.onRotateConfirm}
        onClickConfirm={captcha.onClickConfirm}
        onSlideConfirm={captcha.onSlideConfirm}
        captchaRef={captcha.captchaRef}
      />
    </div>
  );
}
