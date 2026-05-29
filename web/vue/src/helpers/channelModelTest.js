import { API } from './api';

export const MODEL_TEST_OK = 'ok';
export const MODEL_TEST_FAIL = 'fail';
export const MODEL_TEST_SKIP = 'skip';
export const MODEL_TEST_TESTING = 'testing';

function rowTestStatus(row) {
  if (row?.skipped) return MODEL_TEST_SKIP;
  if (row?.success) return MODEL_TEST_OK;
  return MODEL_TEST_FAIL;
}

function applyResultRow(status, messages, meta, row) {
  const model = String(row.model || '').trim();
  if (!model) return;
  const st = typeof row.success === 'boolean' ? rowTestStatus(row) : MODEL_TEST_FAIL;
  status[model] = st;
  if (st === MODEL_TEST_FAIL) {
    messages[model] = row.message || '';
  }
  meta[model] = {
    testedAt: row.tested_at || 0,
    elapsedMs: row.elapsed_ms || 0,
    message: row.message || '',
    testKind: row.test_kind || '',
    skipped: !!row.skipped,
  };
}

export function summarizeModelTestReport(rows = []) {
  let ok = 0;
  let fail = 0;
  let skip = 0;
  for (const row of rows || []) {
    if (row.skipped) {
      skip += 1;
    } else if (row.success) {
      ok += 1;
    } else {
      fail += 1;
    }
  }
  return { ok, fail, skip, total: (rows || []).length };
}

export function normalizeModelTestReportRows(jobData, storedResults = []) {
  const fromJob = (jobData?.results || []).map((row) => ({
    model: row.model,
    success: !!row.success,
    skipped: !!row.skipped,
    test_kind: row.test_kind || '',
    test_protocol: row.test_protocol || row.detail?.wire_protocol || '',
    timed_out: !!row.timed_out,
    message: row.message || '',
    detail: row.detail || null,
    started_at: row.started_at,
    elapsed_ms: row.elapsed_ms,
    time: row.time,
    tested_at: row.tested_at,
  }));
  if (fromJob.length > 0) {
    return fromJob;
  }
  return (storedResults || []).map((row) => ({
    model: row.model,
    success: !!row.success,
    skipped: !!row.skipped,
    test_kind: row.test_kind || '',
    test_protocol: row.test_protocol || row.detail?.wire_protocol || '',
    timed_out: !!row.timed_out,
    message: row.message || '',
    detail: row.detail || null,
    started_at: row.started_at,
    elapsed_ms: row.elapsed_ms,
    time: row.time,
    tested_at: row.tested_at,
  }));
}

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
    controlApiPath: `${root}/jobs/control`,
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
  for (const row of results || []) {
    applyResultRow(status, messages, meta, row);
  }
  let ok = 0;
  let fail = 0;
  for (const st of Object.values(status)) {
    if (st === MODEL_TEST_OK) ok += 1;
    else if (st === MODEL_TEST_FAIL) fail += 1;
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
    applyResultRow(status, messages, meta, row);
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
      paused: !!jobData?.paused,
      cancelReq: !!jobData?.cancel_req,
      state: String(jobData?.state || ''),
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
  const ok = (jobData?.results || []).filter((r) => r.success && !r.skipped).length;
  const fail = (jobData?.results || []).filter((r) => !r.success && !r.skipped).length;
  const skip = (jobData?.results || []).filter((r) => r.skipped).length;
  if (jobData?.running) {
    const current = String(jobData.current_model || '').trim();
    if (jobData?.cancel_req || jobData?.state === 'cancelling') {
      return t('channel.edit.messages.test_models_cancelling', {
        completed,
        total,
      });
    }
    if (jobData?.paused || jobData?.state === 'paused') {
      return t('channel.edit.messages.test_models_paused', {
        completed,
        total,
        model: current || '…',
      });
    }
    return t('channel.edit.messages.test_models_running', {
      completed,
      total,
      model: current || '…',
    });
  }
  if (jobData?.interrupted || jobData?.state === 'cancelled') {
    return t('channel.edit.messages.test_models_cancelled', {
      completed,
      total,
    });
  }
  if (total > 0) {
    return t('channel.edit.messages.test_models_done', { ok, fail, skip, total });
  }
  return '';
}

export async function controlChannelModelTestJob(payload, action) {
  const { controlApiPath, statusParams } = payload;
  const res = await API.post(controlApiPath, {
    ...statusParams,
    action,
  });
  const { success, message, data } = res.data || {};
  if (!success) {
    throw new Error(message || 'control job failed');
  }
  return { message: message || '', job: data || null };
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
  const ok = (lastSnapshot?.results || []).filter((r) => r.success && !r.skipped).length;
  const fail = (lastSnapshot?.results || []).filter((r) => !r.success && !r.skipped).length;
  const skip = (lastSnapshot?.results || []).filter((r) => r.skipped).length;
  return { ok, fail, skip, total: list.length, job: lastSnapshot };
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
  if (status === MODEL_TEST_SKIP) classNames.push('model-tag--skip');
  if (status === MODEL_TEST_TESTING) classNames.push('model-tag--testing');
  const failMsg = modelTestMessages?.[model];
  const meta = modelTestMeta?.[model];
  return {
    content: item.text,
    className: classNames.join(' '),
    title: formatTestTooltip(status, meta, failMsg),
  };
}
