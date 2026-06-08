import { API } from './api';

function catalogBase(tenantConsole = false) {
  return tenantConsole ? '/api/tenant_console/meta/model_catalog' : '/api/model_catalog';
}

export async function fetchModelCatalogProviders(query = '', limit = 30, tenantConsole = false) {
  const params = new URLSearchParams();
  if (String(query || '').trim()) params.set('q', String(query).trim());
  params.set('limit', String(limit));
  const res = await API.get(`${catalogBase(tenantConsole)}/providers?${params.toString()}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'load providers failed');
  }
  return res.data.data || [];
}

export async function fetchModelCatalogModelIds(filters = {}, tenantConsole = false) {
  const params = new URLSearchParams();
  Object.entries(filters || {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value).trim());
    }
  });
  const qs = params.toString();
  const res = await API.get(
    `${catalogBase(tenantConsole)}/model_ids${qs ? `?${qs}` : ''}`
  );
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'load model ids failed');
  }
  return res.data.data?.model_ids || [];
}

export function providerFilterValue(option) {
  if (!option) return '';
  return String(option.key || option.display || option.label || '').trim();
}

export function providerOptionsToDropdown(items) {
  return (items || []).map((item) => {
    const key = String(item.key || '').trim();
    const value = key || String(item.display || item.label || '').trim();
    return {
      key: value,
      value,
      text: item.label || item.display || value,
    };
  });
}

/** 渠道填入：下拉选中用精确 provider_key；手工输入仍走模糊 filter_provider。 */
export function providerCatalogModelFilters(value, options = []) {
  const v = String(value || '').trim();
  if (!v) return null;
  if ((options || []).some((o) => o.value === v)) {
    return { filter_provider_key: v };
  }
  return { filter_provider: v };
}
