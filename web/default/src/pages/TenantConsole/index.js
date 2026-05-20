import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Button, Card, Header, Message } from 'semantic-ui-react';
import { API, showError } from '../../helpers';

export default function TenantConsoleHome() {
  const { t } = useTranslation();
  const [tenantMeta, setTenantMeta] = useState(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/tenant_console/meta/tenant');
        const { success, data, message } = res.data || {};
        if (!success || !data) {
          if (!cancelled && message) showError(message);
          return;
        }
        if (!cancelled) setTenantMeta(data);
      } catch (e) {
        if (!cancelled) showError(e.message);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('tenant_console.home.title')}
          </Card.Header>
          {tenantMeta ? (
            <Message info>
              <Message.Header>租户 ID（子账号登录「租户登录」页时请填写此项）</Message.Header>
              <p style={{ marginTop: 8, marginBottom: 0 }}>
                <strong>{tenantMeta.tenant_id}</strong>
                {tenantMeta.name ? ` — ${tenantMeta.name}` : ''}
                {tenantMeta.slug ? ` (${tenantMeta.slug})` : ''}
              </p>
            </Message>
          ) : null}
          <Header as='h4' dividing>
            {t('tenant_console.home.subtitle')}
          </Header>
          <Button.Group>
            <Button as={Link} to='/tenant-console/users' primary>
              {t('header.tenant_subusers')}
            </Button>
            <Button as={Link} to='/tenant-console/channels'>
              {t('header.tenant_channels')}
            </Button>
          </Button.Group>
        </Card.Content>
      </Card>
    </div>
  );
}
