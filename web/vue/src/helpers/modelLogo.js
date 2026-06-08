// 模型品牌 logo 解析：根据模型 ID（主）与 provider（辅）推断品牌，返回品牌名、主题色、
// 短标识与可选内置 SVG。常见品牌内置简洁 SVG 图标，其余回落为「品牌色 + 首字母色块」。
//
// 设计为纯前端、零外部依赖（离线可用）。新增品牌只需往 BRANDS 里加一条；要换成真实矢量
// logo，给该品牌补一个 svg 字段即可，组件无需改动。

// 极简内置 SVG（自绘抽象标记，非官方矢量），viewBox 统一 0 0 24 24。
const SVG = {
  gemini:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2c.45 5.05 2.5 7.1 8 7.5-5.5.4-7.55 2.45-8 7.5-.45-5.05-2.5-7.1-8-7.5 5.5-.4 7.55-2.45 8-7.5z" fill="#fff"/></svg>',
  anthropic:
    '<svg viewBox="0 0 24 24" aria-hidden="true" stroke="#fff" stroke-width="2.4" stroke-linecap="round"><path d="M12 3v18M3 12h18M5.6 5.6l12.8 12.8M18.4 5.6 5.6 18.4"/></svg>',
  mistral:
    '<svg viewBox="0 0 24 24" aria-hidden="true" fill="#fff"><rect x="3" y="5" width="18" height="3.4"/><rect x="3" y="10.3" width="18" height="3.4"/><rect x="3" y="15.6" width="18" height="3.4"/></svg>',
  xai:
    '<svg viewBox="0 0 24 24" aria-hidden="true" stroke="#fff" stroke-width="2.6" stroke-linecap="round"><path d="M5 5l14 14M19 5L5 19"/></svg>',
};

// 品牌注册表：match 命中模型 ID（小写）则采用；color 为色块底色；short 为兜底首字母；svg 可选。
const BRANDS = [
  { key: 'openai', name: 'OpenAI', color: '#10A37F', short: 'AI', match: /^(gpt|o[1-9]|chatgpt|codex|davinci|babbage|text-embedding|text-moderation|omni-moderation|whisper|tts|dall-e|dalle|sora)\b/ },
  { key: 'anthropic', name: 'Anthropic', color: '#D97757', short: 'An', svg: SVG.anthropic, match: /claude/ },
  { key: 'google', name: 'Google', color: '#1A73E8', short: 'G', svg: SVG.gemini, match: /(gemini|gemma|palm|bison|imagen|veo)/ },
  { key: 'deepseek', name: 'DeepSeek', color: '#4D6BFE', short: 'DS', match: /deepseek/ },
  { key: 'qwen', name: '通义千问', color: '#615CED', short: 'Qw', match: /(qwen|qwq|qvq|tongyi)/ },
  { key: 'meta', name: 'Meta Llama', color: '#0866FF', short: 'Ll', match: /(llama|meta-llama|codellama)/ },
  { key: 'mistral', name: 'Mistral', color: '#FA500F', short: 'Mi', svg: SVG.mistral, match: /(mistral|mixtral|codestral|ministral|pixtral|magistral|devstral)/ },
  { key: 'xai', name: 'xAI Grok', color: '#000000', short: 'x', svg: SVG.xai, match: /grok/ },
  { key: 'moonshot', name: 'Moonshot Kimi', color: '#16191E', short: 'Ki', match: /(moonshot|kimi)/ },
  { key: 'zhipu', name: '智谱 GLM', color: '#3859FF', short: 'GLM', match: /(glm|chatglm|zhipu|cogview|cogvideo)/ },
  { key: 'baidu', name: '文心一言', color: '#2932E1', short: '文', match: /(ernie|wenxin|baidu)/ },
  { key: 'doubao', name: '豆包', color: '#2F6BFF', short: '豆', match: /(doubao|volc)/ },
  { key: 'stepfun', name: '阶跃星辰', color: '#005AFF', short: 'St', match: /step-/ },
  { key: 'minimax', name: 'MiniMax', color: '#E8454E', short: 'MM', match: /(abab|minimax|hailuo)/ },
  { key: 'yi', name: '零一万物', color: '#003425', short: 'Yi', match: /(^yi-|01-ai|yi-vl|yi-large)/ },
  { key: 'cohere', name: 'Cohere', color: '#39594D', short: 'Co', match: /(command|cohere|rerank|aya)/ },
  { key: 'microsoft', name: 'Microsoft Phi', color: '#00A4EF', short: 'Ph', match: /(^phi-|phi3|phi-3|wizardlm)/ },
  { key: 'nvidia', name: 'NVIDIA', color: '#76B900', short: 'Nv', match: /(nemotron|nvidia)/ },
  { key: 'perplexity', name: 'Perplexity', color: '#20808D', short: 'Pe', match: /(sonar|pplx|perplexity)/ },
  { key: 'stability', name: 'Stable Diffusion', color: '#7C3AED', short: 'SD', match: /(stable-diffusion|stable_diffusion|sdxl|^sd[-3]|flux)/ },
  { key: 'midjourney', name: 'Midjourney', color: '#111111', short: 'MJ', match: /midjourney/ },
];

// 兜底配色池（按字符串散列稳定取色），用于未知品牌的首字母色块。
const FALLBACK_COLORS = ['#5B6B8C', '#3F7CAC', '#5E8B7E', '#A86B3C', '#7E5A9B', '#46707E', '#8C5B6B', '#566B3F'];

function hashColor(s) {
  let h = 0;
  for (let i = 0; i < s.length; i += 1) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return FALLBACK_COLORS[h % FALLBACK_COLORS.length];
}

function initials(s) {
  const t = String(s || '').trim();
  if (!t) return '?';
  const m = t.match(/[A-Za-z0-9一-龥]/g);
  if (!m) return '?';
  return m.slice(0, 2).join('').toUpperCase();
}

// resolveModelBrand 返回 { key, name, color, short, svg|null }。
// modelId 为主依据；providerKey / providerDisplay 作为兜底（用于命名与首字母）。
export function resolveModelBrand(modelId, providerKey = '', providerDisplay = '') {
  const id = String(modelId || '').toLowerCase().trim();
  const pk = String(providerKey || '').toLowerCase().trim();
  for (const b of BRANDS) {
    if (b.match.test(id) || (pk && b.match.test(pk))) {
      return { key: b.key, name: b.name, color: b.color, short: b.short, svg: b.svg || null };
    }
  }
  const label = providerDisplay || providerKey || modelId || '';
  return {
    key: pk || 'unknown',
    name: providerDisplay || providerKey || '其他',
    color: hashColor(label),
    short: initials(label),
    svg: null,
  };
}
