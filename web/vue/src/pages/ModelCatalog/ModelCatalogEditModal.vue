<template>
  <a-modal
    :open="open"
    width="900px"
    class="model-catalog-edit-modal"
    :title="editRow ? t('model_catalog.modal_edit_title') : t('model_catalog.modal_add_title')"
    @cancel="emit('close')"
  >
    <div class="model-catalog-edit-modal__content">
      <a-form id="model-catalog-edit-form" layout="vertical" @submit.prevent="handleSubmit">
        <a-divider orientation="center" class="model-catalog-form-section">
          <span>{{ t('model_catalog.form_section_basic') }}</span>
        </a-divider>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_model_id')" required>
              <a-input v-model:value="form.model_id" placeholder="gpt-4o" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_model_name')">
              <a-input v-model:value="form.model_name" :placeholder="t('model_catalog.form_model_name_ph')" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_owned_by')">
              <a-input v-model:value="form.owned_by" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_tags')">
              <a-input v-model:value="form.tags" :placeholder="t('model_catalog.form_tags_ph')" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_source')">
              <a-input v-if="editRow" v-model:value="form.source" />
              <a-input v-else value="manual" readonly />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_enabled')" class="model-catalog-form-enabled">
              <a-switch v-model:checked="form.enabled" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-divider orientation="center" class="model-catalog-form-section">
          <span>{{ t('model_catalog.form_section_provider') }}</span>
        </a-divider>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_provider_key')">
              <a-input v-model:value="form.provider_key" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_provider_name')">
              <a-input v-model:value="form.provider_display" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_family')">
              <a-input v-model:value="form.family" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-divider orientation="center" class="model-catalog-form-section">
          <span>{{ t('model_catalog.form_section_capability') }}</span>
        </a-divider>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_modalities_in')">
              <a-input v-model:value="form.modalities_in" placeholder="text,image" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_modalities_out')">
              <a-input v-model:value="form.modalities_out" placeholder="text" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_context_limit')">
              <a-input v-model:value="form.context_limit" type="number" :min="0" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_output_limit')">
              <a-input v-model:value="form.output_limit" type="number" :min="0" />
            </a-form-item>
          </a-col>
        </a-row>
        <div class="model-catalog-form-bool-row">
          <a-checkbox v-model:checked="form.reasoning">{{ t('model_catalog.col_reasoning') }}</a-checkbox>
          <a-checkbox v-model:checked="form.tool_call">{{ t('model_catalog.col_tool_call') }}</a-checkbox>
          <a-checkbox v-model:checked="form.temperature_ok">{{ t('model_catalog.col_temperature') }}</a-checkbox>
          <a-checkbox v-model:checked="form.attachment_ok">{{ t('model_catalog.col_attachment') }}</a-checkbox>
          <a-checkbox v-model:checked="form.open_weights">{{ t('model_catalog.col_open_weights') }}</a-checkbox>
        </div>

        <a-divider orientation="center" class="model-catalog-form-section">
          <span>{{ t('model_catalog.form_section_pricing') }}</span>
        </a-divider>
        <a-row :gutter="16">
          <a-col :span="6">
            <a-form-item :label="t('model_catalog.col_cost_input')">
              <a-input v-model:value="form.cost_input" type="number" step="any" :min="0" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item :label="t('model_catalog.col_cost_output')">
              <a-input v-model:value="form.cost_output" type="number" step="any" :min="0" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item :label="t('model_catalog.col_cost_cache_read')">
              <a-input v-model:value="form.cost_cache_read" type="number" step="any" :min="0" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item :label="t('model_catalog.col_cost_cache_write')">
              <a-input v-model:value="form.cost_cache_write" type="number" step="any" :min="0" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-divider orientation="center" class="model-catalog-form-section">
          <span>{{ t('model_catalog.form_section_meta') }}</span>
        </a-divider>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_knowledge')">
              <a-input v-model:value="form.knowledge_cutoff" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_release_date')">
              <a-input v-model:value="form.release_date" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_last_updated')">
              <a-input v-model:value="form.last_updated" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_npm')">
              <a-input v-model:value="form.npm_package" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('model_catalog.col_api_base')">
              <a-input v-model:value="form.api_base" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('model_catalog.col_doc')">
          <a-input v-model:value="form.doc_url" placeholder="https://" />
        </a-form-item>
        <a-form-item :label="t('model_catalog.col_notes')">
          <a-textarea v-model:value="form.notes" :rows="3" />
        </a-form-item>
      </a-form>
    </div>

    <template #footer>
      <a-button type="button" :disabled="saving" @click="emit('close')">
        {{ t('model_catalog.btn_cancel') }}
      </a-button>
      <a-button type="primary" :loading="saving" :disabled="saving" @click="handleSubmit">
        {{ t('model_catalog.btn_save') }}
      </a-button>
    </template>
  </a-modal>
</template>

