import { API } from '@/api/client';
import { isAdminUser } from '@/lib/modelCatalog';

/** 与后端 common.PageInfo 默认一致 */
export const LOG_PAGE_SIZE = 10;

const QUOTA_DIVISOR = 500_000;

export const LOG_TYPE_META = {
  1: { label: '充值', tagClass: 'logs-type-topup' },
  2: { label: '消费', tagClass: 'logs-type-usage' },
  3: { label: '管理', tagClass: 'logs-type-admin' },
  4: { label: '系统', tagClass: 'logs-type-system' },
  5: { label: '测试', tagClass: 'logs-type-test' },
  6: { label: '错误', tagClass: 'logs-type-error' },
  7: { label: '退款', tagClass: 'logs-type-refund' },
};

/** 类型筛选下拉（与表格「类型」列一致） */
export const LOG_TYPE_FILTER_OPTIONS = [
  { value: 0, label: '全部类型' },
  ...Object.entries(LOG_TYPE_META)
    .sort(([a], [b]) => Number(a) - Number(b))
    .map(([value, meta]) => ({ value: Number(value), label: meta.label })),
];

export function formatLogType(type) {
  const meta = LOG_TYPE_META[Number(type)] || { label: '未知', tagClass: 'logs-type-unknown' };
  return meta;
}

const MODEL_TAG_CLASSES = ['tag-blue', 'tag-amber', 'tag-cyan', 'tag-purple', 'tag-green'];

export function defaultLogTimeRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 6);
  return {
    start_timestamp: toDatetimeLocalValue(start),
    end_timestamp: toDatetimeLocalValue(end),
  };
}

export function toDatetimeLocalValue(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  const h = String(date.getHours()).padStart(2, '0');
  const min = String(date.getMinutes()).padStart(2, '0');
  return `${y}-${m}-${d}T${h}:${min}`;
}

export function parseDatetimeLocal(value) {
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? Math.floor(Date.now() / 1000) : Math.floor(ts / 1000);
}

export function resolveLogQueryByType(logType) {
  const type = Number(logType) || 0;
  return {
    logType: type,
    includeErrors: type === 0,
  };
}

function buildLogListUrl({
  page,
  isAdmin,
  logType,
  includeErrors,
  startTs,
  endTs,
  filters,
}) {
  const q = new URLSearchParams();
  q.set('p', String(page));
  q.set('page_size', String(LOG_PAGE_SIZE));
  q.set('type', String(logType));
  q.set('start_timestamp', String(startTs));
  q.set('end_timestamp', String(endTs));
  if (filters.token_name) q.set('token_name', filters.token_name);
  if (filters.model_name) q.set('model_name', filters.model_name);
  if (filters.group) q.set('group', filters.group);
  if (filters.request_id) q.set('request_id', filters.request_id);
  if (includeErrors) q.set('include_errors', '1');

  if (isAdmin) {
    if (filters.username) q.set('username', filters.username);
    if (filters.channel) q.set('channel', filters.channel);
    return `/api/log/?${q.toString()}`;
  }
  return `/api/log/self/?${q.toString()}`;
}

function buildLogStatUrl({ isAdmin, logType, includeErrors, startTs, endTs, filters }) {
  const q = new URLSearchParams();
  q.set('type', String(logType));
  q.set('start_timestamp', String(startTs));
  q.set('end_timestamp', String(endTs));
  if (filters.token_name) q.set('token_name', filters.token_name);
  if (filters.model_name) q.set('model_name', filters.model_name);
  if (filters.group) q.set('group', filters.group);
  if (filters.request_id) q.set('request_id', filters.request_id);
  if (includeErrors) q.set('include_errors', '1');

  if (isAdmin) {
    if (filters.username) q.set('username', filters.username);
    if (filters.channel) q.set('channel', filters.channel);
    return `/api/log/stat?${q.toString()}`;
  }
  return `/api/log/self/stat?${q.toString()}`;
}

export async function fetchLogPage(user, params) {
  const isAdmin = isAdminUser(user);
  const url = buildLogListUrl({ isAdmin, ...params });
  const res = await API.get(url);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载日志失败');
  }
  const data = res.data.data;
  const items = data?.items !== undefined ? data.items : data;
  const total =
    data && typeof data.total === 'number'
      ? data.total
      : Array.isArray(items)
        ? items.length
        : 0;
  return {
    items: Array.isArray(items) ? items.filter((l) => !l.deleted) : [],
    total,
  };
}

export async function fetchLogStat(user, params) {
  const isAdmin = isAdminUser(user);
  const url = buildLogStatUrl({ isAdmin, ...params });
  const res = await API.get(url);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载统计失败');
  }
  return res.data.data || { quota: 0, rpm: 0, tpm: 0 };
}

export function parseLogOther(other) {
  const s = other != null ? String(other).trim() : '';
  if (!s) return null;
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) return o;
  } catch {
    /* 非 JSON */
  }
  return null;
}

