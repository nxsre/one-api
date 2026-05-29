<template>
  <a-modal
    v-if="detail"
    :open="open"
    width="900px"
    :title="t('channel.edit.test_report.detail_title', { model: row?.model || '—' })"
    @cancel="emit('close')"
  >
    <div class="channel-model-test-detail">
      <div v-if="detail.wire_protocol" class="channel-model-test-detail__meta">
        {{ t('channel.edit.test_report.wire_protocol') }}: {{ detail.wire_protocol }}
      </div>
      <div v-if="requestLine" class="channel-model-test-detail__meta">{{ requestLine }}</div>
      <DetailBlock :title="t('channel.edit.test_report.request_headers')">
        {{ formatHeaders(detail.request_headers) }}
      </DetailBlock>
      <DetailBlock :title="t('channel.edit.test_report.request_body')">
        {{ formatBody(detail.request_body) }}
      </DetailBlock>
      <DetailBlock :title="t('channel.edit.test_report.response_status')">
        {{ detail.response_status ? String(detail.response_status) : '—' }}
      </DetailBlock>
      <DetailBlock :title="t('channel.edit.test_report.response_headers')">
        {{ formatHeaders(detail.response_headers) }}
      </DetailBlock>
      <DetailBlock :title="t('channel.edit.test_report.response_body')">
        {{ formatBody(detail.response_body) }}
      </DetailBlock>
    </div>
    <template #footer>
      <a-button @click="emit('close')">{{ t('channel.edit.test_report.close') }}</a-button>
    </template>
  </a-modal>
</template>

<script setup>
import { computed, h } from 'vue';
import { useI18n } from 'vue-i18n';

const props = defineProps({
  open: { type: Boolean, default: false },
  row: { type: Object, default: null },
});
const emit = defineEmits(['close']);

const { t } = useI18n();

const detail = computed(() => props.row?.detail);

const requestLine = computed(() => {
  const d = detail.value;
  if (!d) return '';
  return [d.request_method, d.request_url || d.request_path].filter(Boolean).join(' ');
});

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

const DetailBlock = (props2, { slots }) =>
  h('div', { class: 'channel-model-test-detail__block' }, [
    h('div', { class: 'channel-model-test-detail__block-title' }, props2.title),
    h('pre', { class: 'channel-model-test-detail__pre' }, slots.default?.()),
  ]);
DetailBlock.props = ['title'];
</script>

<style scoped>
.channel-model-test-detail {
  max-height: calc(100vh - 16rem);
  overflow: auto;
}

.channel-model-test-detail__meta {
  margin-bottom: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.88rem;
  color: rgba(0, 0, 0, 0.7);
  word-break: break-all;
}

.channel-model-test-detail__block {
  margin-bottom: 0.85rem;
}

.channel-model-test-detail__block-title {
  font-weight: 600;
  margin-bottom: 0.35rem;
}

.channel-model-test-detail__pre {
  margin: 0;
  padding: 0.65rem 0.75rem;
  background: #f7f7f7;
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 4px;
  max-height: 16rem;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 0.82rem;
  line-height: 1.45;
}
</style>
