import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Header,
  Icon,
  Label,
  Loader,
  Message,
  Modal,
  Table,
} from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import SettingMonacoField from '../../components/SettingMonacoField';

/** 与 Nacos console-ui-next Namespace 及 GET /api/nacos/namespaces 对齐 */
const isPublicNs = (row) => row && row.type === 0;

const NacosNamespaces = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailData, setDetailData] = useState(null);
  const [selected, setSelected] = useState(null);
  const [saving, setSaving] = useState(false);

  const [formId, setFormId] = useState('');
  const [formName, setFormName] = useState('');
  const [formDesc, setFormDesc] = useState('');
  const [formBaseline, setFormBaseline] = useState({
    id: '',
    name: '',
    desc: '',
  });

  const resetForm = () => {
    setFormId('');
    setFormName('');
    setFormDesc('');
    setFormBaseline({ id: '', name: '', desc: '' });
    setSelected(null);
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/nacos/namespaces');
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      setRows(Array.isArray(res.data.data) ? res.data.data : []);
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleCreate = async () => {
    if (!formName.trim()) {
      showError(t('nacos.ns_name_required'));
      return;
    }
    setSaving(true);
    try {
      const res = await API.post('/api/nacos/namespaces', {
        customNamespaceId: formId.trim(),
        namespaceName: formName.trim(),
        namespaceDesc: formDesc.trim(),
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'create failed');
        return;
      }
      showSuccess(t('nacos.ns_created'));
      setCreateOpen(false);
      resetForm();
      load();
    } catch (e) {
      showError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const openEdit = (ns) => {
    setSelected(ns);
    const name = ns.namespaceShowName || ns.namespace || '';
    const desc = ns.namespaceDesc || '';
    setFormName(name);
    setFormDesc(desc);
    setFormBaseline({
      id: ns.namespace || '',
      name,
      desc,
    });
    setEditOpen(true);
  };

  const handleEdit = async () => {
    if (!selected || !formName.trim()) {
      showError(t('nacos.ns_name_required'));
      return;
    }
    setSaving(true);
    try {
      const res = await API.put('/api/nacos/namespaces', {
        namespace: selected.namespace,
        namespaceShowName: formName.trim(),
        namespaceDesc: formDesc.trim(),
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'update failed');
        return;
      }
      showSuccess(t('nacos.ns_updated'));
      setEditOpen(false);
      resetForm();
      load();
    } catch (e) {
      showError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const openDelete = (ns) => {
    setSelected(ns);
    setDeleteOpen(true);
  };

  const handleDelete = async () => {
    if (!selected) return;
    try {
      const res = await API.delete(
        `/api/nacos/namespaces/${encodeURIComponent(selected.namespace)}`,
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      showSuccess(t('nacos.ns_deleted'));
      setDeleteOpen(false);
      setSelected(null);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const openDetail = async (ns) => {
    setSelected(ns);
    setDetailOpen(true);
    setDetailData(null);
    setDetailLoading(true);
    try {
      const res = await API.get('/api/nacos/namespaces/detail', {
        params: { namespaceId: ns.namespace },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'detail failed');
        setDetailData(ns);
        return;
      }
      setDetailData(res.data.data || ns);
    } catch (e) {
      showError(e.message);
      setDetailData(ns);
    } finally {
      setDetailLoading(false);
    }
  };

  const descCell = (ns) => {
    if (isPublicNs(ns)) {
      return ns.namespaceDesc || t('nacos.ns_public_desc');
    }
    return ns.namespaceDesc || '—';
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              flexWrap: 'wrap',
              gap: 12,
              marginBottom: 12,
            }}
          >
            <Card.Header className='header' style={{ marginBottom: 0 }}>
              {t('nacos.ns_native_title')}
            </Card.Header>
            <div>
              <Button icon labelPosition='left' onClick={load} disabled={loading}>
                <Icon name='refresh' />
                {t('nacos.ns_refresh')}
              </Button>
              <Button
                primary
                icon
                labelPosition='left'
                style={{ marginLeft: 8 }}
                onClick={() => {
                  resetForm();
                  setCreateOpen(true);
                }}
              >
                <Icon name='plus' />
                {t('nacos.ns_add')}
              </Button>
            </div>
          </div>
          <Message info>{t('nacos.ns_manage_hint')}</Message>
          {loading ? (
            <Loader active />
          ) : rows.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '48px 16px', color: '#888' }}>
              <Icon name='globe' size='huge' style={{ opacity: 0.35, marginBottom: 12 }} />
              <Header as='h4' disabled>
                {t('nacos.ns_no_data')}
              </Header>
            </div>
          ) : (
            <Table celled compact size='small'>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>{t('nacos.ns_col_name')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.ns_col_id')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.ns_col_tenant_id')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.ns_col_tenant_name')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.ns_col_desc')}</Table.HeaderCell>
                  <Table.HeaderCell textAlign='center'>
                    {t('nacos.ns_col_config_count')}
                  </Table.HeaderCell>
                  <Table.HeaderCell textAlign='right'>
                    {t('nacos.ns_col_actions')}
                  </Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((ns) => (
                  <Table.Row key={ns.namespace || 'row'}>
                    <Table.Cell>
                      <strong>{ns.namespaceShowName || ns.namespace}</strong>
                      {isPublicNs(ns) ? (
                        <Label size='mini' style={{ marginLeft: 8 }}>
                          {t('nacos.ns_public_badge')}
                        </Label>
                      ) : null}
                    </Table.Cell>
                    <Table.Cell>
                      <code style={{ fontSize: 12 }}>{ns.namespace || '(public)'}</code>
                    </Table.Cell>
                    <Table.Cell>
                      {ns.ownerTenantId != null && ns.ownerTenantId > 0 ? (
                        <code style={{ fontSize: 12 }}>{ns.ownerTenantId}</code>
                      ) : (
                        '—'
                      )}
                    </Table.Cell>
                    <Table.Cell title={ns.ownerTenantName || ''}>
                      {ns.ownerTenantName || '—'}
                    </Table.Cell>
                    <Table.Cell
                      style={{ maxWidth: 280, overflow: 'hidden', textOverflow: 'ellipsis' }}
                      title={descCell(ns)}
                    >
                      {descCell(ns)}
                    </Table.Cell>
                    <Table.Cell textAlign='center'>
                      <Label basic>{ns.configCount ?? 0}</Label>
                    </Table.Cell>
                    <Table.Cell textAlign='right'>
                      <Button size='mini' basic onClick={() => openDetail(ns)}>
                        {t('nacos.ns_detail')}
                      </Button>
                      <Button
                        size='mini'
                        basic
                        disabled={isPublicNs(ns)}
                        onClick={() => openEdit(ns)}
                      >
                        {t('nacos.ns_edit')}
                      </Button>
                      <Button
                        size='mini'
                        basic
                        color='red'
                        disabled={isPublicNs(ns)}
                        onClick={() => openDelete(ns)}
                      >
                        {t('nacos.skills_action_delete')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table>
          )}
          {!loading && rows.length > 0 ? (
            <div style={{ marginTop: 12, fontSize: 13, color: '#64748b' }}>
              {t('nacos.ns_total', { total: rows.length })}
            </div>
          ) : null}
        </Card.Content>
      </Card>

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} size='small'>
        <Modal.Header>{t('nacos.ns_add')}</Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              required
              label={t('nacos.ns_col_name')}
              hint={t('nacos.ns_placeholder_name')}
              value={formName}
              originValue={formBaseline.name}
              onChange={(v) => setFormName(v)}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.ns_col_id')}
              hint={t('nacos.ns_placeholder_id')}
              value={formId}
              originValue={formBaseline.id}
              onChange={(v) => setFormId(v)}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.ns_col_desc')}
              hint={t('nacos.ns_placeholder_desc')}
              value={formDesc}
              originValue={formBaseline.desc}
              onChange={(v) => setFormDesc(v)}
              height={120}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => { setCreateOpen(false); resetForm(); }}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary loading={saving} onClick={handleCreate}>
            {t('nacos.ns_confirm')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={editOpen} onClose={() => setEditOpen(false)} size='small'>
        <Modal.Header>{t('nacos.ns_edit_modal_title')}</Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.ns_col_id')}
              value={selected?.namespace || ''}
              originValue={formBaseline.id}
              readOnly
              enableDiff={false}
              height={72}
              onChange={() => {}}
            />
            <SettingMonacoField
              required
              label={t('nacos.ns_col_name')}
              hint={t('nacos.ns_placeholder_name')}
              value={formName}
              originValue={formBaseline.name}
              onChange={(v) => setFormName(v)}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.ns_col_desc')}
              hint={t('nacos.ns_placeholder_desc')}
              value={formDesc}
              originValue={formBaseline.desc}
              onChange={(v) => setFormDesc(v)}
              height={120}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => { setEditOpen(false); resetForm(); }}>
            {t('nacos.skills_close')}
          </Button>
          <Button primary loading={saving} onClick={handleEdit}>
            {t('nacos.ns_confirm')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={deleteOpen} onClose={() => setDeleteOpen(false)} size='small'>
        <Modal.Header>{t('nacos.skills_action_delete')}</Modal.Header>
        <Modal.Content>
          <p>{t('nacos.ns_delete_confirm_named', { name: selected?.namespaceShowName || selected?.namespace })}</p>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDeleteOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button negative onClick={handleDelete}>
            {t('nacos.ns_confirm')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={detailOpen} onClose={() => setDetailOpen(false)} size='small'>
        <Modal.Header>{t('nacos.ns_detail')}</Modal.Header>
        <Modal.Content>
          {detailLoading ? <Loader active inline='centered' /> : null}
          {!detailLoading && detailData ? (
            <Table basic='very' collapsing>
              <Table.Body>
                <Table.Row>
                  <Table.Cell><strong>{t('nacos.ns_col_name')}</strong></Table.Cell>
                  <Table.Cell>{detailData.namespaceShowName}</Table.Cell>
                </Table.Row>
                <Table.Row>
                  <Table.Cell><strong>{t('nacos.ns_col_id')}</strong></Table.Cell>
                  <Table.Cell>
                    <code>{detailData.namespace || '(public)'}</code>
                  </Table.Cell>
                </Table.Row>
                <Table.Row>
                  <Table.Cell><strong>{t('nacos.ns_col_tenant_id')}</strong></Table.Cell>
                  <Table.Cell>
                    {detailData.ownerTenantId != null && detailData.ownerTenantId > 0
                      ? detailData.ownerTenantId
                      : '—'}
                  </Table.Cell>
                </Table.Row>
                <Table.Row>
                  <Table.Cell><strong>{t('nacos.ns_col_tenant_name')}</strong></Table.Cell>
                  <Table.Cell>{detailData.ownerTenantName || '—'}</Table.Cell>
                </Table.Row>
                <Table.Row>
                  <Table.Cell><strong>{t('nacos.ns_config_quota_label')}</strong></Table.Cell>
                  <Table.Cell>
                    {t('nacos.ns_config_quota', {
                      used: detailData.configCount ?? 0,
                      quota: detailData.quota ?? 128,
                    })}
                  </Table.Cell>
                </Table.Row>
                <Table.Row>
                  <Table.Cell><strong>{t('nacos.ns_col_desc')}</strong></Table.Cell>
                  <Table.Cell>{detailData.namespaceDesc || '—'}</Table.Cell>
                </Table.Row>
              </Table.Body>
            </Table>
          ) : null}
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDetailOpen(false)}>{t('nacos.skills_close')}</Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
};

export default NacosNamespaces;
