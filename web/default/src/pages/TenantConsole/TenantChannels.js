import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Card, Table } from 'semantic-ui-react';
import { API, showError } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { renderNumber } from '../../helpers/render';

export default function TenantChannels() {
  const { t } = useTranslation();
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [pageIdx, setPageIdx] = useState(0);

  const load = async (p) => {
    setLoading(true);
    try {
      const res = await API.get(`/api/tenant_console/channels?p=${p}&page_size=${ITEMS_PER_PAGE}`);
      const { success, message, data } = res.data;
      if (!success) {
        if (String(message).includes('请先在页面顶部选择')) {
          setRows([]);
        } else {
          showError(message);
        }
        return;
      }
      setRows(Array.isArray(data) ? data : []);
      setPageIdx(p);
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load(0).then();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('tenant_console.channels.title')}
          </Card.Header>
          <div style={{ marginBottom: 12 }}>
            <Button positive as={Link} to='/tenant-console/channels/add'>
              {t('tenant_console.channels.add')}
            </Button>
          </div>
          <Table celled compact selectable unstackable loading={loading}>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>ID</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.channels.col_name')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.channels.col_type')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.channels.col_status')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.channels.col_priority')}</Table.HeaderCell>
                <Table.HeaderCell>API 调用次数</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.channels.col_actions')}</Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {rows.map((ch) => (
                <Table.Row key={ch.id}>
                  <Table.Cell>{ch.id}</Table.Cell>
                  <Table.Cell>{ch.name}</Table.Cell>
                  <Table.Cell>{renderNumber(ch.type)}</Table.Cell>
                  <Table.Cell>{ch.status}</Table.Cell>
                  <Table.Cell>{ch.priority ?? '—'}</Table.Cell>
                  <Table.Cell>{ch.api_call_count ?? 0}</Table.Cell>
                  <Table.Cell>
                    <Button
                      size='mini'
                      as={Link}
                      to={`/tenant-console/channels/edit/${ch.id}`}
                      state={{ channel: ch }}
                    >
                      {t('tenant_console.channels.edit')}
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
          <div style={{ marginTop: 12 }}>
            <Button
              type='button'
              disabled={pageIdx <= 0 || loading}
              onClick={() => load(pageIdx - 1)}
            >
              {t('tenant_console.users.prev')}
            </Button>
            <Button
              type='button'
              disabled={rows.length < ITEMS_PER_PAGE || loading}
              onClick={() => load(pageIdx + 1)}
            >
              {t('tenant_console.users.next')}
            </Button>
          </div>
        </Card.Content>
      </Card>
    </div>
  );
}
