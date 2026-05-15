import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
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
import NacosMonacoEditor from '../../components/nacos/NacosMonacoEditor';
import NacosMonacoDiffModal from '../../components/nacos/NacosMonacoDiffModal';
import SettingMonacoField from '../../components/SettingMonacoField';
import {
  monacoLanguageFromNacosType,
  normalizeConfigEOL,
} from '../../components/nacos/monacoLanguage';

const fmtTime = (ms) => {
  if (!ms) return '—';
  try {
    return new Date(ms).toLocaleString();
  } catch {
    return String(ms);
  }
};

const preview = (s, n = 96) => {
  if (s == null) return '';
  const t = String(s);
  return t.length <= n ? t : `${t.slice(0, n)}…`;
};

const NacosCsConfigs = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState([]);
  const [namespace, setNamespace] = useState(() => getStoredNacosNamespace());

  const [editOpen, setEditOpen] = useState(false);
  const [editDataId, setEditDataId] = useState('');
  const [editGroup, setEditGroup] = useState('');
  const [editType, setEditType] = useState('');
  const [editContent, setEditContent] = useState('');
  const [lockKeys, setLockKeys] = useState(false);
  const [editKeysBaseline, setEditKeysBaseline] = useState({
    dataId: '',
    group: '',
    type: '',
  });

  const [histOpen, setHistOpen] = useState(false);
  const [histLoading, setHistLoading] = useState(false);
  const [histRows, setHistRows] = useState([]);
  const [histKey, setHistKey] = useState({ dataId: '', group: '' });

  const [viewOpen, setViewOpen] = useState(false);
  const [viewContent, setViewContent] = useState('');
  const [viewLang, setViewLang] = useState('plaintext');

  const [histConfigType, setHistConfigType] = useState('');

  const [diffOpen, setDiffOpen] = useState(false);
  const [diffOriginal, setDiffOriginal] = useState('');
  const [diffModified, setDiffModified] = useState('');
  const [diffLang, setDiffLang] = useState('plaintext');

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteKey, setDeleteKey] = useState({ dataId: '', group: '' });

  const load = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/nacos/cs/configs', {
        params: { namespace, page: 1, size: 200 },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      setRows(res.data.data?.pageItems || []);
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

  const openCreate = () => {
    setLockKeys(false);
    setEditDataId('');
    setEditGroup('DEFAULT_GROUP');
    setEditType('yaml');
    setEditContent('# example\nkey: value\n');
    setEditKeysBaseline({ dataId: '', group: 'DEFAULT_GROUP', type: 'yaml' });
    setEditOpen(true);
  };

  const openEdit = async (dataId, group) => {
    setHistKey({ dataId, group });
    try {
      const res = await API.get('/api/nacos/cs/configs/detail', {
        params: { namespace, dataId, groupName: group },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      const d = res.data.data;
      setEditDataId(d.dataId || '');
      setEditGroup(d.groupName || d.group || '');
      setEditType(d.type || '');
      setEditContent(d.content ?? '');
      setLockKeys(true);
      setEditKeysBaseline({
        dataId: d.dataId || '',
        group: d.groupName || d.group || '',
        type: d.type || '',
      });
      setEditOpen(true);
    } catch (e) {
      showError(e.message);
    }
  };

  const savePublish = async () => {
    const dataId = editDataId.trim();
    const group = editGroup.trim();
    if (!dataId || !group) {
      showError(t('nacos.cs_keys_required'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/cs/configs/publish', {
        namespace,
        dataId,
        groupName: group,
        content: editContent,
        type: editType,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'publish failed');
        return;
      }
      showSuccess(t('nacos.cs_published'));
      setEditOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const loadHistory = async (dataId, group, configType) => {
    setHistKey({ dataId, group });
    setHistConfigType(configType || '');
    setHistOpen(true);
    setHistLoading(true);
    setHistRows([]);
    try {
      const res = await API.get('/api/nacos/cs/configs/history', {
        params: { namespace, dataId, groupName: group, page: 1, size: 50 },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'history failed');
        setHistOpen(false);
        return;
      }
      setHistRows(res.data.data?.pageItems || []);
    } catch (e) {
      showError(e.message);
      setHistOpen(false);
    } finally {
      setHistLoading(false);
    }
  };

  const doRollback = async (historyId) => {
    try {
      const res = await API.post('/api/nacos/cs/configs/rollback', {
        namespace,
        dataId: histKey.dataId,
        groupName: histKey.group,
        historyId,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'rollback failed');
        return;
      }
      showSuccess(t('nacos.cs_rollback_ok'));
      loadHistory(histKey.dataId, histKey.group, histConfigType);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doDelete = async () => {
    try {
      const res = await API.delete('/api/nacos/cs/configs/item', {
        params: {
          namespace,
          dataId: deleteKey.dataId,
          groupName: deleteKey.group,
        },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      showSuccess(t('nacos.cs_deleted'));
      setDeleteOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const openDiffVsCurrent = async (histRow) => {
    try {
      const res = await API.get('/api/nacos/cs/configs/detail', {
        params: {
          namespace,
          dataId: histKey.dataId,
          groupName: histKey.group,
        },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      const d = res.data.data;
      const lang = monacoLanguageFromNacosType(
        d.type || histConfigType || histRow?.type
      );
      setDiffLang(lang);
      setDiffOriginal(normalizeConfigEOL(histRow?.content ?? ''));
      setDiffModified(normalizeConfigEOL(d.content ?? ''));
      setDiffOpen(true);
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('nacos.cs_title')}
          </Card.Header>
          <Message info>{t('nacos.cs_hint')}</Message>
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
            <Button size='small' onClick={load}>
              {t('nacos.refresh')}
            </Button>
            <Button size='small' primary onClick={openCreate}>
              {t('nacos.cs_new')}
            </Button>
          </div>
          {loading ? (
            <Loader active />
          ) : (
            <Table celled compact>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>{t('nacos.cs_col_dataid')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_col_group')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_col_type')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_col_updated')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_col_actions')}</Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <Table.Row key={`${r.dataId}@@${r.groupName || r.group}`}>
                    <Table.Cell>{r.dataId}</Table.Cell>
                    <Table.Cell>{r.groupName || r.group}</Table.Cell>
                    <Table.Cell>{r.type || '—'}</Table.Cell>
                    <Table.Cell>{fmtTime(r.updateTime)}</Table.Cell>
                    <Table.Cell>
                      <Button
                        size='mini'
                        onClick={() => openEdit(r.dataId, r.groupName || r.group)}
                      >
                        {t('nacos.cs_edit')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          setViewContent(r.content || '');
                          setViewLang(monacoLanguageFromNacosType(r.type));
                          setViewOpen(true);
                        }}
                      >
                        {t('nacos.cs_preview')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() =>
                          loadHistory(
                            r.dataId,
                            r.groupName || r.group,
                            r.type
                          )
                        }
                      >
                        {t('nacos.cs_history')}
                      </Button>
                      <Button
                        size='mini'
                        negative
                        onClick={() => {
                          setDeleteKey({
                            dataId: r.dataId,
                            group: r.groupName || r.group,
                          });
                          setDeleteOpen(true);
                        }}
                      >
                        {t('nacos.cs_delete')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table>
          )}
        </Card.Content>
      </Card>

      <Modal open={editOpen} onClose={() => setEditOpen(false)} size='large'>
        <Modal.Header>{t('nacos.cs_edit_title')}</Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.cs_col_dataid')}
              value={editDataId}
              originValue={editKeysBaseline.dataId}
              onChange={(v) => setEditDataId(v)}
              readOnly={lockKeys}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.cs_col_group')}
              value={editGroup}
              originValue={editKeysBaseline.group}
              onChange={(v) => setEditGroup(v)}
              readOnly={lockKeys}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.cs_col_type')}
              hint='yaml / json / text'
              value={editType}
              originValue={editKeysBaseline.type}
              onChange={(v) => setEditType(v)}
              height={88}
            />
            <Form.Field>
              <label>{t('nacos.cs_content')}</label>
              <NacosMonacoEditor
                language={monacoLanguageFromNacosType(editType)}
                value={editContent}
                onChange={setEditContent}
                height={420}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setEditOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary onClick={savePublish}>
            {t('nacos.cs_publish')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={histOpen} onClose={() => setHistOpen(false)} size='large'>
        <Modal.Header>
          {t('nacos.cs_history_title')}: {histKey.dataId} / {histKey.group}
        </Modal.Header>
        <Modal.Content scrolling>
          {histLoading ? (
            <Loader active />
          ) : (
            <Table celled compact size='small'>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>id</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_hist_action')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_hist_operator')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_hist_time')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_hist_preview')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.cs_col_actions')}</Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {histRows.map((h) => (
                  <Table.Row key={h.id}>
                    <Table.Cell>{h.id}</Table.Cell>
                    <Table.Cell>{h.action}</Table.Cell>
                    <Table.Cell>
                      {h.operatorName || (h.operatorId ? `#${h.operatorId}` : '—')}
                    </Table.Cell>
                    <Table.Cell>{fmtTime(h.createdAt)}</Table.Cell>
                    <Table.Cell>
                      <code style={{ fontSize: 12 }}>{preview(h.content)}</code>
                    </Table.Cell>
                    <Table.Cell>
                      <Button
                        size='mini'
                        type='button'
                        onClick={() => {
                          setViewContent(h.content || '');
                          setViewLang(monacoLanguageFromNacosType(histConfigType));
                          setViewOpen(true);
                        }}
                      >
                        {t('nacos.cs_preview')}
                      </Button>
                      <Button
                        size='mini'
                        type='button'
                        onClick={() => openDiffVsCurrent(h)}
                      >
                        {t('nacos.cs_compare')}
                      </Button>
                      <Button
                        size='mini'
                        primary
                        type='button'
                        onClick={() => doRollback(h.id)}
                      >
                        {t('nacos.cs_rollback')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table>
          )}
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setHistOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={viewOpen} onClose={() => setViewOpen(false)} size='large'>
        <Modal.Header>{t('nacos.cs_content')}</Modal.Header>
        <Modal.Content>
          <NacosMonacoEditor
            readOnly
            language={viewLang}
            value={viewContent}
            height={Math.min(520, Math.round(window.innerHeight * 0.55))}
          />
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setViewOpen(false)}>
            {t('nacos.skills_close')}
          </Button>
        </Modal.Actions>
      </Modal>

      <NacosMonacoDiffModal
        open={diffOpen}
        onClose={() => setDiffOpen(false)}
        original={diffOriginal}
        modified={diffModified}
        language={diffLang}
      />

      <Confirm
        open={deleteOpen}
        header={t('nacos.cs_delete_confirm')}
        content={`${deleteKey.dataId} @ ${deleteKey.group} / ${namespace}`}
        onCancel={() => {
          setDeleteOpen(false);
          setDeleteKey({ dataId: '', group: '' });
        }}
        onConfirm={doDelete}
      />
    </div>
  );
};

export default NacosCsConfigs;
