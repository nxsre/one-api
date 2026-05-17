import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Checkbox,
  Form,
  Icon,
  Message,
  Modal,
  Pagination,
  Table,
} from 'semantic-ui-react';
import {
  API,
  isAdmin,
  showError,
  showSuccess,
} from '../../helpers';
import './ModelCatalog.css';

const SYNC_OPENAI = 'openai_compatible';
const SYNC_MODELS_DEV = 'models_dev';
const SYNC_OPENROUTER = 'openrouter';
const SYNC_ANTHROPIC = 'anthropic';
const SYNC_GEMINI = 'gemini';
const SYNC_CHANNEL = 'channel';

const MODEL_CATALOG_PAGE_SIZE_OPTIONS = [10, 20, 50, 100];
const MODEL_CATALOG_DEFAULT_PAGE_SIZE = 20;

function fmtPrice(n) {
  if (n === null || n === undefined || Number.isNaN(Number(n))) return '—';
  const x = Number(n);
  if (x === 0) return '0';
  const s = x.toFixed(6).replace(/\.?0+$/, '');
  return s;
}

function fmtInt(n) {
  if (!n && n !== 0) return '—';
  return String(n);
}

export default function ModelCatalogPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortColumn, setSortColumn] = useState(null);
  const [sortDirection, setSortDirection] = useState('ascending');
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(MODEL_CATALOG_DEFAULT_PAGE_SIZE);
  const [totalMatched, setTotalMatched] = useState(0);
  const [grandTotal, setGrandTotal] = useState(0);

  const [modalOpen, setModalOpen] = useState(false);
  const [editRow, setEditRow] = useState(null);
  const [formModelId, setFormModelId] = useState('');
  const [formOwnedBy, setFormOwnedBy] = useState('');
  const [formEnabled, setFormEnabled] = useState(true);
  const [formNotes, setFormNotes] = useState('');

  const [openSync, setOpenSync] = useState(false);
  const [syncSource, setSyncSource] = useState(SYNC_MODELS_DEV);
  const [syncBaseURL, setSyncBaseURL] = useState('');
  const [syncApiKey, setSyncApiKey] = useState('');
  const [syncChannelId, setSyncChannelId] = useState('');
  const [syncBusy, setSyncBusy] = useState(false);

  useEffect(() => {
    if (!isAdmin()) {
      showError(t('model_catalog.forbidden'));
      navigate('/');
    }
  }, [navigate, t]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.set('page', String(activePage));
      params.set('page_size', String(pageSize));
      const q = String(searchQuery || '').trim();
      if (q) params.set('search', q);
      if (sortColumn) {
        params.set('sort', sortColumn);
        params.set('order', sortDirection === 'ascending' ? 'asc' : 'desc');
      }
      const res = await API.get(`/api/model_catalog?${params.toString()}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || 'load failed');
        return;
      }
      if (data && Array.isArray(data.items)) {
        setRows(data.items);
        setTotalMatched(Number(data.total) || 0);
        setGrandTotal(Number(data.grand_total) || Number(data.total) || 0);
      } else if (Array.isArray(data)) {
        setRows(data);
        setTotalMatched(data.length);
        setGrandTotal(data.length);
      } else {
        setRows([]);
        setTotalMatched(0);
        setGrandTotal(0);
      }
    } catch (e) {
      /* axios interceptor */
    } finally {
      setLoading(false);
    }
  }, [activePage, pageSize, searchQuery, sortColumn, sortDirection]);

  useEffect(() => {
    if (isAdmin()) load().then();
  }, [load]);

  const openAddModal = () => {
    setEditRow(null);
    setFormModelId('');
    setFormOwnedBy('');
    setFormEnabled(true);
    setFormNotes('');
    setModalOpen(true);
  };

  const openEditModal = (row) => {
    setEditRow(row);
    setFormModelId(row.model_id || '');
    setFormOwnedBy(row.owned_by || '');
    setFormEnabled(!!row.enabled);
    setFormNotes(row.notes || '');
    setModalOpen(true);
  };

  const saveEdit = async () => {
    const mid = String(formModelId || '').trim();
    if (!mid) {
      showError(t('model_catalog.col_model_id'));
      return;
    }
    try {
      if (editRow) {
        const res = await API.put('/api/model_catalog', {
          id: editRow.id,
          model_id: mid,
          owned_by: String(formOwnedBy || '').trim(),
          enabled: !!formEnabled,
          source: editRow.source || 'manual',
          notes: String(formNotes || '').trim(),
        });
        if (!res.data?.success) {
          showError(res.data?.message || 'fail');
          return;
        }
      } else {
        const res = await API.post('/api/model_catalog', {
          model_id: mid,
          owned_by: String(formOwnedBy || '').trim(),
          enabled: !!formEnabled,
          notes: String(formNotes || '').trim(),
        });
        if (!res.data?.success) {
          showError(res.data?.message || 'fail');
          return;
        }
      }
      showSuccess(t('model_catalog.saved'));
      setModalOpen(false);
      await load();
    } catch {
      /* noop */
    }
  };

  const removeRow = async (row) => {
    if (!window.confirm(t('model_catalog.confirm_delete'))) return;
    try {
      const res = await API.delete(`/api/model_catalog/${row.id}`);
      if (!res.data?.success) {
        showError(res.data?.message || 'fail');
        return;
      }
      await load();
    } catch {
      /* noop */
    }
  };

  const runSync = async () => {
    setSyncBusy(true);
    try {
      const body = { source: syncSource };
      if (syncSource === SYNC_MODELS_DEV) {
        const u = String(syncBaseURL || '').trim();
        if (u) body.base_url = u;
      } else if (syncSource === SYNC_OPENAI || syncSource === SYNC_OPENROUTER) {
        body.base_url = String(syncBaseURL || '').trim();
        body.api_key = String(syncApiKey || '').trim();
      } else if (
        syncSource === SYNC_ANTHROPIC ||
        syncSource === SYNC_GEMINI
      ) {
        body.api_key = String(syncApiKey || '').trim();
      } else if (syncSource === SYNC_CHANNEL) {
        const n = parseInt(String(syncChannelId || '').trim(), 10);
        body.channel_id = Number.isFinite(n) ? n : 0;
      }
      const res = await API.post('/api/model_catalog/sync', body);
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || 'sync failed');
        return;
      }
      const msg = t('model_catalog.sync_done', {
        added: data?.added ?? 0,
        updated: data?.updated ?? 0,
        total: data?.total_upstream ?? 0,
      });
      showSuccess(msg);
      setOpenSync(false);
      await load();
    } catch {
      /* noop */
    } finally {
      setSyncBusy(false);
    }
  };

  const syncOptions = [
    { value: SYNC_MODELS_DEV, text: t('model_catalog.sync_src_models_dev') },
    { value: SYNC_OPENAI, text: t('model_catalog.sync_src_openai') },
    { value: SYNC_OPENROUTER, text: t('model_catalog.sync_src_openrouter') },
    { value: SYNC_ANTHROPIC, text: t('model_catalog.sync_src_anthropic') },
    { value: SYNC_GEMINI, text: t('model_catalog.sync_src_gemini') },
    { value: SYNC_CHANNEL, text: t('model_catalog.sync_src_channel') },
  ];

  const totalPages = Math.max(1, Math.ceil(totalMatched / pageSize));

  const pageRangeStart =
    totalMatched === 0 ? 0 : (activePage - 1) * pageSize + 1;
  const pageRangeEnd = Math.min(activePage * pageSize, totalMatched);

  useEffect(() => {
    setActivePage(1);
  }, [searchQuery]);

  useEffect(() => {
    setActivePage((p) => Math.min(Math.max(1, p), totalPages));
  }, [totalPages]);

  const toggleSort = (key) => {
    setActivePage(1);
    if (sortColumn !== key) {
      setSortColumn(key);
      setSortDirection('ascending');
    } else {
      setSortDirection((d) => (d === 'ascending' ? 'descending' : 'ascending'));
    }
  };

  const sortProps = (key) => ({
    sorted: sortColumn === key ? sortDirection : undefined,
    onClick: () => toggleSort(key),
    style: { cursor: 'pointer', userSelect: 'none' },
  });

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>{t('model_catalog.title')}</Card.Header>
          <Message info>{t('model_catalog.hint')}</Message>
          <Button primary onClick={openAddModal} style={{ marginBottom: 12 }}>
            <Icon name='add' /> {t('model_catalog.add')}
          </Button>
          <Button
            color='teal'
            onClick={() => setOpenSync(true)}
            style={{ marginBottom: 12, marginLeft: 8 }}
          >
            <Icon name='cloud download' /> {t('model_catalog.sync')}
          </Button>
          <Button
            basic
            icon
            labelPosition='left'
            onClick={() => load()}
            loading={loading}
            disabled={loading}
            style={{ marginBottom: 12, marginLeft: 8 }}
          >
            <Icon name='refresh' /> {t('model_catalog.refresh')}
          </Button>

          <Form className='model-catalog-toolbar-form'>
            <Form.Input
              className='model-catalog-search-input'
              icon='search'
              placeholder={t('model_catalog.search_placeholder')}
              value={searchQuery}
              onChange={(e, { value }) => setSearchQuery(value)}
            />
            <span className='model-catalog-filter-count'>
              {t('model_catalog.visible_rows', {
                matched: totalMatched,
                grand_total: grandTotal,
              })}
            </span>
          </Form>

          <div className='model-catalog-table-wrap'>
            <Table celled striped compact size='small' sortable>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell {...sortProps('id')}>
                    {t('model_catalog.col_pk')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('model_id')}>
                    {t('model_catalog.col_model_id')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('model_name')}>
                    {t('model_catalog.col_model_name')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('provider_key')}>
                    {t('model_catalog.col_provider_key')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('provider_display')}>
                    {t('model_catalog.col_provider_name')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('family')}>
                    {t('model_catalog.col_family')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('modalities_in')}>
                    {t('model_catalog.col_modalities_in')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('modalities_out')}>
                    {t('model_catalog.col_modalities_out')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('context_limit')}>
                    {t('model_catalog.col_context_limit')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('output_limit')}>
                    {t('model_catalog.col_output_limit')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('cost_input')}>
                    {t('model_catalog.col_cost_input')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('cost_output')}>
                    {t('model_catalog.col_cost_output')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('cost_cache_read')}>
                    {t('model_catalog.col_cost_cache_read')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('cost_cache_write')}>
                    {t('model_catalog.col_cost_cache_write')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('reasoning')}>
                    {t('model_catalog.col_reasoning')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('tool_call')}>
                    {t('model_catalog.col_tool_call')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('temperature_ok')}>
                    {t('model_catalog.col_temperature')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('attachment_ok')}>
                    {t('model_catalog.col_attachment')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('open_weights')}>
                    {t('model_catalog.col_open_weights')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('knowledge_cutoff')}>
                    {t('model_catalog.col_knowledge')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('release_date')}>
                    {t('model_catalog.col_release_date')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('last_updated')}>
                    {t('model_catalog.col_last_updated')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('npm_package')}>
                    {t('model_catalog.col_npm')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('api_base')}>
                    {t('model_catalog.col_api_base')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('doc_url')}>
                    {t('model_catalog.col_doc')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('owned_by')}>
                    {t('model_catalog.col_owned_by')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('enabled')}>
                    {t('model_catalog.col_enabled')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('source')}>
                    {t('model_catalog.col_source')}
                  </Table.HeaderCell>
                  <Table.HeaderCell {...sortProps('notes')}>
                    {t('model_catalog.col_notes')}
                  </Table.HeaderCell>
                  <Table.HeaderCell>{t('model_catalog.col_actions')}</Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <Table.Row key={r.id}>
                    <Table.Cell>{r.id}</Table.Cell>
                    <Table.Cell>{r.model_id}</Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.model_name}>
                      {r.model_name || '—'}
                    </Table.Cell>
                    <Table.Cell>{r.provider_key || '—'}</Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.provider_display}>
                      {r.provider_display || '—'}
                    </Table.Cell>
                    <Table.Cell>{r.family || '—'}</Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.modalities_in}>
                      {r.modalities_in || '—'}
                    </Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.modalities_out}>
                      {r.modalities_out || '—'}
                    </Table.Cell>
                    <Table.Cell>{fmtInt(r.context_limit)}</Table.Cell>
                    <Table.Cell>{fmtInt(r.output_limit)}</Table.Cell>
                    <Table.Cell>{fmtPrice(r.cost_input)}</Table.Cell>
                    <Table.Cell>{fmtPrice(r.cost_output)}</Table.Cell>
                    <Table.Cell>{fmtPrice(r.cost_cache_read)}</Table.Cell>
                    <Table.Cell>{fmtPrice(r.cost_cache_write)}</Table.Cell>
                    <Table.Cell>
                      {r.reasoning ? t('model_catalog.yes') : t('model_catalog.no')}
                    </Table.Cell>
                    <Table.Cell>
                      {r.tool_call ? t('model_catalog.yes') : t('model_catalog.no')}
                    </Table.Cell>
                    <Table.Cell>
                      {r.temperature_ok ? t('model_catalog.yes') : t('model_catalog.no')}
                    </Table.Cell>
                    <Table.Cell>
                      {r.attachment_ok ? t('model_catalog.yes') : t('model_catalog.no')}
                    </Table.Cell>
                    <Table.Cell>
                      {r.open_weights ? t('model_catalog.yes') : t('model_catalog.no')}
                    </Table.Cell>
                    <Table.Cell>{r.knowledge_cutoff || '—'}</Table.Cell>
                    <Table.Cell>{r.release_date || '—'}</Table.Cell>
                    <Table.Cell>{r.last_updated || '—'}</Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.npm_package}>
                      {r.npm_package || '—'}
                    </Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.api_base}>
                      {r.api_base || '—'}
                    </Table.Cell>
                    <Table.Cell>
                      {r.doc_url ? (
                        <a href={r.doc_url} target='_blank' rel='noreferrer'>
                          {t('model_catalog.link_doc')}
                        </a>
                      ) : (
                        '—'
                      )}
                    </Table.Cell>
                    <Table.Cell className='collapsible-text' title={r.owned_by}>
                      {r.owned_by}
                    </Table.Cell>
                    <Table.Cell>
                      {r.enabled ? t('model_catalog.yes') : t('model_catalog.no')}
                    </Table.Cell>
                    <Table.Cell>{r.source}</Table.Cell>
                    <Table.Cell
                      className='collapsible-text'
                      title={r.notes}
                    >
                      {r.notes}
                    </Table.Cell>
                    <Table.Cell>
                      <Button size='small' onClick={() => openEditModal(r)}>
                        {t('model_catalog.edit')}
                      </Button>
                      <Button size='small' negative onClick={() => removeRow(r)}>
                        {t('model_catalog.delete')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table>
          </div>

          <div className='model-catalog-pagination'>
            <div className='model-catalog-pagination__size'>
              <span className='model-catalog-pagination__label'>
                {t('model_catalog.page_size_label')}
              </span>
              <Form.Dropdown
                selection
                compact
                disabled={loading}
                options={MODEL_CATALOG_PAGE_SIZE_OPTIONS.map((n) => ({
                  text: String(n),
                  value: n,
                }))}
                value={pageSize}
                onChange={(e, { value }) => {
                  const n = Number(value);
                  setPageSize(Number.isFinite(n) ? n : MODEL_CATALOG_DEFAULT_PAGE_SIZE);
                  setActivePage(1);
                }}
              />
            </div>
            {totalMatched > 0 && totalPages > 1 ? (
              <Pagination
                className='model-catalog-pagination__pages'
                activePage={activePage}
                totalPages={totalPages}
                firstItem={null}
                lastItem={null}
                siblingRange={1}
                boundaryRange={1}
                onPageChange={(e, { activePage: next }) =>
                  setActivePage(Math.min(Math.max(1, next), totalPages))
                }
              />
            ) : null}
            <span className='model-catalog-pagination__range'>
              {t('model_catalog.page_range', {
                start: pageRangeStart,
                end: pageRangeEnd,
                total: totalMatched,
              })}
            </span>
          </div>
        </Card.Content>
      </Card>

      <Modal open={modalOpen} onClose={() => setModalOpen(false)} size='small'>
        <Modal.Header>
          {editRow
            ? t('model_catalog.modal_edit_title')
            : t('model_catalog.modal_add_title')}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Input
              label={t('model_catalog.col_model_id')}
              value={formModelId}
              onChange={(e, { value }) => setFormModelId(value)}
              required
            />
            <Form.Input
              label={t('model_catalog.col_owned_by')}
              value={formOwnedBy}
              onChange={(e, { value }) => setFormOwnedBy(value)}
            />
            <Form.Field>
              <Checkbox
                label={t('model_catalog.col_enabled')}
                checked={formEnabled}
                onChange={(e, { checked }) => setFormEnabled(!!checked)}
              />
            </Form.Field>
            <Form.TextArea
              label={t('model_catalog.col_notes')}
              value={formNotes}
              onChange={(e, { value }) => setFormNotes(value)}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setModalOpen(false)}>
            {t('model_catalog.btn_cancel')}
          </Button>
          <Button primary onClick={saveEdit}>
            {t('model_catalog.btn_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={openSync} onClose={() => !syncBusy && setOpenSync(false)} size='small'>
        <Modal.Header>{t('model_catalog.modal_sync_title')}</Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Dropdown
              selection
              label={t('model_catalog.sync_source')}
              options={syncOptions}
              value={syncSource}
              onChange={(e, { value }) => setSyncSource(value)}
            />
            {syncSource === SYNC_MODELS_DEV && (
              <>
                <Message info size='small'>
                  {t('model_catalog.sync_models_dev_url_hint')}
                </Message>
                <Form.Input
                  label={t('model_catalog.sync_models_dev_url_label')}
                  value={syncBaseURL}
                  onChange={(e, { value }) => setSyncBaseURL(value)}
                  placeholder='https://models.dev'
                />
              </>
            )}
            {(syncSource === SYNC_OPENAI ||
              syncSource === SYNC_OPENROUTER) && (
              <>
                <Form.Input
                  label={t('model_catalog.base_url')}
                  value={syncBaseURL}
                  onChange={(e, { value }) => setSyncBaseURL(value)}
                  placeholder='https://api.openai.com/v1'
                />
                <Form.Input
                  label={t('model_catalog.api_key')}
                  value={syncApiKey}
                  onChange={(e, { value }) => setSyncApiKey(value)}
                  type='password'
                />
              </>
            )}
            {(syncSource === SYNC_ANTHROPIC ||
              syncSource === SYNC_GEMINI) && (
              <Form.Input
                label={t('model_catalog.api_key')}
                value={syncApiKey}
                onChange={(e, { value }) => setSyncApiKey(value)}
                type='password'
              />
            )}
            {syncSource === SYNC_CHANNEL && (
              <Form.Input
                label={t('model_catalog.channel_id')}
                value={syncChannelId}
                onChange={(e, { value }) => setSyncChannelId(value)}
              />
            )}
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button disabled={syncBusy} onClick={() => setOpenSync(false)}>
            {t('model_catalog.btn_cancel')}
          </Button>
          <Button primary loading={syncBusy} onClick={runSync}>
            {t('model_catalog.sync_run')}
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
}
