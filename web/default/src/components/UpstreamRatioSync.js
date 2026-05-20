import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Checkbox,
  Dropdown,
  Form,
  Header,
  Icon,
  Label,
  Loader,
  Message,
  Modal,
  Segment,
  Table,
} from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess, timestamp2string } from '../helpers';
import './UpstreamRatioSync.css';

const FIELD_LABELS = {
  model_ratio: '模型倍率',
  completion_ratio: '补全倍率',
  model_price: '模型价格',
  cache_ratio: '缓存倍率',
  create_cache_ratio: '缓存创建倍率',
  image_ratio: '图片倍率',
  audio_ratio: '音频倍率',
  audio_completion_ratio: '音频补全倍率',
  billing_mode: '计费模式',
  billing_expr: '分层计费表达式',
};

const PRICING_APPLIED_EVENT = 'one-api-pricing-applied';
const PAGE_SIZE = 50;

function rowKey(model, field) {
  return `${model}::${field}`;
}

function pickDefaultUpstream(row, preferred, upstreamNames) {
  const names = upstreamNames || Object.keys(row.upstreams || {}).filter(
    (n) => row.upstreams[n] !== 'same' && row.upstreams[n] != null
  );
  if (preferred && names.includes(preferred)) return preferred;
  const trusted = names.find((n) => row.confidence?.[n] !== false);
  return trusted || names[0] || '';
}

function formatValue(v) {
  if (v == null) return '—';
  if (typeof v === 'object') return JSON.stringify(v);
  if (typeof v === 'number' && Number.isFinite(v)) {
    if (Number.isInteger(v)) return String(v);
    return String(parseFloat(v.toFixed(6)));
  }
  if (typeof v === 'string') {
    const trimmed = v.trim();
    if (trimmed !== '' && !Number.isNaN(Number(trimmed))) {
      const n = Number(trimmed);
      if (Number.isInteger(n)) return String(n);
      return String(parseFloat(n.toFixed(6)));
    }
  }
  return String(v);
}

function upstreamNamesFromRow(row) {
  return Object.keys(row.upstreams || {}).filter(
    (n) => row.upstreams[n] !== 'same' && row.upstreams[n] != null
  );
}

function PagerBar({ page, totalPages, total, onPageChange, t, loading = false }) {
  if (!totalPages || totalPages <= 1) return null;

  const pageItems = [];
  const addPage = (p) => {
    if (p >= 1 && p <= totalPages && !pageItems.includes(p)) pageItems.push(p);
  };
  addPage(1);
  for (let p = page - 2; p <= page + 2; p += 1) addPage(p);
  addPage(totalPages);
  pageItems.sort((a, b) => a - b);

  const buttons = [];
  let prev = 0;
  pageItems.forEach((p) => {
    if (p - prev > 1) buttons.push({ type: 'ellipsis', key: `ellipsis-${p}` });
    buttons.push({ type: 'page', value: p, key: `page-${p}` });
    prev = p;
  });

  const go = (nextPage) => {
    if (loading || nextPage < 1 || nextPage > totalPages || nextPage === page) return;
    onPageChange(nextPage);
  };

  return (
    <div className='upstream-ratio-sync__pager-bar'>
      <span className='upstream-ratio-sync__pager-info'>
        {t('setting.operation.ratio.upstream_sync.page_info', { page, totalPages, total })}
      </span>
      <Button
        type='button'
        size='mini'
        icon='angle left'
        aria-label={t('setting.operation.ratio.upstream_sync.page_prev')}
        disabled={loading || page <= 1}
        onClick={() => go(page - 1)}
      />
      {buttons.map((item) =>
        item.type === 'ellipsis' ? (
          <span key={item.key} className='upstream-ratio-sync__pager-ellipsis'>
            …
          </span>
        ) : (
          <Button
            key={item.key}
            type='button'
            size='mini'
            primary={item.value === page}
            basic={item.value !== page}
            disabled={loading}
            onClick={() => go(item.value)}
          >
            {item.value}
          </Button>
        )
      )}
      <Button
        type='button'
        size='mini'
        icon='angle right'
        aria-label={t('setting.operation.ratio.upstream_sync.page_next')}
        disabled={loading || page >= totalPages}
        onClick={() => go(page + 1)}
      />
    </div>
  );
}

