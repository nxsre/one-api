import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Input,
  Loader,
  Message,
  Table,
} from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const roleLabel = (role) => {
  if (role >= 100) return 'Root';
  if (role >= 10) return 'Admin';
  return 'User';
};

const NacosPermissions = () => {
  const { t, i18n } = useTranslation();
  const [permCatalog, setPermCatalog] = useState([]);
  const [rules, setRules] = useState({});
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);

  const [keyword, setKeyword] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [selectedIds, setSelectedIds] = useState(() => new Set());
  const [aclSource, setAclSource] = useState(null);

  const normalizeCatalogItem = (raw) => {
    if (!raw || typeof raw !== 'object') return null;
    const key = raw.key ?? raw.Key ?? '';
    if (!key) return null;
    return {
      key,
      label_zh: raw.label_zh ?? raw.LabelZh ?? key,
      label_en: raw.label_en ?? raw.LabelEn ?? key,
    };
  };

  const loadInfo = async () => {
    const ir = await API.get('/api/nacos/registry/info');
    if (!ir.data?.success || !ir.data.data) {
      return;
    }
    const d = ir.data.data;
    if (Array.isArray(d.permission_catalog) && d.permission_catalog.length) {
      const list = d.permission_catalog
        .map(normalizeCatalogItem)
        .filter(Boolean);
      setPermCatalog(list);
    } else if (Array.isArray(d.permission_keys)) {
      setPermCatalog(
        d.permission_keys.map((k) => ({
          key: k,
          label_zh: k,
          label_en: k,
        }))
      );
    }
  };

  const searchUsers = async () => {
    const q = keyword.trim();
    if (q.length < 1) {
      showError(t('nacos.perm_search_need_keyword'));
      return;
    }
    setSearching(true);
    try {
      const res = await API.get('/api/user/search', { params: { keyword: q } });
      if (!res.data?.success) {
        showError(res.data?.message || 'search failed');
        return;
      }
      const list = Array.isArray(res.data.data) ? res.data.data : [];
      setSearchResults(list);
      setSelectedIds(new Set());
      setAclSource(null);
      if (list.length === 0) {
        showError(t('nacos.perm_no_results'));
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setSearching(false);
    }
  };

  const toggleUser = (id) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const selectAllInResults = () => {
    const next = new Set(searchResults.map((u) => u.user_id));
    setSelectedIds(next);
  };

  const clearSelection = () => {
    setSelectedIds(new Set());
  };

  const loadAcl = async () => {
    if (selectedIds.size !== 1) {
      showError(t('nacos.perm_select_one_load'));
      return;
    }
    const id = [...selectedIds][0];
    setLoading(true);
    try {
      const res = await API.get(`/api/nacos/users/${id}/acl`);
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      setRules(res.data.data?.rules || {});
      const u = searchResults.find((x) => x.user_id === id);
      setAclSource({
        id,
        username: u?.username || String(id),
      });
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  const saveAcl = async () => {
    if (selectedIds.size < 1) {
      showError(t('nacos.perm_select_users_save'));
      return;
    }
    setLoading(true);
    const ids = [...selectedIds];
    let ok = 0;
    try {
      for (const id of ids) {
        const res = await API.put(`/api/nacos/users/${id}/acl`, { rules });
        if (!res.data?.success) {
          showError(
            res.data?.message ||
              `save failed for user #${id}`
          );
          return;
        }
        ok += 1;
      }
      showSuccess(t('nacos.perm_batch_saved', { count: ok }));
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadInfo();
  }, []);

  const toggle = (k) => (e, { checked }) => {
    setRules((prev) => ({ ...prev, [k]: !!checked }));
  };

  const selectAllPermissions = () => {
    setRules((prev) => {
      const next = { ...prev };
      permCatalog.forEach((item) => {
        next[item.key] = true;
      });
      return next;
    });
  };

  const clearAllPermissions = () => {
    setRules((prev) => {
      const next = { ...prev };
      permCatalog.forEach((item) => {
        next[item.key] = false;
      });
      return next;
    });
  };

  const useZh =
    i18n.language && i18n.language.toLowerCase().startsWith('zh');

  const selectedCount = selectedIds.size;

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('nacos.perm_title')}
          </Card.Header>
          <Message warning>{t('nacos.perm_hint')}</Message>

          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              gap: '8px 10px',
              marginBottom: 16,
            }}
          >
            <Input
              placeholder={t('nacos.perm_search_placeholder')}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  searchUsers();
                }
              }}
              style={{
                minWidth: 200,
                flex: '1 1 220px',
                marginBottom: 0,
              }}
            />
            <Button
              type='button'
              primary
              loading={searching}
              disabled={searching}
              onClick={searchUsers}
            >
              {t('nacos.perm_search_btn')}
            </Button>
            <Button
              size='small'
              type='button'
              onClick={selectAllInResults}
              disabled={!searchResults.length}
            >
              {t('nacos.perm_select_all')}
            </Button>
            <Button
              size='small'
              type='button'
              onClick={clearSelection}
              disabled={selectedCount === 0}
            >
              {t('nacos.perm_select_none')}
            </Button>
            <span style={{ color: '#666', fontSize: '0.95em' }}>
              {t('nacos.perm_selected_count', { count: selectedCount })}
            </span>
          </div>

          {searchResults.length > 0 ? (
            <>
              <Table celled compact size='small'>
                <Table.Header>
                  <Table.Row>
                    <Table.HeaderCell collapsing textAlign='center'>
                      {t('nacos.perm_col_select')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>{t('nacos.perm_col_id')}</Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.perm_col_username')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.perm_col_display')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.perm_col_email')}
                    </Table.HeaderCell>
                    <Table.HeaderCell>
                      {t('nacos.perm_col_role')}
                    </Table.HeaderCell>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {searchResults.map((u) => (
                    <Table.Row key={u.user_id} active={selectedIds.has(u.user_id)}>
                      <Table.Cell collapsing textAlign='center'>
                        <Checkbox
                          checked={selectedIds.has(u.user_id)}
                          onChange={() => toggleUser(u.user_id)}
                        />
                      </Table.Cell>
                      <Table.Cell>{u.user_id}</Table.Cell>
                      <Table.Cell>{u.username}</Table.Cell>
                      <Table.Cell>{u.display_name || '—'}</Table.Cell>
                      <Table.Cell>{u.email || '—'}</Table.Cell>
                      <Table.Cell>{roleLabel(u.role)}</Table.Cell>
                    </Table.Row>
                  ))}
                </Table.Body>
              </Table>
            </>
          ) : null}

          <div style={{ marginTop: 16, marginBottom: 8 }}>
            <Button type='button' onClick={loadAcl} disabled={loading || selectedCount !== 1}>
              {t('nacos.load_acl')}
            </Button>
            <Button type='button' primary onClick={saveAcl} disabled={loading || selectedCount < 1}>
              {t('nacos.save_acl')}
            </Button>
          </div>

          {aclSource ? (
            <Message info size='small'>
              {t('nacos.perm_load_source', {
                id: aclSource.id,
                name: aclSource.username,
              })}
            </Message>
          ) : null}

          {loading ? <Loader active inline='centered' /> : null}

          <div style={{ marginTop: 16 }}>
            {permCatalog.length === 0 ? (
              <Message size='small'>
                {t('nacos.perm_catalog_empty')}
              </Message>
            ) : (
              <>
                <div
                  style={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    alignItems: 'center',
                    gap: '8px 12px',
                    marginBottom: 12,
                  }}
                >
                  <span style={{ fontWeight: 600, marginRight: 4 }}>
                    {t('nacos.perm_rules_heading')}
                  </span>
                  <Button size='small' type='button' onClick={selectAllPermissions}>
                    {t('nacos.perm_rules_select_all')}
                  </Button>
                  <Button size='small' type='button' onClick={clearAllPermissions}>
                    {t('nacos.perm_rules_clear_all')}
                  </Button>
                </div>
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns:
                      'repeat(auto-fill, minmax(min(100%, 260px), 1fr))',
                    gap: '12px 16px',
                    alignItems: 'stretch',
                  }}
                >
                  {permCatalog.map((item) => {
                    const desc = useZh ? item.label_zh : item.label_en;
                    const k = item.key;
                    return (
                      <div
                        key={k}
                        style={{
                          display: 'flex',
                          alignItems: 'flex-start',
                          gap: 10,
                          padding: '10px 12px',
                          border: '1px solid rgba(34,36,38,.12)',
                          borderRadius: 6,
                          background: 'rgba(0,0,0,.02)',
                          minWidth: 0,
                        }}
                      >
                        <Checkbox
                          checked={!!rules[k]}
                          onChange={toggle(k)}
                          style={{ flexShrink: 0, paddingTop: 2 }}
                        />
                        <div
                          role='presentation'
                          style={{ cursor: 'pointer', flex: 1, minWidth: 0 }}
                          onClick={() =>
                            setRules((prev) => ({ ...prev, [k]: !prev[k] }))
                          }
                        >
                          <div style={{ fontWeight: 500, lineHeight: 1.35 }}>
                            {desc}
                          </div>
                          <div
                            style={{
                              fontSize: '0.85em',
                              color: '#767676',
                              marginTop: 4,
                              wordBreak: 'break-word',
                              overflowWrap: 'anywhere',
                            }}
                          >
                            {k}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </>
            )}
          </div>
        </Card.Content>
      </Card>
    </div>
  );
};

export default NacosPermissions;
