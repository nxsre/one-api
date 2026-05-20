import { API } from './api';

export const MODEL_TEST_OK = 'ok';
export const MODEL_TEST_FAIL = 'fail';
export const MODEL_TEST_TESTING = 'testing';

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export function normalizeChannelTestBaseUrl(baseUrl, channelType, channelTypeOptions = []) {
  let base = String(baseUrl || '').trim();
  if (base.endsWith('/')) {
    base = base.slice(0, -1);
  }
  if (!base && channelType) {
    const opt = (channelTypeOptions || []).find((o) => o.value === channelType);
    const def = opt?.default_base_url ? String(opt.default_base_url).trim() : '';
    if (def) {
      base = def.endsWith('/') ? def.slice(0, -1) : def;
    }
  }
  return base;
}

export function buildChannelModelTestPayload({
  channelId,
  type,
  baseUrl,
  key,
  config,
  modelMapping,
  channelTypeOptions = [],
  concurrency = 3,
  tenantConsole = false,
}) {
  let cfg = { ...(config || {}) };
  if ((type === 14 || type === 42) && !cfg.api_version) {
    cfg.api_version = 'v1';
  }
  const base = normalizeChannelTestBaseUrl(baseUrl, type, channelTypeOptions);
  const root = tenantConsole
    ? '/api/tenant_console/meta/channel/test_models'
    : '/api/channel/test_models';
  return {
    jobApiPath: `${root}/jobs`,
    statusApiPath: `${root}/jobs/status`,
    body: {
      channel_id: channelId ? Number(channelId) : 0,
      type,
      base_url: base,
      key: String(key || '').trim(),
      config: JSON.stringify(cfg),
      model_mapping: String(modelMapping || '').trim(),
      concurrency: Math.min(10, Math.max(1, Number(concurrency) || 3)),
    },
    statusParams: {
      channel_id: channelId ? Number(channelId) : 0,
      base_url: base,
      channel_type: type,
    },
  };
}

export async function fetchChannelModelTestResults({
  channelId,
  baseUrl,
  channelType,
  channelTypeOptions = [],
  tenantConsole = false,
}) {
  if (!channelId) return { results: [], job: null };
  const base = normalizeChannelTestBaseUrl(baseUrl, channelType, channelTypeOptions);
  const params = new URLSearchParams();
  params.set('base_url', base);
  if (channelType) params.set('channel_type', String(channelType));
  const prefix = tenantConsole
    ? '/api/tenant_console/meta/channel'
    : '/api/channel';
  const statusPath = tenantConsole
    ? '/api/tenant_console/meta/channel/test_models/jobs/status'
    : '/api/channel/test_models/jobs/status';
  const [historyRes, jobRes] = await Promise.all([
    API.get(`${prefix}/${channelId}/model_test_results?${params.toString()}`),
    API.get(statusPath, {
      params: {
        channel_id: channelId,
        base_url: base,
        channel_type: channelType || '',
      },
    }),
  ]);
  if (!historyRes.data?.success) {
    throw new Error(historyRes.data?.message || 'load model test results failed');
  }
  const job = jobRes.data?.success ? jobRes.data.data : null;
  return {
    results: historyRes.data.data?.results || [],
    job,
  };
}

export function applyStoredModelTestResults(results) {
  const status = {};
  const messages = {};
  const meta = {};
  let ok = 0;
  let fail = 0;
  for (const row of results || []) {
    const model = String(row.model || '').trim();
    if (!model) continue;
    if (row.success) {
      status[model] = MODEL_TEST_OK;
      ok += 1;
    } else {
      status[model] = MODEL_TEST_FAIL;
      messages[model] = row.message || '';
      fail += 1;
    }
    meta[model] = {
      testedAt: row.tested_at || 0,
      elapsedMs: row.elapsed_ms || 0,
      message: row.message || '',
    };
  }
  return { status, messages, meta, ok, fail };
}

export function applyJobSnapshotToModelState(jobData, modelList = []) {
  const allowed = new Set((modelList || []).map((m) => String(m)));
  const status = {};
  const messages = {};
  const meta = {};
  for (const row of jobData?.results || []) {
    const model = String(row.model || '').trim();
    if (!model || (allowed.size > 0 && !allowed.has(model))) continue;
    status[model] = row.success ? MODEL_TEST_OK : MODEL_TEST_FAIL;
    if (!row.success) messages[model] = row.message || '';
    meta[model] = {
      testedAt: row.tested_at || 0,
      elapsedMs: row.elapsed_ms || 0,
      message: row.message || '',
    };
  }
  if (jobData?.running) {
    String(jobData.current_model || '')
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
      .forEach((model) => {
        if (allowed.size === 0 || allowed.has(model)) {
          status[model] = MODEL_TEST_TESTING;
        }
      });
  }
  return {
    status,
    messages,
    meta,
    progress: {
      total: Number(jobData?.total) || 0,
      completed: Number(jobData?.completed) || 0,
      currentModel: String(jobData?.current_model || '').trim(),
      concurrency: Number(jobData?.concurrency) || 3,
      running: !!jobData?.running,
    },
  };
}

