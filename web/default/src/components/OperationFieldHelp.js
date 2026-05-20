import React from 'react';
import { useTranslation } from 'react-i18next';
import { Icon, Popup } from 'semantic-ui-react';
import './OperationFieldHelp.css';

export function FieldLabelWithHelp({ label, helpContent }) {
  return (
    <label className='operation-field-label-with-help'>
      <span>{label}</span>
      <Popup
        wide
        hoverable
        position='top left'
        className='operation-billing-help-popup-wrap'
        trigger={
          <span
            className='operation-field-help-trigger'
            role='button'
            tabIndex={0}
            onClick={(e) => e.preventDefault()}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') e.preventDefault();
            }}
          >
            <Icon
              name='question circle outline'
              className='operation-field-help-icon'
              aria-label='help'
            />
          </span>
        }
        content={
          <div className='operation-billing-help-popup'>{helpContent}</div>
        }
      />
    </label>
  );
}

function HelpSection({ title, children }) {
  return (
    <div className='operation-billing-help-section'>
      {title ? (
        <div className='operation-billing-help-section-title'>{title}</div>
      ) : null}
      {children}
    </div>
  );
}

function HelpCode({ children }) {
  return <pre className='operation-billing-help-code'>{children}</pre>;
}

export function BillingModeHelpContent() {
  const { t } = useTranslation();
  return (
    <>
      <HelpSection title={t('setting.operation.ratio.billing_mode.help.intro_title')}>
        <p>{t('setting.operation.ratio.billing_mode.help.intro')}</p>
      </HelpSection>
      <HelpSection title={t('setting.operation.ratio.billing_mode.help.values_title')}>
        <ul>
          <li>
            <strong>ratio</strong> — {t('setting.operation.ratio.billing_mode.help.ratio_desc')}
          </li>
          <li>
            <strong>tiered_expr</strong> —{' '}
            {t('setting.operation.ratio.billing_mode.help.tiered_desc')}
          </li>
        </ul>
      </HelpSection>
      <HelpSection title={t('setting.operation.ratio.billing_mode.help.example_title')}>
        <HelpCode>{`{
  "claude-sonnet-4-6": "tiered_expr",
  "gpt-4o": "ratio"
}`}</HelpCode>
        <p>{t('setting.operation.ratio.billing_mode.help.example_note')}</p>
      </HelpSection>
    </>
  );
}

export function BillingExprHelpContent() {
  const { t } = useTranslation();
  return (
    <>
      <HelpSection title={t('setting.operation.ratio.billing_expr.help.intro_title')}>
        <p>{t('setting.operation.ratio.billing_expr.help.intro')}</p>
      </HelpSection>
      <HelpSection title={t('setting.operation.ratio.billing_expr.help.vars_title')}>
        <table className='operation-billing-help-vars'>
          <thead>
            <tr>
              <th>{t('setting.operation.ratio.billing_expr.help.var_col')}</th>
              <th>{t('setting.operation.ratio.billing_expr.help.meaning_col')}</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>
                <code>p</code> / <code>c</code>
              </td>
              <td>{t('setting.operation.ratio.billing_expr.help.var_pc')}</td>
            </tr>
            <tr>
              <td>
                <code>len</code>
              </td>
              <td>{t('setting.operation.ratio.billing_expr.help.var_len')}</td>
            </tr>
            <tr>
              <td>
                <code>cr</code> / <code>cc</code> / <code>cc1h</code>
              </td>
              <td>{t('setting.operation.ratio.billing_expr.help.var_cache')}</td>
            </tr>
            <tr>
              <td>
                <code>img</code> / <code>ai</code> / <code>img_o</code> / <code>ao</code>
              </td>
              <td>{t('setting.operation.ratio.billing_expr.help.var_media')}</td>
            </tr>
          </tbody>
        </table>
        <p>{t('setting.operation.ratio.billing_expr.help.vars_note')}</p>
      </HelpSection>
      <HelpSection title={t('setting.operation.ratio.billing_expr.help.tier_title')}>
        <p>{t('setting.operation.ratio.billing_expr.help.tier_desc')}</p>
        <HelpCode>{`tier("tier_name", p * 3 + c * 15 + cr * 0.3)`}</HelpCode>
        <p>{t('setting.operation.ratio.billing_expr.help.price_unit')}</p>
      </HelpSection>
      <HelpSection title={t('setting.operation.ratio.billing_expr.help.examples_title')}>
        <p>{t('setting.operation.ratio.billing_expr.help.ex1_label')}</p>
        <HelpCode>{`{
  "claude-sonnet-4-6": "tier(\\"base\\", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6)"
}`}</HelpCode>
        <p>{t('setting.operation.ratio.billing_expr.help.ex2_label')}</p>
        <HelpCode>{`{
  "claude-sonnet-4-6": "len <= 200000 ? tier(\\"standard\\", p * 1.5 + c * 7.5) : tier(\\"long_context\\", p * 3.0 + c * 11.25)"
}`}</HelpCode>
      </HelpSection>
      <HelpSection>
        <p>{t('setting.operation.ratio.billing_expr.help.relation')}</p>
      </HelpSection>
    </>
  );
}
