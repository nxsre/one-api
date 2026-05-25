import React from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Modal } from 'semantic-ui-react';
import './ChannelModelTestReportModal.css';

function formatBody(text) {
  const raw = String(text || '').trim();
  if (!raw) return '—';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function formatHeaders(headers) {
  if (!headers || typeof headers !== 'object') return '—';
  const lines = [];
  for (const [key, value] of Object.entries(headers)) {
    const vals = Array.isArray(value) ? value : [value];
    for (const v of vals) {
      lines.push(`${key}: ${v}`);
    }
  }
  return lines.length ? lines.join('\n') : '—';
}

function DetailBlock({ title, children }) {
  return (
    <div className='channel-model-test-detail__block'>
      <div className='channel-model-test-detail__block-title'>{title}</div>
      <pre className='channel-model-test-detail__pre'>{children}</pre>
    </div>
  );
}

export default function ChannelModelTestDetailModal({ open, onClose, row }) {
  const { t } = useTranslation();
  const detail = row?.detail;

  if (!detail) {
    return null;
  }

  const requestLine = [detail.request_method, detail.request_url || detail.request_path]
    .filter(Boolean)
    .join(' ');

  return (
    <Modal open={open} onClose={onClose} size='large' closeIcon>
      <Modal.Header>
        {t('channel.edit.test_report.detail_title', { model: row?.model || '—' })}
      </Modal.Header>
      <Modal.Content scrolling>
        <div className='channel-model-test-detail'>
          {detail.wire_protocol ? (
            <div className='channel-model-test-detail__meta'>
              {t('channel.edit.test_report.wire_protocol')}: {detail.wire_protocol}
            </div>
          ) : null}
          {requestLine ? (
            <div className='channel-model-test-detail__meta'>{requestLine}</div>
          ) : null}
          <DetailBlock title={t('channel.edit.test_report.request_headers')}>
            {formatHeaders(detail.request_headers)}
          </DetailBlock>
          <DetailBlock title={t('channel.edit.test_report.request_body')}>
            {formatBody(detail.request_body)}
          </DetailBlock>
          <DetailBlock title={t('channel.edit.test_report.response_status')}>
            {detail.response_status ? String(detail.response_status) : '—'}
          </DetailBlock>
          <DetailBlock title={t('channel.edit.test_report.response_headers')}>
            {formatHeaders(detail.response_headers)}
          </DetailBlock>
          <DetailBlock title={t('channel.edit.test_report.response_body')}>
            {formatBody(detail.response_body)}
          </DetailBlock>
        </div>
      </Modal.Content>
      <Modal.Actions>
        <Button onClick={onClose}>{t('channel.edit.test_report.close')}</Button>
      </Modal.Actions>
    </Modal>
  );
}