const UpstreamRatioSync = () => {
  const { t } = useTranslation();
  const [channels, setChannels] = useState([]);
  const [selected, setSelected] = useState([]);
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);
  const [savingSelections, setSavingSelections] = useState(false);

  const [batches, setBatches] = useState([]);
  const [activeBatchId, setActiveBatchId] = useState(null);
  const [activeBatch, setActiveBatch] = useState(null);
  const [reviewOpen, setReviewOpen] = useState(false);

  const [diffRows, setDiffRows] = useState([]);
  const [diffTotal, setDiffTotal] = useState(0);
  const [diffTotalPages, setDiffTotalPages] = useState(1);
  const [diffPage, setDiffPage] = useState(1);
  const [diffLoading, setDiffLoading] = useState(false);
  const diffTableRef = useRef(null);
  const [modelSearch, setModelSearch] = useState('');
  const [modelQuery, setModelQuery] = useState('');

  const [selectionDraft, setSelectionDraft] = useState({});
  const [defaultUpstream, setDefaultUpstream] = useState('');
  const [applySource, setApplySource] = useState('');
  const [activateOnApply, setActivateOnApply] = useState(true);

  const [compareOpen, setCompareOpen] = useState(false);
  const [compareLeft, setCompareLeft] = useState('current');
  const [compareRight, setCompareRight] = useState('');
  const [compareRows, setCompareRows] = useState([]);
  const [compareTotal, setCompareTotal] = useState(0);
  const [compareTotalPages, setCompareTotalPages] = useState(1);
  const [comparePage, setComparePage] = useState(1);
  const compareTableRef = useRef(null);
  const [compareModelQuery, setCompareModelQuery] = useState('');
  const [compareLoading, setCompareLoading] = useState(false);

  const [versionBlocks, setVersionBlocks] = useState([]);
  const [versionLoading, setVersionLoading] = useState(false);
  const [versionModalOpen, setVersionModalOpen] = useState(false);
  const [versionModalBlock, setVersionModalBlock] = useState(null);
  const [blockVersions, setBlockVersions] = useState([]);
  const [blockActiveVersion, setBlockActiveVersion] = useState(0);
  const [activatingVersion, setActivatingVersion] = useState(false);

  const loadVersionBlocks = useCallback(async () => {
    setVersionLoading(true);
    try {
      const res = await API.get('/api/ratio_sync/versions');
      if (res.data?.success) setVersionBlocks(res.data.data || []);
    } catch {
      /* ignore */
    }
    setVersionLoading(false);
  }, []);

  const loadBatches = useCallback(async () => {
    try {
      const res = await API.get('/api/ratio_sync/batches?limit=30');
      if (res.data?.success) {
        setBatches(res.data.data || []);
      }
    } catch {
      /* ignore */
    }
  }, []);

  const loadBatchDetail = useCallback(async (batchId) => {
    if (!batchId) return null;
    const res = await API.get(`/api/ratio_sync/batches/${batchId}`);
    if (res.data?.success) {
      setActiveBatch(res.data.data);
      return res.data.data;
    }
    return null;
  }, []);

  const loadDiffPage = useCallback(
    async (batchId, page, model) => {
      if (!batchId) return;
      setDiffLoading(true);
      try {
        const res = await API.get(`/api/ratio_sync/batches/${batchId}/diffs`, {
          params: { page, page_size: PAGE_SIZE, model: model || undefined },
        });
        if (res.data?.success) {
          const data = res.data.data || {};
          setDiffRows(data.items || []);
          const total = data.total || 0;
          setDiffTotal(total);
          setDiffTotalPages(
            Math.max(1, data.total_pages || Math.ceil(total / PAGE_SIZE) || 1)
          );
          setDiffPage(data.page || page);
          const draft = {};
          (data.items || []).forEach((row) => {
            const key = rowKey(row.model, row.field);
            draft[key] = {
              model: row.model,
              field: row.field,
              upstream_name:
                row.upstream_name ||
                pickDefaultUpstream(row, defaultUpstream, upstreamNamesFromRow(row)),
              selected: row.selected ?? false,
            };
          });
          setSelectionDraft((prev) => ({ ...prev, ...draft }));
          if (diffTableRef.current) {
            diffTableRef.current.scrollTop = 0;
          }
        } else {
          showError(res.data?.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
        }
      } catch (e) {
        showError(e.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
      } finally {
        setDiffLoading(false);
      }
    },
    [defaultUpstream, t]
  );

  useEffect(() => {
    API.get('/api/ratio_sync/channels')
      .then((res) => {
        if (res.data?.success) setChannels(res.data.data || []);
      })
      .catch(() => {});
    loadVersionBlocks();
    loadBatches();
    API.get('/api/ratio_sync/batches/latest')
      .then((res) => {
        if (res.data?.success && res.data.data?.id) {
          setActiveBatchId(res.data.data.id);
          setActiveBatch(res.data.data);
        }
      })
      .catch(() => {});
  }, [loadVersionBlocks, loadBatches]);

  const options = channels.map((ch) => ({
    key: ch.id,
    value: ch.id,
    text: ch.name + (ch.base_url ? ` (${ch.base_url})` : ''),
  }));

  const batchOptions = useMemo(
    () =>
      batches.map((b) => ({
        key: b.id,
        value: b.id,
        text: `#${b.id} · ${timestamp2string(b.created_time)} · ${b.diff_count || 0} 项 · ${b.status}`,
      })),
    [batches]
  );

  const compareSideOptions = useMemo(() => {
    const opts = [{ key: 'current', value: 'current', text: '当前生效价目' }];
    batches
      .filter((b) => b.status === 'completed')
      .forEach((b) => {
        opts.push({
          key: `batch-${b.id}`,
          value: `batch:${b.id}`,
          text: `批次 #${b.id}`,
        });
      });
    versionBlocks.forEach((block) => {
      opts.push({
        key: `block-${block.block_id}`,
        value: `version:${block.block_id}:${block.active_version || 0}`,
        text: `${block.title} v${block.active_version || '—'}`,
        disabled: !block.active_version,
      });
    });
    return opts;
  }, [batches, versionBlocks]);

  const pollBatchUntilDone = async (batchId) => {
    for (let i = 0; i < 120; i++) {
      const detail = await loadBatchDetail(batchId);
      if (!detail) break;
      if (detail.status === 'completed' || detail.status === 'failed') {
        return detail;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    return null;
  };

  const fetchUpstream = async () => {
    if (selected.length === 0) {
      showError(t('setting.operation.ratio.upstream_sync.select_upstream'));
      return;
    }
    setLoading(true);
    try {
      const res = await API.post('/api/ratio_sync/fetch', {
        channel_ids: selected,
        timeout: 15,
      });
      const batchId = res.data?.data?.batch_id;
      if (!batchId) {
        showError(res.data?.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
        setLoading(false);
        return;
      }
      setActiveBatchId(batchId);
      const detail = await pollBatchUntilDone(batchId);
      await loadBatches();
      if (!detail) {
        showError(t('setting.operation.ratio.upstream_sync.fetch_timeout'));
        setLoading(false);
        return;
      }
      if (detail.status === 'failed') {
        showError(detail.error_message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
        setLoading(false);
        return;
      }
      const pref =
        (detail.test_results || []).find((r) => r.status === 'success')?.name || '';
      setDefaultUpstream(pref);
      setApplySource(pref);
      setSelectionDraft({});
      setModelSearch('');
      setModelQuery('');
      setDiffPage(1);
      await loadDiffPage(batchId, 1, '');
      setReviewOpen(true);
      const hasError = (detail.test_results || []).some((r) => r.status === 'error');
      showSuccess(
        hasError
          ? t('setting.operation.ratio.upstream_sync.fetch_partial')
          : t('setting.operation.ratio.upstream_sync.fetch_ok')
      );
    } catch (e) {
      showError(e.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
    }
    setLoading(false);
  };

  const openBatchReview = async (batchId) => {
    if (!batchId) return;
    setActiveBatchId(batchId);
    const batch = await loadBatchDetail(batchId);
    const count = batch?.diff_count || 0;
    if (count > 0) {
      setDiffTotal(count);
      setDiffTotalPages(Math.max(1, Math.ceil(count / PAGE_SIZE)));
    }
    setDiffPage(1);
    setModelQuery('');
    setModelSearch('');
    await loadDiffPage(batchId, 1, '');
    setReviewOpen(true);
  };

  const applyDefaultUpstreamToPage = (name) => {
    setDefaultUpstream(name);
    setSelectionDraft((prev) => {
      const next = { ...prev };
      diffRows.forEach((row) => {
        const key = rowKey(row.model, row.field);
        next[key] = {
          model: row.model,
          field: row.field,
          upstream_name: pickDefaultUpstream(row, name, upstreamNamesFromRow(row)),
          selected: next[key]?.selected ?? false,
        };
      });
      return next;
    });
  };

  const toggleRowSelection = (row, checked) => {
    const key = rowKey(row.model, row.field);
    setSelectionDraft((prev) => ({
      ...prev,
      [key]: {
        model: row.model,
        field: row.field,
        upstream_name:
          prev[key]?.upstream_name ||
          pickDefaultUpstream(row, defaultUpstream, upstreamNamesFromRow(row)),
        selected: checked,
      },
    }));
  };

  const selectAllBatch = async () => {
    if (!activeBatchId) return;
    setSavingSelections(true);
    try {
      const res = await API.post(`/api/ratio_sync/batches/${activeBatchId}/selections/select_all`, {
        upstream_name: defaultUpstream,
        selected: true,
      });
      if (res.data?.success) {
        showSuccess(t('setting.operation.ratio.upstream_sync.select_all_ok'));
        await loadBatchDetail(activeBatchId);
        await loadDiffPage(activeBatchId, diffPage, modelQuery);
      } else {
        showError(res.data?.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setSavingSelections(false);
  };

  const saveSelections = async () => {
    if (!activeBatchId) return;
    const items = Object.values(selectionDraft).filter((x) => x.selected);
    if (items.length === 0) {
      showError(t('setting.operation.ratio.upstream_sync.apply_none'));
      return;
    }
    setSavingSelections(true);
    try {
      const res = await API.put(`/api/ratio_sync/batches/${activeBatchId}/selections`, {
        items,
      });
      if (res.data?.success) {
        showSuccess(res.data.message || t('setting.operation.ratio.upstream_sync.selections_saved'));
        await loadBatchDetail(activeBatchId);
      } else {
        showError(res.data?.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setSavingSelections(false);
  };

  const applySelected = async () => {
    if (!activeBatchId) return;
    const savedCount = activeBatch?.selection_count ?? 0;
    if (savedCount === 0) {
      showError(t('setting.operation.ratio.upstream_sync.apply_no_saved'));
      return;
    }
    if (diffTotal === 0) {
      showError(t('setting.operation.ratio.upstream_sync.no_diff'));
      return;
    }
    setApplying(true);
    try {
      const res = await API.post(`/api/ratio_sync/batches/${activeBatchId}/apply`, {
        activate: activateOnApply,
        source: applySource || defaultUpstream,
        label: t('setting.operation.ratio.upstream_sync.apply_label_default'),
      });
      if (res.data?.success) {
        showSuccess(res.data.message || t('setting.operation.ratio.upstream_sync.apply_ok'));
        window.dispatchEvent(new CustomEvent(PRICING_APPLIED_EVENT));
        await loadVersionBlocks();
        setReviewOpen(false);
      } else {
        showError(res.data?.message || t('setting.operation.ratio.upstream_sync.apply_fail'));
      }
    } catch (e) {
      showError(e.message || t('setting.operation.ratio.upstream_sync.apply_fail'));
    }
    setApplying(false);
  };

  const loadComparePage = async (page = 1, model = compareModelQuery) => {
    if (!compareLeft || !compareRight) return;
    setCompareLoading(true);
    try {
      const res = await API.get('/api/ratio_sync/compare/diffs', {
        params: {
          left: compareLeft,
          right: compareRight,
          page,
          page_size: PAGE_SIZE,
          model: model || undefined,
        },
      });
      if (res.data?.success) {
        const data = res.data.data || {};
        setCompareRows(data.items || []);
        const total = data.total || 0;
        setCompareTotal(total);
        setCompareTotalPages(
          Math.max(1, data.total_pages || Math.ceil(total / PAGE_SIZE) || 1)
        );
        setComparePage(data.page || page);
        if (compareTableRef.current) {
          compareTableRef.current.scrollTop = 0;
        }
      } else {
        showError(res.data?.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setCompareLoading(false);
  };

  const openBlockVersions = async (block) => {
    setVersionModalBlock(block);
    setVersionModalOpen(true);
    setBlockVersions([]);
    try {
      const res = await API.get(`/api/ratio_sync/versions/${block.block_id}`);
      if (res.data?.success) {
        setBlockVersions(res.data.data?.versions || []);
        setBlockActiveVersion(res.data.data?.active_version || 0);
      } else {
        showError(res.data?.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const activateVersion = async (versionId) => {
    if (!versionModalBlock) return;
    setActivatingVersion(true);
    try {
      const res = await API.post('/api/ratio_sync/versions/activate', {
        block_id: versionModalBlock.block_id,
        version_id: versionId,
      });
      if (res.data?.success) {
        showSuccess(res.data.message || t('setting.operation.ratio.upstream_sync.version_activated'));
        setBlockActiveVersion(versionId);
        window.dispatchEvent(new CustomEvent(PRICING_APPLIED_EVENT));
        await loadVersionBlocks();
        await openBlockVersions(versionModalBlock);
      } else {
        showError(res.data?.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setActivatingVersion(false);
  };

  const selectedCount = Object.values(selectionDraft).filter((x) => x.selected).length;

  const renderDiffTable = (rows, readOnly = false) => (
    <Table compact celled striped size='small'>
      <Table.Header>
        <Table.Row>
          {!readOnly && <Table.HeaderCell collapsing />}
          <Table.HeaderCell>{t('setting.operation.ratio.editor.col_model')}</Table.HeaderCell>
          <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_field')}</Table.HeaderCell>
          <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_change')}</Table.HeaderCell>
          {!readOnly && (
            <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_pick')}</Table.HeaderCell>
          )}
          {readOnly && (
            <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_upstream')}</Table.HeaderCell>
          )}
        </Table.Row>
      </Table.Header>
      <Table.Body>
        {rows.map((row) => {
          const key = rowKey(row.model, row.field);
          const names = upstreamNamesFromRow(row);
          const draft = selectionDraft[key];
          const upstream =
            draft?.upstream_name || pickDefaultUpstream(row, defaultUpstream, names);
          const upVal = row.upstreams?.[upstream];
          const lowConf = row.confidence?.[upstream] === false;
          const targetName = names.find((n) => row.upstreams[n] !== 'same') || names[0];
          const compareVal = targetName ? row.upstreams[targetName] : null;
          return (
            <Table.Row key={key} warning={lowConf}>
              {!readOnly && (
                <Table.Cell collapsing>
                  <Checkbox
                    checked={!!draft?.selected}
                    onChange={(_, { checked }) => toggleRowSelection(row, !!checked)}
                  />
                </Table.Cell>
              )}
              <Table.Cell className='upstream-ratio-sync__model-cell' title={row.model}>
                {row.model}
              </Table.Cell>
              <Table.Cell className='upstream-ratio-sync__field-cell'>
                {FIELD_LABELS[row.field] || row.field}
              </Table.Cell>
              <Table.Cell className='upstream-ratio-sync__change-cell'>
                <span>{formatValue(row.current)}</span>
                <span className='upstream-ratio-sync__change-arrow'>→</span>
                <span className='upstream-ratio-sync__change-new'>
                  {formatValue(readOnly ? compareVal : upVal)}
                </span>
                {lowConf && (
                  <Label size='mini' color='orange' style={{ marginLeft: 4 }}>
                    {t('setting.operation.ratio.upstream_sync.low_confidence')}
                  </Label>
                )}
              </Table.Cell>
              {!readOnly && (
                <Table.Cell className='upstream-ratio-sync__pick-cell'>
                  <select
                    className='upstream-ratio-sync__native-select'
                    value={upstream}
                    onChange={(e) =>
                      setSelectionDraft((prev) => ({
                        ...prev,
                        [key]: {
                          model: row.model,
                          field: row.field,
                          upstream_name: e.target.value,
                          selected: prev[key]?.selected ?? false,
                        },
                      }))
                    }
                  >
                    {names.map((n) => (
                      <option key={n} value={n}>
                        {n}
                      </option>
                    ))}
                  </select>
                </Table.Cell>
              )}
              {readOnly && (
                <Table.Cell>{targetName || '—'}</Table.Cell>
              )}
            </Table.Row>
          );
        })}
      </Table.Body>
    </Table>
  );

  return (
    <Segment>
      <Header as='h4'>{t('setting.operation.ratio.upstream_sync.title')}</Header>
      <Message info size='small'>
        {t('setting.operation.ratio.upstream_sync.hint')}
      </Message>
      <Form onSubmit={(e) => e.preventDefault()}>
        <Form.Dropdown
          fluid
          multiple
          search
          selection
          options={options}
          value={selected}
          placeholder={t('setting.operation.ratio.upstream_sync.select_placeholder')}
          onChange={(_, { value }) => setSelected(value)}
        />
        <div className='upstream-ratio-sync__actions'>
          <Button type='button' primary loading={loading} disabled={loading} onClick={fetchUpstream}>
            {t('setting.operation.ratio.upstream_sync.fetch')}
          </Button>
          {activeBatchId && (
            <Button type='button' basic onClick={() => openBatchReview(activeBatchId)}>
              {t('setting.operation.ratio.upstream_sync.open_latest_batch', { id: activeBatchId })}
            </Button>
          )}
          <Button type='button' basic onClick={() => setCompareOpen(true)}>
            {t('setting.operation.ratio.upstream_sync.compare_open')}
          </Button>
        </div>
      </Form>

      {batches.length > 0 && (
        <>
          <Header as='h5' dividing style={{ marginTop: '1.5rem' }}>
            {t('setting.operation.ratio.upstream_sync.batch_history')}
          </Header>
          <Table compact celled size='small'>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>ID</Table.HeaderCell>
                <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_time')}</Table.HeaderCell>
                <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_status')}</Table.HeaderCell>
                <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_diff_count')}</Table.HeaderCell>
                <Table.HeaderCell />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {batches.slice(0, 10).map((b) => (
                <Table.Row key={b.id} positive={b.id === activeBatchId}>
                  <Table.Cell>#{b.id}</Table.Cell>
                  <Table.Cell>{b.created_time ? timestamp2string(b.created_time) : '—'}</Table.Cell>
                  <Table.Cell>
                    <Label
                      size='mini'
                      color={
                        b.status === 'completed'
                          ? 'green'
                          : b.status === 'failed'
                          ? 'red'
                          : 'yellow'
                      }
                      title={b.error_message || undefined}
                    >
                      {b.status}
                    </Label>
                  </Table.Cell>
                  <Table.Cell>{b.diff_count || 0}</Table.Cell>
                  <Table.Cell collapsing>
                    <Button
                      type='button'
                      size='tiny'
                      basic
                      disabled={b.status !== 'completed'}
                      onClick={() => openBatchReview(b.id)}
                    >
                      {t('setting.operation.ratio.upstream_sync.review_batch')}
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        </>
      )}

      <Header as='h5' dividing style={{ marginTop: '1.5rem' }}>
        {t('setting.operation.ratio.upstream_sync.versions_title')}
      </Header>
      {versionLoading && versionBlocks.length === 0 ? (
        <Message size='small'>{t('setting.operation.ratio.upstream_sync.versions_loading')}</Message>
      ) : (
        <Table compact celled size='small'>
          <Table.Header>
            <Table.Row>
              <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_block')}</Table.HeaderCell>
              <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_active')}</Table.HeaderCell>
              <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_count')}</Table.HeaderCell>
              <Table.HeaderCell />
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {versionBlocks.map((block) => (
              <Table.Row key={block.block_id}>
                <Table.Cell>{block.title}</Table.Cell>
                <Table.Cell>
                  {block.active_version > 0 ? (
                    <Label color='green'>v{block.active_version}</Label>
                  ) : (
                    '—'
                  )}
                </Table.Cell>
                <Table.Cell>{block.version_count || 0}</Table.Cell>
                <Table.Cell collapsing>
                  <Button type='button' size='tiny' basic onClick={() => openBlockVersions(block)}>
                    {t('setting.operation.ratio.upstream_sync.manage_versions')}
                  </Button>
                </Table.Cell>
              </Table.Row>
            ))}
          </Table.Body>
        </Table>
      )}

      <Modal open={reviewOpen} onClose={() => setReviewOpen(false)} className='upstream-ratio-sync-diff-modal'>
        <Modal.Header>{t('setting.operation.ratio.upstream_sync.diff_title')}</Modal.Header>
        <Modal.Content className='upstream-ratio-sync__modal-content'>
          {(activeBatch?.test_results || []).length > 0 && (
            <div className='upstream-ratio-sync__fetch-status'>
              {(activeBatch.test_results || []).map((r) => (
                <Label
                  key={r.name}
                  color={r.status === 'success' ? 'green' : 'red'}
                  title={r.error || undefined}
                >
                  <Icon name={r.status === 'success' ? 'check circle' : 'times circle'} />
                  {r.name}
                  {r.error ? `: ${r.error}` : ''}
                </Label>
              ))}
            </div>
          )}
          <div className='upstream-ratio-sync__modal-body'>
          <div className='upstream-ratio-sync__toolbar'>
            <div className='upstream-ratio-sync__toolbar-row upstream-ratio-sync__toolbar-row--review'>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.batch_select')}</label>
                <Dropdown
                  selection
                  fluid
                  options={batchOptions}
                  value={activeBatchId || ''}
                  onChange={(_, { value }) => openBatchReview(value)}
                />
              </div>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.model_search')}</label>
                <Form.Input
                  fluid
                  value={modelSearch}
                  placeholder={t('setting.operation.ratio.editor.filter_placeholder')}
                  onChange={(e, { value }) => setModelSearch(value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      setModelQuery(modelSearch.trim());
                      loadDiffPage(activeBatchId, 1, modelSearch.trim());
                    }
                  }}
                />
              </div>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.default_upstream')}</label>
                <Dropdown
                  selection
                  fluid
                  options={(activeBatch?.test_results || [])
                    .filter((r) => r.status === 'success')
                    .map((r) => ({ key: r.name, value: r.name, text: r.name }))}
                  value={defaultUpstream}
                  onChange={(_, { value }) => applyDefaultUpstreamToPage(value)}
                />
              </div>
            </div>
            <div className='upstream-ratio-sync__toolbar-row upstream-ratio-sync__toolbar-row--review'>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.apply_source')}</label>
                <Form.Input
                  fluid
                  value={applySource}
                  onChange={(e, { value }) => setApplySource(value)}
                  placeholder={defaultUpstream}
                />
              </div>
              <div className='upstream-ratio-sync__toolbar-check'>
                <Checkbox
                  label={t('setting.operation.ratio.upstream_sync.activate_on_apply')}
                  checked={activateOnApply}
                  onChange={(_, { checked }) => setActivateOnApply(!!checked)}
                />
              </div>
            </div>
          </div>
          <div className='upstream-ratio-sync__summary-row'>
            <div className='upstream-ratio-sync__summary'>
              {t('setting.operation.ratio.upstream_sync.diff_summary', {
                total: diffTotal,
                selected: activeBatch?.selection_count ?? selectedCount,
              })}
            </div>
            <PagerBar
              page={diffPage}
              totalPages={diffTotalPages}
              total={diffTotal}
              loading={diffLoading}
              onPageChange={(page) => loadDiffPage(activeBatchId, page, modelQuery)}
              t={t}
            />
          </div>
          <div className='upstream-ratio-sync__table-wrap' ref={diffTableRef}>
            {diffLoading && (
              <div className='upstream-ratio-sync__table-loading'>
                <Loader active inline='centered' size='small' />
              </div>
            )}
            {diffRows.length === 0 && !diffLoading ? (
              <Message info>{t('setting.operation.ratio.upstream_sync.no_diff')}</Message>
            ) : (
              renderDiffTable(diffRows)
            )}
          </div>
          <div className='upstream-ratio-sync__pager-footer'>
            <PagerBar
              page={diffPage}
              totalPages={diffTotalPages}
              total={diffTotal}
              loading={diffLoading}
              onPageChange={(page) => loadDiffPage(activeBatchId, page, modelQuery)}
              t={t}
            />
          </div>
          </div>
        </Modal.Content>
        <Modal.Actions>
          <Button type='button' onClick={() => setReviewOpen(false)}>
            {t('setting.operation.ratio.upstream_sync.close')}
          </Button>
          <Button
            type='button'
            basic
            loading={savingSelections}
            disabled={savingSelections || diffTotal === 0}
            onClick={selectAllBatch}
          >
            {t('setting.operation.ratio.upstream_sync.select_all_batch')}
          </Button>
          <Button type='button' loading={savingSelections} disabled={savingSelections} onClick={saveSelections}>
            {t('setting.operation.ratio.upstream_sync.save_selections')}
          </Button>
          <Button
            type='button'
            primary
            loading={applying}
            disabled={
              applying ||
              (activeBatch?.selection_count ?? 0) === 0 ||
              diffTotal === 0
            }
            onClick={applySelected}
          >
            {t('setting.operation.ratio.upstream_sync.apply_from_batch')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={compareOpen} onClose={() => setCompareOpen(false)} className='upstream-ratio-sync-diff-modal'>
        <Modal.Header>{t('setting.operation.ratio.upstream_sync.compare_title')}</Modal.Header>
        <Modal.Content className='upstream-ratio-sync__modal-content'>
          <div className='upstream-ratio-sync__modal-body'>
          <div className='upstream-ratio-sync__toolbar'>
            <div className='upstream-ratio-sync__toolbar-row upstream-ratio-sync__toolbar-row--compare'>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.compare_left')}</label>
                <Dropdown
                  selection
                  fluid
                  options={compareSideOptions}
                  value={compareLeft}
                  onChange={(_, { value }) => setCompareLeft(value)}
                />
              </div>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.compare_right')}</label>
                <Dropdown
                  selection
                  fluid
                  options={compareSideOptions}
                  value={compareRight}
                  onChange={(_, { value }) => setCompareRight(value)}
                />
              </div>
              <div className='upstream-ratio-sync__toolbar-field'>
                <label>{t('setting.operation.ratio.upstream_sync.model_search')}</label>
                <Form.Input
                  fluid
                  value={compareModelQuery}
                  onChange={(e, { value }) => setCompareModelQuery(value)}
                />
              </div>
              <div className='upstream-ratio-sync__toolbar-check'>
                <Button
                  type='button'
                  primary
                  loading={compareLoading}
                  disabled={compareLoading || !compareRight}
                  onClick={() => loadComparePage(1, compareModelQuery)}
                >
                  {t('setting.operation.ratio.upstream_sync.compare_run')}
                </Button>
              </div>
            </div>
          </div>
          <div className='upstream-ratio-sync__summary-row'>
            <div className='upstream-ratio-sync__summary'>
              {t('setting.operation.ratio.upstream_sync.compare_summary', { total: compareTotal })}
            </div>
            <PagerBar
              page={comparePage}
              totalPages={compareTotalPages}
              total={compareTotal}
              loading={compareLoading}
              onPageChange={(page) => loadComparePage(page, compareModelQuery)}
              t={t}
            />
          </div>
          <div className='upstream-ratio-sync__table-wrap' ref={compareTableRef}>
            {compareLoading && (
              <div className='upstream-ratio-sync__table-loading'>
                <Loader active inline='centered' size='small' />
              </div>
            )}
            {compareRows.length === 0 && !compareLoading ? (
              <Message info>{t('setting.operation.ratio.upstream_sync.compare_empty')}</Message>
            ) : (
              renderDiffTable(compareRows, true)
            )}
          </div>
          <div className='upstream-ratio-sync__pager-footer'>
            <PagerBar
              page={comparePage}
              totalPages={compareTotalPages}
              total={compareTotal}
              loading={compareLoading}
              onPageChange={(page) => loadComparePage(page, compareModelQuery)}
              t={t}
            />
          </div>
          </div>
        </Modal.Content>
        <Modal.Actions>
          <Button type='button' onClick={() => setCompareOpen(false)}>
            {t('setting.operation.ratio.upstream_sync.close')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={versionModalOpen} onClose={() => setVersionModalOpen(false)} size='small'>
        <Modal.Header>
          {versionModalBlock?.title} — {t('setting.operation.ratio.upstream_sync.version_history')}
        </Modal.Header>
        <Modal.Content scrolling>
          {blockVersions.length === 0 ? (
            <Message info size='small'>
              {t('setting.operation.ratio.upstream_sync.no_versions')}
            </Message>
          ) : (
            <Table compact celled size='small'>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>ID</Table.HeaderCell>
                  <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_label')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_source')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('setting.operation.ratio.upstream_sync.col_time')}</Table.HeaderCell>
                  <Table.HeaderCell />
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {[...blockVersions]
                  .sort((a, b) => b.id - a.id)
                  .map((v) => (
                    <Table.Row key={v.id} positive={v.id === blockActiveVersion}>
                      <Table.Cell>
                        v{v.id}
                        {v.id === blockActiveVersion && (
                          <Label size='mini' color='green' style={{ marginLeft: 4 }}>
                            {t('setting.operation.ratio.upstream_sync.active')}
                          </Label>
                        )}
                      </Table.Cell>
                      <Table.Cell>{v.label}</Table.Cell>
                      <Table.Cell>{v.source || '—'}</Table.Cell>
                      <Table.Cell>{v.created_at ? timestamp2string(v.created_at) : '—'}</Table.Cell>
                      <Table.Cell collapsing>
                        {v.id !== blockActiveVersion && (
                          <Button
                            type='button'
                            size='tiny'
                            primary
                            loading={activatingVersion}
                            disabled={activatingVersion}
                            onClick={() => activateVersion(v.id)}
                          >
                            {t('setting.operation.ratio.upstream_sync.set_active')}
                          </Button>
                        )}
                      </Table.Cell>
                    </Table.Row>
                  ))}
              </Table.Body>
            </Table>
          )}
        </Modal.Content>
        <Modal.Actions>
          <Button type='button' onClick={() => setVersionModalOpen(false)}>
            {t('setting.operation.ratio.upstream_sync.close')}
          </Button>
        </Modal.Actions>
      </Modal>
    </Segment>
  );
};

export default UpstreamRatioSync;
