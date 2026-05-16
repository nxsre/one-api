import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
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

const NacosPipelinesRegistry = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState([]);
  const [namespace, setNamespace] = useState(() => getStoredNacosNamespace());
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState(null);

  const detailJson = useMemo(
    () => (detail ? JSON.stringify(detail, null, 2) : ''),
    [detail]
  );

  const load = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/nacos/pipelines', {
        params: { namespace, page: 1, size: 100 },
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

  const runScan = async () => {
    try {
      const res = await API.post(
        `/api/nacos/pipelines/run-scan?namespace=${encodeURIComponent(namespace)}`
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'run failed');
        return;
      }
      showSuccess(t('nacos.pipelines_scan_ok'));
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const openDetail = async (id) => {
    try {
      const res = await API.get('/api/nacos/pipelines/detail', {
        params: { namespace, id },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load detail failed');
        return;
      }
      setDetail(res.data.data);
      setDetailOpen(true);
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('nacos.pipelines_title')}
          </Card.Header>
          <Message info>{t('nacos.pipelines_hint')}</Message>
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
            <Button size='small' primary onClick={runScan}>
              {t('nacos.pipelines_run_scan')}
            </Button>
          </div>
          {loading ? (
            <Loader active />
          ) : (
            <Table celled compact>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>id</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.pipelines_col_type')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.pipelines_col_status')}</Table.HeaderCell>
                  <Table.HeaderCell>message</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.pipelines_col_actions')}</Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <Table.Row key={r.id}>
                    <Table.Cell>{r.id}</Table.Cell>
                    <Table.Cell>{r.jobType}</Table.Cell>
                    <Table.Cell>{r.status}</Table.Cell>
                    <Table.Cell>{r.message || '—'}</Table.Cell>
                    <Table.Cell>
                      <Button size='mini' onClick={() => openDetail(r.id)}>
                        {t('nacos.pipelines_detail')}
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
        <Modal.Header>{t('nacos.pipelines_detail_title')}</Modal.Header>
        <Modal.Content>
          {detail ? (
            <Form.Field>
              <label>{t('nacos.pipelines_json_view')}</label>
              <NacosMonacoEditor
                readOnly
                language='json'
                value={detailJson}
                height={Math.min(560, Math.round(window.innerHeight * 0.5))}
              />
            </Form.Field>
          ) : null}
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDetailOpen(false)}>{t('nacos.skills_close')}</Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
};

export default NacosPipelinesRegistry;
