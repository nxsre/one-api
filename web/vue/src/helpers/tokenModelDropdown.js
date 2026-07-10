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

/** Canonical storage value for a model bound to one channel. */
export function tokenModelChannelKey(channelId, model) {
  return `#${channelId}:${model}`;
}

/** Parse "model" or "#<channelId>:<model>". */
export function parseTokenModelEntry(value) {
  const s = String(value || '').trim();
  const m = /^#(\d+):(.+)$/.exec(s);
  if (!m) return { channelId: 0, model: s, scoped: false };
  return { channelId: Number(m[1]), model: m[2], scoped: true };
}

/**
 * @param {Array<{model:string,channel_id:number,channel_name:string}>} items
 * @param {string[]} selectedModels 当前已选模型（含编辑页已有令牌模型）
 * @param {(key:string)=>string} t i18n translate
 */
export function buildTokenModelOptionsFromDetail(items, selectedModels, t) {
  const seen = new Set();
  const opts = [];
  for (const row of items || []) {
    const m = String(row.model || '').trim();
    const cid = row.channel_id;
    if (!m || !cid) continue;
    const value = tokenModelChannelKey(cid, m);
    if (seen.has(value)) continue;
    seen.add(value);
    const cname = String(row.channel_name || '').trim();
    const label = cname ? `#${cid} ${cname} — ${m}` : `#${cid} — ${m}`;
    opts.push({ key: value, value, text: label });
  }
  const suffix = t('token.edit.model_legacy_suffix');
  const sep = t('token.edit.model_channel_sep');
  for (const raw of selectedModels || []) {
    const ms = String(raw || '').trim();
    if (!ms || seen.has(ms)) continue;
    const parsed = parseTokenModelEntry(ms);
    if (parsed.scoped) {
      seen.add(ms);
      opts.push({
        key: ms,
        value: ms,
        text: `#${parsed.channelId} — ${parsed.model}`,
      });
      continue;
    }
    const chs = [];
    for (const row of items || []) {
      if (String(row.model || '').trim() !== parsed.model) continue;
      const cid = row.channel_id;
      const cname = String(row.channel_name || '').trim();
      if (cid && !chs.some((x) => x.id === cid)) {
        chs.push({ id: cid, name: cname });
      }
    }
    seen.add(ms);
    const chPart = chs
      .map((c) => `#${c.id} ${c.name}`.trim())
      .filter(Boolean)
      .join(sep);
    opts.push({
      key: `${ms}__legacy`,
      value: ms,
      text: chPart ? `${parsed.model} — ${chPart} (${suffix})` : `${parsed.model} (${suffix})`,
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

/**
 * Merge channel-scoped model entries for selected channels into the current selection.
 * Plain model names that duplicate a newly scoped entry are removed.
 */
export function applyBulkChannelsToModels(items, channelIds, existingModels) {
  const ids = new Set(channelIds || []);
  const next = new Set(
    (existingModels || []).map((m) => String(m || '').trim()).filter(Boolean)
  );
  for (const row of items || []) {
    if (!ids.has(row.channel_id)) continue;
    const model = String(row.model || '').trim();
    if (!model) continue;
    next.delete(model);
    next.add(tokenModelChannelKey(row.channel_id, model));
  }
  return Array.from(next).sort((a, b) => String(a).localeCompare(String(b)));
}
