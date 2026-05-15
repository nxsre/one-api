/**
 * AES-256-GCM，与服务端 common.DecryptLoginPayloadAES 对应：Base64(nonce||ciphertext+tag)。
 * @param {string} keyB64 32 字节密钥（Base64，来自 login_enc_key）
 * @param {string} plaintext
 * @returns {Promise<string>} Base64 密文
 */
export async function encryptLoginPayloadAES(keyB64, plaintext) {
  const keyRaw = Uint8Array.from(atob(keyB64), (c) => c.charCodeAt(0));
  if (keyRaw.length !== 32) {
    throw new Error('AES_KEY_INVALID');
  }
  const key = await crypto.subtle.importKey(
    'raw',
    keyRaw,
    { name: 'AES-GCM' },
    false,
    ['encrypt'],
  );
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const data = new TextEncoder().encode(plaintext);
  const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, data);
  const out = new Uint8Array(iv.length + ct.byteLength);
  out.set(iv, 0);
  out.set(new Uint8Array(ct), iv.length);
  let binary = '';
  for (let i = 0; i < out.length; i += 1) {
    binary += String.fromCharCode(out[i]);
  }
  return btoa(binary);
}
