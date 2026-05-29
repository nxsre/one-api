import { API } from './api';
import { encryptLoginPayloadAES } from './loginPasswordAes';
import { isSecurePasswordLoginEnabled } from './utils';

// Normalize the captcha solution across modes. `captcha.answer` is the
// mode-specific object (click: [{x,y}...]; slide: {x,y}; rotate: {angle}).
// `captcha.captcha_clicks` is still accepted for click for backward-compat.
function resolveCaptchaAnswer(captcha) {
  if (!captcha || !captcha.captcha_id) return null;
  const mode = captcha.mode || 'click';
  let answer = captcha.answer;
  if (answer == null && Array.isArray(captcha.captcha_clicks)) {
    answer = captcha.captcha_clicks;
  }
  if (answer == null) return null;
  if (Array.isArray(answer) && answer.length === 0) return null;
  return { mode, answer, id: captcha.captcha_id };
}

/**
 * 构建登录 POST 体。
 * 安全登录关闭：明文 password（依赖 HTTPS）；开启：proof + AES。
 */
export async function buildLoginPayload(username, passwordPlain, captcha, loginProof) {
  const resolved = resolveCaptchaAnswer(captcha);

  if (!isSecurePasswordLoginEnabled()) {
    const out = { username, password: passwordPlain };
    if (resolved) {
      out.captcha_id = resolved.id;
      out.captcha_mode = resolved.mode;
      out.captcha_dots_enc = JSON.stringify(resolved.answer);
    }
    return out;
  }

  let d;
  if (
    loginProof &&
    loginProof.id &&
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
  if (resolved) {
    out.captcha_id = resolved.id;
    out.captcha_mode = resolved.mode;
    out.captcha_dots_enc = await encryptLoginPayloadAES(
      d.login_enc_key,
      JSON.stringify(resolved.answer),
    );
  }
  return out;
}
