/**
 * 静态控制台（next/legacy）所在路径前缀，用于页面跳转。
 * 使用 HashRouter 时 pathname 恒为 /nacos-ui，相对链接 ../legacy 会错误解析到站点根 /legacy/。
 */
export function getNacosConsoleStaticBase(): string {
  const path = window.location.pathname;
  if (path === '/nacos-ui' || path.startsWith('/nacos-ui/')) {
    return '/nacos-ui/';
  }
  const base = path.replace(/\/(next|legacy)(\/.*)?$/, '/');
  return base === '' ? '/' : base;
}

/** 旧版控制台入口（与新版 sibling，挂在同一路径前缀下）。 */
export function getLegacyConsoleHref(): string {
  return `${getNacosConsoleStaticBase()}legacy/`;
}
