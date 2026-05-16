/** 与 Nacos console-ui-next `src/lib/storage.ts` 中 THEME_KEY 一致，便于同源下与 /nacos-ui 切换状态同步 */
export const NACOS_THEME_STORAGE_KEY = 'nacos_theme';

export function getNacosTheme() {
  try {
    const t = localStorage.getItem(NACOS_THEME_STORAGE_KEY);
    if (t === 'dark' || t === 'light') return t;
  } catch {
    /* ignore */
  }
  return 'light';
}

/** 与 Nacos 一致：在 documentElement 上挂 `light` | `dark` */
export function applyNacosThemeClass(theme) {
  const t = theme === 'dark' ? 'dark' : 'light';
  if (typeof document === 'undefined') return;
  document.documentElement.classList.remove('light', 'dark');
  document.documentElement.classList.add(t);
}

export function setNacosTheme(theme) {
  const t = theme === 'dark' ? 'dark' : 'light';
  try {
    localStorage.setItem(NACOS_THEME_STORAGE_KEY, t);
  } catch {
    /* ignore */
  }
  applyNacosThemeClass(t);
  try {
    window.dispatchEvent(new Event('one-api:nacos-theme-changed'));
  } catch {
    /* ignore */
  }
}

export function toggleNacosTheme() {
  const next = getNacosTheme() === 'dark' ? 'light' : 'dark';
  setNacosTheme(next);
  return next;
}
