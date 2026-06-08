<template>
  <a-modal
    :open="open"
    class="channel-model-test-report-modal"
    width="94vw"
    :title="t('channel.edit.test_report.title')"
    @cancel="emit('close')"
  >
    <div class="channel-model-test-report__body">
      <div class="channel-model-test-report__meta">
        <span v-if="channelTypeLabel">
          {{ t('channel.edit.test_report.channel_type') }}: {{ channelTypeLabel }}
        </span>
        <span v-if="baseUrl">{{ t('channel.edit.test_report.base_url') }}: {{ baseUrl }}</span>
      </div>
      <div
        class="channel-model-test-report__filter-bar"
        role="tablist"
        :aria-label="t('channel.edit.test_report.title')"
      >
        <button
          v-for="item in FILTER_ITEMS"
          :key="item.key"
          type="button"
          role="tab"
          :aria-selected="filter === item.key"
          :class="[
            'channel-model-test-report__filter-btn',
            `channel-model-test-report__filter-btn--${item.tone}`,
            { 'is-active': filter === item.key },
          ]"
          @click="filter = item.key"
        >
          <template v-if="item.key === 'all'">
            {{ t('channel.edit.test_report.filter_all') }} {{ allCount }}
          </template>
          <template v-else>
            {{ t(`channel.edit.test_report.${item.labelKey}`, { count: summary[item.countProp] }) }}
          </template>
        </button>
      </div>
      <a-table
        class="channel-model-test-report__table"
        :columns="columns"
        :data-source="filtered"
        :pagination="false"
        size="small"
        bordered
        row-key="model"
      >
        <template #emptyText>
          {{ t('channel.edit.test_report.empty') }}
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'model'">
            <span class="channel-model-test-report__model">{{ record.model }}</span>
          </template>
          <template v-else-if="column.key === 'kind'">
            <span class="channel-model-test-report__protocol">
              {{ record.test_protocol || record.test_kind || '—' }}
            </span>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tooltip :title="statusTooltip(record)">
              <span
                :class="`channel-model-test-report__status channel-model-test-report__status--${statusKey(record)}`"
                :aria-label="statusTooltip(record)"
              />
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'started_at'">
            <span class="channel-model-test-report__started-at">{{ formatStartedAt(record) }}</span>
          </template>
          <template v-else-if="column.key === 'time'">
            {{ formatTime(record) }}
          </template>
          <template v-else-if="column.key === 'message'">
            <span class="channel-model-test-report__message">{{ record.message || '—' }}</span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button size="small" :disabled="!hasTestDetail(record)" @click="detailRow = record">
              {{ t('channel.edit.test_report.view_detail') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </div>
    <template #footer>
      <a-button @click="emit('close')">{{ t('channel.edit.test_report.close') }}</a-button>
    </template>
  </a-modal>
  <ChannelModelTestDetailModal :open="!!detailRow" :row="detailRow" @close="detailRow = null" />
</template>

<script setup>
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { summarizeModelTestReport } from '@/helpers/channelModelTest';
import ChannelModelTestDetailModal from './ChannelModelTestDetailModal.vue';

const props = defineProps({
  open: { type: Boolean, default: false },
  rows: { type: Array, default: () => [] },
  channelTypeLabel: { type: String, default: '' },
  baseUrl: { type: String, default: '' },
  totalModels: { type: Number, default: 0 },
});
const emit = defineEmits(['close']);

const { t } = useI18n();

const FILTER_ITEMS = [
  { key: 'all', tone: 'all', countProp: '', labelKey: 'filter_all' },
  { key: 'done', tone: 'all', countProp: 'total', labelKey: 'count_done' },
  { key: 'ok', tone: 'ok', countProp: 'ok', labelKey: 'count_ok' },
  { key: 'fail', tone: 'fail', countProp: 'fail', labelKey: 'count_fail' },
  { key: 'skip', tone: 'skip', countProp: 'skip', labelKey: 'count_skip' },
];

const filter = ref('all');
const detailRow = ref(null);

const summary = computed(() => summarizeModelTestReport(props.rows));
const allCount = computed(() =>
  Number(props.totalModels) > 0 ? Number(props.totalModels) : summary.value.total
);

const columns = computed(() => [
  { title: t('channel.edit.test_report.col_model'), key: 'model' },
  { title: t('channel.edit.test_report.col_kind'), key: 'kind' },
  { title: t('channel.edit.test_report.col_status'), key: 'status', width: 80 },
  { title: t('channel.edit.test_report.col_started_at'), key: 'started_at' },
  { title: t('channel.edit.test_report.col_time'), key: 'time' },
  { title: t('channel.edit.test_report.col_message'), key: 'message' },
  { title: t('channel.edit.test_report.col_action'), key: 'action' },
]);

const filtered = computed(() => {
  const list = [...(props.rows || [])].sort((a, b) =>
    String(a.model || '').localeCompare(String(b.model || ''))
  );
  if (filter.value === 'ok') {
    return list.filter((r) => r.success && !r.skipped);
  }
  if (filter.value === 'fail') {
    return list.filter((r) => !r.success && !r.skipped);
  }
  if (filter.value === 'skip') {
    return list.filter((r) => r.skipped);
  }
  return list;
});

function statusKey(row) {
  if (row?.skipped) return 'skip';
  if (row?.timed_out) return 'timeout';
  if (row?.success) return 'ok';
  return 'fail';
}

function statusTooltip(row) {
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

function formatTime(row) {
  if (row.elapsed_ms != null) return `${(Number(row.elapsed_ms) / 1000).toFixed(2)}s`;
  if (row.time != null) return `${Number(row.time).toFixed(2)}s`;
  return '—';
}
</script>

<style scoped>
.channel-model-test-report__body {
  max-height: calc(100vh - 12rem);
  overflow: auto;
}

.channel-model-test-report__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.25rem;
  margin-bottom: 0.75rem;
  color: rgba(0, 0, 0, 0.65);
  font-size: 0.92rem;
}

.channel-model-test-report__filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.85rem;
}

.channel-model-test-report__filter-btn {
  appearance: none;
  border: 2px solid transparent;
  border-radius: 6px;
  padding: 0.42rem 0.85rem;
  font-size: 0.88rem;
  font-weight: 500;
  line-height: 1.2;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease,
    color 0.15s ease;
}

.channel-model-test-report__filter-btn:focus-visible {
  outline: 2px solid #2185d0;
  outline-offset: 2px;
}

.channel-model-test-report__filter-btn--all {
  background: rgba(0, 0, 0, 0.04);
  border-color: rgba(0, 0, 0, 0.14);
  color: rgba(0, 0, 0, 0.75);
}

.channel-model-test-report__filter-btn--all.is-active {
  background: #fff;
  border-color: #2185d0;
  color: #1678c2;
  box-shadow: 0 0 0 2px rgba(33, 133, 208, 0.18);
}

.channel-model-test-report__filter-btn--ok {
  background: rgba(33, 186, 69, 0.1);
  border-color: rgba(33, 186, 69, 0.35);
  color: #16853c;
}

.channel-model-test-report__filter-btn--ok.is-active {
  background: #21ba45;
  border-color: #21ba45;
  color: #fff;
  box-shadow: 0 0 0 2px rgba(33, 186, 69, 0.28);
}

.channel-model-test-report__filter-btn--fail {
  background: rgba(219, 40, 40, 0.08);
  border-color: rgba(219, 40, 40, 0.32);
  color: #c01c1c;
}

.channel-model-test-report__filter-btn--fail.is-active {
  background: #db2828;
  border-color: #db2828;
  color: #fff;
  box-shadow: 0 0 0 2px rgba(219, 40, 40, 0.25);
}

.channel-model-test-report__filter-btn--skip {
  background: rgba(118, 118, 118, 0.1);
  border-color: rgba(118, 118, 118, 0.32);
  color: #5c5c5c;
}

.channel-model-test-report__filter-btn--skip.is-active {
  background: #767676;
  border-color: #767676;
  color: #fff;
  box-shadow: 0 0 0 2px rgba(118, 118, 118, 0.25);
}

.channel-model-test-report__table {
  font-size: 0.88rem;
}

.channel-model-test-report__model {
  display: inline-block;
  min-width: 10rem;
  max-width: 16rem;
  word-break: break-all;
}

.channel-model-test-report__protocol {
  display: inline-block;
  min-width: 11rem;
  max-width: 14rem;
  word-break: break-word;
}

.channel-model-test-report__started-at {
  white-space: nowrap;
  min-width: 9.5rem;
}

.channel-model-test-report__status {
  display: inline-block;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  vertical-align: middle;
}

.channel-model-test-report__status--ok {
  background-color: #21ba45;
}

.channel-model-test-report__status--fail {
  background-color: #db2828;
}

.channel-model-test-report__status--timeout {
  background-color: #f2711c;
}

.channel-model-test-report__status--skip {
  background-color: #767676;
}

.channel-model-test-report__message {
  display: inline-block;
  max-width: 36rem;
  word-break: break-word;
  white-space: pre-wrap;
}
</style>
