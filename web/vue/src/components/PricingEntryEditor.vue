<template>
  <div class="pricing-entry-editor">
    <div class="pricing-entry-editor__toolbar">
      <a-input-search
        :placeholder="t('setting.operation.ratio.editor.filter_placeholder')"
        :value="searchInput"
        :enter-button="t('setting.operation.ratio.editor.search')"
        style="max-width: 360px"
        @update:value="(v) => (searchInput = v)"
        @search="() => (search = searchInput)"
        @press-enter="() => (search = searchInput)"
      />
      <span class="pricing-entry-editor__count">
        {{ t('setting.operation.ratio.editor.entry_count', { count: total, total }) }}
      </span>
    </div>

    <div class="pricing-entry-editor__add">
      <div class="pricing-entry-editor__add-grid">
        <div class="pricing-entry-editor__add-field">
          <label>{{ keyLabel }}</label>
          <a-input
            v-model:value="newRow.entry_key"
            :list="keyDropdown ? `pe-keys-${blockId}` : undefined"
          />
          <datalist v-if="keyDropdown" :id="`pe-keys-${blockId}`">
            <option v-for="o in keyOptions" :key="o.value" :value="o.value" />
          </datalist>
        </div>
        <div v-if="isGroupGroup" class="pricing-entry-editor__add-field">
          <label>{{ subLabel }}</label>
          <a-input
            v-model:value="newRow.sub_key"
            :list="keyDropdown ? `pe-subkeys-${blockId}` : undefined"
          />
          <datalist v-if="keyDropdown" :id="`pe-subkeys-${blockId}`">
            <option v-for="o in keyOptions" :key="`s-${o.value}`" :value="o.value" />
          </datalist>
        </div>
        <div class="pricing-entry-editor__add-field">
          <label>{{ t('setting.operation.ratio.editor.col_value') }}</label>
          <a-input
            v-model:value="newRow.value_text"
            :type="isStringValue ? 'text' : 'number'"
            :step="isStringValue ? undefined : 'any'"
          />
        </div>
        <div class="pricing-entry-editor__add-field pricing-entry-editor__add-btn">
          <label>&nbsp;</label>
          <a-button type="primary" @click="createRow">
            {{ t('setting.operation.ratio.editor.add_row') }}
          </a-button>
        </div>
      </div>
    </div>

    <div class="pricing-entry-editor__table-wrap">
      <a-table
        :columns="columns"
        :data-source="items"
        :loading="loading"
        :pagination="false"
        size="small"
        bordered
        row-key="id"
      >
        <template #emptyText>
          {{ t('setting.operation.ratio.editor.empty') }}
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'entry_key'">
            <span class="pricing-entry-editor__key-cell">{{ record.entry_key }}</span>
          </template>
          <template v-else-if="column.key === 'sub_key'">
            {{ record.sub_key }}
          </template>
          <template v-else-if="column.key === 'value'">
            <a-input
              :type="isStringValue ? 'text' : 'number'"
              :step="isStringValue ? undefined : 'any'"
              :value="draftValue(record)"
              @update:value="(v) => setDraft(record.id, v)"
            />
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button
              size="small"
              :loading="savingId === record.id"
              :disabled="savingId === record.id"
              :title="t('setting.operation.ratio.buttons.save')"
              @click="saveRow(record)"
            >
              <template #icon><SaveOutlined /></template>
            </a-button>
            <a-button
              size="small"
              danger
              style="margin-left: 6px"
              :title="t('setting.operation.ratio.editor.remove_row')"
              @click="deleteRow(record)"
            >
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </template>
        </template>
      </a-table>
    </div>

    <div class="pricing-entry-editor__footer">
      <a-select
        :value="pageSize"
        style="width: 120px"
        size="small"
        :options="pageSizeOptions"
        @update:value="(v) => (pageSize = Number(v))"
      />
      <a-pagination
        v-if="totalPages > 1"
        size="small"
        :current="page"
        :total="total"
        :page-size="pageSize"
        :show-size-changer="false"
        @change="(p) => (page = Number(p))"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Modal } from 'ant-design-vue';
import { SaveOutlined, DeleteOutlined } from '@ant-design/icons-vue';
import { API, showError, showSuccess } from '@/helpers';

const PAGE_SIZE_OPTIONS = [20, 50, 100];

const props = defineProps({
  blockId: { type: String, required: true },
  valueKind: { type: String, default: 'number' },
  keyColumnLabel: { type: String, default: '' },
  subKeyColumnLabel: { type: String, default: '' },
  keyOptions: { type: Array, default: () => [] },
  keyDropdown: { type: Boolean, default: false },
  onMutate: { type: Function, default: null },
});

const { t } = useI18n();

