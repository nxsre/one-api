import React, { useEffect, useState, useCallback } from 'react';
import {
  Button,
  Form,
  Label,
  Pagination,
  Popup,
  Table,
  Dropdown,
  Icon,
} from 'semantic-ui-react';
import { Link } from 'react-router-dom';
import { API, showError, showSuccess } from '../helpers';
import { useTranslation } from 'react-i18next';

import { ITEMS_PER_PAGE } from '../constants';
import {
  renderGroup,
  renderNumber,
  renderQuota,
  renderText,
} from '../helpers/render';

function renderRole(role, t) {
  switch (role) {
    case 1:
      return <Label>{t('user.table.role_types.normal')}</Label>;
    case 10:
      return <Label color='yellow'>{t('user.table.role_types.admin')}</Label>;
    case 20:
      return <Label color='teal'>{t('user.table.role_types.tenant_admin')}</Label>;
    case 50:
      return <Label color='violet'>{t('user.table.role_types.super_admin')}</Label>;
    case 100:
      return <Label color='orange'>{t('user.table.role_types.root')}</Label>;
    default:
      return <Label color='red'>{t('user.table.role_types.unknown')}</Label>;
  }
}

const UsersTable = () => {
  const { t } = useTranslation();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [totalCount, setTotalCount] = useState(0);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [orderBy, setOrderBy] = useState('');
  const [scope, setScope] = useState('independent');
  const [tenantId, setTenantId] = useState('');
  const [roles, setRoles] = useState([]);
  const [tenantOptions, setTenantOptions] = useState([]);

  const fetchTenants = async () => {
    try {
      const res = await API.get('/api/platform/tenants/options');
      const { success, data } = res.data;
      if (success) {
        setTenantOptions(
          data.map((t) => ({
            key: t.id,
            text: t.name,
            value: t.id,
          }))
        );
      }
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchTenants();
  }, []);

  const loadUsers = useCallback(
    async (page) => {
      setLoading(true);
      try {
        let url = `/api/user/paged?p=${page - 1}&page_size=${ITEMS_PER_PAGE}&order=${orderBy}&scope=${scope}`;
        if (scope === 'tenant' && tenantId) {
          url += `&tenant_id=${tenantId}`;
        }
        if (roles.length > 0) {
          url += `&roles=${roles.join(',')}`;
        }
        if (searchKeyword) {
          url += `&keyword=${encodeURIComponent(searchKeyword)}`;
        }
        const res = await API.get(url);
        const { success, message, data } = res.data;
        if (success) {
          setUsers(data.items || []);
          setTotalCount(data.total || 0);
        } else {
          showError(message);
        }
      } catch (err) {
        showError(err.message);
      }
      setLoading(false);
    },
    [orderBy, scope, tenantId, roles, searchKeyword]
  );

  const onPaginationChange = (e, { activePage }) => {
    setActivePage(activePage);
    loadUsers(activePage);
  };

  useEffect(() => {
    setActivePage(1);
    loadUsers(1);
  }, [loadUsers]);

  const searchUsers = async () => {
    setActivePage(1);
    await loadUsers(1);
  };

  const handleKeywordChange = (e, { value }) => {
    setSearchKeyword(value);
  };

  const sortUser = (key) => {
    let newOrder = key;
    if (orderBy === key) {
      newOrder = key + '_desc';
    }
    setOrderBy(newOrder);
  };

  const handleOrderByChange = (e, { value }) => {
    setOrderBy(value);
  };

  const manageUser = async (username, action, idx) => {
    try {
      const res = await API.post('/api/user/manage', { username, action });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('user.messages.operation_success'));
        let user = res.data.data;
        let newUsers = [...users];
        if (action === 'delete') {
          newUsers.splice(idx, 1);
        } else {
          newUsers[idx].status = user.status;
          newUsers[idx].role = user.role;
        }
        setUsers(newUsers);
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const renderStatus = (status) => {
    switch (status) {
      case 1:
        return <Label basic>{t('user.table.status_types.activated')}</Label>;
      case 2:
        return (
          <Label basic color='red'>
            {t('user.table.status_types.banned')}
          </Label>
        );
      default:
        return (
          <Label basic color='grey'>
            {t('user.table.status_types.unknown')}
          </Label>
        );
    }
  };

  const getTenantName = (tid) => {
    const t = tenantOptions.find((o) => o.value === tid);
    return t ? t.text : `租户 ${tid}`;
  };

  return (
    <>
      <Form onSubmit={searchUsers} style={{ marginBottom: '1rem' }}>
        <Form.Group>
          <Form.Field>
            <Dropdown
              selection
              options={[
                { key: 'independent', text: t('user.table.scope.independent', '独立用户'), value: 'independent' },
                { key: 'tenant', text: t('user.table.scope.tenant', '租户用户'), value: 'tenant' },
                { key: 'all', text: t('user.table.scope.all', '全部用户'), value: 'all' },
              ]}
              value={scope}
              onChange={(e, { value }) => setScope(value)}
            />
          </Form.Field>
          {scope === 'tenant' && (
            <Form.Field>
              <Dropdown
                selection
                clearable
                search
                placeholder={t('user.table.select_tenant', '选择租户')}
                options={tenantOptions}
                value={tenantId}
                onChange={(e, { value }) => setTenantId(value)}
              />
            </Form.Field>
          )}
          <Form.Field>
            <Dropdown
              multiple
              selection
              clearable
              placeholder={t('user.table.filter_role', '筛选角色')}
              options={[
                { key: 1, text: t('user.table.role_types.normal'), value: 1 },
                { key: 10, text: t('user.table.role_types.admin'), value: 10 },
                { key: 20, text: t('user.table.role_types.tenant_admin'), value: 20 },
                { key: 50, text: t('user.table.role_types.super_admin'), value: 50 },
                { key: 100, text: t('user.table.role_types.root'), value: 100 },
              ]}
              value={roles}
              onChange={(e, { value }) => setRoles(value)}
            />
          </Form.Field>
          <Form.Input
            icon='search'
            placeholder={t('user.search')}
            value={searchKeyword}
            loading={loading}
            onChange={handleKeywordChange}
          />
          <Form.Button primary type='submit'>
            {t('user.search')}
          </Form.Button>
        </Form.Group>
      </Form>

      <Table basic={'very'} compact size='small'>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell style={{ cursor: 'pointer' }} onClick={() => sortUser('id')}>
              {t('user.table.id')}
            </Table.HeaderCell>
            <Table.HeaderCell style={{ cursor: 'pointer' }} onClick={() => sortUser('username')}>
              {t('user.table.username')}
            </Table.HeaderCell>
            <Table.HeaderCell style={{ cursor: 'pointer' }} onClick={() => sortUser('group')}>
              {t('user.table.group')}
            </Table.HeaderCell>
            <Table.HeaderCell>
              {t('user.table.tenant', '租户')}
            </Table.HeaderCell>
            <Table.HeaderCell style={{ cursor: 'pointer' }} onClick={() => sortUser('quota')}>
              {t('user.table.quota')}
            </Table.HeaderCell>
            <Table.HeaderCell style={{ cursor: 'pointer' }} onClick={() => sortUser('role')}>
              {t('user.table.role_text')}
            </Table.HeaderCell>
            <Table.HeaderCell style={{ cursor: 'pointer' }} onClick={() => sortUser('status')}>
              {t('user.table.status_text')}
            </Table.HeaderCell>
            <Table.HeaderCell>{t('user.table.actions')}</Table.HeaderCell>
          </Table.Row>
        </Table.Header>

        <Table.Body>
          {users.map((user, idx) => (
            <Table.Row key={user.user_id}>
              <Table.Cell>{user.user_id}</Table.Cell>
              <Table.Cell>
                <Popup
                  content={user.email ? user.email : '未绑定邮箱地址'}
                  key={user.username}
                  header={user.display_name ? user.display_name : user.username}
                  trigger={<span>{renderText(user.username, 15)}</span>}
                  hoverable
                />
              </Table.Cell>
              <Table.Cell>{renderGroup(user.group)}</Table.Cell>
              <Table.Cell>
                {user.tenant_id ? (
                  <Label basic color="blue">
                    {getTenantName(user.tenant_id)}
                  </Label>
                ) : (
                  <span style={{ color: 'gray' }}>独立用户</span>
                )}
              </Table.Cell>
              <Table.Cell>
                <Popup
                  content={t('user.table.remaining_quota')}
                  trigger={<Label basic>{renderQuota(user.quota, t)}</Label>}
                />
                <Popup
                  content={t('user.table.used_quota')}
                  trigger={<Label basic>{renderQuota(user.used_quota, t)}</Label>}
                />
                <Popup
                  content={t('user.table.request_count')}
                  trigger={<Label basic>{renderNumber(user.request_count)}</Label>}
                />
              </Table.Cell>
              <Table.Cell>{renderRole(user.role, t)}</Table.Cell>
              <Table.Cell>{renderStatus(user.status)}</Table.Cell>
              <Table.Cell>
                <div>
                  <Popup
                    trigger={
                      <Button size='tiny' negative disabled={user.role >= 50}>
                        {t('user.buttons.delete')}
                      </Button>
                    }
                    on='click'
                    flowing
                    hoverable
                  >
                    <Button
                      negative
                      size={'tiny'}
                      onClick={() => manageUser(user.username, 'delete', idx)}
                    >
                      {t('user.buttons.delete_user')} {user.username}
                    </Button>
                  </Popup>
                  <Button
                    size={'tiny'}
                    onClick={() =>
                      manageUser(
                        user.username,
                        user.status === 1 ? 'disable' : 'enable',
                        idx
                      )
                    }
                    disabled={user.role >= 50}
                  >
                    {user.status === 1 ? t('user.buttons.disable') : t('user.buttons.enable')}
                  </Button>
                  <Button size={'tiny'} as={Link} to={'/user/edit/' + user.user_id}>
                    {t('user.buttons.edit')}
                  </Button>
                </div>
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>

        <Table.Footer>
          <Table.Row>
            <Table.HeaderCell colSpan='8'>
              <Button size='small' as={Link} to='/user/add' loading={loading}>
                {t('user.buttons.add')}
              </Button>
              <Dropdown
                placeholder={t('user.table.sort_by')}
                selection
                options={[
                  { key: '', text: t('user.table.sort.default'), value: '' },
                  { key: 'quota', text: t('user.table.sort.quota'), value: 'quota' },
                  { key: 'used_quota', text: t('user.table.sort.used_quota'), value: 'used_quota' },
                  { key: 'request_count', text: t('user.table.sort.request_count'), value: 'request_count' },
                ]}
                value={orderBy}
                onChange={handleOrderByChange}
                style={{ marginLeft: '10px' }}
              />
              <Pagination
                floated='right'
                activePage={activePage}
                onPageChange={onPaginationChange}
                size='small'
                siblingRange={1}
                totalPages={Math.ceil(totalCount / ITEMS_PER_PAGE)}
              />
            </Table.HeaderCell>
          </Table.Row>
        </Table.Footer>
      </Table>
    </>
  );
};

export default UsersTable;
