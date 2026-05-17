/** @param {unknown} raw @param {Record<string, unknown>} fallback */
export function safeParseObject(raw, fallback) {
  try {
    const s = String(raw ?? '').trim();
    if (!s) return { ...fallback };
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      return { ...fallback, ...o };
    }
    return { ...fallback };
  } catch {
    return { ...fallback };
  }
}

export function toInt(v, def = 0) {
  const n = parseInt(String(v), 10);
  return Number.isFinite(n) ? n : def;
}

export function toFloat(v, def = 0) {
  const n = parseFloat(String(v));
  return Number.isFinite(n) ? n : def;
}

/** @returns {string} */
export function stringifyPolicy(obj) {
  try {
    return JSON.stringify(obj, null, 0);
  } catch {
    return '{}';
  }
}

export const EMPTY_ROUTING_POLICY = {
  selection_mode: 'weighted_random',
  consistent_hash_seed: '',
  sticky_source: 'token_id',
  direction_enabled: true,
  direction_probe_ratio: 0.1,
  direction_min_probe_ratio: 0.02,
  direction_latency_weight: 1,
  direction_error_weight: 2,
  probe_pick_strategy: 'secondary_uniform',
  auto_adaptive_enabled: true,
  adaptive_interval_sec: 60,
  circuit_fail_threshold: 8,
  circuit_cooldown_sec: 60,
  half_open_probe_ms: 8000,
  adaptive_aggressive_penalty: 0.92,
  adaptive_gentle_boost: 1.01,
  adaptive_err_ratio_threshold: 0.25,
  auto_weight_min: 0.05,
  auto_weight_max: 3,
  aggregation_interval_sec: 60,
};

export const EMPTY_RELAY_RETRY = {
  max_retries_override: null,
  base_backoff_ms: 150,
  max_backoff_ms: 4000,
  retry_http_status_allowlist: [408, 409, 429, 500, 502, 503],
  retry_http_status_denylist: [],
  force_different_channel_each_attempt: true,
};

export const EMPTY_ALIAS_POLICY = {
  max_chain_steps: 32,
  chains: {},
  aliases: {},
  defaults: {},
  group_overrides: {},
};

export const EMPTY_RATE_LIMIT_POLICY = {
  enabled: true,
  by_token: { '*': { qps: 80, burst: 120, concurrency: 20, daily_quota: 0 } },
  by_user: {},
  by_group: {},
};

export const HTTP_RETRY_CODE_OPTIONS = [
  400, 401, 403, 404, 408, 409, 413, 422, 429,
  500, 501, 502, 503, 504,
].map((n) => ({ key: String(n), text: String(n), value: n }));
