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
