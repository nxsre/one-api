import React from 'react';
import { Icon, Label } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import './PricingEntryBlockPanel.css';

export default function PricingEntryBlockPanel({
  title,
  help,
  expanded,
  onToggle,
  entryCount,
  children,
}) {
  const { t } = useTranslation();
  const panelId = `pricing-block-${String(title).replace(/\s+/g, '-')}`;

  return (
    <div className={`pricing-entry-block-panel${expanded ? ' is-expanded' : ''}`}>
      <button
        type='button'
        className='pricing-entry-block-panel__header'
        aria-expanded={expanded}
        aria-controls={panelId}
        onClick={onToggle}
      >
        <Icon name={expanded ? 'chevron down' : 'chevron right'} />
        <span className='pricing-entry-block-panel__title'>{title}</span>
        {help ? (
          <span
            className='pricing-entry-block-panel__help'
            onClick={(e) => e.stopPropagation()}
          >
            {help}
          </span>
        ) : null}
        {entryCount != null ? (
          <Label size='tiny' basic className='pricing-entry-block-panel__count'>
            {t('setting.operation.ratio.editor.block_entry_count', {
              count: entryCount,
              defaultValue: `${entryCount} 条`,
            })}
          </Label>
        ) : null}
      </button>
      {expanded ? (
        <div id={panelId} className='pricing-entry-block-panel__body'>
          {children}
        </div>
      ) : null}
    </div>
  );
}
