import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, Modal, Table } from 'semantic-ui-react';
import { API, showError, showSuccess, timestamp2string, noAutofillFormProps, noAutofillPasswordProps, noAutofillTextProps, useNoAutofillUnlock } from '../../helpers';

/** Unix 秒 -> datetime-local 字符串（本地时区） */
function tsToDatetimeLocal(ts) {
  if (ts == null || Number.isNaN(Number(ts))) return '';
  const d = new Date(Number(ts) * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const TenantManagement = () => {
  const [tenants, setTenants] = useState([]);
  const createAutofillUnlock = useNoAutofillUnlock();

  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState(null);
  const [editInputs, setEditInputs] = useState({
    discount_ratio: 1.0,
    price_per_1k_api_call: -1.0,
  });

  const [billingRulesModalOpen, setBillingRulesModalOpen] = useState(false);
  const [billingRules, setBillingRules] = useState([]);
  const [billingRuleEditingId, setBillingRuleEditingId] = useState(null);
  const [billingChannelList, setBillingChannelList] = useState([]);
  const [billingRuleInputs, setBillingRuleInputs] = useState({
    channel_id: '0',
    rule_type: 1,
    value: '',
    start_time: '',
    end_time: '',
    status: 1,
  });

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [createInputs, setCreateInputs] = useState({
    name: '',
    slug: '',
    remark: '',
    admin_username: '',
    admin_password: '',
    admin_display_name: '',
  });

  const handleCreateChange = (e, { name, value }) => {
    setCreateInputs((prev) => ({ ...prev, [name]: value }));
  };

  const submitCreate = async () => {
    if (!createInputs.name || !createInputs.slug || !createInputs.admin_username || !createInputs.admin_password) {
      showError('名称、Slug、管理员账号及密码均不能为空');
      return;
    }
    setCreateSubmitting(true);
    try {
      const res = await API.post('/api/platform/tenants', createInputs);
      const { success, message } = res.data;
      if (success) {
        showSuccess('租户创建成功');
        setCreateModalOpen(false);
        setCreateInputs({
          name: '',
          slug: '',
          remark: '',
          admin_username: '',
          admin_password: '',
          admin_display_name: '',
        });
        await loadTenants();
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setCreateSubmitting(false);
    }
  };

  const loadTenants = async () => {
    const res = await API.get('/api/platform/tenants');
    const { success, message, data } = res.data;
    if (success) {
      setTenants(data || []);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    loadTenants();
  }, []);

  const openEditModal = (tenant) => {
    setEditingTenant(tenant);
    setEditInputs({
      discount_ratio: tenant.discount_ratio || 1.0,
      price_per_1k_api_call: tenant.price_per_1k_api_call !== undefined ? tenant.price_per_1k_api_call : -1.0,
    });
    setEditModalOpen(true);
  };

  const submitEdit = async () => {
    const res = await API.put(`/api/platform/tenants/${editingTenant.id}/billing`, {
      discount_ratio: parseFloat(editInputs.discount_ratio),
      price_per_1k_api_call: parseFloat(editInputs.price_per_1k_api_call),
    });
    if (res.data.success) {
      showSuccess('保存成功');
      setEditModalOpen(false);
      loadTenants();
    } else {
      showError(res.data.message);
    }
  };

  const loadBillingRules = async (tenantId) => {
    const res = await API.get(`/api/platform/tenants/${tenantId}/billing_rules`);
    if (res.data.success) {
      setBillingRules(res.data.data || []);
    }
  };

  useEffect(() => {
    if (!billingRulesModalOpen || !editingTenant) {
      return undefined;
    }
    let cancelled = false;
    (async () => {
      const tid = billingRuleInputs.rule_type === 2 ? editingTenant.id : null;
      const base = '/api/channel/?p=0&page_size=500';
      const url = tid != null ? `${base}&tenant_id=${tid}` : base;
      try {
        const res = await API.get(url);
        if (cancelled) return;
        if (res.data.success) {
          setBillingChannelList(res.data.data || []);
        }
      } catch {
        if (!cancelled) setBillingChannelList([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [billingRulesModalOpen, editingTenant, billingRuleInputs.rule_type]);

  const channelDropdownOptions = useMemo(() => {
    const all = [{ key: 'ch-0', text: '全部（全局）', value: '0' }];
    const fromApi = (billingChannelList || []).map((ch) => ({
      key: `ch-${ch.id}`,
      text: `${ch.id} · ${ch.name || ''}`,
      value: String(ch.id),
    }));
    const ids = new Set(fromApi.map((o) => o.value));
    const sel = String(billingRuleInputs.channel_id ?? '0');
    if (sel !== '0' && !ids.has(sel)) {
      fromApi.unshift({
        key: `ch-missing-${sel}`,
        text: `${sel}（当前规则，列表中未找到该渠道）`,
        value: sel,
      });
    }
    return [...all, ...fromApi];
  }, [billingChannelList, billingRuleInputs.channel_id]);

  const resetBillingRuleFormDraft = (ruleType) => ({
    channel_id: '0',
    rule_type: Number(ruleType),
    value: '',
    start_time: '',
    end_time: '',
    status: 1,
  });

  const openBillingRulesModal = async (tenant) => {
    setEditingTenant(tenant);
    setBillingRuleEditingId(null);
    setBillingRuleInputs(resetBillingRuleFormDraft(1));
    setBillingRulesModalOpen(true);
    await loadBillingRules(tenant.id);
  };

  const cancelBillingRuleEdit = () => {
    setBillingRuleEditingId(null);
    setBillingRuleInputs(resetBillingRuleFormDraft(billingRuleInputs.rule_type));
  };

  const startEditBillingRule = (r) => {
    setBillingRuleEditingId(r.id);
    setBillingRuleInputs({
      channel_id: String(r.channel_id ?? 0),
      rule_type: r.rule_type,
      value: r.value !== undefined && r.value !== null ? String(r.value) : '',
      start_time: tsToDatetimeLocal(r.start_time),
      end_time: tsToDatetimeLocal(r.end_time),
      status: r.status,
    });
  };

  const submitBillingRule = async () => {
    if (!billingRuleInputs.value || !billingRuleInputs.start_time || !billingRuleInputs.end_time) {
      showError('请填写完整信息');
      return;
    }
    const body = {
      channel_id: parseInt(billingRuleInputs.channel_id || 0, 10),
      rule_type: parseInt(billingRuleInputs.rule_type, 10),
      value: parseFloat(billingRuleInputs.value),
      start_time: Math.floor(new Date(billingRuleInputs.start_time).getTime() / 1000),
      end_time: Math.floor(new Date(billingRuleInputs.end_time).getTime() / 1000),
      status: parseInt(billingRuleInputs.status, 10),
    };
    let res;
    if (billingRuleEditingId != null) {
      res = await API.put(`/api/platform/tenants/${editingTenant.id}/billing_rules/${billingRuleEditingId}`, body);
    } else {
      res = await API.post(`/api/platform/tenants/${editingTenant.id}/billing_rules`, body);
    }
    if (res.data.success) {
      showSuccess(billingRuleEditingId != null ? '更新成功' : '添加成功');
      setBillingRuleEditingId(null);
      setBillingRuleInputs(resetBillingRuleFormDraft(billingRuleInputs.rule_type));
      loadBillingRules(editingTenant.id);
    } else {
      showError(res.data.message);
    }
  };

  const deleteBillingRule = async (ruleId) => {
    const res = await API.delete(`/api/platform/tenants/${editingTenant.id}/billing_rules/${ruleId}`);
    if (res.data.success) {
      showSuccess('删除成功');
      if (billingRuleEditingId === ruleId) {
        cancelBillingRuleEdit();
      }
      loadBillingRules(editingTenant.id);
    } else {
      showError(res.data.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>租户管理</Card.Header>
          <div style={{ marginTop: 10, marginBottom: 10 }}>
            <Button size='small' primary onClick={() => setCreateModalOpen(true)}>
              新建租户
            </Button>
          </div>
          <Table celled>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>ID</Table.HeaderCell>
                <Table.HeaderCell>企业名称</Table.HeaderCell>
                <Table.HeaderCell>标识 (Slug)</Table.HeaderCell>
                <Table.HeaderCell>全局折扣</Table.HeaderCell>
                <Table.HeaderCell>私有渠道千次调用单价</Table.HeaderCell>
                <Table.HeaderCell>创建时间</Table.HeaderCell>
                <Table.HeaderCell>操作</Table.HeaderCell>
              </Table.Row>
            </Table.Header>

            <Table.Body>
              {tenants.map((t) => (
                <Table.Row key={t.id}>
                  <Table.Cell>{t.id}</Table.Cell>
                  <Table.Cell>{t.name}</Table.Cell>
                  <Table.Cell>{t.slug}</Table.Cell>
                  <Table.Cell>{t.discount_ratio || 1.0}</Table.Cell>
                  <Table.Cell>{t.price_per_1k_api_call !== undefined && t.price_per_1k_api_call >= 0 ? t.price_per_1k_api_call : '默认'}</Table.Cell>
                  <Table.Cell>{timestamp2string(t.created_time)}</Table.Cell>
                  <Table.Cell>
                    <Button size='mini' onClick={() => openEditModal(t)}>
                      编辑兜底计费
                    </Button>
                    <Button size='mini' onClick={() => openBillingRulesModal(t)}>
                      时效规则
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        </Card.Content>
      </Card>

      <Modal open={createModalOpen} onClose={() => setCreateModalOpen(false)} size='small'>
        <Modal.Header>新建租户</Modal.Header>
        <Modal.Content>
          <Form loading={createSubmitting} {...noAutofillFormProps}>
            <Form.Input
              label='租户名称'
              required
              name='name'
              value={createInputs.name}
              onChange={handleCreateChange}
              placeholder='例如：某某企业'
            />
            <Form.Input
              label='标识 (Slug)'
              required
              name='slug'
              value={createInputs.slug}
              onChange={handleCreateChange}
              placeholder='例如：my-company'
            />
            <Form.Input
              label='备注'
              name='remark'
              value={createInputs.remark}
              onChange={handleCreateChange}
            />
            <Form.Group widths='equal'>
              <Form.Input
                label='租户管理员账号'
                required
                name='admin_username'
                value={createInputs.admin_username}
                onChange={handleCreateChange}
                placeholder='登录用户名'
                {...noAutofillTextProps}
                {...createAutofillUnlock}
              />
              <Form.Input
                label='管理员密码'
                required
                type='password'
                name='admin_password'
                value={createInputs.admin_password}
                onChange={handleCreateChange}
                {...noAutofillPasswordProps}
                {...createAutofillUnlock}
              />
            </Form.Group>
            <Form.Input
              label='管理员显示名'
              name='admin_display_name'
              value={createInputs.admin_display_name}
              onChange={handleCreateChange}
              placeholder='可选，默认同账号名'
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setCreateModalOpen(false)}>取消</Button>
          <Button primary onClick={submitCreate} loading={createSubmitting}>
            创建
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={editModalOpen} onClose={() => setEditModalOpen(false)} size='tiny'>
        <Modal.Header>编辑租户计费配置</Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Input
              label='全局折扣率 (如 0.8 表示 8 折)'
              type='number'
              step='0.1'
              value={editInputs.discount_ratio}
              onChange={(e, { value }) => setEditInputs({ ...editInputs, discount_ratio: value })}
            />
            <Form.Input
              label='私有渠道默认千次调用单价 (-1 表示使用系统默认配置)'
              type='number'
              step='0.001'
              value={editInputs.price_per_1k_api_call}
              onChange={(e, { value }) => setEditInputs({ ...editInputs, price_per_1k_api_call: value })}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setEditModalOpen(false)}>取消</Button>
          <Button primary onClick={submitEdit}>保存</Button>
        </Modal.Actions>
      </Modal>

      <Modal
        open={billingRulesModalOpen}
        onClose={() => {
          setBillingRulesModalOpen(false);
          setBillingRuleEditingId(null);
        }}
        size='large'
      >
        <Modal.Header>管理时效计费规则 - {editingTenant?.name}</Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Group widths='equal'>
              <Form.Dropdown
                label='规则类型'
                selection
                options={[
                  { key: 1, text: '平台渠道折扣率 (如0.8)', value: 1 },
                  { key: 2, text: '私有渠道千次调用单价', value: 2 },
                ]}
                value={billingRuleInputs.rule_type}
                onChange={(e, { value }) => {
                  setBillingRuleInputs({
                    ...resetBillingRuleFormDraft(value),
                  });
                  setBillingRuleEditingId(null);
                }}
              />
              <Form.Dropdown
                label='渠道（可搜索；选「全部」表示全局）'
                placeholder='选择或输入搜索'
                fluid
                selection
                search
                options={channelDropdownOptions}
                value={String(billingRuleInputs.channel_id)}
                onChange={(e, { value }) =>
                  setBillingRuleInputs({ ...billingRuleInputs, channel_id: String(value) })}
              />
              <Form.Input
                label='设定值'
                type='number'
                step='0.01'
                value={billingRuleInputs.value}
                onChange={(e, { value }) => setBillingRuleInputs({ ...billingRuleInputs, value: value })}
              />
            </Form.Group>
            <Form.Group widths='equal'>
              <Form.Input
                label='生效时间'
                type='datetime-local'
                value={billingRuleInputs.start_time}
                onChange={(e, { value }) => setBillingRuleInputs({ ...billingRuleInputs, start_time: value })}
              />
              <Form.Input
                label='失效时间'
                type='datetime-local'
                value={billingRuleInputs.end_time}
                onChange={(e, { value }) => setBillingRuleInputs({ ...billingRuleInputs, end_time: value })}
              />
              <Form.Dropdown
                label='状态'
                selection
                options={[
                  { key: 'st-1', text: '启用', value: 1 },
                  { key: 'st-2', text: '禁用', value: 2 },
                ]}
                value={billingRuleInputs.status}
                onChange={(e, { value }) =>
                  setBillingRuleInputs({ ...billingRuleInputs, status: value })}
              />
              <Form.Field>
                <label>&nbsp;</label>
                <Button.Group>
                  <Button primary onClick={submitBillingRule}>
                    {billingRuleEditingId != null ? '保存修改' : '添加'}
                  </Button>
                  {billingRuleEditingId != null && (
                    <Button type='button' onClick={cancelBillingRuleEdit}>
                      取消编辑
                    </Button>
                  )}
                </Button.Group>
              </Form.Field>
            </Form.Group>
          </Form>
          <Table celled>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>ID</Table.HeaderCell>
                <Table.HeaderCell>类型</Table.HeaderCell>
                <Table.HeaderCell>渠道</Table.HeaderCell>
                <Table.HeaderCell>数值</Table.HeaderCell>
                <Table.HeaderCell>生效时间</Table.HeaderCell>
                <Table.HeaderCell>失效时间</Table.HeaderCell>
                <Table.HeaderCell>状态</Table.HeaderCell>
                <Table.HeaderCell>操作</Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {billingRules.map((r) => (
                <Table.Row key={r.id}>
                  <Table.Cell>{r.id}</Table.Cell>
                  <Table.Cell>{r.rule_type === 1 ? '折扣率' : '单价'}</Table.Cell>
                  <Table.Cell>{r.channel_id === 0 ? '全局' : r.channel_id}</Table.Cell>
                  <Table.Cell>{r.value}</Table.Cell>
                  <Table.Cell>{timestamp2string(r.start_time)}</Table.Cell>
                  <Table.Cell>{timestamp2string(r.end_time)}</Table.Cell>
                  <Table.Cell>{r.status === 1 ? '启用' : '禁用'}</Table.Cell>
                  <Table.Cell>
                    <Button size='mini' onClick={() => startEditBillingRule(r)}>
                      编辑
                    </Button>
                    <Button size='mini' negative onClick={() => deleteBillingRule(r.id)}>
                      删除
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        </Modal.Content>
      </Modal>
    </div>
  );
};

export default TenantManagement;
