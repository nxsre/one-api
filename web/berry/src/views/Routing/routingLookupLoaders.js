import { ITEMS_PER_PAGE } from 'constants';

/** @param {unknown} raw */
export function parseChannelModelsField(raw) {
  if (raw == null || raw === '') return [];
  return [
    ...new Set(
      String(raw)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    ),
  ];
}

/** @returns {Promise<string[]>} */
export async function fetchRoutingGroups(API) {
  const res = await API.get('/api/group/');
  if (!res.data?.success) return [];
  return (res.data.data || []).map(String).filter(Boolean);
}

/** @returns {Promise<string[]>} */
export async function fetchRoutingModelCatalog(API) {
  const res = await API.get('/api/routing/model-catalog');
  if (!res.data?.success) return [];
  const rows = res.data.data;
  return Array.isArray(rows) ? rows.map(String).filter(Boolean) : [];
}

/** @returns {Promise<{ id: number; name: string; models: string[] }[]>} */
export async function fetchRoutingChannels(API, maxPages = 100) {
  const merged = new Map();
  for (let p = 0; p < maxPages; p++) {
    const res = await API.get(`/api/channel/?p=${p}`);
    if (!res.data?.success) break;
    const rows = res.data.data || [];
    if (!rows.length) break;
    for (const ch of rows) {
      merged.set(ch.id, {
        id: ch.id,
        name: ch.name || '',
        models: parseChannelModelsField(ch.models),
      });
    }
    if (rows.length < ITEMS_PER_PAGE) break;
  }
  return [...merged.values()].sort((a, b) => b.id - a.id);
}
