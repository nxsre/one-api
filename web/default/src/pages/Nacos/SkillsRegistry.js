import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Confirm,
  Dropdown,
  Form,
  Icon,
  Input,
  Loader,
  Message,
  Modal,
  Table,
} from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  getStoredNacosNamespace,
  setStoredNacosNamespace,
  isRoot,
  showError,
  showSuccess,
} from '../../helpers';
import NacosNamespaceSelect from '../../components/NacosNamespaceSelect';
import SettingMonacoField from '../../components/SettingMonacoField';

const fmtBytes = (n) => {
  if (n == null || !Number.isFinite(Number(n))) return '';
  const v = Number(n);
  if (v < 1024) return `${v} B`;
  if (v < 1024 * 1024) return `${(v / 1024).toFixed(1)} KiB`;
  return `${(v / (1024 * 1024)).toFixed(1)} MiB`;
};

const NacosSkillsRegistry = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [info, setInfo] = useState(null);
  const [rows, setRows] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [pageNo, setPageNo] = useState(1);
  const [pageSize, setPageSize] = useState(12);
  const [namespace, setNamespace] = useState(() => getStoredNacosNamespace());

  const [searchInput, setSearchInput] = useState('');
  const [bizTagInput, setBizTagInput] = useState('');
  const [ownerInput, setOwnerInput] = useState('');
  const [filterScope, setFilterScope] = useState('');
  const [orderBy, setOrderBy] = useState('');
  const [filterOnlyMine, setFilterOnlyMine] = useState(false);
  const [query, setQuery] = useState({
    name: '',
    biz: '',
    scope: '',
    order: '',
    owner: '',
    onlyMine: false,
  });
  const [selected, setSelected] = useState({});
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState(null);

  const [editOpen, setEditOpen] = useState(false);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editBiz, setEditBiz] = useState('');
  const [editScope, setEditScope] = useState('PUBLIC');
  const [editEnable, setEditEnable] = useState(true);
  const [editBaseline, setEditBaseline] = useState({
    desc: '',
    biz: '',
    scope: 'PUBLIC',
  });

  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadFile, setUploadFile] = useState(null);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteName, setDeleteName] = useState('');

  const [submitOpen, setSubmitOpen] = useState(false);
  const [submitName, setSubmitName] = useState('');
  const [submitVersion, setSubmitVersion] = useState('');
  const [submitVersionBaseline, setSubmitVersionBaseline] = useState('');

  const [publishOpen, setPublishOpen] = useState(false);
  const [publishName, setPublishName] = useState('');
  const [publishVersion, setPublishVersion] = useState('');
  const [publishVersionBaseline, setPublishVersionBaseline] = useState('');
  const [publishUpdateLatest, setPublishUpdateLatest] = useState(true);
  const [publishForce, setPublishForce] = useState(false);
  const [publishCandidates, setPublishCandidates] = useState([]);

  const [labelsOpen, setLabelsOpen] = useState(false);
  const [labelsName, setLabelsName] = useState('');
  const [labelsText, setLabelsText] = useState('{}');
  const [labelsReplace, setLabelsReplace] = useState(false);
  const [labelsBaseline, setLabelsBaseline] = useState('{}');

  /** 详情内版本上线/下线：{ name, version, op: 'on' | 'off' } */
  const [versionOpBusy, setVersionOpBusy] = useState(null);

  const openPublish = async (name, reviewingFromList) => {
    setPublishName(name);
    setPublishUpdateLatest(true);
    setPublishForce(false);
    let candidates = [];
    if (reviewingFromList) {
      candidates = [reviewingFromList];
    }
    try {
      const res = await API.get('/api/nacos/skills/detail', {
        params: { namespace, name },
      });
      if (res.data?.success) {
        const rev = (res.data.data?.versions || [])
          .filter((v) => v.status === 'reviewing')
          .map((v) => v.version);
        if (rev.length) {
          candidates = rev;
        }
      }
    } catch (_) {
      /* ignore */
    }
    setPublishCandidates(candidates);
    const initial = candidates[0] || '';
    setPublishVersion(initial);
    setPublishVersionBaseline(initial);
    setPublishOpen(true);
  };

  const localUser = () => {
    try {
      const raw = localStorage.getItem('user');
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  };

  const applyFilters = () => {
    const u = localUser();
    setQuery({
      name: searchInput.trim(),
      biz: bizTagInput.trim(),
      scope: filterScope,
      order: orderBy,
      owner: isRoot() ? ownerInput.trim() : '',
      onlyMine: !isRoot() && filterOnlyMine && !!u?.username,
    });
    setPageNo(1);
  };

  const resetFilters = () => {
    setSearchInput('');
    setBizTagInput('');
    setOwnerInput('');
    setFilterScope('');
    setOrderBy('');
    setFilterOnlyMine(false);
    setQuery({
      name: '',
      biz: '',
      scope: '',
      order: '',
      owner: '',
      onlyMine: false,
    });
    setPageNo(1);
  };

  const setScopeFilter = (v) => {
    const scope = !v || v === '_all' ? '' : v;
    setFilterScope(scope);
    setQuery((q) => ({ ...q, scope }));
    setPageNo(1);
  };

  const setSortOrder = (v) => {
    const order = !v || v === '_' ? '' : v;
    setOrderBy(order);
    setQuery((q) => ({ ...q, order }));
    setPageNo(1);
  };

  const toggleFilterOnlyMine = () => {
    const u = localUser();
    const next = !filterOnlyMine;
    setFilterOnlyMine(next);
    setQuery((q) => ({
      ...q,
      onlyMine: next && !isRoot() && !!u?.username,
    }));
    setPageNo(1);
  };

  const selectedNames = () =>
    Object.keys(selected).filter((k) => selected[k]);

  const toggleSelect = (name) => {
    setSelected((s) => ({ ...s, [name]: !s[name] }));
  };

  const toggleSelectAllPage = () => {
    const names = rows.map((r) => r.name);
    const allOn =
      names.length > 0 && names.every((n) => selected[n]);
    const next = { ...selected };
    if (allOn) {
      names.forEach((n) => {
        delete next[n];
      });
    } else {
      names.forEach((n) => {
        next[n] = true;
      });
    }
    setSelected(next);
  };

  const clearSelection = () => setSelected({});

  const load = async () => {
    setLoading(true);
    try {
      const ir = await API.get('/api/nacos/registry/info');
      if (!ir.data?.success) {
        showError(ir.data?.message || 'load info failed');
        return;
      }
      setInfo(ir.data.data);
      const u = localUser();
      const params = {
        namespace,
        page: pageNo,
        size: pageSize,
      };
      if (query.name) {
        params.skillName = query.name;
        params.search = 'blur';
      }
      if (query.biz) {
        params.bizTag = query.biz;
      }
      if (query.scope) {
        params.scope = query.scope;
      }
      if (query.order) {
        params.orderBy = query.order;
      }
      if (isRoot() && query.owner) {
        params.owner = query.owner;
      } else if (query.onlyMine && u?.username) {
        params.owner = u.username;
      }
      const sr = await API.get('/api/nacos/skills', { params });
      if (!sr.data?.success) {
        showError(sr.data?.message || 'load skills failed');
        return;
      }
      setRows(sr.data.data?.pageItems || []);
      setTotalCount(sr.data.data?.totalCount ?? 0);
      setSelected((prev) => {
        const next = { ...prev };
        const nameSet = new Set((sr.data.data?.pageItems || []).map((r) => r.name));
        Object.keys(next).forEach((k) => {
          if (!nameSet.has(k)) delete next[k];
        });
        return next;
      });
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setStoredNacosNamespace(namespace);
    setPageNo(1);
  }, [namespace]);

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [namespace, pageNo, pageSize, query]);

  const openDetail = async (name) => {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      const res = await API.get('/api/nacos/skills/detail', {
        params: { namespace, name },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load detail failed');
        setDetailOpen(false);
        return;
      }
      setDetail(res.data.data);
    } catch (e) {
      showError(e.message);
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const openEdit = (r) => {
    setEditName(r.name);
    const desc = r.description || '';
    const biz = r.bizTags || '';
    const scope = r.scope || 'PUBLIC';
    setEditDesc(desc);
    setEditBiz(biz);
    setEditScope(scope);
    setEditEnable(!!r.enable);
    setEditBaseline({ desc, biz, scope });
    setEditOpen(true);
  };

  const postSkillVersionOnline = async (skillName, version) => {
    setVersionOpBusy({ name: skillName, version, op: 'on' });
    try {
      const res = await API.post('/api/nacos/skills/version/online', {
        namespace,
        name: skillName,
        version,
        updateLatestLabel: true,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'online failed');
        return;
      }
      showSuccess(t('nacos.skills_version_online_ok'));
      load();
      if (detailOpen && detail?.name === skillName) {
        openDetail(skillName);
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setVersionOpBusy(null);
    }
  };

  const postSkillVersionOffline = async (skillName, version) => {
    setVersionOpBusy({ name: skillName, version, op: 'off' });
    try {
      const res = await API.post('/api/nacos/skills/version/offline', {
        namespace,
        name: skillName,
        version,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'offline failed');
        return;
      }
      showSuccess(t('nacos.skills_version_offline_ok'));
      load();
      if (detailOpen && detail?.name === skillName) {
        openDetail(skillName);
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setVersionOpBusy(null);
    }
  };

  const isVersionOpBusy = (skillName, version, op) =>
    versionOpBusy?.name === skillName &&
    versionOpBusy?.version === version &&
    versionOpBusy?.op === op;

  const saveMetadata = async () => {
    try {
      const res = await API.put('/api/nacos/skills/metadata', {
        namespace,
        name: editName,
        description: editDesc,
        bizTags: editBiz,
        scope: editScope,
        enable: editEnable,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'save failed');
        return;
      }
      showSuccess(t('nacos.skills_saved'));
      setEditOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doUpload = async () => {
    if (!uploadFile) {
      showError(t('nacos.skills_pick_file'));
      return;
    }
    const fd = new FormData();
    fd.append('file', uploadFile);
    try {
      const res = await API.post(
        `/api/nacos/skills/upload?namespace=${encodeURIComponent(namespace)}`,
        fd
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'upload failed');
        return;
      }
      showSuccess(t('nacos.skills_upload_ok'));
      setUploadOpen(false);
      setUploadFile(null);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doDelete = async () => {
    try {
      const res = await API.delete('/api/nacos/skills/item', {
        params: { namespace, name: deleteName },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      showSuccess(t('nacos.skills_deleted'));
      setDeleteOpen(false);
      setDeleteName('');
      clearSelection();
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doBatchDelete = async () => {
    const names = selectedNames();
    if (names.length === 0) return;
    try {
      for (const name of names) {
        const res = await API.delete('/api/nacos/skills/item', {
          params: { namespace, name },
        });
        if (!res.data?.success) {
          showError(res.data?.message || `${name}`);
          return;
        }
      }
      showSuccess(t('nacos.skills_batch_deleted'));
      clearSelection();
      setBatchDeleteOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doSubmit = async () => {
    try {
      const res = await API.post('/api/nacos/skills/submit', {
        namespace,
        name: submitName,
        version: submitVersion || undefined,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'submit failed');
        return;
      }
      showSuccess(t('nacos.skills_submit_ok'));
      setSubmitOpen(false);
      setSubmitVersion('');
      setSubmitVersionBaseline('');
      load();
      if (detailOpen && detail?.name === submitName) {
        openDetail(submitName);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const doPublish = async () => {
    if (!publishVersion.trim()) {
      showError(t('nacos.skills_version_required'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/skills/publish', {
        namespace,
        name: publishName,
        version: publishVersion.trim(),
        updateLatest: publishUpdateLatest,
        forcePublish: publishForce,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'publish failed');
        return;
      }
      showSuccess(t('nacos.skills_publish_ok'));
      setPublishOpen(false);
      setPublishVersion('');
      load();
      if (detailOpen && detail?.name === publishName) {
        openDetail(publishName);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const downloadVersionZip = async (name, ver) => {
    try {
      const res = await API.get('/api/nacos/skills/download', {
        params: { namespace, name, version: ver },
        responseType: 'blob',
      });
      const blob = new Blob([res.data], { type: 'application/zip' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${name}-${ver}.zip`;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (e) {
      showError(e.message);
    }
  };

  const saveLabels = async () => {
    let labels;
    try {
      labels = JSON.parse(labelsText || '{}');
    } catch {
      showError(t('nacos.skills_labels_invalid'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/skills/labels', {
        namespace,
        name: labelsName,
        labels,
        replace: labelsReplace,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'save labels failed');
        return;
      }
      showSuccess(t('nacos.skills_labels_saved'));
      setLabelsOpen(false);
      load();
      if (detailOpen && detail?.name === labelsName) {
        openDetail(labelsName);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize) || 1);

  const skillVersionStatusLabel = (status) => {
    switch (status) {
      case 'draft':
        return t('nacos.skills_ver_draft');
      case 'reviewing':
        return t('nacos.skills_ver_reviewing');
      case 'reviewed':
        return t('nacos.skills_ver_reviewed');
      case 'online':
        return t('nacos.skills_ver_online');
      case 'offline':
        return t('nacos.skills_ver_offline');
      default:
        return status || '—';
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card nacos-registry-table-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('nacos.skills_title')}
          </Card.Header>
          {info && (
            <Message info>
              {t('nacos.registry_storage')}: {info.zip_storage}
              {info.zip_local_dir ? ` | 本地: ${info.zip_local_dir}` : ''}
              {info.s3_remote_configured ? ' | S3 已配置' : ''}
              {info.max_upload_bytes
                ? ` | ${t('nacos.skills_max_upload')}: ${fmtBytes(
                    info.max_upload_bytes
                  )}`
                : ''}
            </Message>
          )}
          <div style={{ marginBottom: 8, color: '#666', fontSize: 13 }}>
            {t('nacos.skills_total', { total: totalCount })}
          </div>
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              columnGap: 12,
              rowGap: 8,
              marginBottom: 12,
            }}
          >
            <span style={{ flexShrink: 0, color: '#666' }}>namespace</span>
            <div
              style={{ flex: '1 1 220px', minWidth: 180, maxWidth: 420 }}
            >
              <NacosNamespaceSelect
                value={namespace}
                onChange={(v) => setNamespace(v || 'public')}
              />
            </div>
            <Button size='small' style={{ margin: 0 }} onClick={load}>
              {t('nacos.refresh')}
            </Button>
            <Button
              size='small'
              primary
              style={{ margin: 0 }}
              onClick={() => {
                setUploadFile(null);
                setUploadOpen(true);
              }}
            >
              {t('nacos.skills_upload')}
            </Button>
          </div>
          <div
            className='nacos-skills-filter-row'
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'flex-end',
              columnGap: 22,
              rowGap: 14,
              marginBottom: 16,
              paddingTop: 2,
            }}
          >
            <Form.Field style={{ margin: 0, minWidth: 168, flex: '0 1 auto' }}>
              <label
                style={{
                  fontSize: 12,
                  color: '#666',
                  display: 'block',
                  marginBottom: 8,
                  fontWeight: 500,
                }}
              >
                {t('nacos.skills_search_name')}
              </label>
              <Input
                size='small'
                placeholder={t('nacos.skills_search_placeholder')}
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
              />
            </Form.Field>
            <Form.Field style={{ margin: 0, minWidth: 168, flex: '0 1 auto' }}>
              <label
                style={{
                  fontSize: 12,
                  color: '#666',
                  display: 'block',
                  marginBottom: 8,
                  fontWeight: 500,
                }}
              >
                {t('nacos.skills_filter_biztag')}
              </label>
              <Input
                size='small'
                placeholder={t('nacos.skills_filter_biztag_ph')}
                value={bizTagInput}
                onChange={(e) => setBizTagInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
              />
            </Form.Field>
            {isRoot() ? (
              <Form.Field style={{ margin: 0, minWidth: 168, flex: '0 1 auto' }}>
                <label
                  style={{
                    fontSize: 12,
                    color: '#666',
                    display: 'block',
                    marginBottom: 8,
                    fontWeight: 500,
                  }}
                >
                  {t('nacos.skills_filter_owner')}
                </label>
                <Input
                  size='small'
                  placeholder={t('nacos.skills_filter_owner_ph')}
                  value={ownerInput}
                  onChange={(e) => setOwnerInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
                />
              </Form.Field>
            ) : (
              <Button
                size='small'
                toggle
                active={filterOnlyMine}
                type='button'
                style={{ alignSelf: 'flex-end', marginBottom: 2 }}
                onClick={toggleFilterOnlyMine}
              >
                {t('nacos.skills_filter_only_mine')}
              </Button>
            )}
            <Form.Field style={{ margin: 0, minWidth: 140, flex: '0 1 auto' }}>
              <label
                style={{
                  fontSize: 12,
                  color: '#666',
                  display: 'block',
                  marginBottom: 8,
                  fontWeight: 500,
                }}
              >
                {t('nacos.skills_filter_scope')}
              </label>
              <select
                style={{
                  minHeight: 32,
                  padding: '6px 10px',
                  width: '100%',
                  boxSizing: 'border-box',
                }}
                value={filterScope || '_all'}
                onChange={(e) => setScopeFilter(e.target.value)}
              >
                <option value='_all'>
                  {t('nacos.skills_filter_scope_all')}
                </option>
                <option value='PUBLIC'>PUBLIC</option>
                <option value='PRIVATE'>PRIVATE</option>
              </select>
            </Form.Field>
            <Form.Field style={{ margin: 0, minWidth: 168, flex: '0 1 auto' }}>
              <label
                style={{
                  fontSize: 12,
                  color: '#666',
                  display: 'block',
                  marginBottom: 8,
                  fontWeight: 500,
                }}
              >
                {t('nacos.skills_sort')}
              </label>
              <select
                style={{
                  minHeight: 32,
                  padding: '6px 10px',
                  width: '100%',
                  boxSizing: 'border-box',
                }}
                value={orderBy || '_'}
                onChange={(e) => setSortOrder(e.target.value)}
              >
                <option value='_'>{t('nacos.skills_sort_default')}</option>
                <option value='download_count'>
                  {t('nacos.skills_sort_downloads')}
                </option>
              </select>
            </Form.Field>
            <div
              style={{
                display: 'flex',
                gap: 10,
                alignItems: 'center',
                flexWrap: 'wrap',
                marginLeft: 4,
                alignSelf: 'flex-end',
                marginBottom: 2,
              }}
            >
              <Button size='small' primary type='button' onClick={applyFilters}>
                {t('nacos.skills_search_btn')}
              </Button>
              <Button size='small' type='button' onClick={resetFilters}>
                {t('nacos.skills_reset_filters')}
              </Button>
            </div>
            {selectedNames().length > 0 ? (
              <div
                style={{
                  marginLeft: 'auto',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  flexWrap: 'wrap',
                }}
              >
                <span style={{ fontSize: 12, color: '#666' }}>
                  {t('nacos.skills_selected', { n: selectedNames().length })}
                </span>
                <Button
                  negative
                  size='small'
                  type='button'
                  onClick={() => setBatchDeleteOpen(true)}
                >
                  {t('nacos.skills_batch_delete')}
                </Button>
                <Button size='small' type='button' onClick={clearSelection}>
                  {t('nacos.skills_clear_selection')}
                </Button>
              </div>
            ) : null}
          </div>
          {loading ? (
            <Loader active />
          ) : (
            <>
              <Table
                celled
                compact
                stackable
                className='nacos-skills-table'
                style={{ overflow: 'visible' }}
              >
                <Table.Header>
                  <Table.Row>
                    <Table.HeaderCell collapsing>
                      <Checkbox
                        checked={
                          rows.length > 0 &&
                          rows.every((r) => selected[r.name])
                        }
                        indeterminate={
                          selectedNames().length > 0 &&
                          !rows.every((r) => selected[r.name])
                        }
                        onChange={toggleSelectAllPage}
                      />
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_name')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_desc')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_owner')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_enable')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_scope')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_draft_review')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_online_cnt')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_downloads')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_actions')}
                    </Table.HeaderCell>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {rows.map((r) => (
                    <Table.Row key={r.name}>
                      <Table.Cell collapsing>
                        <Checkbox
                          checked={!!selected[r.name]}
                          onChange={() => toggleSelect(r.name)}
                        />
                      </Table.Cell>
                      <Table.Cell>
                        <span style={{ wordBreak: 'break-word' }}>{r.name}</span>
                      </Table.Cell>
                      <Table.Cell
                        title={r.description || ''}
                        style={{
                          maxWidth: 280,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {r.description || '—'}
                      </Table.Cell>
                      <Table.Cell>{r.owner || '—'}</Table.Cell>
                      <Table.Cell collapsing textAlign='center'>
                        {r.enable ? (
                          <Icon name='check circle' color='green' />
                        ) : (
                          <span style={{ color: '#999' }}>—</span>
                        )}
                      </Table.Cell>
                      <Table.Cell collapsing>{r.scope || 'PUBLIC'}</Table.Cell>
                      <Table.Cell>
                        <div style={{ fontSize: 12, lineHeight: 1.45 }}>
                          <div>
                            <span style={{ color: '#888' }}>
                              {t('nacos.skills_col_draft_short')}
                            </span>{' '}
                            {r.editingVersion || '—'}
                          </div>
                          <div>
                            <span style={{ color: '#888' }}>
                              {t('nacos.skills_col_reviewing_short')}
                            </span>{' '}
                            {r.reviewingVersion || '—'}
                          </div>
                        </div>
                      </Table.Cell>
                      <Table.Cell>
                        {r.onlineCnt != null ? r.onlineCnt : '—'}
                      </Table.Cell>
                      <Table.Cell>
                        {r.downloadCount != null ? r.downloadCount : '—'}
                      </Table.Cell>
                      <Table.Cell
                        collapsing
                        style={{ overflow: 'visible', verticalAlign: 'middle' }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            flexWrap: 'wrap',
                            gap: 6,
                            alignItems: 'center',
                            maxWidth: 340,
                            position: 'relative',
                            overflow: 'visible',
                            zIndex: 1,
                          }}
                        >
                          <Button.Group size='mini'>
                            <Button
                              type='button'
                              onClick={() => openDetail(r.name)}
                            >
                              {t('nacos.skills_action_detail')}
                            </Button>
                            <Button
                              type='button'
                              onClick={() => openEdit(r)}
                            >
                              {t('nacos.skills_action_edit')}
                            </Button>
                          </Button.Group>
                          <Dropdown
                            button
                            floating
                            upward
                            direction='left'
                            className='mini nacos-skills-actions-dropdown'
                            icon='dropdown'
                            text={t('nacos.skills_more_actions')}
                          >
                            <Dropdown.Menu>
                              <Dropdown.Item
                                text={t('nacos.skills_action_submit')}
                                onClick={() => {
                                  setSubmitName(r.name);
                                  setSubmitVersion('');
                                  setSubmitVersionBaseline('');
                                  setSubmitOpen(true);
                                }}
                              />
                              <Dropdown.Item
                                text={t('nacos.skills_action_publish')}
                                onClick={() =>
                                  openPublish(r.name, r.reviewingVersion)
                                }
                              />
                              <Dropdown.Item
                                text={t('nacos.skills_action_labels')}
                                onClick={() => {
                                  const lt = JSON.stringify(
                                    r.labels || {},
                                    null,
                                    2
                                  );
                                  setLabelsName(r.name);
                                  setLabelsText(lt);
                                  setLabelsBaseline(lt);
                                  setLabelsReplace(false);
                                  setLabelsOpen(true);
                                }}
                              />
                              <Dropdown.Divider />
                              <Dropdown.Item
                                text={t('nacos.skills_action_delete')}
                                onClick={() => {
                                  setDeleteName(r.name);
                                  setDeleteOpen(true);
                                }}
                                style={{ color: '#db2828' }}
                              />
                            </Dropdown.Menu>
                          </Dropdown>
                        </div>
                      </Table.Cell>
                    </Table.Row>
                  ))}
                </Table.Body>
              </Table>
              {totalCount > pageSize ? (
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'flex-end',
                    gap: 8,
                    marginTop: 12,
                    flexWrap: 'wrap',
                  }}
                >
                  <span style={{ fontSize: 13, color: '#666' }}>
                    {t('nacos.skills_page_summary', {
                      pageNo,
                      totalPages,
                      total: totalCount,
                    })}
                  </span>
                  <select
                    value={pageSize}
                    style={{ minHeight: 32, padding: '0 8px' }}
                    onChange={(e) => {
                      setPageSize(Number(e.target.value));
                      setPageNo(1);
                    }}
                  >
                    <option value={12}>12 / {t('nacos.skills_page')}</option>
                    <option value={24}>24 / {t('nacos.skills_page')}</option>
                    <option value={48}>48 / {t('nacos.skills_page')}</option>
                  </select>
                  <Button
                    size='small'
                    disabled={pageNo <= 1}
                    type='button'
                    onClick={() => setPageNo((p) => Math.max(1, p - 1))}
                  >
                    {t('nacos.skills_prev_page')}
                  </Button>
                  <Button
                    size='small'
                    disabled={pageNo >= totalPages}
                    type='button'
                    onClick={() =>
                      setPageNo((p) => Math.min(totalPages, p + 1))
                    }
                  >
                    {t('nacos.skills_next_page')}
                  </Button>
                </div>
              ) : null}
            </>
          )}
        </Card.Content>
      </Card>

      <Modal open={detailOpen} onClose={() => setDetailOpen(false)} size='large'>
        <Modal.Header>{t('nacos.skills_detail_title')}</Modal.Header>
        <Modal.Content scrolling>
          {detailLoading ? (
            <Loader active />
          ) : detail ? (
            <>
              <p>
                <strong>{detail.name}</strong> — {detail.description}
              </p>
              <p>
                {t('nacos.skills_detail_owner')}: {detail.owner || '—'} |{' '}
                {t('nacos.skills_col_enable')}:{' '}
                {detail.enable ? t('nacos.skills_yes') : t('nacos.skills_no')}{' '}
                | {t('nacos.skills_col_scope')}: {detail.scope || 'PUBLIC'} |
                bizTags: {detail.bizTags || '—'}
              </p>
              {detail.labels && Object.keys(detail.labels).length > 0 ? (
                <p>
                  labels: {JSON.stringify(detail.labels)}
                </p>
              ) : null}
              <Table celled compact size='small'>
                <Table.Header>
                  <Table.Row>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_version')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_status')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_commit')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.skills_col_actions')}
                    </Table.HeaderCell>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {(detail.versions || []).map((v) => (
                    <Table.Row key={v.version}>
                      <Table.Cell>{v.version}</Table.Cell>
                      <Table.Cell>
                        {skillVersionStatusLabel(v.status)}
                      </Table.Cell>
                      <Table.Cell>{v.commitMsg || '—'}</Table.Cell>
                      <Table.Cell>
                        <div
                          style={{
                            display: 'flex',
                            flexWrap: 'wrap',
                            gap: 6,
                            alignItems: 'center',
                          }}
                        >
                          {v.status === 'online' ? (
                            <Button
                              size='mini'
                              type='button'
                              onClick={() =>
                                downloadVersionZip(detail.name, v.version)
                              }
                            >
                              {t('nacos.skills_download')}
                            </Button>
                          ) : (
                            <span style={{ color: '#999', fontSize: 12 }}>
                              {t('nacos.skills_download_online_only')}
                            </span>
                          )}
                          {v.status === 'online' ? (
                            <Button
                              size='mini'
                              type='button'
                              loading={isVersionOpBusy(
                                detail.name,
                                v.version,
                                'off'
                              )}
                              disabled={
                                !!versionOpBusy &&
                                !isVersionOpBusy(
                                  detail.name,
                                  v.version,
                                  'off'
                                )
                              }
                              onClick={() =>
                                postSkillVersionOffline(detail.name, v.version)
                              }
                            >
                              {t('nacos.skills_action_offline')}
                            </Button>
                          ) : null}
                          {v.status === 'offline' ? (
                            <Button
                              positive
                              size='mini'
                              type='button'
                              loading={isVersionOpBusy(
                                detail.name,
                                v.version,
                                'on'
                              )}
                              disabled={
                                !!versionOpBusy &&
                                !isVersionOpBusy(
                                  detail.name,
                                  v.version,
                                  'on'
                                )
                              }
                              onClick={() =>
                                postSkillVersionOnline(detail.name, v.version)
                              }
                            >
                              {t('nacos.skills_action_online')}
                            </Button>
                          ) : null}
                        </div>
                      </Table.Cell>
                    </Table.Row>
                  ))}
                </Table.Body>
              </Table>
            </>
          ) : null}
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDetailOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={editOpen} onClose={() => setEditOpen(false)}>
        <Modal.Header>
          {t('nacos.skills_edit_title')}: {editName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.skills_col_desc')}
              value={editDesc}
              originValue={editBaseline.desc}
              onChange={(v) => setEditDesc(v)}
              height={160}
            />
            <SettingMonacoField
              label='bizTags'
              value={editBiz}
              originValue={editBaseline.biz}
              onChange={(v) => setEditBiz(v)}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.skills_col_scope')}
              hint='PUBLIC / PRIVATE'
              value={editScope}
              originValue={editBaseline.scope}
              onChange={(v) => setEditScope(v)}
              height={88}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.skills_col_enable')}
                checked={editEnable}
                onChange={(e, { checked }) => setEditEnable(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setEditOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary onClick={saveMetadata}>
            {t('nacos.skills_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={uploadOpen} onClose={() => setUploadOpen(false)}>
        <Modal.Header>{t('nacos.skills_upload_title')}</Modal.Header>
        <Modal.Content>
          <p>{t('nacos.skills_upload_hint')}</p>
          <input
            type='file'
            accept='.zip,application/zip'
            onChange={(e) =>
              setUploadFile(e.target.files && e.target.files[0])
            }
          />
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setUploadOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary onClick={doUpload}>
            {t('nacos.skills_upload')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={submitOpen} onClose={() => setSubmitOpen(false)}>
        <Modal.Header>
          {t('nacos.skills_submit_title')}: {submitName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.skills_version_optional')}
              hint={t('nacos.skills_submit_hint')}
              value={submitVersion}
              originValue={submitVersionBaseline}
              onChange={(v) => setSubmitVersion(v)}
              height={88}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setSubmitOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary onClick={doSubmit}>
            {t('nacos.skills_action_submit')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={publishOpen} onClose={() => setPublishOpen(false)}>
        <Modal.Header>
          {t('nacos.skills_publish_title')}: {publishName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.skills_publish_version')}
              hint={`${t('nacos.skills_publish_hint')} · ${
                publishCandidates.length
                  ? publishCandidates.join(', ')
                  : '—'
              }`}
              value={publishVersion}
              originValue={publishVersionBaseline}
              onChange={(v) => setPublishVersion(v)}
              height={96}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.skills_update_latest')}
                checked={publishUpdateLatest}
                onChange={(e, { checked }) =>
                  setPublishUpdateLatest(!!checked)
                }
              />
            </Form.Field>
            <Form.Field>
              <Checkbox
                label={t('nacos.skills_force_publish')}
                checked={publishForce}
                onChange={(e, { checked }) => setPublishForce(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setPublishOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
          <Button positive onClick={doPublish}>
            {t('nacos.skills_action_publish')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={labelsOpen} onClose={() => setLabelsOpen(false)} size='large'>
        <Modal.Header>
          {t('nacos.skills_labels_title')}: {labelsName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label='labels (JSON object)'
              language='json'
              enableJsonFormat
              minimap
              value={labelsText}
              originValue={labelsBaseline}
              onChange={(v) => setLabelsText(v)}
              height={360}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.skills_labels_replace')}
                checked={labelsReplace}
                onChange={(e, { checked }) => setLabelsReplace(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setLabelsOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary onClick={saveLabels}>
            {t('nacos.skills_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Confirm
        open={batchDeleteOpen}
        header={t('nacos.skills_batch_delete')}
        content={t('nacos.skills_batch_delete_confirm', {
          n: selectedNames().length,
        })}
        onCancel={() => setBatchDeleteOpen(false)}
        onConfirm={doBatchDelete}
      />

      <Confirm
        open={deleteOpen}
        header={t('nacos.skills_delete_confirm')}
        content={`${deleteName} @ ${namespace}`}
        onCancel={() => {
          setDeleteOpen(false);
          setDeleteName('');
        }}
        onConfirm={doDelete}
      />
    </div>
  );
};

export default NacosSkillsRegistry;
