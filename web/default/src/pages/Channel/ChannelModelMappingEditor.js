import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Form, Icon, Table } from 'semantic-ui-react';
import { showError, verifyJSON } from '../../helpers';

const MODEL_MAP_EXAMPLE_LINES = `{
  "gpt-3.5-turbo-0301": "gpt-3.5-turbo",
  "gpt-4-0314": "gpt-4"
}`;

function parseMappingToRows(str) {
  const s = String(str ?? '').trim();
  if (!s) {
    return [{ from: '', to: '' }];
  }
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      const ent = Object.entries(o);
      if (ent.length === 0) {
        return [{ from: '', to: '' }];
      }
      return ent.map(([from, to]) => ({
        from: String(from),
        to: String(to ?? ''),
      }));
    }
  } catch {
    /* fallthrough */
  }
  return [{ from: '', to: '' }];
}

function rowsToMappingJson(rows) {
  const o = {};
  for (const r of rows) {
    const k = String(r.from ?? '').trim();
    if (!k) continue;
    o[k] = String(r.to ?? '');
  }
  if (Object.keys(o).length === 0) {
    return '';
  }
  return JSON.stringify(o, null, 2);
}

/** @param {{ value: string, onChange: (v: string) => void, exampleHint?: string, upstreamModelOptions?: string[] }} props */
export default function ChannelModelMappingEditor({
  value,
  onChange,
  exampleHint,
  upstreamModelOptions = [],
}) {
  const { t } = useTranslation();
  const [rows, setRows] = useState(() => parseMappingToRows(value));
  const [jsonRaw, setJsonRaw] = useState('');
  const [jsonMode, setJsonMode] = useState(false);

  const upstreamOptions = React.useMemo(() => {
    const seen = new Set();
    const out = [];
    (upstreamModelOptions || []).forEach((id) => {
      const s = String(id || '').trim();
      if (!s || seen.has(s)) return;
      seen.add(s);
      out.push({ key: s, value: s, text: s });
    });
    return out;
  }, [upstreamModelOptions]);

  useEffect(() => {
    setRows(parseMappingToRows(value));
  }, [value]);

  const emitRows = (next) => {
    setRows(next);
    onChange(rowsToMappingJson(next));
  };

  const updateRow = (i, field, v) => {
    const next = rows.map((r, j) =>
      j === i ? { ...r, [field]: v } : r
    );
    emitRows(next);
  };

  const addRow = () => {
    emitRows([...rows, { from: '', to: '' }]);
  };

  const removeRow = (i) => {
    const next = rows.filter((_, j) => j !== i);
    emitRows(next.length ? next : [{ from: '', to: '' }]);
  };

  const applyJsonPaste = () => {
    const raw = jsonRaw.trim();
    if (!raw) {
      emitRows([{ from: '', to: '' }]);
      setJsonMode(false);
      return;
    }
    if (!verifyJSON(raw)) {
      showError(t('channel.edit.messages.model_mapping_invalid'));
      return;
    }
    let o;
    try {
      o = JSON.parse(raw);
    } catch {
      showError(t('channel.edit.messages.model_mapping_invalid'));
      return;
    }
    if (typeof o !== 'object' || o === null || Array.isArray(o)) {
      showError(t('channel.edit.messages.model_mapping_invalid'));
      return;
    }
    const next = parseMappingToRows(raw);
    emitRows(next);
    setJsonMode(false);
    setJsonRaw('');
  };

  return (
    <div className='channel-model-mapping-editor'>
      <p className='channel-edit-field-hint'>
        {exampleHint ||
          `${t('channel.edit.model_mapping_placeholder')}\n${MODEL_MAP_EXAMPLE_LINES}`}
      </p>
      {!jsonMode ? (
        <>
          <Table compact celled size='small' className='channel-model-mapping-table'>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>
                  {t('channel.edit.model_mapping_col_request')}
                </Table.HeaderCell>
                <Table.HeaderCell>
                  {t('channel.edit.model_mapping_col_upstream')}
                </Table.HeaderCell>
                <Table.HeaderCell collapsing />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {rows.map((r, i) => (
                <Table.Row key={i}>
                  <Table.Cell>
                    <Form.Input
                      fluid
                      placeholder={t(
                        'channel.edit.model_mapping_ph_request'
                      )}
                      value={r.from}
                      onChange={(e, { value: v }) =>
                        updateRow(i, 'from', v)
                      }
                    />
                  </Table.Cell>
                  <Table.Cell>
                    {upstreamOptions.length > 0 ? (
                      <Form.Dropdown
                        fluid
                        search
                        selection
                        allowAdditions
                        openOnFocus={false}
                        selectOnBlur={false}
                        options={(() => {
                          const current = String(r.to || '').trim();
                          if (!current || upstreamOptions.some((o) => o.value === current)) {
                            return upstreamOptions;
                          }
                          return [
                            { key: `custom-${current}`, value: current, text: current },
                            ...upstreamOptions,
                          ];
                        })()}
                        value={r.to}
                        placeholder={t(
                          'channel.edit.model_mapping_ph_upstream'
                        )}
                        onChange={(_, { value: v }) => updateRow(i, 'to', v || '')}
                      />
                    ) : (
                      <Form.Input
                        fluid
                        placeholder={t(
                          'channel.edit.model_mapping_ph_upstream'
                        )}
                        value={r.to}
                        onChange={(e, { value: v }) => updateRow(i, 'to', v)}
                      />
                    )}
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      type='button'
                      icon
                      basic
                      size='small'
                      disabled={rows.length <= 1}
                      onClick={() => removeRow(i)}
                      title={t('channel.edit.model_mapping_remove')}
                    >
                      <Icon name='trash' />
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
          <div className='channel-model-mapping-editor__toolbar'>
            <Button type='button' size='small' basic onClick={addRow}>
              <Icon name='plus' />
              {t('channel.edit.model_mapping_add')}
            </Button>
            <Button
              type='button'
              size='small'
              basic
              onClick={() => {
                setJsonRaw(value?.trim() ? value : MODEL_MAP_EXAMPLE_LINES);
                setJsonMode(true);
              }}
            >
              {t('channel.edit.model_mapping_json_toggle')}
            </Button>
          </div>
        </>
      ) : (
        <div className='channel-model-mapping-editor__json'>
          <Form.TextArea
            rows={12}
            value={jsonRaw}
            onChange={(e, { value: v }) => setJsonRaw(v)}
            placeholder={t('channel.edit.model_mapping_json_placeholder')}
          />
          <div className='channel-model-mapping-editor__toolbar'>
            <Button type='button' size='small' onClick={applyJsonPaste}>
              {t('channel.edit.model_mapping_json_apply')}
            </Button>
            <Button
              type='button'
              size='small'
              basic
              onClick={() => {
                setJsonMode(false);
                setJsonRaw('');
              }}
            >
              {t('channel.edit.model_mapping_json_cancel')}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
