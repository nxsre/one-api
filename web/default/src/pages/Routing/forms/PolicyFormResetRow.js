import React from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Icon } from 'semantic-ui-react';

/** 表单内「恢复默认」：与页脚「保存此项」视觉区分，并提示仅本地重置 */
export default function PolicyFormResetRow({ disabled, onReset }) {
  const { t } = useTranslation();
  return (
    <div className='routing-policy-form-reset-row'>
      <Button
        type='button'
        size='small'
        basic
        icon
        labelPosition='left'
        className='routing-policy-btn-reset'
        disabled={disabled}
        onClick={onReset}
      >
        <Icon name='undo' />
        {t('routing.form_reset_defaults')}
      </Button>
      <span className='routing-policy-reset-hint'>
        {t('routing.form_reset_local_only_hint')}
      </span>
    </div>
  );
}
