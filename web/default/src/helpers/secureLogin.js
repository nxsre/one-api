import { API } from './api';
import { encryptLoginPayloadAES } from './loginPasswordAes';
import { isSecurePasswordLoginEnabled } from './utils';

/**
 * 构建登录 POST 体。
 * 安全登录关闭：明文 password（依赖 HTTPS）；开启：proof + AES。
 */
export async function buildLoginPayload(username, passwordPlain, captcha, loginProof) {
  if (!isSecurePasswordLoginEnabled()) {
    const out = { username, password: passwordPlain };
    if (
      captcha?.captcha_clicks &&
      captcha.captcha_clicks.length > 0 &&
      captcha?.captcha_id
    ) {
      out.captcha_id = captcha.captcha_id;
      out.captcha_dots_enc = JSON.stringify(captcha.captcha_clicks);
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
  if (
    captcha?.captcha_clicks &&
    captcha.captcha_clicks.length > 0 &&
    captcha?.captcha_id
  ) {
    out.captcha_id = captcha.captcha_id;
    out.captcha_dots_enc = await encryptLoginPayloadAES(
      d.login_enc_key,
      JSON.stringify(captcha.captcha_clicks),
    );
  }
  return out;
}
