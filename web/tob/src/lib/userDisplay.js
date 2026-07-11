/** 从用户信息取展示用全名 */
export function getUserDisplayName(user) {
  const name = String(user?.display_name || user?.username || '').trim();
  return name || '用户';
}

/**
 * 取姓（头像用）：中文名取首字；含空格时取第一段首字；否则取首字符大写。
 */
export function getUserSurname(user) {
  const raw = String(user?.display_name || user?.username || '').trim();
  if (!raw) return '用';

  if (/[\u4e00-\u9fff\u3400-\u4dbf]/.test(raw)) {
    return raw.charAt(0);
  }

  const segment = raw.split(/\s+/).filter(Boolean)[0] || raw;
  return segment.charAt(0).toUpperCase();
}
