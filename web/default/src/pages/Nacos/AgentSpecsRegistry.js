import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Confirm,
  Form,
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

const NacosAgentSpecsRegistry = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [info, setInfo] = useState(null);
  const [rows, setRows] = useState([]);
  const [namespace, setNamespace] = useState(() => getStoredNacosNamespace());

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
  const [uploadOverwrite, setUploadOverwrite] = useState(false);

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

  const openPublish = async (name, reviewingFromList) => {
    setPublishName(name);
    setPublishUpdateLatest(true);
    setPublishForce(false);
    let candidates = [];
    if (reviewingFromList) {
      candidates = [reviewingFromList];
    }
    try {
      const res = await API.get('/api/nacos/agentspecs/detail', {
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

  const load = async () => {
    setLoading(true);
    try {
      const ir = await API.get('/api/nacos/registry/info');
      if (!ir.data?.success) {
        showError(ir.data?.message || 'load info failed');
        return;
      }
      setInfo(ir.data.data);
      const sr = await API.get('/api/nacos/agentspecs', {
        params: { namespace, page: 1, size: 100 },
      });
      if (!sr.data?.success) {
        showError(sr.data?.message || 'load agentspecs failed');
        return;
      }
      setRows(sr.data.data?.pageItems || []);
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setStoredNacosNamespace(namespace);
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [namespace]);

  const openDetail = async (name) => {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      const res = await API.get('/api/nacos/agentspecs/detail', {
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
    const desc = r.description == null ? '' : String(r.description);
    const biz = r.bizTags == null ? '' : String(r.bizTags);
    const scope = r.scope || 'PUBLIC';
    setEditDesc(desc);
    setEditBiz(biz);
    setEditScope(scope);
    setEditEnable(!!r.enable);
    setEditBaseline({ desc, biz, scope });
    setEditOpen(true);
  };

  const saveMetadata = async () => {
    try {
      const res = await API.put('/api/nacos/agentspecs/metadata', {
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
      showSuccess(t('nacos.agentspecs_saved'));
      setEditOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doUpload = async () => {
    if (!uploadFile) {
      showError(t('nacos.agentspecs_pick_file'));
      return;
    }
    const fd = new FormData();
    fd.append('file', uploadFile);
    try {
      const res = await API.post(
        `/api/nacos/agentspecs/upload?namespace=${encodeURIComponent(
          namespace
        )}&overwrite=${uploadOverwrite ? 'true' : 'false'}`,
        fd
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'upload failed');
        return;
      }
      showSuccess(t('nacos.agentspecs_upload_ok'));
      setUploadOpen(false);
      setUploadFile(null);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doDelete = async () => {
    try {
      const res = await API.delete('/api/nacos/agentspecs/item', {
        params: { namespace, name: deleteName },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      showSuccess(t('nacos.agentspecs_deleted'));
      setDeleteOpen(false);
      setDeleteName('');
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doSubmit = async () => {
    try {
      const res = await API.post('/api/nacos/agentspecs/submit', {
        namespace,
        name: submitName,
        version: submitVersion || undefined,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'submit failed');
        return;
      }
      showSuccess(t('nacos.agentspecs_submit_ok'));
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
      showError(t('nacos.agentspecs_version_required'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/agentspecs/publish', {
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
      showSuccess(t('nacos.agentspecs_publish_ok'));
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
      const res = await API.get('/api/nacos/agentspecs/download', {
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
      showError(t('nacos.agentspecs_labels_invalid'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/agentspecs/labels', {
        namespace,
        name: labelsName,
        labels,
        replace: labelsReplace,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'save labels failed');
        return;
      }
      showSuccess(t('nacos.agentspecs_labels_saved'));
      setLabelsOpen(false);
      load();
      if (detailOpen && detail?.name === labelsName) {
        openDetail(labelsName);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('nacos.agentspecs_title')}
          </Card.Header>
          {info && (
            <Message info>
              {t('nacos.registry_storage')}: {info.zip_storage}
              {info.zip_local_dir ? ` | 本地: ${info.zip_local_dir}` : ''}
              {info.s3_remote_configured ? ' | S3 已配置' : ''}
              {info.max_upload_bytes
                ? ` | ${t('nacos.agentspecs_max_upload')}: ${fmtBytes(
                    info.max_upload_bytes
                  )}`
                : ''}
            </Message>
          )}
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
                setUploadOverwrite(false);
                setUploadOpen(true);
              }}
            >
              {t('nacos.agentspecs_upload')}
            </Button>
          </div>
          {loading ? (
            <Loader active />
          ) : (
            <Table celled compact>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>
                    {t('nacos.agentspecs_col_name')}
                  </Table.HeaderCell>
                  <Table.HeaderCell>
                    {t('nacos.agentspecs_col_desc')}
                  </Table.HeaderCell>
                  <Table.HeaderCell>
                    {t('nacos.agentspecs_col_enable')}
                  </Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.agentspecs_col_scope')}</Table.HeaderCell>
                  <Table.HeaderCell>editing</Table.HeaderCell>
                  <Table.HeaderCell>reviewing</Table.HeaderCell>
                  <Table.HeaderCell>online</Table.HeaderCell>
                  <Table.HeaderCell>
                    {t('nacos.agentspecs_col_actions')}
                  </Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <Table.Row key={r.name}>
                    <Table.Cell>{r.name}</Table.Cell>
                    <Table.Cell>
                      {r.description == null ? '' : String(r.description)}
                    </Table.Cell>
                    <Table.Cell>{r.enable ? '✓' : '—'}</Table.Cell>
                    <Table.Cell>{r.scope || 'PUBLIC'}</Table.Cell>
                    <Table.Cell>{r.editingVersion}</Table.Cell>
                    <Table.Cell>{r.reviewingVersion}</Table.Cell>
                    <Table.Cell>
                      {r.onlineCnt != null ? r.onlineCnt : '-'}
                    </Table.Cell>
                    <Table.Cell>
                      <Button
                        size='mini'
                        onClick={() => openDetail(r.name)}
                      >
                        {t('nacos.agentspecs_action_detail')}
                      </Button>
                      <Button size='mini' onClick={() => openEdit(r)}>
                        {t('nacos.agentspecs_action_edit')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          setSubmitName(r.name);
                          setSubmitVersion('');
                          setSubmitVersionBaseline('');
                          setSubmitOpen(true);
                        }}
                      >
                        {t('nacos.agentspecs_action_submit')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => openPublish(r.name, r.reviewingVersion)}
                      >
                        {t('nacos.agentspecs_action_publish')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          const lt = JSON.stringify(r.labels || {}, null, 2);
                          setLabelsName(r.name);
                          setLabelsText(lt);
                          setLabelsBaseline(lt);
                          setLabelsReplace(false);
                          setLabelsOpen(true);
                        }}
                      >
                        {t('nacos.agentspecs_action_labels')}
                      </Button>
                      <Button
                        size='mini'
                        negative
                        onClick={() => {
                          setDeleteName(r.name);
                          setDeleteOpen(true);
                        }}
                      >
                        {t('nacos.agentspecs_action_delete')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table>
          )}
        </Card.Content>
      </Card>

      <Modal open={detailOpen} onClose={() => setDetailOpen(false)} size='large'>
        <Modal.Header>{t('nacos.agentspecs_detail_title')}</Modal.Header>
        <Modal.Content scrolling>
          {detailLoading ? (
            <Loader active />
          ) : detail ? (
            <>
              <p>
                <strong>{detail.name}</strong> — {detail.description}
              </p>
              <p>
                enable: {detail.enable ? 'yes' : 'no'} | scope:{' '}
                {detail.scope || 'PUBLIC'} | bizTags: {detail.bizTags || '—'}
              </p>
              {detail.labels && Object.keys(detail.labels).length > 0 ? (
                <p>
                  labels: {JSON.stringify(detail.labels)}
                </p>
              ) : null}
              <Table celled compact size='small'>
                <Table.Header>
                  <Table.Row>
                    <Table.HeaderCell>version</Table.HeaderCell>
                    <Table.HeaderCell>status</Table.HeaderCell>
                    <Table.HeaderCell>commit</Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.agentspecs_col_actions')}
                    </Table.HeaderCell>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {(detail.versions || []).map((v) => (
                    <Table.Row key={v.version}>
                      <Table.Cell>{v.version}</Table.Cell>
                      <Table.Cell>{v.status}</Table.Cell>
                      <Table.Cell>{v.commitMsg || '—'}</Table.Cell>
                      <Table.Cell>
                        <Button
                          size='mini'
                          type='button'
                          onClick={() =>
                            downloadVersionZip(detail.name, v.version)
                          }
                        >
                          {t('nacos.agentspecs_download')}
                        </Button>
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
            {t('nacos.agentspecs_close')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={editOpen} onClose={() => setEditOpen(false)}>
        <Modal.Header>
          {t('nacos.agentspecs_edit_title')}: {editName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.agentspecs_col_desc')}
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
              label={t('nacos.agentspecs_col_scope')}
              hint='PUBLIC / PRIVATE'
              value={editScope}
              originValue={editBaseline.scope}
              onChange={(v) => setEditScope(v)}
              height={88}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.agentspecs_col_enable')}
                checked={editEnable}
                onChange={(e, { checked }) => setEditEnable(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setEditOpen(false)}>
            {t('nacos.agentspecs_close')}
          </Button>
          <Button primary onClick={saveMetadata}>
            {t('nacos.agentspecs_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={uploadOpen} onClose={() => setUploadOpen(false)}>
        <Modal.Header>{t('nacos.agentspecs_upload_title')}</Modal.Header>
        <Modal.Content>
          <p>{t('nacos.agentspecs_upload_hint')}</p>
          <input
            type='file'
            accept='.zip,application/zip'
            onChange={(e) =>
              setUploadFile(e.target.files && e.target.files[0])
            }
          />
          <Form.Field style={{ marginTop: 12 }}>
            <Checkbox
              label={t('nacos.agentspecs_upload_overwrite')}
              checked={uploadOverwrite}
              onChange={(e, { checked }) => setUploadOverwrite(!!checked)}
            />
          </Form.Field>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setUploadOpen(false)}>
            {t('nacos.agentspecs_close')}
          </Button>
          <Button primary onClick={doUpload}>
            {t('nacos.agentspecs_upload')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={submitOpen} onClose={() => setSubmitOpen(false)}>
        <Modal.Header>
          {t('nacos.agentspecs_submit_title')}: {submitName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.agentspecs_version_optional')}
              hint={t('nacos.agentspecs_submit_hint')}
              value={submitVersion}
              originValue={submitVersionBaseline}
              onChange={(v) => setSubmitVersion(v)}
              height={88}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setSubmitOpen(false)}>
            {t('nacos.agentspecs_close')}
          </Button>
          <Button primary onClick={doSubmit}>
            {t('nacos.agentspecs_action_submit')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={publishOpen} onClose={() => setPublishOpen(false)}>
        <Modal.Header>
          {t('nacos.agentspecs_publish_title')}: {publishName}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.agentspecs_publish_version')}
              hint={`${t('nacos.agentspecs_publish_hint')} · ${
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
                label={t('nacos.agentspecs_update_latest')}
                checked={publishUpdateLatest}
                onChange={(e, { checked }) =>
                  setPublishUpdateLatest(!!checked)
                }
              />
            </Form.Field>
            <Form.Field>
              <Checkbox
                label={t('nacos.agentspecs_force_publish')}
                checked={publishForce}
                onChange={(e, { checked }) => setPublishForce(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setPublishOpen(false)}>
            {t('nacos.agentspecs_close')}
          </Button>
          <Button positive onClick={doPublish}>
            {t('nacos.agentspecs_action_publish')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={labelsOpen} onClose={() => setLabelsOpen(false)} size='large'>
        <Modal.Header>
          {t('nacos.agentspecs_labels_title')}: {labelsName}
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
                label={t('nacos.agentspecs_labels_replace')}
                checked={labelsReplace}
                onChange={(e, { checked }) => setLabelsReplace(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setLabelsOpen(false)}>
            {t('nacos.agentspecs_close')}
          </Button>
          <Button primary onClick={saveLabels}>
            {t('nacos.agentspecs_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Confirm
        open={deleteOpen}
        header={t('nacos.agentspecs_delete_confirm')}
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

export default NacosAgentSpecsRegistry;
