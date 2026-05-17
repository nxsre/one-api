import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Form,
  Header,
  Icon,
  Segment,
  Table,
} from 'semantic-ui-react';
import {
  EMPTY_ALIAS_POLICY,
  safeParseObject,
  stringifyPolicy,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow';

/** @param {{ value: string, onChange: (json: string) => void, disabled?: boolean, modelOpts: any[], groupOpts: any[], onAddModel: (v: string) => void, onAddGroup: (v: string) => void, lookupReady: boolean }} props */
export default function ModelAliasPolicyForm({
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
  const [st, setSt] = useState(() => fromAliasJSON(value));

  useEffect(() => {
    setSt(fromAliasJSON(value));
  }, [value]);

  const commit = (next) => {
    setSt(next);
    onChange(toAliasJSON(next));
  };

  const emptyTarget = () => ({ model: '', weight: 10, priority: 100 });

  return (
    <Segment basic className="routing-policy-form-segment">
      <Form>
        <Form.Input
          label={t('routing.form_alias_max_chain_steps')}
          type="number"
          value={st.max_chain_steps}
          onChange={(e, { value: v }) => commit({ ...st, max_chain_steps: toInt(v, 32) })}
          disabled={disabled}
        />
      </Form>

      <Header as="h5">{t('routing.form_alias_chains')}</Header>
      <p className="routing-form-hint">{t('routing.form_alias_chains_hint')}</p>
      <Table compact celled size="small">
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell>{t('routing.form_alias_chain_from')}</Table.HeaderCell>
            <Table.HeaderCell>{t('routing.form_alias_chain_to')}</Table.HeaderCell>
            <Table.HeaderCell collapsing />
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {st.chainRows.map((row, i) => (
            <Table.Row key={i}>
              <Table.Cell>
                <Form.Dropdown
                  fluid
                  search
                  selection
                  allowAdditions
                  additionLabel={`${t('routing.combo_add_custom')} `}
                  disabled={!lookupReady || disabled}
                  options={modelOpts}
                  value={row.from}
                  onAddItem={(e, { value: v }) => onAddModel(v)}
                  onChange={(e, { value: v }) => {
                    const next = [...st.chainRows];
                    next[i] = { ...next[i], from: v };
                    commit({ ...st, chainRows: next });
                  }}
                />
              </Table.Cell>
              <Table.Cell>
                <Form.Dropdown
                  fluid
                  search
                  selection
                  allowAdditions
                  additionLabel={`${t('routing.combo_add_custom')} `}
                  disabled={!lookupReady || disabled}
                  options={modelOpts}
                  value={row.to}
                  onAddItem={(e, { value: v }) => onAddModel(v)}
                  onChange={(e, { value: v }) => {
                    const next = [...st.chainRows];
                    next[i] = { ...next[i], to: v };
                    commit({ ...st, chainRows: next });
                  }}
                />
              </Table.Cell>
              <Table.Cell>
                <Button
                  type="button"
                  icon
                  basic
                  disabled={disabled || st.chainRows.length <= 1}
                  onClick={() => {
                    const next = st.chainRows.filter((_, j) => j !== i);
                    commit({ ...st, chainRows: next.length ? next : [{ from: '', to: '' }] });
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
        onClick={() => commit({ ...st, chainRows: [...st.chainRows, { from: '', to: '' }] })}
      >
        {t('routing.form_add_chain_row')}
      </Button>

      <Header as="h5" style={{ marginTop: '1.25rem' }}>
        {t('routing.form_alias_aliases')}
      </Header>
      <p className="routing-form-hint">{t('routing.form_alias_aliases_hint')}</p>
      {st.aliasBlocks.map((block, bi) => (
        <Segment key={bi} style={{ marginBottom: '1rem' }}>
          <Form.Group widths="equal">
            <Form.Dropdown
              label={t('routing.form_alias_logical_model')}
              fluid
              search
              selection
              allowAdditions
              additionLabel={`${t('routing.combo_add_custom')} `}
              disabled={!lookupReady || disabled}
              options={modelOpts}
              value={block.logical}
              onAddItem={(e, { value: v }) => onAddModel(v)}
              onChange={(e, { value: v }) => {
                const blocks = [...st.aliasBlocks];
                blocks[bi] = { ...blocks[bi], logical: v };
                commit({ ...st, aliasBlocks: blocks });
              }}
            />
            <Form.Field>
              <label>&nbsp;</label>
              <Button
                type="button"
                negative
                basic
                disabled={disabled || st.aliasBlocks.length <= 1}
                onClick={() => {
                  const blocks = st.aliasBlocks.filter((_, j) => j !== bi);
                  commit({
                    ...st,
                    aliasBlocks: blocks.length
                      ? blocks
                      : [{ logical: '', targets: [emptyTarget()] }],
                  });
                }}
              >
                {t('routing.form_remove_alias_block')}
              </Button>
            </Form.Field>
          </Form.Group>
          <Table compact size="small">
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>{t('routing.model')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.multiplier')}</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.form_alias_priority')}</Table.HeaderCell>
                <Table.HeaderCell collapsing />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {block.targets.map((tg, ti) => (
                <Table.Row key={ti}>
                  <Table.Cell>
                    <Form.Dropdown
                      fluid
                      search
                      selection
                      allowAdditions
                      additionLabel={`${t('routing.combo_add_custom')} `}
                      disabled={!lookupReady || disabled}
                      options={modelOpts}
                      value={tg.model}
                      onAddItem={(e, { value: v }) => onAddModel(v)}
                      onChange={(e, { value: v }) => {
                        const blocks = [...st.aliasBlocks];
                        const tgs = [...blocks[bi].targets];
                        tgs[ti] = { ...tgs[ti], model: v };
                        blocks[bi] = { ...blocks[bi], targets: tgs };
                        commit({ ...st, aliasBlocks: blocks });
                      }}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={tg.weight}
                      onChange={(e, { value: v }) => {
                        const blocks = [...st.aliasBlocks];
                        const tgs = [...blocks[bi].targets];
                        tgs[ti] = { ...tgs[ti], weight: toInt(v, 0) };
                        blocks[bi] = { ...blocks[bi], targets: tgs };
                        commit({ ...st, aliasBlocks: blocks });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Form.Input
                      type="number"
                      value={tg.priority}
                      onChange={(e, { value: v }) => {
                        const blocks = [...st.aliasBlocks];
                        const tgs = [...blocks[bi].targets];
                        tgs[ti] = { ...tgs[ti], priority: toInt(v, 0) };
                        blocks[bi] = { ...blocks[bi], targets: tgs };
                        commit({ ...st, aliasBlocks: blocks });
                      }}
                      disabled={disabled}
                    />
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      type="button"
                      icon
                      basic
                      disabled={disabled || block.targets.length <= 1}
                      onClick={() => {
                        const blocks = [...st.aliasBlocks];
                        const tgs = blocks[bi].targets.filter((_, j) => j !== ti);
                        blocks[bi] = { ...blocks[bi], targets: tgs.length ? tgs : [emptyTarget()] };
                        commit({ ...st, aliasBlocks: blocks });
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
            onClick={() => {
              const blocks = [...st.aliasBlocks];
              blocks[bi] = {
                ...blocks[bi],
                targets: [...blocks[bi].targets, emptyTarget()],
              };
              commit({ ...st, aliasBlocks: blocks });
            }}
          >
            {t('routing.form_add_target_row')}
          </Button>
        </Segment>
      ))}
      <Button
        type="button"
        size="small"
        basic
        disabled={disabled}
        onClick={() =>
          commit({
            ...st,
            aliasBlocks: [...st.aliasBlocks, { logical: '', targets: [emptyTarget()] }],
          })
        }
      >
        {t('routing.form_add_alias_block')}
      </Button>

      <Header as="h5" style={{ marginTop: '1.5rem' }}>
        {t('routing.form_alias_group_overrides')}
      </Header>
      <p className="routing-form-hint">{t('routing.form_alias_group_hint')}</p>
      {st.groupBlocks.map((gb, gi) => (
        <Segment key={gi} style={{ marginBottom: '1rem' }} className="routing-group-override-segment">
          <Header as="h5">
            {t('routing.group')}: {gb.group || '—'}
          </Header>
              <Form.Dropdown
                label={t('routing.group')}
                fluid
                search
                selection
                allowAdditions
                additionLabel={`${t('routing.combo_add_custom')} `}
                disabled={!lookupReady || disabled}
                options={groupOpts}
                value={gb.group}
                onAddItem={(e, { value: v }) => onAddGroup(v)}
                onChange={(e, { value: v }) => {
                  const gbs = [...st.groupBlocks];
                  gbs[gi] = { ...gbs[gi], group: v };
                  commit({ ...st, groupBlocks: gbs });
                }}
              />
                <Header as="h6">{t('routing.form_alias_chains')}</Header>
                <Table compact size="small">
                  <Table.Body>
                    {gb.chainRows.map((row, i) => (
                      <Table.Row key={i}>
                        <Table.Cell>
                          <Form.Dropdown
                            fluid
                            search
                            selection
                            allowAdditions
                            disabled={!lookupReady || disabled}
                            options={modelOpts}
                            value={row.from}
                            onAddItem={(e, { value: v }) => onAddModel(v)}
                            onChange={(e, { value: v }) => {
                              const gbs = [...st.groupBlocks];
                              const rows = [...gbs[gi].chainRows];
                              rows[i] = { ...rows[i], from: v };
                              gbs[gi] = { ...gbs[gi], chainRows: rows };
                              commit({ ...st, groupBlocks: gbs });
                            }}
                          />
                        </Table.Cell>
                        <Table.Cell>
                          <Form.Dropdown
                            fluid
                            search
                            selection
                            allowAdditions
                            disabled={!lookupReady || disabled}
                            options={modelOpts}
                            value={row.to}
                            onAddItem={(e, { value: v }) => onAddModel(v)}
                            onChange={(e, { value: v }) => {
                              const gbs = [...st.groupBlocks];
                              const rows = [...gbs[gi].chainRows];
                              rows[i] = { ...rows[i], to: v };
                              gbs[gi] = { ...gbs[gi], chainRows: rows };
                              commit({ ...st, groupBlocks: gbs });
                            }}
                          />
                        </Table.Cell>
                        <Table.Cell collapsing>
                          <Button
                            type="button"
                            icon
                            basic
                            disabled={disabled || gb.chainRows.length <= 1}
                            onClick={() => {
                              const gbs = [...st.groupBlocks];
                              const rows = gb.chainRows.filter((_, j) => j !== i);
                              gbs[gi] = { ...gbs[gi], chainRows: rows.length ? rows : [{ from: '', to: '' }] };
                              commit({ ...st, groupBlocks: gbs });
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
                  size="mini"
                  basic
                  disabled={disabled}
                  onClick={() => {
                    const gbs = [...st.groupBlocks];
                    gbs[gi] = { ...gbs[gi], chainRows: [...gbs[gi].chainRows, { from: '', to: '' }] };
                    commit({ ...st, groupBlocks: gbs });
                  }}
                >
                  {t('routing.form_add_chain_row')}
                </Button>

                <Header as="h6">{t('routing.form_alias_aliases')}</Header>
                {gb.aliasBlocks.map((block, bi) => (
                  <Segment key={bi} size="small">
                    <Form.Dropdown
                      label={t('routing.form_alias_logical_model')}
                      fluid
                      search
                      selection
                      allowAdditions
                      disabled={!lookupReady || disabled}
                      options={modelOpts}
                      value={block.logical}
                      onAddItem={(e, { value: v }) => onAddModel(v)}
                      onChange={(e, { value: v }) => {
                        const gbs = [...st.groupBlocks];
                        const ab = [...gbs[gi].aliasBlocks];
                        ab[bi] = { ...ab[bi], logical: v };
                        gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
                        commit({ ...st, groupBlocks: gbs });
                      }}
                    />
                    <Table compact size="small">
                      <Table.Body>
                        {block.targets.map((tg, ti) => (
                          <Table.Row key={ti}>
                            <Table.Cell>
                              <Form.Dropdown
                                fluid
                                search
                                selection
                                allowAdditions
                                disabled={!lookupReady || disabled}
                                options={modelOpts}
                                value={tg.model}
                                onAddItem={(e, { value: v }) => onAddModel(v)}
                                onChange={(e, { value: v }) => {
                                  const gbs = [...st.groupBlocks];
                                  const ab = [...gbs[gi].aliasBlocks];
                                  const tgs = [...ab[bi].targets];
                                  tgs[ti] = { ...tgs[ti], model: v };
                                  ab[bi] = { ...ab[bi], targets: tgs };
                                  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
                                  commit({ ...st, groupBlocks: gbs });
                                }}
                              />
                            </Table.Cell>
                            <Table.Cell>
                              <Form.Input
                                type="number"
                                value={tg.weight}
                                onChange={(e, { value: v }) => {
                                  const gbs = [...st.groupBlocks];
                                  const ab = [...gbs[gi].aliasBlocks];
                                  const tgs = [...ab[bi].targets];
                                  tgs[ti] = { ...tgs[ti], weight: toInt(v, 0) };
                                  ab[bi] = { ...ab[bi], targets: tgs };
                                  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
                                  commit({ ...st, groupBlocks: gbs });
                                }}
                                disabled={disabled}
                              />
                            </Table.Cell>
                            <Table.Cell>
                              <Form.Input
                                type="number"
                                value={tg.priority}
                                onChange={(e, { value: v }) => {
                                  const gbs = [...st.groupBlocks];
                                  const ab = [...gbs[gi].aliasBlocks];
                                  const tgs = [...ab[bi].targets];
                                  tgs[ti] = { ...tgs[ti], priority: toInt(v, 0) };
                                  ab[bi] = { ...ab[bi], targets: tgs };
                                  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
                                  commit({ ...st, groupBlocks: gbs });
                                }}
                                disabled={disabled}
                              />
                            </Table.Cell>
                            <Table.Cell collapsing>
                              <Button
                                type="button"
                                icon
                                basic
                                disabled={disabled || block.targets.length <= 1}
                                onClick={() => {
                                  const gbs = [...st.groupBlocks];
                                  const ab = [...gbs[gi].aliasBlocks];
                                  const tgs = ab[bi].targets.filter((_, j) => j !== ti);
                                  ab[bi] = { ...ab[bi], targets: tgs.length ? tgs : [emptyTarget()] };
                                  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
                                  commit({ ...st, groupBlocks: gbs });
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
                      size="mini"
                      basic
                      disabled={disabled}
                      onClick={() => {
                        const gbs = [...st.groupBlocks];
                        const ab = [...gbs[gi].aliasBlocks];
                        ab[bi] = { ...ab[bi], targets: [...ab[bi].targets, emptyTarget()] };
                        gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
                        commit({ ...st, groupBlocks: gbs });
                      }}
                    >
                      {t('routing.form_add_target_row')}
                    </Button>
                    <Button
                      type="button"
                      size="mini"
                      negative
                      basic
                      disabled={disabled || gb.aliasBlocks.length <= 1}
                      onClick={() => {
                        const gbs = [...st.groupBlocks];
                        const ab = gbs[gi].aliasBlocks.filter((_, j) => j !== bi);
                        gbs[gi] = {
                          ...gbs[gi],
                          aliasBlocks: ab.length ? ab : [{ logical: '', targets: [emptyTarget()] }],
                        };
                        commit({ ...st, groupBlocks: gbs });
                      }}
                    >
                      {t('routing.form_remove_alias_block')}
                    </Button>
                  </Segment>
                ))}
                <Button
                  type="button"
                  size="mini"
                  basic
                  disabled={disabled}
                  onClick={() => {
                    const gbs = [...st.groupBlocks];
                    gbs[gi] = {
                      ...gbs[gi],
                      aliasBlocks: [...gbs[gi].aliasBlocks, { logical: '', targets: [emptyTarget()] }],
                    };
                    commit({ ...st, groupBlocks: gbs });
                  }}
                >
                  {t('routing.form_add_alias_block')}
                </Button>

                <Button
                  type="button"
                  negative
                  basic
                  size="small"
                  style={{ marginTop: '0.75rem' }}
                  disabled={disabled}
                  onClick={() => {
                    commit({ ...st, groupBlocks: st.groupBlocks.filter((_, j) => j !== gi) });
                  }}
                >
                  {t('routing.form_remove_group_override')}
                </Button>
        </Segment>
      ))}

      <Button
        type="button"
        size="small"
        basic
        style={{ marginTop: '0.75rem' }}
        disabled={disabled}
        onClick={() =>
          commit({
            ...st,
            groupBlocks: [
              ...st.groupBlocks,
              {
                group: '',
                chainRows: [{ from: '', to: '' }],
                aliasBlocks: [{ logical: '', targets: [emptyTarget()] }],
              },
            ],
          })
        }
      >
        {t('routing.form_add_group_override')}
      </Button>

      <PolicyFormResetRow
        disabled={disabled}
        onReset={() =>
          commit(fromAliasJSON(stringifyPolicy(EMPTY_ALIAS_POLICY)))
        }
      />
    </Segment>
  );
}

function fromAliasJSON(raw) {
  const o = safeParseObject(raw, EMPTY_ALIAS_POLICY);
  const chainRows = Object.entries(o.chains || {}).map(([from, to]) => ({
    from,
    to,
  }));
  const aliasBlocks = Object.entries(o.aliases || {}).map(([logical, targets]) => ({
    logical,
    targets: (Array.isArray(targets) ? targets : []).map((t) => ({
      model: t?.model ?? '',
      weight: toInt(t?.weight, 0),
      priority: Number(t?.priority) || 0,
    })),
  }));
  const groupBlocks = Object.entries(o.group_overrides || {}).map(([group, frag]) => {
    const f = frag && typeof frag === 'object' ? frag : {};
    const cr = Object.entries(f.chains || {}).map(([from, to]) => ({ from, to }));
    const ab = Object.entries(f.aliases || {}).map(([logical, targets]) => ({
      logical,
      targets: (Array.isArray(targets) ? targets : []).map((t) => ({
        model: t?.model ?? '',
        weight: toInt(t?.weight, 0),
        priority: Number(t?.priority) || 0,
      })),
    }));
    return {
      group,
      chainRows: cr.length ? cr : [{ from: '', to: '' }],
      aliasBlocks: ab.length ? ab : [{ logical: '', targets: [{ model: '', weight: 10, priority: 100 }] }],
    };
  });
  return {
    max_chain_steps: toInt(o.max_chain_steps, 32),
    chainRows: chainRows.length ? chainRows : [{ from: '', to: '' }],
    aliasBlocks: aliasBlocks.length
      ? aliasBlocks
      : [{ logical: '', targets: [{ model: '', weight: 10, priority: 100 }] }],
    groupBlocks,
  };
}

function toAliasJSON(st) {
  const chains = {};
  for (const r of st.chainRows) {
    const a = String(r.from || '').trim();
    const b = String(r.to || '').trim();
    if (a && b) chains[a] = b;
  }
  const aliases = {};
  for (const b of st.aliasBlocks) {
    const k = String(b.logical || '').trim();
    if (!k) continue;
    const arr = (b.targets || [])
      .filter((t) => String(t.model || '').trim())
      .map((t) => ({
        model: String(t.model).trim(),
        weight: toInt(t.weight, 0),
        priority: Number(t.priority) || 0,
      }));
    if (arr.length) aliases[k] = arr;
  }
  const group_overrides = {};
  for (const g of st.groupBlocks) {
    const gn = String(g.group || '').trim();
    if (!gn) continue;
    const gc = {};
    for (const r of g.chainRows || []) {
      const a = String(r.from || '').trim();
      const b = String(r.to || '').trim();
      if (a && b) gc[a] = b;
    }
    const ga = {};
    for (const b of g.aliasBlocks || []) {
      const k = String(b.logical || '').trim();
      if (!k) continue;
      const arr = (b.targets || [])
        .filter((t) => String(t.model || '').trim())
        .map((t) => ({
          model: String(t.model).trim(),
          weight: toInt(t.weight, 0),
          priority: Number(t.priority) || 0,
        }));
      if (arr.length) ga[k] = arr;
    }
    const frag = {};
    if (Object.keys(gc).length) frag.chains = gc;
    if (Object.keys(ga).length) frag.aliases = ga;
    if (Object.keys(frag).length) group_overrides[gn] = frag;
  }
  return stringifyPolicy({
    max_chain_steps: toInt(st.max_chain_steps, 32),
    chains,
    aliases,
    defaults: {},
    group_overrides,
  });
}