<script>
export const emptyCatalogForm = () => ({
  model_id: '',
  model_name: '',
  owned_by: '',
  enabled: true,
  source: 'manual',
  tags: '',
  provider_key: '',
  provider_display: '',
  family: '',
  modalities_in: '',
  modalities_out: '',
  context_limit: '',
  output_limit: '',
  cost_input: '',
  cost_output: '',
  cost_cache_read: '',
  cost_cache_write: '',
  reasoning: false,
  tool_call: false,
  temperature_ok: false,
  attachment_ok: false,
  open_weights: false,
  knowledge_cutoff: '',
  release_date: '',
  last_updated: '',
  npm_package: '',
  api_base: '',
  doc_url: '',
  notes: '',
});

export function rowToCatalogForm(row) {
  if (!row) return emptyCatalogForm();
  const numStr = (n) => {
    if (n === null || n === undefined || Number.isNaN(Number(n))) return '';
    if (Number(n) === 0) return '';
    return String(n);
  };
  return {
    model_id: row.model_id || '',
    model_name: row.model_name || '',
    owned_by: row.owned_by || '',
    enabled: !!row.enabled,
    source: row.source || 'manual',
    tags: row.tags || '',
    provider_key: row.provider_key || '',
    provider_display: row.provider_display || '',
    family: row.family || '',
    modalities_in: row.modalities_in || '',
    modalities_out: row.modalities_out || '',
    context_limit: row.context_limit ? String(row.context_limit) : '',
    output_limit: row.output_limit ? String(row.output_limit) : '',
    cost_input: numStr(row.cost_input),
    cost_output: numStr(row.cost_output),
    cost_cache_read: numStr(row.cost_cache_read),
    cost_cache_write: numStr(row.cost_cache_write),
    reasoning: !!row.reasoning,
    tool_call: !!row.tool_call,
    temperature_ok: !!row.temperature_ok,
    attachment_ok: !!row.attachment_ok,
    open_weights: !!row.open_weights,
    knowledge_cutoff: row.knowledge_cutoff || '',
    release_date: row.release_date || '',
    last_updated: row.last_updated || '',
    npm_package: row.npm_package || '',
    api_base: row.api_base || '',
    doc_url: row.doc_url || '',
    notes: row.notes || '',
  };
}

function parseIntField(v) {
  const s = String(v ?? '').trim();
  if (!s) return 0;
  const n = parseInt(s, 10);
  return Number.isFinite(n) ? n : 0;
}

function parseFloatField(v) {
  const s = String(v ?? '').trim();
  if (!s) return 0;
  const n = parseFloat(s);
  return Number.isFinite(n) ? n : 0;
}

export function catalogFormToPayload(form, editRow) {
  const body = {
    model_id: String(form.model_id || '').trim(),
    model_name: String(form.model_name || '').trim(),
    owned_by: String(form.owned_by || '').trim(),
    enabled: !!form.enabled,
    notes: String(form.notes || '').trim(),
    provider_key: String(form.provider_key || '').trim(),
    provider_display: String(form.provider_display || '').trim(),
    family: String(form.family || '').trim(),
    npm_package: String(form.npm_package || '').trim(),
    api_base: String(form.api_base || '').trim(),
    doc_url: String(form.doc_url || '').trim(),
    modalities_in: String(form.modalities_in || '').trim(),
    modalities_out: String(form.modalities_out || '').trim(),
    context_limit: parseIntField(form.context_limit),
    output_limit: parseIntField(form.output_limit),
    cost_input: parseFloatField(form.cost_input),
    cost_output: parseFloatField(form.cost_output),
    cost_cache_read: parseFloatField(form.cost_cache_read),
    cost_cache_write: parseFloatField(form.cost_cache_write),
    reasoning: !!form.reasoning,
    tool_call: !!form.tool_call,
    temperature_ok: !!form.temperature_ok,
    attachment_ok: !!form.attachment_ok,
    open_weights: !!form.open_weights,
    knowledge_cutoff: String(form.knowledge_cutoff || '').trim(),
    release_date: String(form.release_date || '').trim(),
    last_updated: String(form.last_updated || '').trim(),
    tags: String(form.tags || '').trim(),
  };
  if (editRow) {
    body.id = editRow.id;
    body.source = String(form.source || editRow.source || 'manual').trim();
  }
  return body;
}
</script>

<script setup>
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

const props = defineProps({
  open: { type: Boolean, default: false },
  editRow: { type: Object, default: null },
  saving: { type: Boolean, default: false },
});
const emit = defineEmits(['close', 'save']);

const { t } = useI18n();

const form = ref(emptyCatalogForm());

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      form.value = rowToCatalogForm(props.editRow);
    }
  }
);

function handleSubmit() {
  emit('save', form.value);
}
</script>

<style scoped>
.model-catalog-edit-modal__content {
  max-height: 70vh;
  overflow-y: auto;
}
.model-catalog-form-bool-row {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1rem;
}
</style>
