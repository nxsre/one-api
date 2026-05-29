<template>
  <div class="channel-model-mapping-editor">
    <p class="channel-edit-field-hint">
      {{ exampleHint || `${t('channel.edit.model_mapping_placeholder')}\n${MODEL_MAP_EXAMPLE_LINES}` }}
    </p>
    <template v-if="!jsonMode">
      <a-table
        class="channel-model-mapping-table"
        :columns="columns"
        :data-source="rows"
        :pagination="false"
        size="small"
        bordered
        :row-key="(_, i) => i"
      >
        <template #bodyCell="{ column, index, record }">
          <template v-if="column.key === 'from'">
            <a-input
              :placeholder="t('channel.edit.model_mapping_ph_request')"
              :value="record.from"
              @update:value="(v) => updateRow(index, 'from', v)"
            />
          </template>
          <template v-else-if="column.key === 'to'">
            <a-select
              v-if="upstreamOptions.length > 0"
              show-search
              :options="optionsForRow(record)"
              :field-names="{ label: 'text', value: 'value' }"
              :value="record.to"
              :placeholder="t('channel.edit.model_mapping_ph_upstream')"
              style="width: 100%"
              :filter-option="filterOption"
              @change="(v) => updateRow(index, 'to', v || '')"
            />
            <a-input
              v-else
              :placeholder="t('channel.edit.model_mapping_ph_upstream')"
              :value="record.to"
              @update:value="(v) => updateRow(index, 'to', v)"
            />
          </template>
          <template v-else-if="column.key === 'op'">
            <a-button
              size="small"
              :disabled="rows.length <= 1"
              :title="t('channel.edit.model_mapping_remove')"
              @click="removeRow(index)"
            >
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </template>
        </template>
      </a-table>
      <div class="channel-model-mapping-editor__toolbar">
        <a-button size="small" @click="addRow">
          <template #icon><PlusOutlined /></template>
          {{ t('channel.edit.model_mapping_add') }}
        </a-button>
        <a-button size="small" @click="openJsonMode">
          {{ t('channel.edit.model_mapping_json_toggle') }}
        </a-button>
      </div>
    </template>
    <div v-else class="channel-model-mapping-editor__json">
      <MonacoEditor
        v-model:value="jsonRaw"
        language="json"
        :height="240"
      />
      <div class="channel-model-mapping-editor__toolbar">
        <a-button size="small" type="primary" @click="applyJsonPaste">
          {{ t('channel.edit.model_mapping_json_apply') }}
        </a-button>
        <a-button size="small" @click="cancelJsonMode">
          {{ t('channel.edit.model_mapping_json_cancel') }}
        </a-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue';
import { showError, verifyJSON } from '@/helpers';
import MonacoEditor from '@/components/MonacoEditor.vue';

const props = defineProps({
  value: { type: String, default: '' },
  exampleHint: { type: String, default: '' },
  upstreamModelOptions: { type: Array, default: () => [] },
});
const emit = defineEmits(['change', 'update:value']);

const { t } = useI18n();

const MODEL_MAP_EXAMPLE_LINES = `{
  "gpt-3.5-turbo-0301": "gpt-3.5-turbo",
  "gpt-4-0314": "gpt-4"
}`;

function parseMappingToRows(str) {
  const s = String(str ?? '').trim();
  if (!s) {
    return [{ from: '', to: '' }];
  }
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      const ent = Object.entries(o);
      if (ent.length === 0) {
        return [{ from: '', to: '' }];
      }
      return ent.map(([from, to]) => ({ from: String(from), to: String(to ?? '') }));
    }
  } catch {
    /* fallthrough */
  }
  return [{ from: '', to: '' }];
}

function rowsToMappingJson(rowList) {
  const o = {};
  for (const r of rowList) {
    const k = String(r.from ?? '').trim();
    if (!k) continue;
    o[k] = String(r.to ?? '');
  }
  if (Object.keys(o).length === 0) {
    return '';
  }
  return JSON.stringify(o, null, 2);
}

const rows = ref(parseMappingToRows(props.value));
const jsonRaw = ref('');
const jsonMode = ref(false);

const columns = computed(() => [
  { title: t('channel.edit.model_mapping_col_request'), key: 'from' },
  { title: t('channel.edit.model_mapping_col_upstream'), key: 'to' },
  { title: '', key: 'op', width: 60 },
]);

const upstreamOptions = computed(() => {
  const seen = new Set();
  const out = [];
  (props.upstreamModelOptions || []).forEach((id) => {
    const s = String(id || '').trim();
    if (!s || seen.has(s)) return;
    seen.add(s);
    out.push({ key: s, value: s, text: s });
  });
  return out;
});

function optionsForRow(record) {
  const current = String(record.to || '').trim();
  if (!current || upstreamOptions.value.some((o) => o.value === current)) {
    return upstreamOptions.value;
  }
  return [{ key: `custom-${current}`, value: current, text: current }, ...upstreamOptions.value];
}

function filterOption(input, option) {
  return String(option.text || option.value || '')
    .toLowerCase()
    .includes(String(input).toLowerCase());
}

watch(
  () => props.value,
  (v) => {
    rows.value = parseMappingToRows(v);
  }
);

function emitRows(next) {
  rows.value = next;
  const v = rowsToMappingJson(next);
  emit('change', v);
  emit('update:value', v);
}

function updateRow(i, field, v) {
  const next = rows.value.map((r, j) => (j === i ? { ...r, [field]: v } : r));
  emitRows(next);
}

function addRow() {
  emitRows([...rows.value, { from: '', to: '' }]);
}

function removeRow(i) {
  const next = rows.value.filter((_, j) => j !== i);
  emitRows(next.length ? next : [{ from: '', to: '' }]);
}

function openJsonMode() {
  jsonRaw.value = props.value?.trim() ? props.value : MODEL_MAP_EXAMPLE_LINES;
  jsonMode.value = true;
}

function cancelJsonMode() {
  jsonMode.value = false;
  jsonRaw.value = '';
}

function applyJsonPaste() {
  const raw = jsonRaw.value.trim();
  if (!raw) {
    emitRows([{ from: '', to: '' }]);
    jsonMode.value = false;
    return;
  }
  if (!verifyJSON(raw)) {
    showError(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  let o;
  try {
    o = JSON.parse(raw);
  } catch {
    showError(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  if (typeof o !== 'object' || o === null || Array.isArray(o)) {
    showError(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  emitRows(parseMappingToRows(raw));
  jsonMode.value = false;
  jsonRaw.value = '';
}
</script>
