import { API } from 'utils/api';
import JSEncrypt from 'jsencrypt';

export function encryptLoginPasswordRSA(pemPublicKey, plaintext) {
  const enc = new JSEncrypt();
  enc.setPublicKey(pemPublicKey);
  const out = enc.encrypt(plaintext);
  if (!out) {
    throw new Error('RSA_ENCRYPT_FAILED');
  }
  return out;
}

/**
 * @param {string} username
 * @param {string} passwordPlain
 * @param {{ captcha_id?: string, captcha_clicks?: { x: number, y: number }[] }} [captcha]
 * @param {{ id: string, ts: number, sig: string } | null} [loginProof]
 */
export async function buildLoginPayload(username, passwordPlain, captcha, loginProof) {
  const st = await API.get('/api/status');
  const pem = st.data?.data?.login_password_rsa_public_key;
  if (!pem) {
    const base = { username, password: passwordPlain };
    if (captcha?.captcha_id) base.captcha_id = captcha.captcha_id;
    return base;
  }
  let d;
  if (
    loginProof &&
    loginProof.id &&
    loginProof.sig != null &&
    loginProof.ts != null
  ) {
    d = {
      login_request_id: loginProof.id,
      login_request_ts: loginProof.ts,
      login_request_sig: loginProof.sig,
    };
  } else {
    const pr = await API.get('/api/user/login/request-proof');
    d = pr.data?.data;
    if (!pr.data?.success || !d?.login_request_id) {
      const msg =
        typeof pr.data?.message === 'string' && pr.data.message.trim()
          ? pr.data.message.trim()
          : '无法获取登录凭证，请刷新后重试';
      throw new Error(msg);
    }
  }
  const enc = encryptLoginPasswordRSA(pem, passwordPlain);
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
    out.captcha_dots_enc = encryptLoginPasswordRSA(
      pem,
      JSON.stringify(captcha.captcha_clicks)
    );
  }
  return out;
}
