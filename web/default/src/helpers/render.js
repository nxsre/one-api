import { Label, Message } from 'semantic-ui-react';
import { getChannelOption } from './helper';
import React from 'react';

export function renderText(text, limit) {
  if (text.length > limit) {
    return text.slice(0, limit - 3) + '...';
  }
  return text;
}

export function renderGroup(group) {
  if (group === '') {
    return <Label>default</Label>;
  }
  let groups = group.split(',');
  groups.sort();
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: '2px',
        rowGap: '6px',
      }}
    >
      {groups.map((group) => {
        if (group === 'vip' || group === 'pro') {
          return <Label color='yellow'>{group}</Label>;
        } else if (group === 'svip' || group === 'premium') {
          return <Label color='red'>{group}</Label>;
        }
        return <Label>{group}</Label>;
      })}
    </div>
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
  } else {
    return x;
  }
}

/** 本地缓存的「每单位货币对应额度」无效时（未加载、0、非数字）返回 null，避免 $NaN */
function quotaPerUnitFromStorage() {
  const raw = localStorage.getItem('quota_per_unit');
  const n = parseFloat(raw === null || raw === '' ? 'NaN' : raw);
  if (!Number.isFinite(n) || n <= 0) {
    return null;
  }
  return n;
}

export function renderQuota(quota, t, precision = 2) {
  const displayInCurrency =
    localStorage.getItem('display_in_currency') === 'true';
  const safeQuota = Number.isFinite(Number(quota)) ? Number(quota) : 0;
  const quotaPerUnit = quotaPerUnitFromStorage();

  if (displayInCurrency && quotaPerUnit != null) {
    const amount = (safeQuota / quotaPerUnit).toFixed(precision);
    return t('common.quota.display_short', { amount });
  }

  return renderNumber(safeQuota);
}

export function renderQuotaWithPrompt(quota, t) {
  const displayInCurrency =
    localStorage.getItem('display_in_currency') === 'true';
  const safeQuota = Number.isFinite(Number(quota)) ? Number(quota) : 0;
  const quotaPerUnit = quotaPerUnitFromStorage();

  if (displayInCurrency && quotaPerUnit != null) {
    const amount = (safeQuota / quotaPerUnit).toFixed(2);
    return ` (${t('common.quota.display', { amount })})`;
  }

  return '';
}

const colors = [
  'red',
  'orange',
  'yellow',
  'olive',
  'green',
  'teal',
  'blue',
  'violet',
  'purple',
  'pink',
  'brown',
  'grey',
  'black',
];

export function renderColorLabel(text) {
  let hash = 0;
  for (let i = 0; i < text.length; i++) {
    hash = text.charCodeAt(i) + ((hash << 5) - hash);
  }
  let index = Math.abs(hash % colors.length);
  return (
    <Label basic color={colors[index]}>
      {text}
    </Label>
  );
}

export function renderChannelTip(channelId) {
  let channel = getChannelOption(channelId);
  if (channel === undefined || channel.tip === undefined) {
    return <></>;
  }
  return (
    <Message>
      <div dangerouslySetInnerHTML={{ __html: channel.tip }}></div>
    </Message>
  );
}
