import { API } from '@/api/client';

export const MODEL_PAGE_SIZE = 12;
const CATALOG_MAX_PAGE_SIZE = 100;

export const MODEL_FILTERS = [
  { key: 'all', label: '全部' },
  { key: 'language', label: '语言模型' },
  { key: 'reasoning', label: '推理模型' },
  { key: 'multimodal', label: '多模态' },
  { key: 'code', label: '代码模型' },
  { key: 'image', label: '图像生成' },
];

/** 与 ModelCatalog fmtPrice 一致 */
export function fmtPrice(n) {
  if (n === null || n === undefined || Number.isNaN(Number(n))) return null;
  const x = Number(n);
  if (x === 0) return '0';
  return x.toFixed(6).replace(/\.?0+$/, '');
}

export function formatContextLimit(n) {
  const v = Number(n);
  if (!v || Number.isNaN(v)) return null;
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(v % 1_000_000 === 0 ? 0 : 1)}M ctx`;
  if (v >= 1000) return `${Math.round(v / 1000)}K ctx`;
  return `${v} ctx`;
}

function fmtPricePerM(n) {
  const s = fmtPrice(n);
  if (!s) return null;
  return `$${s}/Mt`;
}

export function formatPriceLabel(row) {
  const input = fmtPricePerM(row?.cost_input);
  const output = fmtPricePerM(row?.cost_output);
  if (!input && !output) return '按平台计费';
  if (input && output && input !== output) {
    return `输入 ${input} / 输出 ${output}`;
  }
  return input || output;
}

export function getCardTheme(row) {
  const key = `${row?.provider_key || ''} ${row?.model_id || ''}`.toLowerCase();
  if (/deepseek/.test(key)) return 'deepseek';
  if (/qwen|tongyi|ali/.test(key)) return 'qwen';
  if (/kimi|moonshot/.test(key)) return 'kimi';
  if (/glm|zhipu|chatglm/.test(key)) return 'gemini';
  if (/doubao|bytedance/.test(key)) return 'gpt';
  if (/wenxin|ernie|baidu/.test(key)) return 'claude';
  if (/hunyuan|tencent/.test(key)) return 'gemini';
  if (/spark|xfyun|讯飞/.test(key)) return 'claude';
  if (/minimax/.test(key)) return 'qwen';
  if (/yi-|01\.ai|零一/.test(key)) return 'kimi';
  if (/openai|gpt|o1|o3/.test(key)) return 'gpt';
  if (/claude|anthropic/.test(key)) return 'claude';
  if (/gemini|google/.test(key)) return 'gemini';
  return 'gemini';
}

export function getModelIcon(row) {
  const theme = getCardTheme(row);
  const map = {
    gpt: '🤖',
    claude: '🧠',
    gemini: '✨',
    deepseek: '🔮',
    qwen: '🌟',
    kimi: '🌙',
  };
  return map[theme] || '✨';
}

export function getBadge(row) {
  if (row?.reasoning) return { text: '推理', tag: 'tag-amber' };
  if (row?.tool_call) return { text: '工具调用', tag: 'tag-green' };
  const tags = String(row?.tags || '').toLowerCase();
  if (tags.includes('new') || tags.includes('latest')) return { text: '最新', tag: 'tag-green' };
  if (tags.includes('hot')) return { text: '热门', tag: 'tag-amber' };
  if (row?.open_weights) return { text: '开源', tag: 'tag-cyan' };
  return null;
}

export function buildTagList(row) {
  const tags = [];
  const modIn = String(row?.modalities_in || '').toLowerCase();
  const modOut = String(row?.modalities_out || '').toLowerCase();
  if (modOut.includes('text') || modIn.includes('text')) tags.push({ label: '语言', cls: 'tag-blue' });
  if (row?.reasoning) tags.push({ label: '推理', cls: 'tag-amber' });
  if (row?.tool_call) tags.push({ label: '工具调用', cls: 'tag-cyan' });
  if (row?.attachment_ok || modIn.includes('image') || modOut.includes('image')) {
    tags.push({ label: '多模态', cls: 'tag-purple' });
  }
  if (/code|coder/.test(String(row?.model_id || '').toLowerCase())) {
    tags.push({ label: '代码', cls: 'tag-cyan' });
  }
  if (row?.family) tags.push({ label: row.family, cls: 'tag-gray' });
  const extra = String(row?.tags || '')
    .split(/[,，]/)
    .map((t) => t.trim())
    .filter(Boolean)
    .slice(0, 2);
  extra.forEach((t) => tags.push({ label: t, cls: 'tag-gray' }));
  const seen = new Set();
  return tags.filter((t) => {
    if (seen.has(t.label)) return false;
    seen.add(t.label);
    return true;
  }).slice(0, 4);
}

export function getModelDescription(row) {
  const notes = String(row?.notes || '').trim();
  if (notes) return notes.length > 120 ? `${notes.slice(0, 120)}…` : notes;
  const parts = [];
  if (row?.provider_display) parts.push(`${row.provider_display} 提供`);
  if (row?.reasoning) parts.push('支持深度推理');
  if (row?.tool_call) parts.push('支持工具调用');
  if (row?.attachment_ok) parts.push('支持多模态输入');
  const ctx = formatContextLimit(row?.context_limit);
  if (ctx) parts.push(`上下文 ${ctx}`);
  return parts.length
    ? `${parts.join('，')}。通过统一 OpenAI 兼容接口即可调用。`
    : '可通过平台统一 API 接口调用该模型。';
}

export function matchesCategory(row, filterKey) {
  if (filterKey === 'all') return true;
  const id = String(row?.model_id || '').toLowerCase();
  const modIn = String(row?.modalities_in || '').toLowerCase();
  const modOut = String(row?.modalities_out || '').toLowerCase();
  const family = String(row?.family || '').toLowerCase();

  if (filterKey === 'image') {
    return (
      /dall|flux|midjourney|stable-diffusion|image-|gpt-image|ideogram|recraft/.test(id) ||
      modOut === 'image' ||
      modOut.includes('image') && !modOut.includes('text')
    );
  }
  if (filterKey === 'reasoning') {
    return row?.reasoning || /reason|think|o1|o3|deepseek-r|qwq/.test(id);
  }
  if (filterKey === 'multimodal') {
    return (
      row?.attachment_ok ||
      /vision|vl-|4o|gemini|claude-3|qwen-vl|glm-4v/.test(id) ||
      modIn.includes('image') ||
      modIn.includes('video')
    );
  }
  if (filterKey === 'code') {
    return /code|coder|codex|starcoder/.test(id) || family.includes('code');
  }
  if (filterKey === 'language') {
    if (matchesCategory(row, 'image')) return false;
    return modOut.includes('text') || modOut === '' || !modOut;
  }
  return true;
}

/** 分类筛选下推到 GET /api/model_catalog?filter_category= */
export function filterToApiParams(filterKey) {
  if (!filterKey || filterKey === 'all') return {};
  return { filterCategory: filterKey };
}

export function matchesSearch(row, query) {
  const q = String(query || '').trim().toLowerCase();
  if (!q) return true;
  const hay = [
    row?.model_id,
    row?.model_name,
    row?.provider_key,
    row?.provider_display,
    row?.family,
    row?.tags,
    row?.owned_by,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();
  return hay.includes(q);
}

export function isAdminUser(user) {
  return Number(user?.role) >= 10;
}

/** GET /api/user/available_models */
export async function fetchUserAvailableModelIds() {
  const res = await API.get('/api/user/available_models');
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载可用模型失败');
  }
  const list = res.data.data;
  if (!Array.isArray(list)) return [];
  return [...new Set(list.map((id) => String(id || '').trim()).filter(Boolean))];
}

/** GET /api/model_catalog 单页（与 ModelCatalog 页一致） */
export async function fetchModelCatalogPage({
  page = 1,
  pageSize = MODEL_PAGE_SIZE,
  search = '',
  filterModelName = '',
  filterProvider = '',
  filterFamily = '',
  filterModalitiesIn = '',
  filterModalitiesOut = '',
  filterCategory = '',
} = {}) {
  const params = new URLSearchParams();
  params.set('page', String(page));
  params.set('page_size', String(Math.min(Math.max(pageSize, 1), CATALOG_MAX_PAGE_SIZE)));
  if (search) params.set('search', search);
  if (filterModelName) params.set('filter_model_name', filterModelName);
  if (filterProvider) params.set('filter_provider', filterProvider);
  if (filterFamily) params.set('filter_family', filterFamily);
  if (filterModalitiesIn) params.set('filter_modalities_in', filterModalitiesIn);
  if (filterModalitiesOut) params.set('filter_modalities_out', filterModalitiesOut);
  if (filterCategory) params.set('filter_category', filterCategory);

  const res = await API.get(`/api/model_catalog?${params.toString()}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载模型目录失败');
  }
  const data = res.data.data;
  if (data && Array.isArray(data.items)) {
    return {
      items: data.items,
      total: Number(data.total) || 0,
      grandTotal: Number(data.grand_total) || Number(data.total) || 0,
    };
  }
  if (Array.isArray(data)) {
    return { items: data, total: data.length, grandTotal: data.length };
  }
  return { items: [], total: 0, grandTotal: 0 };
}

