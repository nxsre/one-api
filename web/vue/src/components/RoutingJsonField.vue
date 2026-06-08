<template>
  <div class="routing-json-field">
    <div class="routing-json-segment">
      <div class="routing-json-toolbar">
        <a-button-group size="small">
          <a-button type="button" :disabled="locked" :title="t('routing.json_format')" @click="formatJson">
            <template #icon><MenuOutlined /></template>
            {{ t('routing.json_format') }}
          </a-button>
          <a-button type="button" :disabled="locked" :title="t('routing.json_minify')" @click="minifyJson">
            <template #icon><ShrinkOutlined /></template>
            {{ t('routing.json_minify') }}
          </a-button>
          <a-button type="button" :title="t('routing.json_copy')" @click="copyAll">
            <template #icon><CopyOutlined /></template>
            {{ t('routing.json_copy') }}
          </a-button>
          <a-button type="button" :disabled="locked" :title="t('routing.json_expand')" @click="openExpand">
            <template #icon><ExpandOutlined /></template>
            {{ t('routing.json_expand') }}
          </a-button>
        </a-button-group>
        <span class="routing-json-status">
          <a-tag v-if="analysis.kind === 'empty'" color="default">
            {{ t('routing.json_empty_badge') }}
          </a-tag>
          <a-tag v-else-if="analysis.ok" color="green">
            <CheckOutlined /> {{ t('routing.json_ok') }}
          </a-tag>
          <a-tag v-else color="red">
            <WarningOutlined /> {{ t('routing.json_error') }}
          </a-tag>
        </span>
      </div>

      <a-alert
        v-if="!analysis.ok && analysis.message && analysis.message !== 'empty'"
        type="error"
        class="routing-json-parse-msg"
        :message="t('routing.json_error')"
      >
        <template #description>
          <pre class="routing-json-error-pre">{{ analysis.message }}</pre>
        </template>
      </a-alert>

      <MonacoEditor
        :id="id"
        :value="text"
        language="json"
        :read-only="!!readOnly || !!disabled"
        :height="monacoHeight"
        :class="
          'routing-json-textarea' +
          (!analysis.ok && analysis.message !== 'empty' ? ' routing-json-textarea-invalid' : '')
        "
        @update:value="handleAreaChange"
      />
    </div>

    <a-modal
      v-model:open="expanded"
      :title="t('routing.json_expand')"
      width="90vw"
      :mask-closable="false"
      :keyboard="false"
    >
      <MonacoEditor
        v-if="expanded"
        :value="draft"
        language="json"
        :height="560"
        class="routing-json-textarea routing-json-textarea-modal"
        @update:value="(v) => (draft = v)"
      />
      <template #footer>
        <a-button @click="expanded = false">{{ t('routing.json_expand_cancel') }}</a-button>
        <a-button type="primary" :disabled="locked" @click="applyExpand">
          {{ t('routing.json_expand_apply') }}
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  MenuOutlined,
  ShrinkOutlined,
  CopyOutlined,
  ExpandOutlined,
  CheckOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue';
import { copy, showError, showSuccess } from '@/helpers';
import MonacoEditor from '@/components/MonacoEditor.vue';

const props = defineProps({
  id: { type: String, default: undefined },
  value: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  readOnly: { type: Boolean, default: false },
  minRows: { type: Number, default: 12 },
  allowEmpty: { type: Boolean, default: true },
  placeholder: { type: String, default: '' },
});
const emit = defineEmits(['update:value', 'change']);

const { t } = useI18n();

function analyzeJson(textVal, allowEmpty) {
  const raw = textVal ?? '';
  const trimmed = raw.trim();
  if (!trimmed) {
    if (allowEmpty) {
      return { ok: true, kind: 'empty', message: null };
    }
    return { ok: false, kind: 'error', message: 'empty' };
  }
  try {
    JSON.parse(trimmed);
    return { ok: true, kind: 'json', message: null };
  } catch (e) {
    return { ok: false, kind: 'error', message: e instanceof Error ? e.message : String(e) };
  }
}

const text = ref(props.value ?? '');
const expanded = ref(false);
const draft = ref('');

watch(
  () => props.value,
  (v) => {
    text.value = v ?? '';
  }
);

const analysis = computed(() => analyzeJson(text.value, props.allowEmpty));
const locked = computed(() => props.disabled || props.readOnly);
const monacoHeight = computed(() => Math.max(160, props.minRows * 20));

function commit(next) {
  text.value = next;
  emit('update:value', next);
  emit('change', next);
}

function handleAreaChange(v) {
  commit(v);
}

function formatJson() {
  const a = analyzeJson(text.value, props.allowEmpty);
  if (a.kind === 'empty') {
    commit('');
    return;
  }
  if (!a.ok) {
    showError(t('routing.json_error') + ': ' + (a.message || ''));
    return;
  }
  try {
    const obj = JSON.parse(text.value.trim());
    commit(JSON.stringify(obj, null, 2));
  } catch (e) {
    showError(e.message || String(e));
  }
}

function minifyJson() {
  const a = analyzeJson(text.value, props.allowEmpty);
  if (a.kind === 'empty') {
    commit('');
    return;
  }
  if (!a.ok) {
    showError(t('routing.json_error') + ': ' + (a.message || ''));
    return;
  }
  try {
    const obj = JSON.parse(text.value.trim());
    commit(JSON.stringify(obj));
  } catch (e) {
    showError(e.message || String(e));
  }
}

async function copyAll() {
  const ok = await copy(text.value);
  if (ok) {
    showSuccess(t('routing.json_copied'));
  } else {
    showError(t('routing.json_copy_failed'));
  }
}

function openExpand() {
  draft.value = text.value;
  expanded.value = true;
}

function applyExpand() {
  const a = analyzeJson(draft.value, props.allowEmpty);
  if (!a.ok && a.message !== 'empty') {
    showError(t('routing.json_error') + ': ' + (a.message || ''));
    return;
  }
  commit(draft.value);
  expanded.value = false;
}
</script>

<style scoped>
.routing-json-segment {
  border: 1px solid rgba(34, 36, 38, 0.15);
  border-radius: 4px;
  padding: 0.5rem;
}
.routing-json-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.routing-json-parse-msg {
  margin-bottom: 0.5rem;
}
.routing-json-error-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
}
.routing-json-textarea-invalid {
  border: 1px solid #db2828;
}
</style>
