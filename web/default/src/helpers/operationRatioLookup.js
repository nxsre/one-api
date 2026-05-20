import { API } from './api';

/** @returns {Promise<{ key: string, text: string, value: string }[]>} */
export async function fetchOperationModelOptions() {
  const res = await API.get('/api/channel/models');
  const list = res.data?.data;
  if (!Array.isArray(list)) return [];
  const seen = new Set();
  const opts = [];
  for (const item of list) {
    const id = String(item?.id ?? item ?? '').trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    opts.push({ key: id, text: id, value: id });
  }
  opts.sort((a, b) => a.value.localeCompare(b.value));
  return opts;
}

/** @returns {Promise<{ key: string, text: string, value: string }[]>} */
export async function fetchOperationGroupOptions() {
  const res = await API.get('/api/group/');
  const list = res.data?.data;
  if (!Array.isArray(list)) return [];
  const opts = list
    .map((g) => String(g ?? '').trim())
    .filter(Boolean)
    .map((g) => ({ key: g, text: g, value: g }));
  opts.sort((a, b) => a.value.localeCompare(b.value));
  return opts;
}

/** @param {{ key: string, text: string, value: string }[]} baseOpts */
export function mergeDropdownOptions(baseOpts, extraKeys) {
  const seen = new Set((baseOpts || []).map((o) => o.value));
  const opts = [...(baseOpts || [])];
  for (const raw of extraKeys || []) {
    const k = String(raw ?? '').trim();
    if (!k || seen.has(k)) continue;
    seen.add(k);
    opts.push({ key: k, text: k, value: k });
  }
  opts.sort((a, b) => String(a.value).localeCompare(String(b.value)));
  return opts;
}

export function keysFromFlatRatioJson(raw) {
  const s = String(raw ?? '').trim();
  if (!s) return [];
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      return Object.keys(o);
    }
  } catch {
    /* ignore */
  }
  return [];
}

export function keysFromGroupGroupRatioJson(raw) {
  const s = String(raw ?? '').trim();
  if (!s) return [];
  try {
    const o = JSON.parse(s);
    if (!o || typeof o !== 'object' || Array.isArray(o)) return [];
    const keys = new Set();
    for (const [outer, inner] of Object.entries(o)) {
      if (outer) keys.add(String(outer));
      if (inner && typeof inner === 'object' && !Array.isArray(inner)) {
        Object.keys(inner).forEach((k) => keys.add(String(k)));
      }
    }
    return [...keys];
  } catch {
    return [];
  }
}
