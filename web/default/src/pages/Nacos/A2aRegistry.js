import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Confirm,
  Form,
  Loader,
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

const defaultCard = () =>
  JSON.stringify({ name: 'example-agent', version: '1.0.0' }, null, 2);

const NacosA2aRegistry = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState([]);
  const [namespace, setNamespace] = useState(() => getStoredNacosNamespace());
  const [editOpen, setEditOpen] = useState(false);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editCard, setEditCard] = useState(defaultCard());
  const [editBiz, setEditBiz] = useState('');
  const [editScope, setEditScope] = useState('PUBLIC');
  const [editEnable, setEditEnable] = useState(true);
  const [editBaseline, setEditBaseline] = useState({
    name: '',
    desc: '',
    card: defaultCard(),
    biz: '',
    scope: 'PUBLIC',
  });
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteName, setDeleteName] = useState('');
  const [nameLocked, setNameLocked] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/nacos/a2a', {
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
    setNameLocked(false);
    const name = '';
    const desc = '';
    const card = defaultCard();
    const biz = '';
    const scope = 'PUBLIC';
    setEditName(name);
    setEditDesc(desc);
    setEditCard(card);
    setEditBiz(biz);
    setEditScope(scope);
    setEditEnable(true);
    setEditBaseline({ name, desc, card, biz, scope });
    setEditOpen(true);
  };

  const openEdit = async (name) => {
    try {
      const res = await API.get('/api/nacos/a2a/detail', {
        params: { namespace, agentName: name },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load detail failed');
        return;
      }
      const d = res.data.data;
      const cardStr = JSON.stringify(d.card || {}, null, 2);
      setEditName(d.agentName);
      setEditDesc(d.description || '');
      setEditCard(cardStr);
      setEditBiz(d.bizTags || '');
      setEditScope(d.scope || 'PUBLIC');
      setEditEnable(!!d.enable);
      setEditBaseline({
        name: d.agentName,
        desc: d.description || '',
        card: cardStr,
        biz: d.bizTags || '',
        scope: d.scope || 'PUBLIC',
      });
      setEditOpen(true);
      setNameLocked(true);
    } catch (e) {
      showError(e.message);
    }
  };

  const save = async () => {
    let card;
    try {
      card = JSON.parse(editCard || '{}');
    } catch {
      showError(t('nacos.a2a_invalid_json'));
      return;
    }
    const name = editName.trim();
    if (!name) {
      showError(t('nacos.a2a_name_required'));
      return;
    }
    try {
      const res = await API.post(
        `/api/nacos/a2a?namespace=${encodeURIComponent(namespace)}`,
        {
          agentName: name,
          description: editDesc,
          card,
          bizTags: editBiz,
          scope: editScope,
          enable: editEnable,
        }
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'save failed');
        return;
      }
      showSuccess(t('nacos.a2a_saved'));
      setEditOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const doDelete = async () => {
    try {
      const res = await API.delete('/api/nacos/a2a', {
        params: { namespace, agentName: deleteName },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      showSuccess(t('nacos.a2a_deleted'));
      setDeleteOpen(false);
      setDeleteName('');
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>{t('nacos.a2a_title')}</Card.Header>
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
              {t('nacos.a2a_new')}
            </Button>
          </div>
          {loading ? (
            <Loader active />
          ) : (
            <Table celled compact>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>{t('nacos.a2a_col_name')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.a2a_col_desc')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.a2a_col_scope')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.a2a_col_enable')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.a2a_col_actions')}</Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <Table.Row key={r.agentName}>
                    <Table.Cell>{r.agentName}</Table.Cell>
                    <Table.Cell>{r.description}</Table.Cell>
                    <Table.Cell>{r.scope || 'PUBLIC'}</Table.Cell>
                    <Table.Cell>{r.enable ? '✓' : '—'}</Table.Cell>
                    <Table.Cell>
                      <Button size='mini' onClick={() => openEdit(r.agentName)}>
                        {t('nacos.a2a_edit')}
                      </Button>
                      <Button
                        size='mini'
                        negative
                        onClick={() => {
                          setDeleteName(r.agentName);
                          setDeleteOpen(true);
                        }}
                      >
                        {t('nacos.a2a_delete')}
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
        <Modal.Header>{t('nacos.a2a_edit_title')}</Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.a2a_col_name')}
              value={editName}
              originValue={editBaseline.name}
              onChange={(v) => setEditName(v)}
              readOnly={nameLocked}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.a2a_col_desc')}
              value={editDesc}
              originValue={editBaseline.desc}
              onChange={(v) => setEditDesc(v)}
              height={140}
            />
            <SettingMonacoField
              label='bizTags'
              value={editBiz}
              originValue={editBaseline.biz}
              onChange={(v) => setEditBiz(v)}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.a2a_col_scope')}
              hint='PUBLIC / PRIVATE'
              value={editScope}
              originValue={editBaseline.scope}
              onChange={(v) => setEditScope(v)}
              height={88}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.a2a_col_enable')}
                checked={editEnable}
                onChange={(e, { checked }) => setEditEnable(!!checked)}
              />
            </Form.Field>
            <SettingMonacoField
              label='card (JSON)'
              language='json'
              enableJsonFormat
              minimap
              value={editCard}
              originValue={editBaseline.card}
              onChange={(v) => setEditCard(v)}
              height={400}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setEditOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button primary onClick={save}>
            {t('nacos.skills_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Confirm
        open={deleteOpen}
        header={t('nacos.a2a_delete_confirm')}
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

export default NacosA2aRegistry;
