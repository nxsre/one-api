import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Header } from 'semantic-ui-react';
import {
  API,
  showError,
  showSuccess,
  noAutofillFormProps,
  noAutofillPasswordProps,
  noAutofillTextProps,
  useNoAutofillUnlock,
} from '../../helpers';
import { renderQuotaWithPrompt } from '../../helpers/render';

export default function TenantEditUser() {
  const { t } = useTranslation();
  const { userId } = useParams();
  const isEdit = Boolean(userId);
  const navigate = useNavigate();
  const [loading, setLoading] = useState(isEdit);
  const [groupOptions, setGroupOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [channelOptions, setChannelOptions] = useState([]);
  const autofillUnlock = useNoAutofillUnlock();
  const [inputs, setInputs] = useState({
    user_id: '',
    username: '',
    display_name: '',
    password: '',
    email: '',
    group: 'default',
    quota: 0,
    status: 1,
    role: 1,
    allowed_models: [],
    allowed_channel_ids: [],
    tenant_permissions: [],
  });

  const handleChange = (e, { name, value }) => {
    if (name === 'quota') {
      const trimmed = String(value).trim();
      const n = trimmed === '' ? 0 : parseInt(trimmed, 10);
      setInputs((prev) => ({
        ...prev,
        quota: Number.isNaN(n) ? prev.quota : n,
      }));
      return;
    }
    setInputs((prev) => ({ ...prev, [name]: value }));
  };

  const loadMeta = async () => {
    try {
      const [gr, mo] = await Promise.all([
        API.get('/api/tenant_console/meta/groups'),
        API.get('/api/tenant_console/meta/all_models'),
      ]);
      if (gr.data?.success && Array.isArray(gr.data.data)) {
        setGroupOptions(
          gr.data.data.map((g) => ({ key: g, text: g, value: g }))
        );
      }
      if (mo.data?.success && Array.isArray(mo.data.data)) {
        setModelOptions(
          mo.data.data.map((m) => ({
            key: m.id,
            text: m.id,
            value: m.id,
          }))
        );
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const loadChannelsForAcl = async (group) => {
    const g = (group && String(group).trim()) || 'default';
    try {
      const res = await API.get(
        `/api/tenant_console/meta/channels_for_acl?group=${encodeURIComponent(g)}`
      );
      if (res.data?.success && Array.isArray(res.data.data)) {
        setChannelOptions(
          res.data.data.map((c) => ({
            key: c.id,
            text: `#${c.id} ${c.name || ''}${
              c.tenant_id == null
                ? ` (${t('tenant_console.edit_user.channel_platform_badge')})`
                : ''
            }`,
            value: c.id,
          }))
        );
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const loadProfile = async () => {
    if (!isEdit) return;
    setLoading(true);
    try {
      const res = await API.get(
        `/api/tenant_console/users/${encodeURIComponent(userId)}/profile`
      );
      const { success, message, data } = res.data;
      if (!success || !data) {
        showError(message || 'load failed');
        return;
      }
      setInputs({
        user_id: data.user_id || '',
        username: data.username || '',
        display_name: data.display_name || '',
        password: '',
        email: data.email || '',
        group: data.group || 'default',
        quota: data.quota ?? 0,
        status: data.status ?? 1,
        role: data.role ?? 1,
        allowed_models: Array.isArray(data.allowed_models)
          ? data.allowed_models
          : [],
        allowed_channel_ids: Array.isArray(data.allowed_channel_ids)
          ? data.allowed_channel_ids
          : [],
        tenant_permissions: Array.isArray(data.tenant_permissions)
          ? data.tenant_permissions
          : [],
      });
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadMeta().then();
    loadProfile().then();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId]);

  useEffect(() => {
    loadChannelsForAcl(inputs.group).then();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inputs.group]);

  const submit = async () => {
    if (!isEdit) {
      if (!inputs.username?.trim() || !inputs.password) {
        showError(t('tenant_console.edit_user.need_credentials'));
        return;
      }
      try {
        const res = await API.post('/api/tenant_console/users', {
          username: inputs.username.trim(),
          password: inputs.password,
          display_name: inputs.display_name?.trim() || inputs.username.trim(),
          group: inputs.group || 'default',
          quota: inputs.quota,
          role: inputs.role,
          email: inputs.email?.trim() || '',
          allowed_models: inputs.allowed_models || [],
          allowed_channel_ids: inputs.allowed_channel_ids || [],
          tenant_permissions: inputs.tenant_permissions || [],
        });
        const { success, message } = res.data;
        if (!success) {
          showError(message);
          return;
        }
        showSuccess(t('tenant_console.edit_user.create_success'));
        navigate('/tenant-console/users');
      } catch (e) {
        showError(e.message);
      }
      return;
    }

    try {
      const res = await API.put('/api/tenant_console/users', {
        user_id: inputs.user_id,
        username: inputs.username?.trim(),
        display_name: inputs.display_name?.trim(),
        password: inputs.password,
        email: inputs.email?.trim() || '',
        group: inputs.group || 'default',
        quota: inputs.quota,
        status: inputs.status,
        role: inputs.role,
        allowed_models: inputs.allowed_models || [],
        allowed_channel_ids: inputs.allowed_channel_ids || [],
        tenant_permissions: inputs.tenant_permissions || [],
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('tenant_console.edit_user.update_success'));
      navigate('/tenant-console/users');
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {isEdit
              ? t('tenant_console.edit_user.title_edit')
              : t('tenant_console.edit_user.title_create')}
          </Card.Header>
          <Header as='h4' dividing>
            {t('tenant_console.edit_user.section_basic')}
          </Header>
          <Form loading={loading} {...noAutofillFormProps}>
            <Form.Field>
              <Form.Input
                label={t('tenant_console.edit_user.username')}
                name='username'
                value={inputs.username}
                onChange={handleChange}
                {...noAutofillTextProps}
                {...autofillUnlock}
                required={!isEdit}
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('tenant_console.edit_user.password')}
                name='password'
                type='password'
                value={inputs.password}
                onChange={handleChange}
                {...noAutofillPasswordProps}
                {...autofillUnlock}
                placeholder={
                  isEdit
                    ? t('tenant_console.edit_user.password_unchanged_hint')
                    : ''
                }
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('tenant_console.edit_user.display_name')}
                name='display_name'
                value={inputs.display_name}
                onChange={handleChange}
                {...noAutofillTextProps}
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('tenant_console.edit_user.email')}
                name='email'
                value={inputs.email}
                onChange={handleChange}
                {...noAutofillTextProps}
              />
            </Form.Field>
            <Form.Field>
              <Form.Dropdown
                label={t('tenant_console.edit_user.group')}
                name='group'
                fluid
                search
                selection
                allowAdditions
                additionLabel={t('tenant_console.edit_user.group_add')}
                options={groupOptions}
                value={inputs.group}
                onChange={handleChange}
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={`${t('tenant_console.edit_user.quota')}${renderQuotaWithPrompt(
                  inputs.quota,
                  t
                )}`}
                name='quota'
                value={
                  inputs.quota === undefined || inputs.quota === null
                    ? ''
                    : String(inputs.quota)
                }
                onChange={handleChange}
              />
            </Form.Field>
            {isEdit ? (
              <Form.Field>
                <Form.Dropdown
                  label={t('tenant_console.edit_user.status')}
                  name='status'
                  fluid
                  selection
                  options={[
                    {
                      key: 1,
                      value: 1,
                      text: t('tenant_console.edit_user.status_enabled'),
                    },
                    {
                      key: 2,
                      value: 2,
                      text: t('tenant_console.edit_user.status_disabled'),
                    },
                  ]}
                  value={inputs.status}
                  onChange={handleChange}
                />
              </Form.Field>
            ) : null}

            <Form.Field>
              <Form.Dropdown
                label='租户权限角色'
                name='role'
                fluid
                selection
                options={[
                  { key: 1, value: 1, text: '普通子账号 (Common)' },
                  { key: 20, value: 20, text: '租户管理员 (Tenant Admin)' },
                ]}
                value={inputs.role}
                onChange={handleChange}
              />
            </Form.Field>
            {inputs.role === 1 ? (
              <Form.Field>
                <Form.Dropdown
                  label='普通子账号额外管理权限'
                  name='tenant_permissions'
                  fluid
                  multiple
                  selection
                  options={[
                    { key: 'manage_users', value: 'manage_users', text: '用户管理 (manage_users)' },
                    { key: 'manage_channels', value: 'manage_channels', text: '渠道管理 (manage_channels)' },
                    { key: 'manage_tokens', value: 'manage_tokens', text: '令牌管理 (manage_tokens)' },
                    { key: 'manage_billing', value: 'manage_billing', text: '账单与报表 (manage_billing)' },
                    { key: 'manage_nacos', value: 'manage_nacos', text: 'Nacos 注册与配置 (manage_nacos)' },
                  ]}
                  value={inputs.tenant_permissions}
                  onChange={handleChange}
                  placeholder='(可选) 授予普通子账号特定的租户控制台管理能力'
                />
              </Form.Field>
            ) : null}

            <Header as='h4' dividing>
              {t('tenant_console.edit_user.section_acl')}
            </Header>
            <p style={{ opacity: 0.85, marginBottom: 12 }}>
              {t('tenant_console.edit_user.acl_hint')}
            </p>
            <Form.Field>
              <Form.Dropdown
                label={t('tenant_console.edit_user.allowed_models')}
                name='allowed_models'
                fluid
                multiple
                search
                selection
                options={modelOptions}
                value={inputs.allowed_models}
                onChange={handleChange}
                placeholder={t('tenant_console.edit_user.allowed_models_ph')}
              />
            </Form.Field>
            <Form.Field>
              <Form.Dropdown
                label={t('tenant_console.edit_user.allowed_channels')}
                name='allowed_channel_ids'
                fluid
                multiple
                search
                selection
                options={channelOptions}
                value={inputs.allowed_channel_ids}
                onChange={handleChange}
                placeholder={t('tenant_console.edit_user.allowed_channels_ph')}
              />
            </Form.Field>

            <Button type='button' onClick={() => navigate('/tenant-console/users')}>
              {t('tenant_console.edit_user.cancel')}
            </Button>
            <Button positive type='button' onClick={() => submit()}>
              {t('tenant_console.edit_user.submit')}
            </Button>
          </Form>
        </Card.Content>
      </Card>
    </div>
  );
}
