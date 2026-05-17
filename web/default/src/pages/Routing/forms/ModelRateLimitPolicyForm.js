import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Checkbox, Form, Icon, Segment, Tab, Table } from 'semantic-ui-react';
import {
  EMPTY_RATE_LIMIT_POLICY,
  safeParseObject,
  stringifyPolicy,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow';

const defaultRule = { qps: 80, burst: 120, concurrency: 20, daily_quota: 0 };

/** 用于判断父组件传入的 JSON 与当前表单序列化结果是否语义一致（忽略空白），避免草稿行（如 user id 为空）被 useEffect 覆盖 */
function canonicalJsonString(s) {
  try {
    return JSON.stringify(JSON.parse(String(s ?? '').trim() || '{}'));
  } catch {
    return String(s ?? '');
  }
}

function ModelRateLimitColGroup() {
  return (
    <colgroup>
      <col className="mrl-col-key" />
      <col className="mrl-col-num" />
      <col className="mrl-col-num" />
      <col className="mrl-col-num" />
      <col className="mrl-col-num" />
      <col className="mrl-col-action" />
    </colgroup>
  );
}

/** @param {{ value: string, onChange: (json: string) => void, disabled?: boolean, modelOpts: any[], groupOpts: any[], onAddModel: (v: string) => void, onAddGroup: (v: string) => void, lookupReady: boolean }} props */
export default function ModelRateLimitPolicyForm({
  value,
  onChange,
  disabled,
  modelOpts,
  groupOpts,
  onAddModel,
  onAddGroup,
  lookupReady,
}) {
  const { t } = useTranslation();
  const [st, setSt] = useState(() => fromRateJSON(value));

  useEffect(() => {
    setSt((prev) => {
      try {
        const prevSerialized = toRateJSON(prev);
        if (canonicalJsonString(prevSerialized) === canonicalJsonString(value)) {
          return prev;
        }
      } catch {
        /* fall through */
      }
      return fromRateJSON(value);
    });
  }, [value]);

  const commit = (next) => {
    setSt(next);
    const json = toRateJSON(next);
    queueMicrotask(() => {
      onChange(json);
    });
  };

  const starOpt = [{ key: '*', text: '*', value: '*' }];

  const tokenRows = st.tokenRows;
  const userRows = st.userRows;
  const groupRows = st.groupRows;

  const panes = [
    {
      menuItem: t('routing.form_rl_by_token'),
      render: () => (
        <Tab.Pane attached={false}>
          <p className="routing-form-hint">{t('routing.form_rl_by_token_hint')}</p>
          <Table compact celled size="small" className="model-rate-limit-policy-table">
            <ModelRateLimitColGroup />
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>{t('routing.form_rl_key_model_glob')}</Table.HeaderCell>
                <Table.HeaderCell>QPS</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_burst')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_concurrency')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_daily_quota')}</Table.HeaderCell>
                <Table.HeaderCell collapsing />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {tokenRows.map((row, i) => (
                <Table.Row key={i}>
                  <Table.Cell>
                    <Form.Dropdown
                      fluid
                      search
                      selection
                      allowAdditions
                      additionLabel={`${t('routing.combo_add_custom')} `}
                      disabled={!lookupReady || disabled}
                      options={[...starOpt, ...modelOpts]}
                      value={row.key}
                      onAddItem={(e, { value: v }) => onAddModel(v)}
                      onChange={(e, { value: v }) => {
                        const next = [...tokenRows];
                        next[i] = { ...next[i], key: v };
                        commit({ ...st, tokenRows: next });
                      }}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.qps}
                      onChange={(e, { value: v }) => {
                        const next = [...tokenRows];
                        next[i] = { ...next[i], qps: toInt(v, 0) };
                        commit({ ...st, tokenRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.burst}
                      onChange={(e, { value: v }) => {
                        const next = [...tokenRows];
                        next[i] = { ...next[i], burst: toInt(v, 0) };
                        commit({ ...st, tokenRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.concurrency}
                      onChange={(e, { value: v }) => {
                        const next = [...tokenRows];
                        next[i] = { ...next[i], concurrency: toInt(v, 0) };
                        commit({ ...st, tokenRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.daily_quota}
                      onChange={(e, { value: v }) => {
                        const next = [...tokenRows];
                        next[i] = { ...next[i], daily_quota: toInt(v, 0) };
                        commit({ ...st, tokenRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      type="button"
                      icon
                      basic
                      disabled={disabled || tokenRows.length <= 1}
                      onClick={() => {
                        const next = tokenRows.filter((_, j) => j !== i);
                        commit({
                          ...st,
                          tokenRows: next.length ? next : [{ key: '*', ...defaultRule }],
                        });
                      }}
                    >
                      <Icon name="trash" />
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
          <Button
            type="button"
            size="small"
            basic
            disabled={disabled}
            onMouseDown={(e) => {
              e.preventDefault();
            }}
            onClick={() =>
              commit({
                ...st,
                tokenRows: [...tokenRows, { key: '*', ...defaultRule }],
              })
            }
          >
            {t('routing.form_add_row')}
          </Button>
        </Tab.Pane>
      ),
    },
    {
      menuItem: t('routing.form_rl_by_user'),
      render: () => (
        <Tab.Pane attached={false}>
          <p className="routing-form-hint">{t('routing.form_rl_by_user_hint')}</p>
          <Table compact celled size="small" className="model-rate-limit-policy-table">
            <ModelRateLimitColGroup />
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>{t('routing.form_rl_key_user_id')}</Table.HeaderCell>
                <Table.HeaderCell>QPS</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_burst')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_concurrency')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_daily_quota')}</Table.HeaderCell>
                <Table.HeaderCell collapsing />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {userRows.map((row, i) => (
                <Table.Row key={i}>
                  <Table.Cell>
                    <Form.Input
                      value={row.key}
                      onChange={(e, { value: v }) => {
                        const next = [...userRows];
                        next[i] = { ...next[i], key: v };
                        commit({ ...st, userRows: next });
                      }}
                      disabled={disabled}
                      placeholder="user id"
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.qps}
                      onChange={(e, { value: v }) => {
                        const next = [...userRows];
                        next[i] = { ...next[i], qps: toInt(v, 0) };
                        commit({ ...st, userRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.burst}
                      onChange={(e, { value: v }) => {
                        const next = [...userRows];
                        next[i] = { ...next[i], burst: toInt(v, 0) };
                        commit({ ...st, userRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.concurrency}
                      onChange={(e, { value: v }) => {
                        const next = [...userRows];
                        next[i] = { ...next[i], concurrency: toInt(v, 0) };
                        commit({ ...st, userRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.daily_quota}
                      onChange={(e, { value: v }) => {
                        const next = [...userRows];
                        next[i] = { ...next[i], daily_quota: toInt(v, 0) };
                        commit({ ...st, userRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      type="button"
                      icon
                      basic
                      disabled={disabled}
                      onClick={() => {
                        commit({ ...st, userRows: userRows.filter((_, j) => j !== i) });
                      }}
                    >
                      <Icon name="trash" />
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
          <Button
            type="button"
            size="small"
            basic
            disabled={disabled}
            onMouseDown={(e) => {
              e.preventDefault();
            }}
            onClick={() =>
              commit({
                ...st,
                userRows: [...userRows, { key: '', ...defaultRule }],
              })
            }
          >
            {t('routing.form_add_row')}
          </Button>
        </Tab.Pane>
      ),
    },
    {
      menuItem: t('routing.form_rl_by_group'),
      render: () => (
        <Tab.Pane attached={false}>
          <p className="routing-form-hint">{t('routing.form_rl_by_group_hint')}</p>
          <Table compact celled size="small" className="model-rate-limit-policy-table">
            <ModelRateLimitColGroup />
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>{t('routing.group')}</Table.HeaderCell>
                <Table.HeaderCell>QPS</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_burst')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_concurrency')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_rl_daily_quota')}</Table.HeaderCell>
                <Table.HeaderCell collapsing />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {groupRows.map((row, i) => (
                <Table.Row key={i}>
                  <Table.Cell>
                    <Form.Dropdown
                      fluid
                      search
                      selection
                      allowAdditions
                      additionLabel={`${t('routing.combo_add_custom')} `}
                      disabled={!lookupReady || disabled}
                      options={groupOpts}
                      value={row.key}
                      onAddItem={(e, { value: v }) => onAddGroup(v)}
                      onChange={(e, { value: v }) => {
                        const next = [...groupRows];
                        next[i] = { ...next[i], key: v };
                        commit({ ...st, groupRows: next });
                      }}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.qps}
                      onChange={(e, { value: v }) => {
                        const next = [...groupRows];
                        next[i] = { ...next[i], qps: toInt(v, 0) };
                        commit({ ...st, groupRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.burst}
                      onChange={(e, { value: v }) => {
                        const next = [...groupRows];
                        next[i] = { ...next[i], burst: toInt(v, 0) };
                        commit({ ...st, groupRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.concurrency}
                      onChange={(e, { value: v }) => {
                        const next = [...groupRows];
                        next[i] = { ...next[i], concurrency: toInt(v, 0) };
                        commit({ ...st, groupRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={row.daily_quota}
                      onChange={(e, { value: v }) => {
                        const next = [...groupRows];
                        next[i] = { ...next[i], daily_quota: toInt(v, 0) };
                        commit({ ...st, groupRows: next });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      type="button"
                      icon
                      basic
                      disabled={disabled}
                      onClick={() => {
                        commit({ ...st, groupRows: groupRows.filter((_, j) => j !== i) });
                      }}
                    >
                      <Icon name="trash" />
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
          <Button
            type="button"
            size="small"
            basic
            disabled={disabled}
            onMouseDown={(e) => {
              e.preventDefault();
            }}
            onClick={() =>
              commit({
                ...st,
                groupRows: [...groupRows, { key: 'default', ...defaultRule }],
              })
            }
          >
            {t('routing.form_add_row')}
          </Button>
        </Tab.Pane>
      ),
    },
  ];

  return (
    <Segment basic className="routing-policy-form-segment">
      <Form>
        <Form.Field>
          <Checkbox
            label={t('routing.form_rl_enabled')}
            checked={!!st.enabled}
            onChange={(e, { checked }) => commit({ ...st, enabled: !!checked })}
            disabled={disabled}
          />
        </Form.Field>
      </Form>
      <Tab panes={panes} />
      <PolicyFormResetRow
        disabled={disabled}
        onReset={() =>
          commit(fromRateJSON(stringifyPolicy(EMPTY_RATE_LIMIT_POLICY)))
        }
      />
    </Segment>
  );
}

function mapToRows(m) {
  const o = m && typeof m === 'object' ? m : {};
  return Object.entries(o).map(([key, r]) => ({
    key,
    qps: toInt(r?.qps, 0),
    burst: toInt(r?.burst, 0),
    concurrency: toInt(r?.concurrency, 0),
    daily_quota: toInt(r?.daily_quota, 0),
  }));
}

function fromRateJSON(raw) {
  const o = safeParseObject(raw, EMPTY_RATE_LIMIT_POLICY);
  const tokenRows = mapToRows(o.by_token);
  const userRows = mapToRows(o.by_user);
  const groupRows = mapToRows(o.by_group);
  return {
    enabled: o.enabled !== false,
    tokenRows: tokenRows.length ? tokenRows : [{ key: '*', ...defaultRule }],
    userRows,
    groupRows,
  };
}

function toRateJSON(st) {
  const toMap = (rows) => {
    const m = {};
    for (const r of rows) {
      const k = String(r.key ?? '').trim();
      if (!k) continue;
      m[k] = {
        qps: toInt(r.qps, 0),
        burst: toInt(r.burst, 0),
        concurrency: toInt(r.concurrency, 0),
        daily_quota: toInt(r.daily_quota, 0),
      };
    }
    return m;
  };
  return stringifyPolicy({
    enabled: !!st.enabled,
    by_token: toMap(st.tokenRows),
    by_user: toMap(st.userRows),
    by_group: toMap(st.groupRows),
  });
}
