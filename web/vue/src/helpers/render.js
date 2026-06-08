import { h } from 'vue';
import { Tag, Alert } from 'ant-design-vue';
import { getChannelOption } from './helper';

export function renderText(text, limit) {
  if (text.length > limit) {
    return text.slice(0, limit - 3) + '...';
  }
  return text;
}

export function renderGroup(group) {
  if (group === '') {
    return h(Tag, {}, () => 'default');
  }
  let groups = group.split(',');
  groups.sort();
  return h(
    'div',
    { style: { display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: '2px', rowGap: '6px' } },
    groups.map((g) => {
      if (g === 'vip' || g === 'pro') {
        return h(Tag, { color: 'gold' }, () => g);
      }
      if (g === 'svip' || g === 'premium') {
        return h(Tag, { color: 'red' }, () => g);
      }
      return h(Tag, {}, () => g);
    })
  );
}

export function renderNumber(num) {
  const n = Number(num);
  const x = Number.isFinite(n) ? n : 0;
  if (x >= 1000000000) {
    return (x / 1000000000).toFixed(1) + 'B';
  } else if (x >= 1000000) {
    return (x / 1000000).toFixed(1) + 'M';
  } else if (x >= 10000) {
    return (x / 1000).toFixed(1) + 'k';
  }
  return x;
}

/** 本地缓存的「每单位货币对应额度」无效时返回 null，避免 $NaN */
function quotaPerUnitFromStorage() {
  const raw = localStorage.getItem('quota_per_unit');
  const n = parseFloat(raw === null || raw === '' ? 'NaN' : raw);
  if (!Number.isFinite(n) || n <= 0) {
    return null;
  }
  return n;
}

/** 返回「每单位货币（$）对应额度」，无效时回退到默认 500000，供金额↔额度换算复用 */
export function getQuotaPerUnit() {
  return quotaPerUnitFromStorage() ?? 500000;
}

export function renderQuota(quota, t, precision = 2) {
  const displayInCurrency = localStorage.getItem('display_in_currency') === 'true';
  const safeQuota = Number.isFinite(Number(quota)) ? Number(quota) : 0;
  const quotaPerUnit = quotaPerUnitFromStorage();
  if (displayInCurrency && quotaPerUnit != null) {
    const amount = (safeQuota / quotaPerUnit).toFixed(precision);
    return t('common.quota.display_short', { amount });
  }
  return renderNumber(safeQuota);
}

export function renderQuotaWithPrompt(quota, t) {
  const displayInCurrency = localStorage.getItem('display_in_currency') === 'true';
  const safeQuota = Number.isFinite(Number(quota)) ? Number(quota) : 0;
  const quotaPerUnit = quotaPerUnitFromStorage();
  if (displayInCurrency && quotaPerUnit != null) {
    const amount = (safeQuota / quotaPerUnit).toFixed(2);
    return ` (${t('common.quota.display', { amount })})`;
  }
  return '';
}

// Ant Design Vue Tag preset colors.
const colors = [
  'red', 'orange', 'gold', 'lime', 'green', 'cyan', 'blue', 'geekblue',
  'purple', 'magenta', 'volcano', 'default',
];

export function renderColorLabel(text) {
  let hash = 0;
  for (let i = 0; i < text.length; i++) {
    hash = text.charCodeAt(i) + ((hash << 5) - hash);
  }
  let index = Math.abs(hash % colors.length);
  return h(Tag, { color: colors[index] }, () => text);
}

export function colorForLabel(text) {
  let hash = 0;
  for (let i = 0; i < text.length; i++) {
    hash = text.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash % colors.length)];
}

export function renderChannelTip(channelId) {
  let channel = getChannelOption(channelId);
  if (channel === undefined || channel.tip === undefined) {
    return null;
  }
  return h(Alert, {
    type: 'info',
    message: h('div', { innerHTML: channel.tip }),
  });
}
