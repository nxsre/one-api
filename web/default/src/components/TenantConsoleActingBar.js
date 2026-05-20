import React, { useEffect, useState } from 'react';
import { Dropdown, Message, Button, Modal, Form } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  isAdmin,
  isTenantAdmin,
  showError,
  showSuccess,
  getTenantConsoleActingTenantId,
  setTenantConsoleActingTenantId,
} from '../helpers';

export default function TenantConsoleActingBar() {
  const { t } = useTranslation();
  const [tenants, setTenants] = useState([]);
  const [loading, setLoading] = useState(true);
  const [value, setValue] = useState(() => getTenantConsoleActingTenantId());

  const loadTenants = async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const res = await API.get('/api/platform/tenants');
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('tenant_console.impersonate.load_failed'));
        return;
      }
      const list = Array.isArray(data) ? data : [];
      setTenants(list);
      let cur = getTenantConsoleActingTenantId();
      if (!cur && list.length > 0) {
        setTenantConsoleActingTenantId(String(list[0].id));
        cur = getTenantConsoleActingTenantId();
      }
      setValue(cur);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!isAdmin() || isTenantAdmin()) return undefined;
    loadTenants();
  }, [t]);

  if (!isAdmin() || isTenantAdmin()) return null;

  const options = tenants.map((row) => ({
    key: String(row.id),
    value: String(row.id),
    text: `${row.name} (#${row.id})`,
  }));

  const onChange = (_, { value: v }) => {
    setTenantConsoleActingTenantId(v);
    setValue(getTenantConsoleActingTenantId());
    window.location.reload();
  };

  return (
    <Message info style={{ marginBottom: '1rem' }}>
      <div
        style={{
          alignItems: 'center',
          display: 'flex',
          flexWrap: 'wrap',
          gap: '12px',
        }}
      >
        <span>{t('tenant_console.impersonate.banner')}</span>
        <Dropdown
          placeholder={t('tenant_console.impersonate.pick_tenant')}
          search
          selection
          loading={loading}
          options={options}
          value={value || ''}
          onChange={onChange}
          style={{ minWidth: 280 }}
        />
      </div>
      {!loading && tenants.length > 0 && !value ? (
        <p style={{ marginBottom: 0, marginTop: 8 }}>
          {t('tenant_console.impersonate.need_select')}
        </p>
      ) : null}
    </Message>
  );
}
