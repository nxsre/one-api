import dayjs from 'dayjs';
import { API } from '@/api/client';

export const TOKEN_PAGE_SIZE = 10;

const QUOTA_DIVISOR = 500_000;

export const TOKEN_STATUS = {
  1: { label: '活跃', tagClass: 'tag-green' },
  2: { label: '已禁用', tagClass: 'tag-gray' },
  3: { label: '已过期', tagClass: 'tag-amber' },
  4: { label: '已耗尽', tagClass: 'tag-gray' },
};

export const TOKEN_TABS = [
  { key: 'all', label: '全部 Key' },
  { key: 'active', label: '活跃' },
  { key: 'disabled', label: '已禁用' },
  { key: 'expired', label: '已过期' },
  { key: 'depleted', label: '已耗尽' },
];

const TAB_STATUS = {
  active: 1,
  disabled: 2,
  expired: 3,
  depleted: 4,
};

export function filterTokensByTab(tokens, tabKey) {
  const status = TAB_STATUS[tabKey];
  if (!status) return tokens.filter((t) => !t.deleted);
  return tokens.filter((t) => !t.deleted && t.status === status);
}

export function getTokenStatusMeta(status) {
  return TOKEN_STATUS[Number(status)] || { label: '未知', tagClass: 'tag-gray' };
}

export async function fetchTokenPage(page = 0, order = '') {
  const res = await API.get(`/api/token/?p=${page}&order=${encodeURIComponent(order || '')}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载令牌失败');
  }
  const data = res.data.data;
  return Array.isArray(data) ? data.filter((t) => !t.deleted) : [];
}

export async function searchTokens(keyword) {
  const res = await API.get(`/api/token/search?keyword=${encodeURIComponent(keyword)}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '搜索失败');
  }
  const data = res.data.data;
  return Array.isArray(data) ? data.filter((t) => !t.deleted) : [];
}

export async function deleteToken(id) {
  const res = await API.delete(`/api/token/${id}/`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '删除失败');
  }
  return res.data;
}

export async function setTokenStatus(id, status) {
  const res = await API.put('/api/token/?status_only=true', { id, status });
  if (!res.data?.success) {
    throw new Error(res.data?.message || '操作失败');
  }
  return res.data.data;
}

export async function fetchTokenById(id) {
  const res = await API.get(`/api/token/${id}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载令牌失败');
  }
  return res.data.data;
}

export async function createToken(payload) {
  const res = await API.post('/api/token/', payload);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '创建失败');
  }
  return res.data;
}

export async function updateToken(payload) {
  const res = await API.put('/api/token/', payload);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '更新失败');
  }
  return res.data;
}

export function toDatetimeLocalValue(timestamp) {
  if (timestamp == null || timestamp === -1) return '';
  const d = new Date(Number(timestamp) * 1000);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const h = String(d.getHours()).padStart(2, '0');
  const min = String(d.getMinutes()).padStart(2, '0');
  return `${y}-${m}-${day}T${h}:${min}`;
}

export function parseExpiredTime(value) {
  if (value == null || value === '' || value === -1) return -1;
  if (dayjs.isDayjs(value)) {
    return value.isValid() ? value.unix() : -1;
  }
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? -1 : Math.ceil(ts / 1000);
}

/** 表单 DatePicker 用 dayjs，-1 表示永不过期 */
export function expiredTimeToDayjs(value) {
  if (value == null || value === '' || value === -1) return null;
  if (dayjs.isDayjs(value)) return value.isValid() ? value : null;
  const d = dayjs(Number(value) * 1000);
  return d.isValid() ? d : dayjs(value);
}

export function normalizeTokenForForm(data) {
  const models =
    data.models === '' || data.models == null
      ? []
      : typeof data.models === 'string'
        ? data.models.split(',').map((s) => s.trim()).filter(Boolean)
        : Array.isArray(data.models)
          ? data.models
          : [];
  return {
    name: data.name || '',
    remain_quota: data.remain_quota ?? 500000,
    expired_time: expiredTimeToDayjs(data.expired_time),
    unlimited_quota: !!data.unlimited_quota,
    models,
    subnet: data.subnet || '',
  };
}

export function buildTokenPayload(form) {
  return {
    name: form.name.trim(),
    remain_quota: parseInt(String(form.remain_quota), 10) || 0,
    expired_time: parseExpiredTime(form.expired_time),
    unlimited_quota: !!form.unlimited_quota,
    models: (form.models || []).join(','),
    subnet: form.subnet || '',
  };
}

export function formatQuotaDisplay(quota) {
  const v = Number(quota) || 0;
  const amount = v / QUOTA_DIVISOR;
  if (amount >= 1) return `${amount.toFixed(2)}`;
  if (amount >= 0.001) return amount.toFixed(3);
  return amount.toFixed(4);
}

export function formatDateTime(timestamp) {
  if (!timestamp || timestamp === -1) return null;
  const d = new Date(Number(timestamp) * 1000);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function formatRelativeAccess(ts) {
  if (!ts) return null;
  const diff = Math.floor(Date.now() / 1000) - Number(ts);
  if (diff < 60) return '刚刚';
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)} 天前`;
  return formatDateTime(ts);
}

export function formatMaskedKey(key) {
  if (!key) return '—';
  const raw = String(key);
  const bare = raw.startsWith('sk-') ? raw.slice(3) : raw;
  if (bare.length <= 8) return raw.startsWith('sk-') ? raw : `sk-${bare}`;
  return `sk-${bare.slice(0, 4)}${'•'.repeat(18)}${bare.slice(-4)}`;
}

export function getCopyKeyValue(key) {
  if (!key) return '';
  const raw = String(key);
  return raw.startsWith('sk-') ? raw : `sk-${raw}`;
}

export function buildTokenMetaParts(token) {
  const parts = [];
  const created = formatDateTime(token.created_time);
  if (created) parts.push(`创建于 ${created}`);
  if (token.unlimited_quota) {
    parts.push('无限额度');
  } else {
    parts.push(`剩余额度 ¥${formatQuotaDisplay(token.remain_quota)}`);
  }
  const accessed = formatRelativeAccess(token.accessed_time);
  if (accessed) parts.push(`最后使用 ${accessed}`);
  const expired =
    token.expired_time === -1
      ? '无过期限制'
      : `过期 ${formatDateTime(token.expired_time) || '—'}`;
  parts.push(expired);
  if (token.models && String(token.models).trim()) {
    const list =
      typeof token.models === 'string'
        ? token.models.split(',').map((s) => s.trim()).filter(Boolean)
        : token.models;
    if (list.length) {
      const preview = list.length > 2 ? `${list.slice(0, 2).join(' / ')} 等` : list.join(' / ');
      parts.push(`模型 ${preview}`);
    }
  }
  return parts;
}

export function computeTokenStats(tokens) {
  const list = tokens.filter((t) => !t.deleted);
  const active = list.filter((t) => t.status === 1).length;
  const used = list.reduce((s, t) => s + (Number(t.used_quota) || 0), 0);
  const remain = list.reduce((s, t) => {
    if (t.unlimited_quota) return s;
    return s + (Number(t.remain_quota) || 0);
  }, 0);
  const unlimited = list.filter((t) => t.unlimited_quota).length;
  return { total: list.length, active, used, remain, unlimited };
}

export async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
