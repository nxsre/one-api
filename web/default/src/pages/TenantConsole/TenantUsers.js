import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Form,
  Table,
  Modal,
} from 'semantic-ui-react';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { renderQuota } from '../../helpers/render';

export default function TenantUsers() {
  const { t } = useTranslation();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [pageIdx, setPageIdx] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [delOpen, setDelOpen] = useState(false);
  const [delTarget, setDelTarget] = useState(null);

  const loadPage = async (p) => {
    setLoading(true);
    try {
      const res = await API.get(`/api/tenant_console/users?p=${p}`);
      const { success, message, data } = res.data;
      if (!success) {
        if (String(message).includes('请先在页面顶部选择')) {
          setUsers([]);
        } else {
          showError(message);
        }
        return;
      }
      setUsers(Array.isArray(data) ? data : []);
      setPageIdx(p);
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  const search = async () => {
    const kw = keyword.trim();
    if (!kw) {
      loadPage(0).then();
      return;
    }
    setSearching(true);
    try {
      const res = await API.get(
        `/api/tenant_console/users/search?keyword=${encodeURIComponent(kw)}`
      );
      const { success, message, data } = res.data;
      if (!success) {
        if (String(message).includes('请先在页面顶部选择')) {
          setUsers([]);
        } else {
          showError(message);
        }
        return;
      }
      setUsers(Array.isArray(data) ? data : []);
      setPageIdx(0);
    } catch (e) {
      showError(e.message);
    } finally {
      setSearching(false);
    }
  };

  useEffect(() => {
    loadPage(0).then();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openDelete = (u) => {
    setDelTarget(u);
    setDelOpen(true);
  };

  const confirmDelete = async () => {
    if (!delTarget) return;
    try {
      const res = await API.delete(
        `/api/tenant_console/users/${encodeURIComponent(delTarget.user_id)}`
      );
      const { success, message } = res.data;
      if (!success) {
        if (String(message).includes('请先在页面顶部选择')) {
          setUsers([]);
        } else {
          showError(message);
        }
        return;
      }
      showSuccess(t('tenant_console.users.delete_success'));
      setDelOpen(false);
      setDelTarget(null);
      loadPage(0).then();
    } catch (e) {
      showError(e.message);
    }
  };

  const pageRows = users;

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('tenant_console.users.title')}
          </Card.Header>
          <Form style={{ marginBottom: 16 }}>
            <Form.Group>
              <Form.Input
                placeholder={t('tenant_console.users.search_placeholder')}
                value={keyword}
                onChange={(e, { value }) => setKeyword(value)}
                width={8}
              />
              <Button type='button' onClick={() => search()} loading={searching}>
                {t('tenant_console.users.search')}
              </Button>
              <Button
                type='button'
                positive
                as={Link}
                to='/tenant-console/users/add'
              >
                {t('tenant_console.users.add')}
              </Button>
            </Form.Group>
          </Form>
          <Table celled compact selectable unstackable loading={loading}>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>
                  {t('tenant_console.users.col_username')}
                </Table.HeaderCell>
                <Table.HeaderCell>
                  {t('tenant_console.users.col_display_name')}
                </Table.HeaderCell>
                <Table.HeaderCell>
                  {t('tenant_console.users.col_group')}
                </Table.HeaderCell>
                <Table.HeaderCell>
                  {t('tenant_console.users.col_quota')}
                </Table.HeaderCell>
                <Table.HeaderCell>
                  {t('tenant_console.users.col_actions')}
                </Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {pageRows.map((u) => (
                <Table.Row key={u.user_id}>
                  <Table.Cell>{renderText(u.username)}</Table.Cell>
                  <Table.Cell>{renderText(u.display_name)}</Table.Cell>
                  <Table.Cell>{renderText(u.group)}</Table.Cell>
                  <Table.Cell>{renderQuota(u.quota, t)}</Table.Cell>
                  <Table.Cell>
                    <Button
                      size='mini'
                      as={Link}
                      to={`/tenant-console/users/edit/${encodeURIComponent(u.user_id)}`}
                    >
                      {t('tenant_console.users.edit')}
                    </Button>
                    <Button
                      size='mini'
                      as={Link}
                      to={`/tenant-console/users/${encodeURIComponent(u.user_id)}/tokens`}
                    >
                      {t('tenant_console.users.tokens')}
                    </Button>
                    <Button size='mini' negative onClick={() => openDelete(u)}>
                      {t('tenant_console.users.delete')}
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
          {!keyword.trim() ? (
            <div style={{ marginTop: 12 }}>
              <Button
                type='button'
                disabled={pageIdx <= 0 || loading}
                onClick={() => loadPage(pageIdx - 1)}
              >
                {t('tenant_console.users.prev')}
              </Button>
              <Button
                type='button'
                disabled={users.length < ITEMS_PER_PAGE || loading}
                onClick={() => loadPage(pageIdx + 1)}
              >
                {t('tenant_console.users.next')}
              </Button>
              <span style={{ marginLeft: 12, opacity: 0.8 }}>
                {t('tenant_console.users.page_hint', { n: pageIdx + 1 })}
              </span>
            </div>
          ) : null}
        </Card.Content>
      </Card>
      <Modal size='small' open={delOpen} onClose={() => setDelOpen(false)}>
        <Modal.Header>{t('tenant_console.users.delete_confirm_title')}</Modal.Header>
        <Modal.Content>
          <p>
            {t('tenant_console.users.delete_confirm_body', {
              name: delTarget?.username || '',
            })}
          </p>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDelOpen(false)}>
            {t('tenant_console.users.cancel')}
          </Button>
          <Button negative onClick={() => confirmDelete()}>
            {t('tenant_console.users.delete')}
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
}

function renderText(v) {
  if (v === undefined || v === null || v === '') return '—';
  return String(v);
}
