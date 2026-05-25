import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Modal, Table } from 'semantic-ui-react';
import {
  summarizeModelTestReport,
} from '../../helpers/channelModelTest';
import ChannelModelTestDetailModal from './ChannelModelTestDetailModal';
import './ChannelModelTestReportModal.css';

const FILTER_ITEMS = [
  { key: 'all', tone: 'all', countProp: '', labelKey: 'filter_all' },
  { key: 'done', tone: 'all', countProp: 'total', labelKey: 'count_done' },
  { key: 'ok', tone: 'ok', countProp: 'ok', labelKey: 'count_ok' },
  { key: 'fail', tone: 'fail', countProp: 'fail', labelKey: 'count_fail' },
  { key: 'skip', tone: 'skip', countProp: 'skip', labelKey: 'count_skip' },
];

function formatStartedAt(row) {
  if (row?.started_at) {
    const d = new Date(Number(row.started_at) * 1000);
    if (!Number.isNaN(d.getTime())) return d.toLocaleString();
  }
  if (row?.tested_at && row?.elapsed_ms != null) {
    const d = new Date(Number(row.tested_at) * 1000 - Number(row.elapsed_ms));
    if (!Number.isNaN(d.getTime())) return d.toLocaleString();
  }
  return '—';
}

function statusKey(row) {
  if (row?.skipped) return 'skip';
  if (row?.timed_out) return 'timeout';
  if (row?.success) return 'ok';
  return 'fail';
}

function statusTooltip(t, row) {
  const key = statusKey(row);
  if (key === 'timeout') return t('channel.edit.test_report.status_timeout');
  if (key === 'skip') return t('channel.edit.test_report.status_skip');
  if (key === 'ok') return t('channel.edit.test_report.status_ok');
  return t('channel.edit.test_report.status_fail');
}

function hasTestDetail(row) {
  const d = row?.detail;
  if (!d || typeof d !== 'object') return false;
  return !!(
    d.request_body ||
    d.request_url ||
    d.request_path ||
    (d.request_headers && Object.keys(d.request_headers).length) ||
    d.response_body ||
    d.response_status ||
    (d.response_headers && Object.keys(d.response_headers).length)
  );
}

export default function ChannelModelTestReportModal({
  open,
  onClose,
  rows = [],
  channelTypeLabel = '',
  baseUrl = '',
  totalModels = 0,
}) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState('all');
  const [detailRow, setDetailRow] = useState(null);

  const summary = useMemo(() => summarizeModelTestReport(rows), [rows]);
  const allCount = Number(totalModels) > 0 ? Number(totalModels) : summary.total;

  const filtered = useMemo(() => {
    const list = [...(rows || [])].sort((a, b) =>
      String(a.model || '').localeCompare(String(b.model || ''))
    );
    if (filter === 'ok') {
      return list.filter((r) => r.success && !r.skipped);
    }
    if (filter === 'fail') {
      return list.filter((r) => !r.success && !r.skipped);
    }
    if (filter === 'skip') {
      return list.filter((r) => r.skipped);
    }
    return list;
  }, [rows, filter]);

  return (
    <>
      <Modal
        open={open}
        onClose={onClose}
        closeIcon
        className='channel-model-test-report-modal'
      >
        <Modal.Header>{t('channel.edit.test_report.title')}</Modal.Header>
        <Modal.Content scrolling>
          <div className='channel-model-test-report__meta'>
            {channelTypeLabel ? (
              <span>
                {t('channel.edit.test_report.channel_type')}: {channelTypeLabel}
              </span>
            ) : null}
            {baseUrl ? (
              <span>
                {t('channel.edit.test_report.base_url')}: {baseUrl}
              </span>
            ) : null}
          </div>
          <div className='channel-model-test-report__filter-bar' role='tablist' aria-label={t('channel.edit.test_report.title')}>
            {FILTER_ITEMS.map(({ key, tone, countProp, labelKey }) => {
              const active = filter === key;
              return (
                <button
                  key={key}
                  type='button'
                  role='tab'
                  aria-selected={active}
                  className={`channel-model-test-report__filter-btn channel-model-test-report__filter-btn--${tone}${active ? ' is-active' : ''}`}
                  onClick={() => setFilter(key)}
                >
                  {key === 'all'
                    ? `${t('channel.edit.test_report.filter_all')} ${allCount}`
                    : t(`channel.edit.test_report.${labelKey}`, { count: summary[countProp] })}
                </button>
              );
            })}
          </div>
          <Table celled compact striped size='small' className='channel-model-test-report__table'>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>{t('channel.edit.test_report.col_model')}</Table.HeaderCell>
                <Table.HeaderCell>{t('channel.edit.test_report.col_kind')}</Table.HeaderCell>
                <Table.HeaderCell collapsing>{t('channel.edit.test_report.col_status')}</Table.HeaderCell>
                <Table.HeaderCell>{t('channel.edit.test_report.col_started_at')}</Table.HeaderCell>
                <Table.HeaderCell>{t('channel.edit.test_report.col_time')}</Table.HeaderCell>
                <Table.HeaderCell>{t('channel.edit.test_report.col_message')}</Table.HeaderCell>
                <Table.HeaderCell>{t('channel.edit.test_report.col_action')}</Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {filtered.length === 0 ? (
                <Table.Row>
                  <Table.Cell colSpan={7} textAlign='center'>
                    {t('channel.edit.test_report.empty')}
                  </Table.Cell>
                </Table.Row>
              ) : (
                filtered.map((row) => (
                  <Table.Row key={row.model}>
                    <Table.Cell className='channel-model-test-report__model'>{row.model}</Table.Cell>
                    <Table.Cell className='channel-model-test-report__protocol'>
                      {row.test_protocol || row.test_kind || '—'}
                    </Table.Cell>
                    <Table.Cell textAlign='center'>
                      <span
                        className={`channel-model-test-report__status channel-model-test-report__status--${statusKey(row)}`}
                        title={statusTooltip(t, row)}
                        aria-label={statusTooltip(t, row)}
                      />
                    </Table.Cell>
                    <Table.Cell className='channel-model-test-report__started-at'>
                      {formatStartedAt(row)}
                    </Table.Cell>
                    <Table.Cell>
                      {row.elapsed_ms != null
                        ? `${(Number(row.elapsed_ms) / 1000).toFixed(2)}s`
                        : row.time != null
                          ? `${Number(row.time).toFixed(2)}s`
                          : '—'}
                    </Table.Cell>
                    <Table.Cell className='channel-model-test-report__message'>
                      {row.message || '—'}
                    </Table.Cell>
                    <Table.Cell collapsing>
                      <Button
                        size='mini'
                        basic
                        disabled={!hasTestDetail(row)}
                        onClick={() => setDetailRow(row)}
                      >
                        {t('channel.edit.test_report.view_detail')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))
              )}
            </Table.Body>
          </Table>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={onClose}>{t('channel.edit.test_report.close')}</Button>
        </Modal.Actions>
      </Modal>
      <ChannelModelTestDetailModal
        open={!!detailRow}
        onClose={() => setDetailRow(null)}
        row={detailRow}
      />
    </>
  );
}
