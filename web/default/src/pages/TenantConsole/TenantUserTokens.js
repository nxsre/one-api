import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Card, Label, Table } from 'semantic-ui-react';
import {
  API,
  copy,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

function tokenStatusLabel(status, t) {
  switch (status) {
    case 1:
      return (
        <Label basic color='green'>
          {t('token.table.status_enabled')}
        </Label>
      );
    case 2:
      return (
        <Label basic color='red'>
          {t('token.table.status_disabled')}
        </Label>
      );
    case 3:
      return (
        <Label basic color='yellow'>
          {t('token.table.status_expired')}
        </Label>
      );
    case 4:
      return (
        <Label basic color='grey'>
          {t('token.table.status_depleted')}
        </Label>
      );
    default:
      return (
        <Label basic color='black'>
          {t('token.table.status_unknown')}
        </Label>
      );
  }
}

export default function TenantUserTokens() {
  const { t } = useTranslation();
  const { ownerId } = useParams();
  const [tokens, setTokens] = useState([]);
  const [loading, setLoading] = useState(true);
  const [pageIdx, setPageIdx] = useState(0);

  const load = async (p) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/tenant_console/users/${encodeURIComponent(ownerId)}/tokens?p=${p}`
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setTokens(Array.isArray(data) ? data : []);
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
  }, [ownerId]);

  const toggleStatus = async (row) => {
    const next = row.status === 1 ? 2 : 1;
    try {
      const res = await API.put(
        `/api/tenant_console/users/${encodeURIComponent(ownerId)}/tokens?status_only=1`,
        { id: row.id, status: next }
      );
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('tenant_console.tokens.status_updated'));
      load(pageIdx).then();
    } catch (e) {
      showError(e.message);
    }
  };

  const pageRows = tokens;

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('tenant_console.tokens.title')}
          </Card.Header>
          <div style={{ marginBottom: 12 }}>
            <Button as={Link} to='/tenant-console/users'>
              {t('tenant_console.tokens.back_users')}
            </Button>
            <Button
              positive
              as={Link}
              to={`/tenant-console/users/${encodeURIComponent(ownerId)}/tokens/add`}
            >
              {t('tenant_console.tokens.add')}
            </Button>
          </div>
          <Table celled compact unstackable loading={loading}>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>{t('tenant_console.tokens.col_name')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.tokens.col_status')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.tokens.col_created')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.tokens.col_key')}</Table.HeaderCell>
                <Table.HeaderCell>{t('tenant_console.tokens.col_actions')}</Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {pageRows.map((row) => (
                <Table.Row key={row.id}>
                  <Table.Cell>{row.name}</Table.Cell>
                  <Table.Cell>{tokenStatusLabel(row.status, t)}</Table.Cell>
                  <Table.Cell>
                    {timestamp2string(row.created_time)}
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      size='mini'
                      type='button'
                      onClick={() =>
                        copy(row.key ? `sk-${row.key}` : '').then(() =>
                          showSuccess(t('tenant_console.tokens.copied'))
                        )
                      }
                    >
                      {t('tenant_console.tokens.copy_sk')}
                    </Button>
                  </Table.Cell>
                  <Table.Cell>
                    <Button size='mini' type='button' onClick={() => toggleStatus(row)}>
                      {row.status === 1
                        ? t('tenant_console.tokens.disable')
                        : t('tenant_console.tokens.enable')}
                    </Button>
                    <Button
                      size='mini'
                      as={Link}
                      to={`/tenant-console/users/${encodeURIComponent(ownerId)}/tokens/edit/${row.id}`}
                    >
                      {t('tenant_console.tokens.edit')}
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
              disabled={tokens.length < ITEMS_PER_PAGE || loading}
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
