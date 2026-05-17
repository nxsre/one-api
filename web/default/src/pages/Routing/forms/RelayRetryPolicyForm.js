import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Checkbox, Form, Segment } from 'semantic-ui-react';
import {
  EMPTY_RELAY_RETRY,
  HTTP_RETRY_CODE_OPTIONS,
  safeParseObject,
  stringifyPolicy,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow';

/** @param {{ value: string, onChange: (json: string) => void, disabled?: boolean }} props */
export default function RelayRetryPolicyForm({ value, onChange, disabled }) {
  const { t } = useTranslation();
  const [p, setP] = useState(() => parseRelay(safeParseObject(value, EMPTY_RELAY_RETRY)));

  useEffect(() => {
    setP(parseRelay(safeParseObject(value, EMPTY_RELAY_RETRY)));
  }, [value]);

  const emit = (next) => {
    setP(next);
    const o = { ...next };
    if (o.max_retries_override === null || o.max_retries_override === undefined || o.max_retries_override === '') {
      delete o.max_retries_override;
    }
    onChange(stringifyPolicy(o));
  };

  const nfi = (name) => (e, { value: v }) => {
    emit({ ...p, [name]: toInt(v, p[name] ?? 0) });
  };

  const nfcb = (name) => (e, { checked }) => {
    emit({ ...p, [name]: !!checked });
  };

  return (
    <Segment basic className="routing-policy-form-segment">
      <Form>
        <Form.Group widths="equal">
          <Form.Input
            label={t('routing.form_max_retries_override')}
            type="number"
            value={p.max_retries_override === null || p.max_retries_override === undefined ? '' : p.max_retries_override}
            onChange={(e, { value: v }) => {
              const s = String(v).trim();
              if (s === '') emit({ ...p, max_retries_override: null });
              else emit({ ...p, max_retries_override: toInt(s, 0) });
            }}
            disabled={disabled}
            placeholder={t('routing.form_placeholder_inherit_system')}
          />
          <Form.Input
            label={t('routing.form_base_backoff_ms')}
            type="number"
            value={p.base_backoff_ms}
            onChange={nfi('base_backoff_ms')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_max_backoff_ms')}
            type="number"
            value={p.max_backoff_ms}
            onChange={nfi('max_backoff_ms')}
            disabled={disabled}
          />
        </Form.Group>
        <Form.Field>
          <label>{t('routing.form_retry_allowlist')}</label>
          <Form.Dropdown
            fluid multiple selection search
            options={HTTP_RETRY_CODE_OPTIONS}
            value={p.retry_http_status_allowlist || []}
            onChange={(e, { value: v }) => emit({ ...p, retry_http_status_allowlist: v })}
            disabled={disabled}
            placeholder={t('routing.form_pick_http_codes')}
          />
        </Form.Field>
        <Form.Field>
          <label>{t('routing.form_retry_denylist')}</label>
          <Form.Dropdown
            fluid multiple selection search
            options={HTTP_RETRY_CODE_OPTIONS}
            value={p.retry_http_status_denylist || []}
            onChange={(e, { value: v }) => emit({ ...p, retry_http_status_denylist: v })}
            disabled={disabled}
            placeholder={t('routing.form_pick_http_codes')}
          />
        </Form.Field>
        <Form.Field>
          <Checkbox
            label={t('routing.form_force_different_channel')}
            checked={!!p.force_different_channel_each_attempt}
            onChange={nfcb('force_different_channel_each_attempt')}
            disabled={disabled}
          />
        </Form.Field>
        <PolicyFormResetRow
          disabled={disabled}
          onReset={() => emit({ ...EMPTY_RELAY_RETRY })}
        />
      </Form>
    </Segment>
  );
}

/** @param {Record<string, unknown>} o */
function parseRelay(o) {
  return {
    max_retries_override:
      o.max_retries_override === undefined ? null : o.max_retries_override,
    base_backoff_ms: toInt(o.base_backoff_ms, 150),
    max_backoff_ms: toInt(o.max_backoff_ms, 4000),
    retry_http_status_allowlist: Array.isArray(o.retry_http_status_allowlist)
      ? o.retry_http_status_allowlist.map((x) => toInt(x, 0))
      : [...EMPTY_RELAY_RETRY.retry_http_status_allowlist],
    retry_http_status_denylist: Array.isArray(o.retry_http_status_denylist)
      ? o.retry_http_status_denylist.map((x) => toInt(x, 0))
      : [],
    force_different_channel_each_attempt: !!o.force_different_channel_each_attempt,
  };
}
