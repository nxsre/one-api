import React, { useCallback, useEffect, useState } from 'react';
import { Button, Loader, Message } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';

const sameOriginNacosUiUrl = () => `${window.location.origin}/nacos-ui/`;

/**
 * 直接访问 /nacos/console 时：检测是否已打入控制台静态资源，并尝试新标签打开 /nacos-ui/（不在 one-api 内嵌 iframe）。
 */
const NacosConsoleExternalOpen = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [serving, setServing] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/nacos/registry/info');
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        setServing(false);
        return;
      }
      const ok = !!res.data.data?.native_console_serving;
      setServing(ok);
      if (ok) {
        window.open(sameOriginNacosUiUrl(), '_blank', 'noopener,noreferrer');
      }
    } catch (e) {
      showError(e.message);
      setServing(false);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const openTab = () => {
    window.open(sameOriginNacosUiUrl(), '_blank', 'noopener,noreferrer');
  };

  const uiUrl = sameOriginNacosUiUrl();

  if (loading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Loader active />
      </div>
    );
  }

  if (!serving) {
    return (
      <div style={{ padding: 16, maxWidth: 56 * 16 }}>
        <h3 style={{ marginTop: 0 }}>{t('nacos.native_console_title')}</h3>
        <Message warning>{t('nacos.native_console_unconfigured')}</Message>
        <p style={{ color: '#64748b', lineHeight: 1.6 }}>{t('nacos.native_console_hint')}</p>
      </div>
    );
  }

  return (
    <div style={{ padding: 16, maxWidth: 56 * 16 }}>
      <h3 style={{ marginTop: 0 }}>{t('nacos.native_console_title')}</h3>
      <Message positive>{t('nacos.native_console_tab_body')}</Message>
      <p>
        <code style={{ fontSize: 12, wordBreak: 'break-all' }}>{uiUrl}</code>
      </p>
      <Button type='button' primary onClick={openTab}>
        {t('nacos.native_console_open_again')}
      </Button>
    </div>
  );
};

export default NacosConsoleExternalOpen;
