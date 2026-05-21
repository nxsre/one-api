import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getApiErrorMessage } from '@/api/client';
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
import { isLoginMathCaptchaEnabled } from '@/lib/systemStatus';
import '@/styles/login.css';
import '@/styles/login-captcha.css';

export default function LoginPage() {
  const navigate = useNavigate();
  const { login: setUser } = useUser();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
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
          if (cached) mergeStatus(JSON.parse(cached));
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
        if (mathCaptchaEnabled) {
          void captcha.loadCaptcha();
        }
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
      if (mathCaptchaEnabled) {
        void captcha.loadCaptcha();
      }
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
      <div className="tob-login-card">
        <div className="tob-login-logo">
          <div className="tob-logo-icon">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="#fff">
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
            </svg>
          </div>
          <span className="tob-logo-text">TokenHub</span>
          <span className="tob-logo-badge">ToB</span>
        </div>

        {step === 'credentials' ? (
          <>
            <h1 className="tob-login-title">登录控制台</h1>
            <p className="tob-login-sub">使用 one-api 账号登录，支持点击验证与 MFA</p>
            {error && <div className="tob-error">{error}</div>}
            <form onSubmit={handleLogin}>
              <div className="tob-field">
                <label>用户名</label>
                <input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  required
                />
              </div>
              <div className="tob-field">
                <label>密码</label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                />
              </div>

              {mathCaptchaEnabled && (
                <div className="tob-captcha-segment">
                  <div className="tob-captcha-segment-title">
                    <span>安全验证</span>
                    <span className="tob-captcha-segment-sub">
                      {captcha.clicks.length} / {captcha.dotNum || '—'}
                    </span>
                  </div>
                  <button
                    type="button"
                    className={`tob-btn-captcha${captcha.isComplete ? ' done' : ''}`}
                    onClick={() => captcha.setModalOpen(true)}
                  >
                    {captcha.isComplete ? '验证已完成' : '打开验证码'}
                  </button>
                  {captcha.loadError && (
                    <p className="tob-error" style={{ marginTop: 8, marginBottom: 0 }}>
                      {captcha.loadError}
                    </p>
                  )}
                </div>
              )}

              <button type="submit" className="tob-btn-primary" disabled={busy}>
                {busy ? '登录中…' : '登录'}
              </button>
            </form>
          </>
        ) : (
          <div className="tob-mfa-overlay">
            <h1 className="tob-login-title">两步验证</h1>
            <p className="tob-mfa-hint">请输入认证器 6 位验证码或 8 位备用码</p>
            {error && <div className="tob-error">{error}</div>}
            <form onSubmit={handle2FA}>
              <div className="tob-field">
                <label>验证码 / 备用码</label>
                <input
                  value={twoFACode}
                  onChange={(e) => setTwoFACode(e.target.value)}
                  autoComplete="one-time-code"
                  required
                />
              </div>
              <button type="submit" className="tob-btn-primary" disabled={busy}>
                {busy ? '验证中…' : '确认登录'}
              </button>
              <button
                type="button"
                className="tob-btn-primary"
                style={{ marginTop: 8, background: 'var(--surface2)', color: 'var(--text)' }}
                onClick={() => {
                  setStep('credentials');
                  setTwoFACode('');
                  setError('');
                }}
              >
                返回
              </button>
            </form>
          </div>
        )}
      </div>

      <LoginCaptchaModal
        open={captcha.modalOpen}
        onClose={() => captcha.setModalOpen(false)}
        thumbSrc={captcha.thumbSrc}
        masterSrc={captcha.masterSrc}
        loading={captcha.loading}
        loadError={captcha.loadError}
        dotNum={captcha.dotNum}
        clicks={captcha.clicks}
        masterSize={captcha.masterSize}
        onMasterLoad={(ev) =>
          captcha.setMasterSize({
            w: ev.target.naturalWidth,
            h: ev.target.naturalHeight,
          })
        }
        onMasterClick={captcha.onMasterClick}
        onClear={() => captcha.setClicks([])}
        onRefresh={() => void captcha.loadCaptcha()}
      />
    </div>
  );
}
