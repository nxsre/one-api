/**
 * 计算控制台 servlet 上下文路径。
 * 嵌入 one-api 的 /nacos-ui/legacy/ 时，API 须走 /nacos/ 前缀，而非 /nacos-ui/v3。
 */
export function getContextPath() {
  const path = window.location.pathname;
  if (path === '/nacos-ui' || path.startsWith('/nacos-ui/')) {
    return '/nacos/';
  }
  return path.replace(/\/(next|legacy)(\/.*)?$/, '/') || '/';
}