export function buildStoredModelTestSummary(t, ok, fail, total) {
  if (!total) return '';
  if (fail === 0 && ok > 0) {
    return t('channel.edit.messages.test_models_history_ok', { count: ok });
  }
  if (ok === 0 && fail > 0) {
    return t('channel.edit.messages.test_models_history_fail', { count: fail });
  }
  if (ok > 0 || fail > 0) {
    return t('channel.edit.messages.test_models_history_mixed', { ok, fail, total });
  }
  return '';
}

export function buildJobProgressSummary(t, jobData) {
  const total = Number(jobData?.total) || 0;
  const completed = Number(jobData?.completed) || 0;
  const ok = (jobData?.results || []).filter((r) => r.success).length;
  const fail = (jobData?.results || []).filter((r) => !r.success).length;
  if (jobData?.running) {
    const current = String(jobData.current_model || '').trim();
    return t('channel.edit.messages.test_models_running', {
      completed,
      total,
      model: current || '…',
    });
  }
  if (total > 0) {
    return t('channel.edit.messages.test_models_done', { ok, fail, total });
  }
  return '';
}

function formatTestTooltip(status, meta, failMsg) {
  const parts = [];
  if (meta?.testedAt) {
    const d = new Date(meta.testedAt * 1000);
    if (!Number.isNaN(d.getTime())) {
      parts.push(d.toLocaleString());
    }
  }
  if (status === MODEL_TEST_FAIL && failMsg) {
    parts.push(failMsg);
  } else if (status === MODEL_TEST_OK && meta?.message) {
    const msg = String(meta.message);
    parts.push(msg.length > 120 ? `${msg.slice(0, 120)}…` : msg);
  }
  return parts.length ? parts.join(' · ') : undefined;
}

/** 启动后台测试任务并轮询进度，直到完成。 */
export async function runChannelModelTestJob(payload, models, handlers = {}) {
  const list = (models || []).map((m) => String(m).trim()).filter(Boolean);
  if (!list.length) {
    throw new Error('no models');
  }
  const { jobApiPath, statusApiPath, body, statusParams } = payload;
  const startRes = await API.post(jobApiPath, { ...body, models: list });
  const startData = startRes.data || {};
  if (!startData.success) {
    if (startData.data?.running) {
      handlers.onJobSnapshot?.(startData.data);
    } else {
      throw new Error(startData.message || 'start job failed');
    }
  } else {
    handlers.onJobSnapshot?.(startData.data);
  }

  let lastSnapshot = startData.data;
  while (true) {
    const res = await API.get(statusApiPath, { params: statusParams });
    const { success, message, data } = res.data || {};
    if (!success) {
      throw new Error(message || 'poll job failed');
    }
    lastSnapshot = data;
    handlers.onJobSnapshot?.(data);
    if (!data?.running) {
      break;
    }
    await sleep(600);
  }
  const ok = (lastSnapshot?.results || []).filter((r) => r.success).length;
  const fail = (lastSnapshot?.results || []).filter((r) => !r.success).length;
  return { ok, fail, total: list.length, job: lastSnapshot };
}

export function filterOutFailedModels(models, modelTestStatus) {
  const list = (models || []).map((m) => String(m).trim()).filter(Boolean);
  const failed = list.filter((m) => modelTestStatus?.[m] === MODEL_TEST_FAIL);
  const remaining = list.filter((m) => modelTestStatus?.[m] !== MODEL_TEST_FAIL);
  return { failed, remaining };
}

export function renderChannelModelLabel(item, modelTestStatus, modelTestMessages, modelTestMeta) {
  const model = item.value;
  const status = modelTestStatus?.[model];
  const classNames = ['model-tag'];
  if (status === MODEL_TEST_OK) classNames.push('model-tag--ok');
  if (status === MODEL_TEST_FAIL) classNames.push('model-tag--fail');
  if (status === MODEL_TEST_TESTING) classNames.push('model-tag--testing');
  const failMsg = modelTestMessages?.[model];
  const meta = modelTestMeta?.[model];
  return {
    content: item.text,
    className: classNames.join(' '),
    title: formatTestTooltip(status, meta, failMsg),
  };
}
