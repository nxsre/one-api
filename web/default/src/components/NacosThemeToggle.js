import React, { useCallback, useEffect, useState } from 'react';
import { Button, Icon } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import {
  getNacosTheme,
  toggleNacosTheme,
} from '../helpers/nacosTheme';

/**
 * 与 Nacos 原生控制台共用 localStorage `nacos_theme`，并监听其它标签页 storage 事件。
 */
export default function NacosThemeToggle({ compact }) {
  const { t } = useTranslation();
  const [dark, setDark] = useState(() => getNacosTheme() === 'dark');

  useEffect(() => {
    const sync = () => setDark(getNacosTheme() === 'dark');
    window.addEventListener('storage', sync);
    window.addEventListener('one-api:nacos-theme-changed', sync);
    return () => {
      window.removeEventListener('storage', sync);
      window.removeEventListener('one-api:nacos-theme-changed', sync);
    };
  }, []);

  const onClick = useCallback(() => {
    const next = toggleNacosTheme();
    setDark(next === 'dark');
  }, []);

  return (
    <Button
      type='button'
      basic
      icon
      size={compact ? 'mini' : 'small'}
      className='app-theme-toggle'
      onClick={onClick}
      title={
        dark
          ? t('header.theme_switch_to_light')
          : t('header.theme_switch_to_dark')
      }
      aria-label={
        dark
          ? t('header.theme_switch_to_light')
          : t('header.theme_switch_to_dark')
      }
    >
      <Icon name={dark ? 'sun outline' : 'moon outline'} />
    </Button>
  );
}