export function extractHttpStatusCode(parsed) {
  if (!parsed || typeof parsed !== 'object') return null;
  for (const key of ['http_status', 'status_code', 'upstream_http_status', 'test_http_status']) {
    const v = parsed[key];
    if (v == null || v === '') continue;
    const n = Number(v);
    if (!Number.isNaN(n) && n > 0) return n;
    return String(v);
  }
  return null;
}

export function getLogStatusMeta(log) {
  const parsed = parseLogOther(log.other);
  const code = extractHttpStatusCode(parsed);
  const n = code != null ? Number(code) : null;

  if (log.type === 6 || (n != null && n >= 500)) {
    return { kind: 'err', code: code ?? 'ERR', pillClass: 'err' };
  }
  if (n != null && n >= 400) {
    return { kind: 'warn', code, pillClass: 'err' };
  }
  if (n != null && n >= 200 && n < 300) {
    return { kind: 'ok', code, pillClass: 'ok' };
  }
  if (log.type === 2) {
    return { kind: 'ok', code: code ?? '200', pillClass: 'ok' };
  }
  return { kind: 'ok', code: code ?? '—', pillClass: 'ok' };
}

export function formatLogTime(createdAt) {
  if (!createdAt) return '—';
  const d = new Date(createdAt * 1000);
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const h = String(d.getHours()).padStart(2, '0');
  const min = String(d.getMinutes()).padStart(2, '0');
  const s = String(d.getSeconds()).padStart(2, '0');
  return `${m}-${day} ${h}:${min}:${s}`;
}

export function formatTokenCell(v) {
  if (v === null || v === undefined) return '—';
  return Number(v).toLocaleString();
}

export function formatLatency(useTime) {
  if (useTime === null || useTime === undefined || useTime === '') return '—';
  const n = Number(useTime);
  if (Number.isNaN(n)) return '—';
  return `${n}s`;
}

export function formatLogQuota(quota) {
  if (quota === null || quota === undefined) return '—';
  const v = Number(quota);
  if (Number.isNaN(v)) return '—';
  const amount = v / QUOTA_DIVISOR;
  return `¥${amount.toFixed(3)}`;
}

export function truncateMiddle(str, head = 4, tail = 4) {
  const s = String(str || '').trim();
  if (!s) return '—';
  if (s.length <= head + tail + 1) return s;
  return `${s.slice(0, head)}…${s.slice(-tail)}`;
}

export function getModelTagClass(modelName) {
  const s = String(modelName || '');
  let hash = 0;
  for (let i = 0; i < s.length; i += 1) {
    hash = (hash + s.charCodeAt(i)) % MODEL_TAG_CLASSES.length;
  }
  return MODEL_TAG_CLASSES[hash];
}

export function buildLogDetailSections(log) {
  const parsed = parseLogOther(log.other);
  const fields = [];
  if (log.request_id) fields.push({ label: '请求 ID', value: log.request_id, mono: true });
  if (log.model_name) fields.push({ label: '模型', value: log.model_name });
  if (log.token_name) fields.push({ label: '令牌名称', value: log.token_name, mono: true });
  if (log.ip) fields.push({ label: 'IP', value: log.ip, mono: true });
  const code = extractHttpStatusCode(parsed);
  if (code != null) fields.push({ label: 'HTTP 状态', value: String(code) });

  const blocks = [];
  if (log.content) {
    const text = String(log.content).trim();
    if (text) blocks.push({ label: '详情', value: text, kind: 'text' });
  }
  if (parsed?.user_input) {
    const text = String(parsed.user_input).trim();
    if (text) blocks.push({ label: '用户输入', value: text, kind: 'code' });
  }
  if (log.other && !parsed) {
    blocks.push({ label: '原始数据', value: String(log.other), kind: 'code' });
  }

  let metadata = null;
  if (parsed) {
    const skip = new Set(['user_input', 'http_status', 'status_code']);
    const rest = Object.fromEntries(
      Object.entries(parsed).filter(([k]) => !skip.has(k))
    );
    if (Object.keys(rest).length) {
      metadata = JSON.stringify(rest, null, 2);
    }
  }

  const badges = [];
  if (log.use_time > 0) badges.push(`${log.use_time}s`);
  if (log.elapsed_time != null && log.elapsed_time !== '') badges.push(`${log.elapsed_time}ms`);
  if (log.is_stream) badges.push('Stream');

  return { fields, blocks, metadata, badges };
}

/** 纯文本，供复制 */
export function buildLogDetailText(log) {
  const { fields, blocks, metadata, badges } = buildLogDetailSections(log);
  const parts = [];
  fields.forEach((f) => parts.push(`${f.label}\n${f.value}`));
  blocks.forEach((b) => parts.push(`${b.label}\n${b.value}`));
  if (metadata) parts.push(`元数据\n${metadata}`);
  if (badges.length) parts.push(badges.join(' · '));
  return parts.join('\n\n') || '—';
}

export { copyText } from './copyText';
