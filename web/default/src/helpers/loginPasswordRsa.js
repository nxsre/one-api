import JSEncrypt from 'jsencrypt';

/**
 * RSA PKCS#1 v1.5 加密，与服务端 rsa.DecryptPKCS1v15 对应。
 * @param {string} pemPublicKey PKIX PEM（/api/status.login_password_rsa_public_key）
 * @param {string} plaintext 原始密码
 * @returns {string} Base64 密文
 */
export function encryptLoginPasswordRSA(pemPublicKey, plaintext) {
  const enc = new JSEncrypt();
  enc.setPublicKey(pemPublicKey);
  const out = enc.encrypt(plaintext);
  if (!out) {
    throw new Error('RSA_ENCRYPT_FAILED');
  }
  return out;
}
