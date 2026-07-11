import { API } from '@/api/client';
import { isAdminUser } from '@/lib/modelCatalog';

export const ROLE_TENANT_ADMIN = 20;

const QUOTA_DIVISOR = 1_000_000;

export const RANGE_PRESETS = [
  { key: 'month', label: '本月' },
  { key: '3months', label: '近3个月' },
  { key: 'halfYear', label: '近半年' },
  { key: 'year', label: '近1年' },
  { key: 'custom', label: '自定义' },
];

export function toDateInputValue(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

export function getDefaultRange() {
  return resolveRangeFromPreset('month');
}

export function resolveRangeFromPreset(presetKey) {
  if (presetKey === 'custom') return null;
  const end = new Date();
  const start = new Date();
  if (presetKey === '3months') {
    start.setMonth(start.getMonth() - 3);
    return { start: toDateInputValue(start), end: toDateInputValue(end) };
  }
  if (presetKey === 'halfYear') {
    start.setMonth(start.getMonth() - 6);
    return { start: toDateInputValue(start), end: toDateInputValue(end) };
  }
  if (presetKey === 'year') {
    start.setFullYear(start.getFullYear() - 1);
    return { start: toDateInputValue(start), end: toDateInputValue(end) };
  }
  start.setDate(1);
  return { start: toDateInputValue(start), end: toDateInputValue(end) };
}

export function toUnixRange(startDate, endDate) {
  const startTs = Math.floor(new Date(startDate).getTime() / 1000);
  const endTs = Math.floor(new Date(endDate).getTime() / 1000) + 86400;
  return { startTs, endTs };
}

export function resolveBillingSource(user) {
  if (isAdminUser(user) && !user?.tenant_id) return 'platform';
  if (Number(user?.role) === ROLE_TENANT_ADMIN) return 'tenant_console';
  return 'dashboard';
}

function billingApiPath(source) {
  if (source === 'platform') return '/api/platform/reports/billing';
  if (source === 'tenant_console') return '/api/tenant_console/reports/billing';
  return null;
}

function billingExportPath(source) {
  if (source === 'platform') return '/api/platform/reports/billing/export';
  if (source === 'tenant_console') return '/api/tenant_console/reports/billing/export';
  return null;
}

/** @returns {Promise<{ rows: object[], source: string }>} */
export async function fetchBillingRows(user, startDate, endDate) {
  const source = resolveBillingSource(user);
  const { startTs, endTs } = toUnixRange(startDate, endDate);

  if (source === 'dashboard') {
    const res = await API.get('/api/user/dashboard');
    if (!res.data?.success) {
      throw new Error(res.data?.message || '加载统计失败');
    }
    const raw = res.data.data || [];
    const startStr = startDate;
    const endStr = endDate;
    const filtered = raw.filter((item) => item.Day >= startStr && item.Day <= endStr);
    return { rows: filtered, source };
  }

  const path = billingApiPath(source);
  const res = await API.get(`${path}?start_time=${startTs}&end_time=${endTs}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载账单失败');
  }
  return { rows: res.data.data || [], source };
}

export function openBillingExport(user, startDate, endDate) {
  const source = resolveBillingSource(user);
  const path = billingExportPath(source);
  if (!path) return;
  const { startTs, endTs } = toUnixRange(startDate, endDate);
  const base = import.meta.env.VITE_API_BASE || '';
  window.open(`${base}${path}?start_time=${startTs}&end_time=${endTs}`, '_blank');
}

function sumField(rows, pick) {
  return rows.reduce((s, r) => s + (Number(pick(r)) || 0), 0);
}

/** dashboard 行：Day + ModelName */
function isDashboardRow(row) {
  return Boolean(row?.Day && row?.ModelName != null);
}

/** 账单行聚合为概览、按日、按模型 */
export function aggregateUsageData(rows, source) {
  if (!rows?.length) {
    return {
      summary: {
        totalTokens: 0,
        promptTokens: 0,
        completionTokens: 0,
        quotaM: 0,
        requestCount: 0,
        modelCount: 0,
        dayCount: 0,
      },
      daily: [],
      byModel: [],
      tableMode: source === 'platform' ? 'tenant' : 'model',
    };
  }

  if (source === 'dashboard' || isDashboardRow(rows[0])) {
    const byDayMap = {};
    const byModelMap = {};

    rows.forEach((item) => {
      const day = item.Day;
      const model = item.ModelName || 'unknown';
      const prompt = item.PromptTokens || 0;
      const completion = item.CompletionTokens || 0;
      const tokens = prompt + completion;
      const quota = (item.Quota || 0) / QUOTA_DIVISOR;
      const requests = item.RequestCount || 0;

      if (!byDayMap[day]) {
        byDayMap[day] = { date: day, tokens: 0, quotaM: 0, requests: 0 };
      }
      byDayMap[day].tokens += tokens;
      byDayMap[day].quotaM += quota;
      byDayMap[day].requests += requests;

      if (!byModelMap[model]) {
        byModelMap[model] = {
          model_name: model,
          request_count: 0,
          prompt_tokens: 0,
          completion_tokens: 0,
          quota: 0,
        };
      }
      byModelMap[model].request_count += requests;
      byModelMap[model].prompt_tokens += prompt;
      byModelMap[model].completion_tokens += completion;
      byModelMap[model].quota += item.Quota || 0;
    });

    const byModel = Object.values(byModelMap).sort((a, b) => b.quota - a.quota);
    const totalQuota = sumField(byModel, (r) => r.quota);
    byModel.forEach((m) => {
      m.quotaM = (m.quota || 0) / QUOTA_DIVISOR;
      m.share = totalQuota > 0 ? (m.quota / totalQuota) * 100 : 0;
    });

    const daily = Object.values(byDayMap).sort((a, b) => a.date.localeCompare(b.date));
    const promptTokens = sumField(byModel, (r) => r.prompt_tokens);
    const completionTokens = sumField(byModel, (r) => r.completion_tokens);

    return {
      summary: {
        totalTokens: promptTokens + completionTokens,
        promptTokens,
        completionTokens,
        quotaM: totalQuota / QUOTA_DIVISOR,
        requestCount: sumField(byModel, (r) => r.request_count),
        modelCount: byModel.length,
        dayCount: daily.length,
      },
      daily,
      byModel,
      tableMode: 'model',
    };
  }

  if (source === 'platform') {
    const totalQuota = sumField(rows, (r) => r.quota);
    const byTenant = rows
      .map((r) => ({
        tenant_id: r.tenant_id,
        tenant_name: formatTenantLabel(r),
        request_count: r.request_count || 0,
        prompt_tokens: r.prompt_tokens || 0,
        completion_tokens: r.completion_tokens || 0,
        quota: r.quota || 0,
        quotaM: (r.quota || 0) / QUOTA_DIVISOR,
        share: totalQuota > 0 ? ((r.quota || 0) / totalQuota) * 100 : 0,
      }))
      .sort((a, b) => b.quota - a.quota);

    const promptTokens = sumField(rows, (r) => r.prompt_tokens);
    const completionTokens = sumField(rows, (r) => r.completion_tokens);

    return {
      summary: {
        totalTokens: promptTokens + completionTokens,
        promptTokens,
        completionTokens,
        quotaM: totalQuota / QUOTA_DIVISOR,
        requestCount: sumField(rows, (r) => r.request_count),
        modelCount: byTenant.length,
        dayCount: 0,
      },
      daily: [],
      byModel: byTenant,
      tableMode: 'tenant',
    };
  }

  const byModelMap = {};
  rows.forEach((r) => {
    const model = r.model_name || 'unknown';
    if (!byModelMap[model]) {
      byModelMap[model] = {
        model_name: model,
        request_count: 0,
        prompt_tokens: 0,
        completion_tokens: 0,
        quota: 0,
      };
    }
    byModelMap[model].request_count += r.request_count || 0;
    byModelMap[model].prompt_tokens += r.prompt_tokens || 0;
    byModelMap[model].completion_tokens += r.completion_tokens || 0;
    byModelMap[model].quota += r.quota || 0;
  });

  const byModel = Object.values(byModelMap).sort((a, b) => b.quota - a.quota);
  const totalQuota = sumField(byModel, (r) => r.quota);
  byModel.forEach((m) => {
    m.quotaM = (m.quota || 0) / QUOTA_DIVISOR;
    m.share = totalQuota > 0 ? (m.quota / totalQuota) * 100 : 0;
  });

  const promptTokens = sumField(byModel, (r) => r.prompt_tokens);
  const completionTokens = sumField(byModel, (r) => r.completion_tokens);

  return {
    summary: {
      totalTokens: promptTokens + completionTokens,
      promptTokens,
      completionTokens,
      quotaM: totalQuota / QUOTA_DIVISOR,
      requestCount: sumField(byModel, (r) => r.request_count),
      modelCount: byModel.length,
      dayCount: 0,
    },
    daily: [],
    byModel,
    tableMode: 'model',
  };
}

export function formatTokenCount(n) {
  const v = Number(n) || 0;
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (v >= 1000) return `${(v / 1000).toFixed(1).replace(/\.0$/, '')}K`;
  return v.toLocaleString();
}

export function formatQuotaM(n) {
  const v = Number(n) || 0;
  return v.toFixed(2);
}

/** 平台报表租户展示：无租户时不显示 undefined */
export function formatTenantLabel(row) {
  const name = String(row?.tenant_name ?? '').trim();
  if (name && name !== 'undefined' && !/^租户\s*undefined$/i.test(name)) {
    return name;
  }
  const id = row?.tenant_id;
  if (id == null || id === '' || Number(id) === 0) {
    return '非租户用户';
  }
  return `租户 #${id}`;
}
