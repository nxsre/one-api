/** 与 default 主题一致：登录密码 AES-GCM 加密 */
export async function encryptLoginPayloadAES(keyB64, plaintext) {
  const keyRaw = Uint8Array.from(atob(keyB64), (c) => c.charCodeAt(0));
  const key = await crypto.subtle.importKey('raw', keyRaw, { name: 'AES-GCM' }, false, ['encrypt']);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encoded = new TextEncoder().encode(plaintext);
  const cipher = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, encoded);
  const combined = new Uint8Array(iv.length + cipher.byteLength);
  combined.set(iv, 0);
  combined.set(new Uint8Array(cipher), iv.length);
  let binary = '';
  combined.forEach((b) => {
    binary += String.fromCharCode(b);
  });
  return btoa(binary);
}