function normalizeCatalogRows(items) {
  return (items || []).filter(
    (row) => row?.enabled !== false && (!row.status || row.status === 'current')
  );
}

/**
 * 分页加载模型广场（对齐 ModelCatalog GET /api/model_catalog）
 * @param {object} opts
 * @param {Set<string>|null} [opts.availableSet] 非管理员可用模型 ID，由页面缓存
 */
export async function fetchModelsPage({
  user,
  page = 1,
  pageSize = MODEL_PAGE_SIZE,
  search = '',
  filterKey = 'all',
  availableSet = null,
}) {
  const admin = isAdminUser(user);
  let allowed = availableSet;
  if (!admin && !allowed) {
    allowed = new Set(await fetchUserAvailableModelIds());
  }

  const apiFilters = filterToApiParams(filterKey);

  try {
    const { items, total, grandTotal } = await fetchModelCatalogPage({
      page,
      pageSize,
      search: String(search || '').trim(),
      ...apiFilters,
    });

    let list = normalizeCatalogRows(items);
    if (!admin && allowed) {
      list = list.filter((row) => allowed.has(String(row.model_id || '').trim()));
    }

    return {
      items: list,
      total,
      grandTotal,
      page,
      pageSize,
      fromCatalog: true,
      availableCount: allowed ? allowed.size : total,
    };
  } catch (err) {
    if (!admin && allowed?.size) {
      const ids = [...allowed].sort();
      const q = String(search || '').trim().toLowerCase();
      const matched = ids.filter((id) => !q || id.toLowerCase().includes(q));
      const filteredIds =
        filterKey && filterKey !== 'all'
          ? matched.filter((id) => matchesCategory({ model_id: id }, filterKey))
          : matched;
      const totalFb = filteredIds.length;
      const start = (page - 1) * pageSize;
      const slice = filteredIds.slice(start, start + pageSize).map((model_id) => ({
        model_id,
        model_name: model_id,
        enabled: true,
        status: 'current',
      }));
      return {
        items: slice,
        total: totalFb,
        grandTotal: totalFb,
        page,
        pageSize,
        fromCatalog: false,
        availableCount: allowed.size,
      };
    }
    throw err;
  }
}
