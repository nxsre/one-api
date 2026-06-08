/**
 * 嵌入同源 /nacos-ui/ 的 console-ui-next 使用 localStorage key `token` 存 accessToken。
 * 主站退出 / 会话失效时需一并清除，避免 Nacos 仍显示已登录。
 */
export const NACOS_EMBEDDED_CONSOLE_TOKEN_KEY = 'token';

export function clearNacosEmbeddedConsoleLocalSession() {
  try {
    localStorage.removeItem(NACOS_EMBEDDED_CONSOLE_TOKEN_KEY);
  } catch {
    /* ignore */
  }
}
