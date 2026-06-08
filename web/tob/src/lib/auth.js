import { API } from '@/api/client';
import { encryptLoginPayloadAES } from '@/lib/loginPasswordAes';
import { isSecurePasswordLoginEnabled, normalizeStatusResponse } from '@/lib/systemStatus';

/**
 * 构建登录 POST 体（与 default secureLogin.js 一致）
 */
export async function buildLoginPayload(username, passwordPlain, captcha, loginProof) {
  const buildCaptchaFields = async (encKey) => {
    if (!captcha?.captcha_id) return {};
    if (captcha.mode === 'slide' && captcha.captcha_point) {
      const payload = JSON.stringify(captcha.captcha_point);
      const base = { captcha_id: captcha.captcha_id, captcha_mode: 'slide' };
      if (encKey) {
        return {
          ...base,
          captcha_dots_enc: await encryptLoginPayloadAES(encKey, payload),
        };
      }
      return { ...base, captcha_dots_enc: payload };
    }
    if (captcha.mode === 'rotate' && captcha.captcha_angle != null) {
      const payload = JSON.stringify({ angle: captcha.captcha_angle });
      const base = { captcha_id: captcha.captcha_id, captcha_mode: 'rotate' };
      if (encKey) {
        return {
          ...base,
          captcha_dots_enc: await encryptLoginPayloadAES(encKey, payload),
        };
      }
      return { ...base, captcha_dots_enc: payload };
    }
    if (captcha?.captcha_clicks?.length > 0) {
      const payload = JSON.stringify(captcha.captcha_clicks);
      const base = { captcha_id: captcha.captcha_id, captcha_mode: 'click' };
      if (encKey) {
        return {
          ...base,
          captcha_dots_enc: await encryptLoginPayloadAES(encKey, payload),
        };
      }
      return { ...base, captcha_dots_enc: payload };
    }
    return {};
  };

  if (!isSecurePasswordLoginEnabled()) {
    const out = { username, password: passwordPlain };
    return { ...out, ...(await buildCaptchaFields(null)) };
  }

  let d;
  if (
    loginProof?.id &&
    loginProof.sig != null &&
    loginProof.ts != null &&
    loginProof.encKey
  ) {
    d = {
      login_request_id: loginProof.id,
      login_request_ts: loginProof.ts,
      login_request_sig: loginProof.sig,
      login_enc_key: loginProof.encKey,
    };
  } else {
    const pr = await API.get('/api/user/login/request-proof');
    const body = pr.data?.data;
    if (!pr.data?.success || !body?.login_request_id || !body?.login_enc_key) {
      const msg =
        typeof pr.data?.message === 'string' && pr.data.message.trim()
          ? pr.data.message.trim()
          : '无法获取登录凭证，请刷新后重试';
      throw new Error(msg);
    }
    d = body;
  }

  const enc = await encryptLoginPayloadAES(d.login_enc_key, passwordPlain);
  const out = {
    username,
    password: enc,
    login_request_id: d.login_request_id,
    login_request_ts: d.login_request_ts,
    login_request_sig: d.login_request_sig,
  };
  return { ...out, ...(await buildCaptchaFields(d.login_enc_key)) };
}

export async function fetchSystemStatus() {
  const res = await API.get('/api/status');
  const payload = normalizeStatusResponse(res.data);
  if (payload && Object.keys(payload).length > 0) {
    localStorage.setItem('status', JSON.stringify(payload));
  }
  return payload;
}

export async function login(username, password, captcha, loginProof) {
  const body = await buildLoginPayload(username, password, captcha, loginProof);
  const res = await API.post('/api/user/login', body);
  return res.data;
}

export async function login2FA(code) {
  const res = await API.post('/api/user/login/2fa', { code: code.trim() });
  return res.data;
}

export function saveUserSession(data) {
  localStorage.setItem('user', JSON.stringify(data));
}

export function clearUserSession() {
  localStorage.removeItem('user');
}

export function getStoredUser() {
  try {
    const raw = localStorage.getItem('user');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function postLoginPath(user) {
  if (user?.require_force_2fa_setup) return '/settings';
  return '/overview';
}

export async function logout() {
  try {
    await API.get('/api/user/logout');
  } finally {
    clearUserSession();
  }
}