const items = ref([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(50);
const totalPages = ref(1);
const search = ref('');
const searchInput = ref('');
const loading = ref(false);
const savingId = ref(null);
const drafts = ref({});
const newRow = ref({ entry_key: '', sub_key: '', value_text: '' });

const isGroupGroup = computed(() => props.valueKind === 'group_group');
const isStringValue = computed(() => props.valueKind === 'string');

const keyLabel = computed(
  () => props.keyColumnLabel || t('setting.operation.ratio.editor.col_key')
);
const subLabel = computed(
  () => props.subKeyColumnLabel || t('setting.operation.ratio.group_group.col_use_group')
);

const pageSizeOptions = computed(() =>
  PAGE_SIZE_OPTIONS.map((n) => ({
    value: n,
    label: `${n} / ${t('setting.operation.ratio.editor.page')}`,
  }))
);

const columns = computed(() => {
  const cols = [{ title: keyLabel.value, key: 'entry_key', dataIndex: 'entry_key' }];
  if (isGroupGroup.value) {
    cols.push({ title: subLabel.value, key: 'sub_key', dataIndex: 'sub_key' });
  }
  cols.push({
    title: t('setting.operation.ratio.editor.col_value'),
    key: 'value',
  });
  cols.push({ title: '', key: 'actions', width: 110 });
  return cols;
});

const load = async () => {
  if (!props.blockId) return;
  loading.value = true;
  try {
    const params = new URLSearchParams({
      block_id: props.blockId,
      page: String(page.value),
      page_size: String(pageSize.value),
    });
    if (search.value.trim()) params.set('q', search.value.trim());
    const res = await API.get(`/api/pricing_entries/?${params.toString()}`);
    const body = res.data || {};
    if (!body.success) {
      showError(body.message || 'load failed');
      return;
    }
    const data = body.data || {};
    items.value = Array.isArray(data.items) ? data.items : [];
    total.value = Number(data.total) || 0;
    totalPages.value = Number(data.total_pages) || 1;
    drafts.value = {};
  } catch (e) {
    showError(e.message || 'load failed');
  } finally {
    loading.value = false;
  }
};

const notifyMutate = async () => {
  if (typeof props.onMutate === 'function') {
    await props.onMutate();
  }
};

// Reset to first page on blockId / pageSize / search change.
watch(
  () => [props.blockId, pageSize.value, search.value],
  () => {
    page.value = 1;
  }
);

watch(
  () => [props.blockId, page.value, pageSize.value, search.value],
  () => {
    load();
  },
  { immediate: true }
);

const draftValue = (row) =>
  drafts.value[row.id] !== undefined ? drafts.value[row.id] : row.value_text;

const setDraft = (id, value) => {
  drafts.value = { ...drafts.value, [id]: value };
};

const saveRow = async (row) => {
  const valueText = String(draftValue(row) ?? '').trim();
  if (!valueText) {
    showError(t('setting.operation.ratio.editor.value_required'));
    return;
  }
  savingId.value = row.id;
  try {
    const res = await API.put(`/api/pricing_entries/${row.id}`, {
      value_text: valueText,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('setting.operation.ratio.editor.row_saved'));
    await load();
    await notifyMutate();
  } catch (e) {
    showError(e.message || 'save failed');
  } finally {
    savingId.value = null;
  }
};

const deleteRow = (row) => {
  Modal.confirm({
    title: t('setting.operation.ratio.editor.confirm_delete'),
    onOk: async () => {
      try {
        const res = await API.delete(`/api/pricing_entries/${row.id}`);
        if (!res.data?.success) {
          showError(res.data?.message || 'delete failed');
          return;
        }
        await load();
        await notifyMutate();
      } catch (e) {
        showError(e.message || 'delete failed');
      }
    },
  });
};

const createRow = async () => {
  const entryKey = String(newRow.value.entry_key ?? '').trim();
  const subKey = String(newRow.value.sub_key ?? '').trim();
  const valueText = String(newRow.value.value_text ?? '').trim();
  if (!entryKey || !valueText || (isGroupGroup.value && !subKey)) {
    showError(t('setting.operation.ratio.editor.create_invalid'));
    return;
  }
  try {
    const res = await API.post('/api/pricing_entries/', {
      block_id: props.blockId,
      entry_key: entryKey,
      sub_key: subKey,
      value_text: valueText,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'create failed');
      return;
    }
    newRow.value = { entry_key: '', sub_key: '', value_text: '' };
    page.value = 1;
    await load();
    await notifyMutate();
  } catch (e) {
    showError(e.message || 'create failed');
  }
};
</script>

<style>
.pricing-entry-editor {
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 6px;
  padding: 0.75rem;
  background: #fafbfc;
}

.pricing-entry-editor__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.pricing-entry-editor__count {
  font-size: 0.85rem;
  color: rgba(0, 0, 0, 0.55);
}

.pricing-entry-editor__add {
  margin-bottom: 0.75rem;
}

.pricing-entry-editor__add-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: flex-end;
}

.pricing-entry-editor__add-field {
  flex: 1 1 160px;
  min-width: 120px;
  display: flex;
  flex-direction: column;
}

.pricing-entry-editor__add-field > label {
  font-size: 0.85rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.pricing-entry-editor__add-btn {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.pricing-entry-editor__table-wrap {
  max-height: 420px;
  overflow: auto;
}

.pricing-entry-editor__key-cell {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.82rem;
  word-break: break-all;
  max-width: 280px;
  display: inline-block;
}

.pricing-entry-editor__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.75rem;
}

html.dark .pricing-entry-editor {
  background: rgba(255, 255, 255, 0.03);
  border-color: rgba(255, 255, 255, 0.12);
}

html.dark .pricing-entry-editor__count {
  color: rgba(255, 255, 255, 0.55);
}
</style>
