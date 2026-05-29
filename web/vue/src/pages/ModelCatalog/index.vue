<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <div class="header" style="font-size: 1.2em; font-weight: 600; margin-bottom: 12px;">
        {{ t('model_catalog.title') }}
      </div>

      <div class="model-catalog-toolbar-actions">
        <div class="model-catalog-toolbar-actions__buttons">
          <a-button type="primary" @click="openAddModal">
            <template #icon><PlusOutlined /></template>
            {{ t('model_catalog.add') }}
          </a-button>
          <a-button style="background:#00b5ad;border-color:#00b5ad;color:#fff" @click="openSync = true">
            <template #icon><CloudDownloadOutlined /></template>
            {{ t('model_catalog.sync') }}
          </a-button>
          <a-button :loading="loading" :disabled="loading" @click="load">
            <template #icon><ReloadOutlined /></template>
            {{ t('model_catalog.refresh') }}
          </a-button>
        </div>
        <div class="model-catalog-toolbar-actions__meta">
          <a-checkbox v-model:checked="includeExpired">
            {{ t('model_catalog.include_expired') }}
          </a-checkbox>
          <span class="model-catalog-filter-count">
            {{ t('model_catalog.visible_rows', { matched: totalMatched, grand_total: grandTotal }) }}
          </span>
        </div>
      </div>

      <a-form class="model-catalog-filters-form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_model_id')">
              <a-input v-model:value="filters.model_id" :placeholder="t('model_catalog.filter_ph_contains')">
                <template #prefix><SearchOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_model_name')">
              <a-input v-model:value="filters.model_name" :placeholder="t('model_catalog.filter_ph_contains')">
                <template #prefix><SearchOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.filter_provider_label')">
              <ModelCatalogProviderSearch
                :placeholder="t('model_catalog.filter_ph_provider')"
                :value="filters.provider"
                @change="onProviderChange"
              />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_family')">
              <a-input v-model:value="filters.family" :placeholder="t('model_catalog.filter_ph_contains')">
                <template #prefix><SearchOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_modalities_in')">
              <a-input v-model:value="filters.modalities_in" :placeholder="t('model_catalog.filter_ph_contains')">
                <template #prefix><SearchOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('model_catalog.col_modalities_out')">
              <a-input v-model:value="filters.modalities_out" :placeholder="t('model_catalog.filter_ph_contains')">
                <template #prefix><SearchOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        <a-button size="small" @click="clearFilters">{{ t('model_catalog.filter_clear') }}</a-button>
      </a-form>

      <div class="model-catalog-table-wrap" style="overflow-x:auto;margin-top:12px;">
        <a-table
          :columns="columns"
          :data-source="rows"
          :loading="loading"
          :pagination="false"
          row-key="id"
          size="small"
          bordered
          :scroll="{ x: 'max-content' }"
        >
          <template #headerCell="{ column }">
            <span
              v-if="column.sortKey"
              style="cursor:pointer;user-select:none;"
              @click="toggleSort(column.sortKey)"
            >
              {{ column.title }}
              <CaretUpOutlined v-if="sortColumn === column.sortKey && sortDirection === 'ascending'" />
              <CaretDownOutlined v-else-if="sortColumn === column.sortKey && sortDirection === 'descending'" />
            </span>
            <span v-else>{{ column.title }}</span>
          </template>

          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'version'">v{{ record.version || 1 }}</template>
            <template v-else-if="column.key === 'status'">
              <a-tag v-if="record.status === 'current'" color="green">{{ t('model_catalog.status_current') }}</a-tag>
              <a-tag v-else color="default">{{ t('model_catalog.status_expired') }}</a-tag>
            </template>
            <template v-else-if="column.key === 'model_name'">
              <span :title="record.model_name">{{ record.model_name || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'provider_key'">{{ record.provider_key || '—' }}</template>
            <template v-else-if="column.key === 'provider_display'">
              <span :title="record.provider_display">{{ record.provider_display || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'family'">{{ record.family || '—' }}</template>
            <template v-else-if="column.key === 'modalities_in'">
              <span :title="record.modalities_in">{{ record.modalities_in || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'modalities_out'">
              <span :title="record.modalities_out">{{ record.modalities_out || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'context_limit'">{{ fmtInt(record.context_limit) }}</template>
            <template v-else-if="column.key === 'output_limit'">{{ fmtInt(record.output_limit) }}</template>
            <template v-else-if="column.key === 'cost_input'">{{ fmtPrice(record.cost_input) }}</template>
            <template v-else-if="column.key === 'cost_output'">{{ fmtPrice(record.cost_output) }}</template>
            <template v-else-if="column.key === 'cost_cache_read'">{{ fmtPrice(record.cost_cache_read) }}</template>
            <template v-else-if="column.key === 'cost_cache_write'">{{ fmtPrice(record.cost_cache_write) }}</template>
            <template v-else-if="column.key === 'reasoning'">{{ record.reasoning ? t('model_catalog.yes') : t('model_catalog.no') }}</template>
            <template v-else-if="column.key === 'tool_call'">{{ record.tool_call ? t('model_catalog.yes') : t('model_catalog.no') }}</template>
            <template v-else-if="column.key === 'temperature_ok'">{{ record.temperature_ok ? t('model_catalog.yes') : t('model_catalog.no') }}</template>
            <template v-else-if="column.key === 'attachment_ok'">{{ record.attachment_ok ? t('model_catalog.yes') : t('model_catalog.no') }}</template>
            <template v-else-if="column.key === 'open_weights'">{{ record.open_weights ? t('model_catalog.yes') : t('model_catalog.no') }}</template>
            <template v-else-if="column.key === 'knowledge_cutoff'">{{ record.knowledge_cutoff || '—' }}</template>
            <template v-else-if="column.key === 'release_date'">{{ record.release_date || '—' }}</template>
            <template v-else-if="column.key === 'last_updated'">{{ record.last_updated || '—' }}</template>
            <template v-else-if="column.key === 'npm_package'">
              <span :title="record.npm_package">{{ record.npm_package || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'api_base'">
              <span :title="record.api_base">{{ record.api_base || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'doc_url'">
              <a v-if="record.doc_url" :href="record.doc_url" target="_blank" rel="noreferrer">
                {{ t('model_catalog.link_doc') }}
              </a>
              <span v-else>—</span>
            </template>
            <template v-else-if="column.key === 'owned_by'">
              <span :title="record.owned_by">{{ record.owned_by }}</span>
            </template>
            <template v-else-if="column.key === 'enabled'">{{ record.enabled ? t('model_catalog.yes') : t('model_catalog.no') }}</template>
            <template v-else-if="column.key === 'notes'">
              <span :title="record.notes">{{ record.notes }}</span>
            </template>
            <template v-else-if="column.key === 'actions'">
              <a-button size="small" @click="openEditModal(record)">{{ t('model_catalog.edit') }}</a-button>
              <a-button size="small" danger style="margin-left:4px" @click="removeRow(record)">{{ t('model_catalog.delete') }}</a-button>
            </template>
          </template>
        </a-table>
      </div>

      <div class="model-catalog-pagination" style="display:flex;align-items:center;gap:16px;margin-top:12px;flex-wrap:wrap;">
        <div class="model-catalog-pagination__size" style="display:flex;align-items:center;gap:8px;">
          <span class="model-catalog-pagination__label">{{ t('model_catalog.page_size_label') }}</span>
          <a-select
            :value="pageSize"
            :disabled="loading"
            style="width: 90px"
            :options="pageSizeOptions"
            @update:value="onPageSizeChange"
          />
        </div>
        <a-pagination
          v-if="totalMatched > 0 && totalPages > 1"
          :current="activePage"
          :total="totalMatched"
          :page-size="pageSize"
          :show-size-changer="false"
          @change="onPageChange"
        />
        <span class="model-catalog-pagination__range">
          {{ t('model_catalog.page_range', { start: pageRangeStart, end: pageRangeEnd, total: totalMatched }) }}
        </span>
      </div>
    </a-card>

    <ModelCatalogEditModal
      :open="modalOpen"
      :edit-row="editRow"
      :saving="formSaving"
      @close="onModalClose"
      @save="saveEdit"
    />

    <a-modal
      v-model:open="openSync"
      :title="t('model_catalog.modal_sync_title')"
      width="520px"
      :mask-closable="!syncBusy"
      :keyboard="!syncBusy"
    >
      <a-form layout="vertical" autocomplete="off">
        <a-form-item :label="t('model_catalog.sync_source')">
          <a-select v-model:value="syncSource" :options="syncOptions" />
        </a-form-item>

        <template v-if="syncSource === 'models_dev'">
          <a-alert type="info" :message="t('model_catalog.sync_models_dev_url_hint')" style="margin-bottom:12px" />
          <a-form-item :label="t('model_catalog.sync_models_dev_url_label')">
            <a-input v-model:value="syncBaseURL" placeholder="https://models.dev" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'basellm'">
          <a-alert type="info" :message="t('model_catalog.sync_basellm_url_hint')" style="margin-bottom:12px" />
          <a-form-item :label="t('model_catalog.sync_basellm_url_label')">
            <a-input v-model:value="syncBaseURL" placeholder="https://basellm.github.io/llm-metadata" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'aliapi'">
          <a-alert type="info" :message="t('model_catalog.sync_aliapi_url_hint')" style="margin-bottom:12px" />
          <a-form-item :label="t('model_catalog.sync_aliapi_url_label')">
            <a-input v-model:value="syncBaseURL" placeholder="https://aliapi.me/models.html" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'anyfast'">
          <a-alert type="info" :message="t('model_catalog.sync_anyfast_url_hint')" style="margin-bottom:12px" />
          <a-form-item :label="t('model_catalog.sync_anyfast_url_label')">
            <a-input v-model:value="syncBaseURL" placeholder="https://www.anyfast.ai" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'new_api_models'">
          <a-alert type="info" :message="t('model_catalog.sync_new_api_models_hint')" style="margin-bottom:12px" />
          <a-form-item :label="t('model_catalog.sync_new_api_url_label')">
            <a-input v-model:value="syncBaseURL" placeholder="https://www.anyfast.ai" />
          </a-form-item>
          <a-form-item :label="t('model_catalog.sync_new_api_user_id')">
            <a-input v-model:value="syncNewApiUserId" placeholder="1096" v-bind="noAutofillTextProps" />
          </a-form-item>
          <a-form-item :label="t('model_catalog.sync_new_api_token')">
            <a-input-password v-model:value="syncApiKey" placeholder="控制台个人设置中的 Access Token" v-bind="noAutofillSecretProps" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'openai_compatible' || syncSource === 'openrouter'">
          <a-form-item :label="t('model_catalog.base_url')">
            <a-input v-model:value="syncBaseURL" placeholder="https://api.openai.com/v1" />
          </a-form-item>
          <a-form-item :label="t('model_catalog.api_key')">
            <a-input-password v-model:value="syncApiKey" v-bind="noAutofillSecretProps" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'anthropic' || syncSource === 'gemini'">
          <a-form-item :label="t('model_catalog.api_key')">
            <a-input-password v-model:value="syncApiKey" v-bind="noAutofillSecretProps" />
          </a-form-item>
        </template>

        <template v-if="syncSource === 'channel'">
          <a-form-item :label="t('model_catalog.channel_id')">
            <a-input v-model:value="syncChannelId" />
          </a-form-item>
        </template>
      </a-form>
      <template #footer>
        <a-button :disabled="syncBusy" @click="openSync = false">{{ t('model_catalog.btn_cancel') }}</a-button>
        <a-button type="primary" :loading="syncBusy" @click="runSync">{{ t('model_catalog.sync_run') }}</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  CloudDownloadOutlined,
  ReloadOutlined,
  SearchOutlined,
  CaretUpOutlined,
  CaretDownOutlined,
} from '@ant-design/icons-vue';
import {
  API,
  isAdmin,
  showError,
  showSuccess,
  noAutofillTextProps,
  noAutofillSecretProps,
} from '@/helpers';
import ModelCatalogEditModal, { catalogFormToPayload } from './ModelCatalogEditModal.vue';
import ModelCatalogProviderSearch from '@/components/ModelCatalogProviderSearch.vue';

const SYNC_OPENAI = 'openai_compatible';
const SYNC_MODELS_DEV = 'models_dev';
const SYNC_BASELLM = 'basellm';
const SYNC_ALIAPI = 'aliapi';
const SYNC_ANYFAST = 'anyfast';
const SYNC_NEW_API_MODELS = 'new_api_models';
const SYNC_OPENROUTER = 'openrouter';
const SYNC_ANTHROPIC = 'anthropic';
const SYNC_GEMINI = 'gemini';
const SYNC_CHANNEL = 'channel';

const MODEL_CATALOG_PAGE_SIZE_OPTIONS = [10, 20, 50, 100];
const MODEL_CATALOG_DEFAULT_PAGE_SIZE = 20;

const EMPTY_FILTERS = {
  model_id: '',
  model_name: '',
  provider: '',
  family: '',
  modalities_in: '',
  modalities_out: '',
};

function fmtPrice(n) {
  if (n === null || n === undefined || Number.isNaN(Number(n))) return '—';
  const x = Number(n);
  if (x === 0) return '0';
  return x.toFixed(6).replace(/\.?0+$/, '');
}

function fmtInt(n) {
  if (!n && n !== 0) return '—';
  return String(n);
}

const { t } = useI18n();
const router = useRouter();

const rows = ref([]);
const loading = ref(false);
const filters = reactive({ ...EMPTY_FILTERS });
const sortColumn = ref(null);
const sortDirection = ref('ascending');
const activePage = ref(1);
const pageSize = ref(MODEL_CATALOG_DEFAULT_PAGE_SIZE);
const includeExpired = ref(false);
const totalMatched = ref(0);
const grandTotal = ref(0);

const modalOpen = ref(false);
const editRow = ref(null);
const formSaving = ref(false);

const openSync = ref(false);
const syncSource = ref(SYNC_MODELS_DEV);
const syncBaseURL = ref('');
const syncApiKey = ref('');
const syncChannelId = ref('');
const syncNewApiUserId = ref('');
const syncBusy = ref(false);

const pageSizeOptions = MODEL_CATALOG_PAGE_SIZE_OPTIONS.map((n) => ({ value: n, label: String(n) }));

const columns = [
  { title: t('model_catalog.col_pk'), dataIndex: 'id', key: 'id', sortKey: 'id' },
  { title: t('model_catalog.col_model_id'), dataIndex: 'model_id', key: 'model_id', sortKey: 'model_id' },
  { title: t('model_catalog.col_version'), key: 'version', sortKey: 'version' },
  { title: t('model_catalog.col_status'), key: 'status', sortKey: 'status' },
  { title: t('model_catalog.col_model_name'), key: 'model_name', sortKey: 'model_name' },
  { title: t('model_catalog.col_provider_key'), key: 'provider_key', sortKey: 'provider_key' },
  { title: t('model_catalog.col_provider_name'), key: 'provider_display', sortKey: 'provider_display' },
  { title: t('model_catalog.col_family'), key: 'family', sortKey: 'family' },
  { title: t('model_catalog.col_modalities_in'), key: 'modalities_in', sortKey: 'modalities_in' },
  { title: t('model_catalog.col_modalities_out'), key: 'modalities_out', sortKey: 'modalities_out' },
  { title: t('model_catalog.col_context_limit'), key: 'context_limit', sortKey: 'context_limit' },
  { title: t('model_catalog.col_output_limit'), key: 'output_limit', sortKey: 'output_limit' },
  { title: t('model_catalog.col_cost_input'), key: 'cost_input', sortKey: 'cost_input' },
  { title: t('model_catalog.col_cost_output'), key: 'cost_output', sortKey: 'cost_output' },
  { title: t('model_catalog.col_cost_cache_read'), key: 'cost_cache_read', sortKey: 'cost_cache_read' },
  { title: t('model_catalog.col_cost_cache_write'), key: 'cost_cache_write', sortKey: 'cost_cache_write' },
  { title: t('model_catalog.col_reasoning'), key: 'reasoning', sortKey: 'reasoning' },
  { title: t('model_catalog.col_tool_call'), key: 'tool_call', sortKey: 'tool_call' },
  { title: t('model_catalog.col_temperature'), key: 'temperature_ok', sortKey: 'temperature_ok' },
  { title: t('model_catalog.col_attachment'), key: 'attachment_ok', sortKey: 'attachment_ok' },
  { title: t('model_catalog.col_open_weights'), key: 'open_weights', sortKey: 'open_weights' },
  { title: t('model_catalog.col_knowledge'), key: 'knowledge_cutoff', sortKey: 'knowledge_cutoff' },
  { title: t('model_catalog.col_release_date'), key: 'release_date', sortKey: 'release_date' },
  { title: t('model_catalog.col_last_updated'), key: 'last_updated', sortKey: 'last_updated' },
  { title: t('model_catalog.col_npm'), key: 'npm_package', sortKey: 'npm_package' },
  { title: t('model_catalog.col_api_base'), key: 'api_base', sortKey: 'api_base' },
  { title: t('model_catalog.col_doc'), key: 'doc_url', sortKey: 'doc_url' },
  { title: t('model_catalog.col_owned_by'), key: 'owned_by', sortKey: 'owned_by' },
  { title: t('model_catalog.col_enabled'), key: 'enabled', sortKey: 'enabled' },
  { title: t('model_catalog.col_source'), dataIndex: 'source', key: 'source', sortKey: 'source' },
  { title: t('model_catalog.col_notes'), key: 'notes', sortKey: 'notes' },
  { title: t('model_catalog.col_actions'), key: 'actions' },
];

const syncOptions = computed(() => [
  { value: SYNC_MODELS_DEV, label: t('model_catalog.sync_src_models_dev') },
  { value: SYNC_BASELLM, label: t('model_catalog.sync_src_basellm') },
  { value: SYNC_ALIAPI, label: t('model_catalog.sync_src_aliapi') },
  { value: SYNC_ANYFAST, label: t('model_catalog.sync_src_anyfast') },
  { value: SYNC_NEW_API_MODELS, label: t('model_catalog.sync_src_new_api_models') },
  { value: SYNC_OPENAI, label: t('model_catalog.sync_src_openai') },
  { value: SYNC_OPENROUTER, label: t('model_catalog.sync_src_openrouter') },
  { value: SYNC_ANTHROPIC, label: t('model_catalog.sync_src_anthropic') },
  { value: SYNC_GEMINI, label: t('model_catalog.sync_src_gemini') },
  { value: SYNC_CHANNEL, label: t('model_catalog.sync_src_channel') },
]);

const totalPages = computed(() => Math.max(1, Math.ceil(totalMatched.value / pageSize.value)));
const pageRangeStart = computed(() => (totalMatched.value === 0 ? 0 : (activePage.value - 1) * pageSize.value + 1));
const pageRangeEnd = computed(() => Math.min(activePage.value * pageSize.value, totalMatched.value));

async function load() {
  loading.value = true;
  try {
    const params = new URLSearchParams();
    params.set('page', String(activePage.value));
    params.set('page_size', String(pageSize.value));
    if (includeExpired.value) params.set('include_expired', 'true');
    const f = filters;
    if (String(f.model_id || '').trim()) params.set('filter_model_id', String(f.model_id).trim());
    if (String(f.model_name || '').trim()) params.set('filter_model_name', String(f.model_name).trim());
    if (String(f.provider || '').trim()) params.set('filter_provider', String(f.provider).trim());
    if (String(f.family || '').trim()) params.set('filter_family', String(f.family).trim());
    if (String(f.modalities_in || '').trim()) params.set('filter_modalities_in', String(f.modalities_in).trim());
    if (String(f.modalities_out || '').trim()) params.set('filter_modalities_out', String(f.modalities_out).trim());
    if (sortColumn.value) {
      params.set('sort', sortColumn.value);
      params.set('order', sortDirection.value === 'ascending' ? 'asc' : 'desc');
    }
    const res = await API.get(`/api/model_catalog?${params.toString()}`);
    const { success, message, data } = res.data || {};
    if (!success) {
      showError(message || 'load failed');
      return;
    }
    if (data && Array.isArray(data.items)) {
      rows.value = data.items;
      totalMatched.value = Number(data.total) || 0;
      grandTotal.value = Number(data.grand_total) || Number(data.total) || 0;
    } else if (Array.isArray(data)) {
      rows.value = data;
      totalMatched.value = data.length;
      grandTotal.value = data.length;
    } else {
      rows.value = [];
      totalMatched.value = 0;
      grandTotal.value = 0;
    }
  } catch (e) {
    /* axios interceptor */
  } finally {
    loading.value = false;
  }
}

watch(
  [activePage, pageSize, () => ({ ...filters }), sortColumn, sortDirection, includeExpired],
  () => {
    if (isAdmin()) load();
  },
  { deep: true }
);

watch(
  () => ({ ...filters }),
  () => {
    activePage.value = 1;
  },
  { deep: true }
);

watch(totalPages, (tp) => {
  activePage.value = Math.min(Math.max(1, activePage.value), tp);
});

function onProviderChange(value) {
  filters.provider = value;
  activePage.value = 1;
}

function clearFilters() {
  Object.assign(filters, { ...EMPTY_FILTERS });
}

function toggleSort(key) {
  activePage.value = 1;
  if (sortColumn.value !== key) {
    sortColumn.value = key;
    sortDirection.value = 'ascending';
  } else {
    sortDirection.value = sortDirection.value === 'ascending' ? 'descending' : 'ascending';
  }
}

function onPageSizeChange(value) {
  const n = Number(value);
  pageSize.value = Number.isFinite(n) ? n : MODEL_CATALOG_DEFAULT_PAGE_SIZE;
  activePage.value = 1;
}

function onPageChange(next) {
  activePage.value = Math.min(Math.max(1, next), totalPages.value);
}

function openAddModal() {
  editRow.value = null;
  modalOpen.value = true;
}

function openEditModal(row) {
  editRow.value = row;
  modalOpen.value = true;
}

function onModalClose() {
  if (!formSaving.value) modalOpen.value = false;
}

async function saveEdit(form) {
  const payload = catalogFormToPayload(form, editRow.value);
  if (!payload.model_id) {
    showError(t('model_catalog.col_model_id'));
    return;
  }
  formSaving.value = true;
  try {
    const res = editRow.value
      ? await API.put('/api/model_catalog', payload)
      : await API.post('/api/model_catalog', payload);
    if (!res.data?.success) {
      showError(res.data?.message || 'fail');
      return;
    }
    showSuccess(t('model_catalog.saved'));
    modalOpen.value = false;
    await load();
  } catch {
    /* noop */
  } finally {
    formSaving.value = false;
  }
}

function removeRow(row) {
  Modal.confirm({
    title: t('model_catalog.confirm_delete'),
    onOk: async () => {
      try {
        const res = await API.delete(`/api/model_catalog/${row.id}`);
        if (!res.data?.success) {
          showError(res.data?.message || 'fail');
          return;
        }
        await load();
      } catch {
        /* noop */
      }
    },
  });
}

async function runSync() {
  syncBusy.value = true;
  try {
    const body = { source: syncSource.value };
    if (
      syncSource.value === SYNC_MODELS_DEV ||
      syncSource.value === SYNC_BASELLM ||
      syncSource.value === SYNC_ALIAPI ||
      syncSource.value === SYNC_ANYFAST
    ) {
      const u = String(syncBaseURL.value || '').trim();
      if (u) body.base_url = u;
    } else if (syncSource.value === SYNC_OPENAI || syncSource.value === SYNC_OPENROUTER) {
      body.base_url = String(syncBaseURL.value || '').trim();
      body.api_key = String(syncApiKey.value || '').trim();
    } else if (syncSource.value === SYNC_ANTHROPIC || syncSource.value === SYNC_GEMINI) {
      body.api_key = String(syncApiKey.value || '').trim();
    } else if (syncSource.value === SYNC_NEW_API_MODELS) {
      const u = String(syncBaseURL.value || '').trim();
      if (u) body.base_url = u;
      body.api_key = String(syncApiKey.value || '').trim();
      const uid = parseInt(String(syncNewApiUserId.value || '').trim(), 10);
      body.new_api_user_id = Number.isFinite(uid) ? uid : 0;
      if (!body.api_key) {
        showError(t('model_catalog.sync_new_api_token_required'));
        syncBusy.value = false;
        return;
      }
      if (!body.new_api_user_id) {
        showError(t('model_catalog.sync_new_api_user_required'));
        syncBusy.value = false;
        return;
      }
    } else if (syncSource.value === SYNC_CHANNEL) {
      const n = parseInt(String(syncChannelId.value || '').trim(), 10);
      body.channel_id = Number.isFinite(n) ? n : 0;
    }
    const res = await API.post('/api/model_catalog/sync', body);
    const { success, message, data } = res.data || {};
    if (!success) {
      showError(message || 'sync failed');
      return;
    }
    const msg = t('model_catalog.sync_done', {
      added: data?.added ?? 0,
      updated: data?.updated ?? 0,
      total: data?.total_upstream ?? 0,
    });
    showSuccess(msg);
    openSync.value = false;
    await load();
  } catch {
    /* noop */
  } finally {
    syncBusy.value = false;
  }
}

onMounted(() => {
  if (!isAdmin()) {
    showError(t('model_catalog.forbidden'));
    router.push('/');
    return;
  }
  load();
});
</script>

<style scoped>
.model-catalog-toolbar-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.model-catalog-toolbar-actions__buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.model-catalog-toolbar-actions__meta {
  display: flex;
  align-items: center;
  gap: 12px;
}
.model-catalog-filter-count {
  opacity: 0.75;
  font-size: 13px;
}
</style>
