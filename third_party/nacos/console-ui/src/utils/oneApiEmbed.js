/**
 * 控制台以同源路径 /nacos-ui/ 嵌入 One API 时的认证跳转（与 console-ui-next 的 one-api-embed-auth 对齐）。
 */

export function isEmbeddedUnderOneApi() {
  const path = window.location.pathname;
  return path === '/nacos-ui' || path.startsWith('/nacos-ui/');
}

/** 登录完成后回到控制台（含 Hash 路由） */
export function buildNacosReturnUrl() {
  return `${window.location.pathname}${window.location.search}${window.location.hash}`;
}

export function redirectToOneApiLoginPage() {
  if (!isEmbeddedUnderOneApi()) {
    window.location.hash = '#/login';
    return;
  }
  const ret = buildNacosReturnUrl() || '/nacos-ui/';
  const redirect = encodeURIComponent(ret);
  window.location.assign(`${window.location.origin}/login?redirect=${redirect}`);
}

/** 与主站「退出登录」一致：清除 Gin Session（需携带 Cookie） */
export async function fetchOneApiLogoutSession() {
  if (!isEmbeddedUnderOneApi()) {
    return;
  }
  try {
    await fetch(`${window.location.origin}/api/user/logout`, {
      method: 'GET',
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    });
  } catch {
    /* ignore */
  }
}

/** 跳转主站「更新用户信息」页修改密码，完成后可经 redirect 回到控制台 */
export function openOneApiUserProfileEdit() {
  const base = `${window.location.origin}/user/edit`;
  if (!isEmbeddedUnderOneApi()) {
    window.location.assign(base);
    return;
  }
  const redirect = encodeURIComponent(buildNacosReturnUrl() || '/nacos-ui/');
  window.location.assign(`${base}?redirect=${redirect}`);
}
