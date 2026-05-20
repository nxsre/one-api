/** 合并接口返回的扁平模型 ID 列表与令牌已保存的模型（后者若不在列表中则带后缀展示）。 */
export function mergeTokenModelDropdownOptions(apiModels, selectedModels, legacySuffix) {
  const seen = new Set();
  const opts = [];
  const list = Array.isArray(apiModels) ? apiModels : [];
  for (const m of list) {
    if (!m || seen.has(m)) continue;
    seen.add(m);
    opts.push({ key: m, text: m, value: m });
  }
  const suffix = legacySuffix || 'saved';
  const sel = Array.isArray(selectedModels) ? selectedModels : [];
  for (const m of sel) {
    if (!m || seen.has(m)) continue;
    seen.add(m);
    opts.push({ key: m, text: `${m} (${suffix})`, value: m });
  }
  opts.sort((a, b) => String(a.value).localeCompare(String(b.value)));
  return opts;
}

/**
 * @param {Array<{model:string,channel_id:number,channel_name:string}>} items
 * @param {string[]} selectedModels 当前已选模型（含编辑页已有令牌模型）
 * @param {(key:string)=>string} t i18n translate
 */
export function buildTokenModelOptionsFromDetail(items, selectedModels, t) {
  const byModel = new Map();
  for (const row of items || []) {
    const m = String(row.model || '').trim();
    if (!m) continue;
    if (!byModel.has(m)) byModel.set(m, []);
    const arr = byModel.get(m);
    const cid = row.channel_id;
    const cname = String(row.channel_name || '').trim();
    if (!arr.some((x) => x.id === cid)) {
      arr.push({ id: cid, name: cname });
    }
  }
  const sep = t('token.edit.model_channel_sep');
  const opts = [];
  for (const [model, chs] of byModel) {
    const chPart = chs
      .map((c) => `#${c.id} ${c.name}`.trim())
      .filter(Boolean)
      .join(sep);
    opts.push({
      key: model,
      value: model,
      text: chPart ? `${model} — ${chPart}` : model,
    });
  }
  const seen = new Set(byModel.keys());
  const suffix = t('token.edit.model_legacy_suffix');
  for (const m of selectedModels || []) {
    const ms = String(m || '').trim();
    if (!ms || seen.has(ms)) continue;
    seen.add(ms);
    opts.push({
      key: `${ms}__legacy`,
      value: ms,
      text: `${ms} (${suffix})`,
    });
  }
  opts.sort((a, b) => String(a.value).localeCompare(String(b.value)));
  return opts;
}

/** 从明细行生成「按渠道批量添加」下拉选项 */
export function distinctChannelsFromModelDetailItems(items) {
  const m = new Map();
  for (const row of items || []) {
    const id = row.channel_id;
    if (id == null || Number(id) <= 0) continue;
    if (!m.has(id)) {
      const nm = String(row.channel_name || '').trim();
      m.set(id, {
        key: id,
        value: id,
        text: nm ? `#${id} ${nm}` : `#${id}`,
      });
    }
  }
  return Array.from(m.values()).sort((a, b) => a.value - b.value);
}
