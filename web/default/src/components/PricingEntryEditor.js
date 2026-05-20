import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Form,
  Icon,
  Input,
  Pagination,
  Table,
} from 'semantic-ui-react';
import { API, showError, showSuccess } from '../helpers';
import './PricingEntryEditor.css';

const PAGE_SIZE_OPTIONS = [20, 50, 100];

export default function PricingEntryEditor({
  blockId,
  valueKind = 'number',
  keyColumnLabel,
  subKeyColumnLabel,
  keyOptions = [],
  keyDropdown = false,
  onMutate,
}) {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalPages, setTotalPages] = useState(1);
  const [search, setSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [savingId, setSavingId] = useState(null);
  const [drafts, setDrafts] = useState({});
  const [newRow, setNewRow] = useState({
    entry_key: '',
    sub_key: '',
    value_text: '',
  });

  const isGroupGroup = valueKind === 'group_group';
  const isStringValue = valueKind === 'string';

  const load = useCallback(async () => {
    if (!blockId) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({
        block_id: blockId,
        page: String(page),
        page_size: String(pageSize),
      });
      if (search.trim()) params.set('q', search.trim());
      const res = await API.get(`/api/pricing_entries/?${params.toString()}`);
      const body = res.data || {};
      if (!body.success) {
        showError(body.message || 'load failed');
        return;
      }
      const data = body.data || {};
      setItems(Array.isArray(data.items) ? data.items : []);
      setTotal(Number(data.total) || 0);
      setTotalPages(Number(data.total_pages) || 1);
      setDrafts({});
    } catch (e) {
      showError(e.message || 'load failed');
    } finally {
      setLoading(false);
    }
  }, [blockId, page, pageSize, search]);

  const notifyMutate = useCallback(async () => {
    if (typeof onMutate === 'function') {
      await onMutate();
    }
  }, [onMutate]);

  useEffect(() => {
    setPage(1);
  }, [blockId, pageSize, search]);

  useEffect(() => {
    load().then();
  }, [load]);

  const draftValue = (row) =>
    drafts[row.id] !== undefined ? drafts[row.id] : row.value_text;

  const saveRow = async (row) => {
    const valueText = String(draftValue(row) ?? '').trim();
    if (!valueText) {
      showError(t('setting.operation.ratio.editor.value_required', '值不能为空'));
      return;
    }
    setSavingId(row.id);
    try {
      const res = await API.put(`/api/pricing_entries/${row.id}`, {
        value_text: valueText,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'save failed');
        return;
      }
      showSuccess(t('setting.operation.ratio.editor.row_saved', '已保存'));
      await load();
      await notifyMutate();
    } catch (e) {
      showError(e.message || 'save failed');
    } finally {
      setSavingId(null);
    }
  };

  const deleteRow = async (row) => {
    if (!window.confirm(t('setting.operation.ratio.editor.confirm_delete', '确定删除此条记录？'))) {
      return;
    }
    try {
      const res = await API.delete(`/api/pricing_entries/${row.id}`);
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      await load();
      await notifyMutate();
    } catch (e) {
      showError(e.message || 'delete failed');
    }
  };

  const createRow = async () => {
    const entryKey = String(newRow.entry_key ?? '').trim();
    const subKey = String(newRow.sub_key ?? '').trim();
    const valueText = String(newRow.value_text ?? '').trim();
    if (!entryKey || !valueText || (isGroupGroup && !subKey)) {
      showError(t('setting.operation.ratio.editor.create_invalid', '请填写完整'));
      return;
    }
    try {
      const res = await API.post('/api/pricing_entries/', {
        block_id: blockId,
        entry_key: entryKey,
        sub_key: subKey,
        value_text: valueText,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'create failed');
        return;
      }
      setNewRow({ entry_key: '', sub_key: '', value_text: '' });
      setPage(1);
      await load();
      await notifyMutate();
    } catch (e) {
      showError(e.message || 'create failed');
    }
  };

  const keyLabel =
    keyColumnLabel || t('setting.operation.ratio.editor.col_key');
  const subLabel =
    subKeyColumnLabel || t('setting.operation.ratio.group_group.col_use_group', '使用分组');

  return (
    <div className='pricing-entry-editor'>
      <div className='pricing-entry-editor__toolbar'>
        <Input
          icon='search'
          placeholder={t('setting.operation.ratio.editor.filter_placeholder')}
          value={searchInput}
          onChange={(e, { value }) => setSearchInput(value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') setSearch(searchInput);
          }}
          action={{
            content: t('setting.operation.ratio.editor.search', '搜索'),
            onClick: () => setSearch(searchInput),
          }}
        />
        <span className='pricing-entry-editor__count'>
          {t('setting.operation.ratio.editor.entry_count', {
            count: total,
            total,
            defaultValue: `共 ${total} 条`,
          })}
        </span>
      </div>

      <div className='pricing-entry-editor__add'>
        <Form.Group widths={isGroupGroup ? 4 : 3}>
          <Form.Input
            label={keyLabel}
            list={keyDropdown ? `pe-keys-${blockId}` : undefined}
            value={newRow.entry_key}
            onChange={(e, { value }) =>
              setNewRow((prev) => ({ ...prev, entry_key: value }))
            }
          />
          {keyDropdown ? (
            <datalist id={`pe-keys-${blockId}`}>
              {keyOptions.map((o) => (
                <option key={o.value} value={o.value} />
              ))}
            </datalist>
          ) : null}
          {isGroupGroup ? (
            <Form.Input
              label={subLabel}
              list={keyDropdown ? `pe-subkeys-${blockId}` : undefined}
              value={newRow.sub_key}
              onChange={(e, { value }) =>
                setNewRow((prev) => ({ ...prev, sub_key: value }))
              }
            />
          ) : null}
          {keyDropdown && isGroupGroup ? (
            <datalist id={`pe-subkeys-${blockId}`}>
              {keyOptions.map((o) => (
                <option key={`s-${o.value}`} value={o.value} />
              ))}
            </datalist>
          ) : null}
          <Form.Input
            label={t('setting.operation.ratio.editor.col_value')}
            type={isStringValue ? 'text' : 'number'}
            step={isStringValue ? undefined : 'any'}
            value={newRow.value_text}
            onChange={(e, { value }) =>
              setNewRow((prev) => ({ ...prev, value_text: value }))
            }
          />
          <Form.Field className='pricing-entry-editor__add-btn'>
            <label>&nbsp;</label>
            <Button primary type='button' onClick={createRow}>
              {t('setting.operation.ratio.editor.add_row')}
            </Button>
          </Form.Field>
        </Form.Group>
      </div>

      <div className='pricing-entry-editor__table-wrap'>
        <Table compact celled size='small' loading={loading}>
          <Table.Header>
            <Table.Row>
              <Table.HeaderCell>{keyLabel}</Table.HeaderCell>
              {isGroupGroup ? (
                <Table.HeaderCell>{subLabel}</Table.HeaderCell>
              ) : null}
              <Table.HeaderCell>
                {t('setting.operation.ratio.editor.col_value')}
              </Table.HeaderCell>
              <Table.HeaderCell collapsing />
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {items.length === 0 ? (
              <Table.Row>
                <Table.Cell
                  colSpan={isGroupGroup ? 4 : 3}
                  textAlign='center'
                >
                  {t('setting.operation.ratio.editor.empty', '暂无记录')}
                </Table.Cell>
              </Table.Row>
            ) : (
              items.map((row) => (
                <Table.Row key={row.id}>
                  <Table.Cell className='pricing-entry-editor__key-cell'>
                    {row.entry_key}
                  </Table.Cell>
                  {isGroupGroup ? (
                    <Table.Cell>{row.sub_key}</Table.Cell>
                  ) : null}
                  <Table.Cell>
                    <Input
                      fluid
                      type={isStringValue ? 'text' : 'number'}
                      step={isStringValue ? undefined : 'any'}
                      value={draftValue(row)}
                      onChange={(e, { value }) =>
                        setDrafts((prev) => ({ ...prev, [row.id]: value }))
                      }
                    />
                  </Table.Cell>
                  <Table.Cell collapsing>
                    <Button
                      icon
                      basic
                      size='mini'
                      loading={savingId === row.id}
                      disabled={savingId === row.id}
                      onClick={() => saveRow(row)}
                      title={t('setting.operation.ratio.buttons.save', '保存')}
                    >
                      <Icon name='save' />
                    </Button>
                    <Button
                      icon
                      basic
                      size='mini'
                      color='red'
                      onClick={() => deleteRow(row)}
                      title={t('setting.operation.ratio.editor.remove_row', '删除')}
                    >
                      <Icon name='trash' />
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))
            )}
          </Table.Body>
        </Table>
      </div>

      <div className='pricing-entry-editor__footer'>
        <Form.Select
          compact
          options={PAGE_SIZE_OPTIONS.map((n) => ({
            key: n,
            value: n,
            text: `${n} / ${t('setting.operation.ratio.editor.page', '页')}`,
          }))}
          value={pageSize}
          onChange={(_, { value }) => setPageSize(Number(value))}
        />
        {totalPages > 1 ? (
          <Pagination
            size='small'
            activePage={page}
            totalPages={totalPages}
            onPageChange={(_, data) => setPage(Number(data.activePage))}
          />
        ) : null}
      </div>
    </div>
  );
}
