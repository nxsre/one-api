<template>
  <a-card class="upstream-ratio-sync">
    <h4>{{ t('setting.operation.ratio.upstream_sync.title') }}</h4>
    <a-alert
      type="info"
      :message="t('setting.operation.ratio.upstream_sync.hint')"
      show-icon
      style="margin-bottom: 1rem"
    />

    <form @submit.prevent>
      <a-select
        mode="multiple"
        show-search
        allow-clear
        style="width: 100%"
        :options="options"
        :value="selected"
        option-filter-prop="label"
        :placeholder="t('setting.operation.ratio.upstream_sync.select_placeholder')"
        @update:value="(v) => (selected = v)"
      />
      <div class="upstream-ratio-sync__actions">
        <a-button
          type="primary"
          :loading="loading"
          :disabled="loading"
          @click="fetchUpstream"
        >
          {{ t('setting.operation.ratio.upstream_sync.fetch') }}
        </a-button>
        <a-button v-if="activeBatchId" @click="openBatchReview(activeBatchId)">
          {{ t('setting.operation.ratio.upstream_sync.open_latest_batch', { id: activeBatchId }) }}
        </a-button>
        <a-button @click="compareOpen = true">
          {{ t('setting.operation.ratio.upstream_sync.compare_open') }}
        </a-button>
      </div>
    </form>

    <template v-if="batches.length > 0">
      <h5 class="upstream-ratio-sync__section-title">
        {{ t('setting.operation.ratio.upstream_sync.batch_history') }}
      </h5>
      <a-table
        :columns="batchColumns"
        :data-source="batches.slice(0, 10)"
        :pagination="false"
        size="small"
        bordered
        row-key="id"
        :row-class-name="(record) => (record.id === activeBatchId ? 'ant-table-row-positive' : '')"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'id'">#{{ record.id }}</template>
          <template v-else-if="column.key === 'time'">
            {{ record.created_time ? timestamp2string(record.created_time) : '—' }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="statusColor(record.status)" :title="record.error_message || undefined">
              {{ record.status }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'diff'">{{ record.diff_count || 0 }}</template>
          <template v-else-if="column.key === 'action'">
            <a-button
              size="small"
              :disabled="record.status !== 'completed'"
              @click="openBatchReview(record.id)"
            >
              {{ t('setting.operation.ratio.upstream_sync.review_batch') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </template>

    <h5 class="upstream-ratio-sync__section-title">
      {{ t('setting.operation.ratio.upstream_sync.versions_title') }}
    </h5>
    <a-alert
      v-if="versionLoading && versionBlocks.length === 0"
      type="info"
      :message="t('setting.operation.ratio.upstream_sync.versions_loading')"
    />
    <a-table
      v-else
      :columns="versionColumns"
      :data-source="versionBlocks"
      :pagination="false"
      size="small"
      bordered
      row-key="block_id"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'block'">{{ record.title }}</template>
        <template v-else-if="column.key === 'active'">
          <a-tag v-if="record.active_version > 0" color="green">v{{ record.active_version }}</a-tag>
          <span v-else>—</span>
        </template>
        <template v-else-if="column.key === 'count'">{{ record.version_count || 0 }}</template>
        <template v-else-if="column.key === 'action'">
          <a-button size="small" @click="openBlockVersions(record)">
            {{ t('setting.operation.ratio.upstream_sync.manage_versions') }}
          </a-button>
        </template>
      </template>
    </a-table>

    <!-- Review modal -->
    <a-modal
      v-model:open="reviewOpen"
      :title="t('setting.operation.ratio.upstream_sync.diff_title')"
      width="min(96vw, 1180px)"
      wrap-class-name="upstream-ratio-sync-diff-modal"
      :body-style="{ maxHeight: 'min(72vh, 720px)', overflow: 'auto' }"
    >
      <div
        v-if="(activeBatch?.test_results || []).length > 0"
        class="upstream-ratio-sync__fetch-status"
      >
        <a-tag
          v-for="r in (activeBatch.test_results || [])"
          :key="r.name"
          :color="r.status === 'success' ? 'green' : 'red'"
          :title="r.error || undefined"
        >
          <CheckCircleOutlined v-if="r.status === 'success'" />
          <CloseCircleOutlined v-else />
          {{ r.name }}{{ r.error ? `: ${r.error}` : '' }}
        </a-tag>
      </div>
      <div class="upstream-ratio-sync__toolbar">
        <div class="upstream-ratio-sync__toolbar-row upstream-ratio-sync__toolbar-row--review">
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.batch_select') }}</label>
            <a-select
              style="width: 100%"
              :options="batchOptions"
              :value="activeBatchId || undefined"
              @update:value="openBatchReview"
            />
          </div>
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.model_search') }}</label>
            <a-input
              :value="modelSearch"
              :placeholder="t('setting.operation.ratio.editor.filter_placeholder')"
              @update:value="(v) => (modelSearch = v)"
              @press-enter="onModelSearchEnter"
            />
          </div>
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.default_upstream') }}</label>
            <a-select
              style="width: 100%"
              :options="defaultUpstreamOptions"
              :value="defaultUpstream"
              @update:value="applyDefaultUpstreamToPage"
            />
          </div>
        </div>
        <div class="upstream-ratio-sync__toolbar-row upstream-ratio-sync__toolbar-row--review">
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.apply_source') }}</label>
            <a-input
              :value="applySource"
              :placeholder="defaultUpstream"
              @update:value="(v) => (applySource = v)"
            />
          </div>
          <div class="upstream-ratio-sync__toolbar-check">
            <a-checkbox
              :checked="activateOnApply"
              @change="(e) => (activateOnApply = e.target.checked)"
            >
              {{ t('setting.operation.ratio.upstream_sync.activate_on_apply') }}
            </a-checkbox>
          </div>
        </div>
      </div>
      <div class="upstream-ratio-sync__summary-row">
        <div class="upstream-ratio-sync__summary">
          {{ t('setting.operation.ratio.upstream_sync.diff_summary', {
            total: diffTotal,
            selected: activeBatch?.selection_count ?? selectedCount,
          }) }}
        </div>
        <PagerBar
          :page="diffPage"
          :total-pages="diffTotalPages"
          :total="diffTotal"
          :loading="diffLoading"
          @page-change="(p) => loadDiffPage(activeBatchId, p, modelQuery)"
        />
      </div>
      <div class="upstream-ratio-sync__table-wrap">
        <a-spin v-if="diffLoading" />
        <a-alert
          v-else-if="diffRows.length === 0"
          type="info"
          :message="t('setting.operation.ratio.upstream_sync.no_diff')"
        />
        <DiffTable v-else :rows="diffRows" :read-only="false" />
      </div>
      <div class="upstream-ratio-sync__pager-footer">
        <PagerBar
          :page="diffPage"
          :total-pages="diffTotalPages"
          :total="diffTotal"
          :loading="diffLoading"
          @page-change="(p) => loadDiffPage(activeBatchId, p, modelQuery)"
        />
      </div>
      <template #footer>
        <a-button @click="reviewOpen = false">
          {{ t('setting.operation.ratio.upstream_sync.close') }}
        </a-button>
        <a-button
          :loading="savingSelections"
          :disabled="savingSelections || diffTotal === 0"
          @click="selectAllBatch"
        >
          {{ t('setting.operation.ratio.upstream_sync.select_all_batch') }}
        </a-button>
        <a-button
          :loading="savingSelections"
          :disabled="savingSelections"
          @click="saveSelections"
        >
          {{ t('setting.operation.ratio.upstream_sync.save_selections') }}
        </a-button>
        <a-button
          type="primary"
          :loading="applying"
          :disabled="applying || (activeBatch?.selection_count ?? 0) === 0 || diffTotal === 0"
          @click="applySelected"
        >
          {{ t('setting.operation.ratio.upstream_sync.apply_from_batch') }}
        </a-button>
      </template>
    </a-modal>

    <!-- Compare modal -->
    <a-modal
      v-model:open="compareOpen"
      :title="t('setting.operation.ratio.upstream_sync.compare_title')"
      width="min(96vw, 1180px)"
      wrap-class-name="upstream-ratio-sync-diff-modal"
      :body-style="{ maxHeight: 'min(72vh, 720px)', overflow: 'auto' }"
    >
      <div class="upstream-ratio-sync__toolbar">
        <div class="upstream-ratio-sync__toolbar-row upstream-ratio-sync__toolbar-row--compare">
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.compare_left') }}</label>
            <a-select
              style="width: 100%"
              :options="compareSideOptions"
              :value="compareLeft"
              @update:value="(v) => (compareLeft = v)"
            />
          </div>
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.compare_right') }}</label>
            <a-select
              style="width: 100%"
              :options="compareSideOptions"
              :value="compareRight || undefined"
              @update:value="(v) => (compareRight = v)"
            />
          </div>
          <div class="upstream-ratio-sync__toolbar-field">
            <label>{{ t('setting.operation.ratio.upstream_sync.model_search') }}</label>
            <a-input
              :value="compareModelQuery"
              @update:value="(v) => (compareModelQuery = v)"
            />
          </div>
          <div class="upstream-ratio-sync__toolbar-check">
            <a-button
              type="primary"
              :loading="compareLoading"
              :disabled="compareLoading || !compareRight"
              @click="loadComparePage(1, compareModelQuery)"
            >
              {{ t('setting.operation.ratio.upstream_sync.compare_run') }}
            </a-button>
          </div>
        </div>
      </div>
      <div class="upstream-ratio-sync__summary-row">
        <div class="upstream-ratio-sync__summary">
          {{ t('setting.operation.ratio.upstream_sync.compare_summary', { total: compareTotal }) }}
        </div>
        <PagerBar
          :page="comparePage"
          :total-pages="compareTotalPages"
          :total="compareTotal"
          :loading="compareLoading"
          @page-change="(p) => loadComparePage(p, compareModelQuery)"
        />
      </div>
      <div class="upstream-ratio-sync__table-wrap">
        <a-spin v-if="compareLoading" />
        <a-alert
          v-else-if="compareRows.length === 0"
          type="info"
          :message="t('setting.operation.ratio.upstream_sync.compare_empty')"
        />
        <DiffTable v-else :rows="compareRows" :read-only="true" />
      </div>
      <div class="upstream-ratio-sync__pager-footer">
        <PagerBar
          :page="comparePage"
          :total-pages="compareTotalPages"
          :total="compareTotal"
          :loading="compareLoading"
          @page-change="(p) => loadComparePage(p, compareModelQuery)"
        />
      </div>
      <template #footer>
        <a-button @click="compareOpen = false">
          {{ t('setting.operation.ratio.upstream_sync.close') }}
        </a-button>
      </template>
    </a-modal>

    <!-- Version history modal -->
    <a-modal
      v-model:open="versionModalOpen"
      :title="`${versionModalBlock?.title || ''} — ${t('setting.operation.ratio.upstream_sync.version_history')}`"
      width="640px"
      :body-style="{ maxHeight: '60vh', overflow: 'auto' }"
    >
      <a-alert
        v-if="blockVersions.length === 0"
        type="info"
        :message="t('setting.operation.ratio.upstream_sync.no_versions')"
      />
      <a-table
        v-else
        :columns="versionHistoryColumns"
        :data-source="sortedBlockVersions"
        :pagination="false"
        size="small"
        bordered
        row-key="id"
        :row-class-name="(record) => (record.id === blockActiveVersion ? 'ant-table-row-positive' : '')"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'id'">
            v{{ record.id }}
            <a-tag v-if="record.id === blockActiveVersion" color="green" style="margin-left: 4px">
              {{ t('setting.operation.ratio.upstream_sync.active') }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'label'">{{ record.label }}</template>
          <template v-else-if="column.key === 'source'">{{ record.source || '—' }}</template>
          <template v-else-if="column.key === 'time'">
            {{ record.created_at ? timestamp2string(record.created_at) : '—' }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button
              v-if="record.id !== blockActiveVersion"
              size="small"
              type="primary"
              :loading="activatingVersion"
              :disabled="activatingVersion"
              @click="activateVersion(record.id)"
            >
              {{ t('setting.operation.ratio.upstream_sync.set_active') }}
            </a-button>
          </template>
        </template>
      </a-table>
      <template #footer>
        <a-button @click="versionModalOpen = false">
          {{ t('setting.operation.ratio.upstream_sync.close') }}
        </a-button>
      </template>
    </a-modal>
  </a-card>
</template>

<script setup>
import { computed, h, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons-vue';
import { Button as AButton } from 'ant-design-vue';
import { API, showError, showSuccess, timestamp2string } from '@/helpers';

const { t } = useI18n();

const FIELD_LABELS = {
  model_ratio: '模型倍率',
  completion_ratio: '补全倍率',
  model_price: '模型价格',
  cache_ratio: '缓存倍率',
  create_cache_ratio: '缓存创建倍率',
  image_ratio: '图片倍率',
  audio_ratio: '音频倍率',
  audio_completion_ratio: '音频补全倍率',
  billing_mode: '计费模式',
  billing_expr: '分层计费表达式',
};

const PRICING_APPLIED_EVENT = 'one-api-pricing-applied';
const PAGE_SIZE = 50;

function rowKey(model, field) {
  return `${model}::${field}`;
}

function upstreamNamesFromRow(row) {
  return Object.keys(row.upstreams || {}).filter(
    (n) => row.upstreams[n] !== 'same' && row.upstreams[n] != null
  );
}

function pickDefaultUpstream(row, preferred, upstreamNames) {
  const names =
    upstreamNames || upstreamNamesFromRow(row);
  if (preferred && names.includes(preferred)) return preferred;
  const trusted = names.find((n) => row.confidence?.[n] !== false);
  return trusted || names[0] || '';
}

function formatValue(v) {
  if (v == null) return '—';
  if (typeof v === 'object') return JSON.stringify(v);
  if (typeof v === 'number' && Number.isFinite(v)) {
    if (Number.isInteger(v)) return String(v);
    return String(parseFloat(v.toFixed(6)));
  }
  if (typeof v === 'string') {
    const trimmed = v.trim();
    if (trimmed !== '' && !Number.isNaN(Number(trimmed))) {
      const n = Number(trimmed);
      if (Number.isInteger(n)) return String(n);
      return String(parseFloat(n.toFixed(6)));
    }
  }
  return String(v);
}

const channels = ref([]);
const selected = ref([]);
const loading = ref(false);
const applying = ref(false);
const savingSelections = ref(false);

const batches = ref([]);
const activeBatchId = ref(null);
const activeBatch = ref(null);
const reviewOpen = ref(false);

const diffRows = ref([]);
const diffTotal = ref(0);
const diffTotalPages = ref(1);
const diffPage = ref(1);
const diffLoading = ref(false);
const modelSearch = ref('');
const modelQuery = ref('');

const selectionDraft = ref({});
const defaultUpstream = ref('');
const applySource = ref('');
const activateOnApply = ref(true);

const compareOpen = ref(false);
const compareLeft = ref('current');
const compareRight = ref('');
const compareRows = ref([]);
const compareTotal = ref(0);
const compareTotalPages = ref(1);
const comparePage = ref(1);
const compareModelQuery = ref('');
const compareLoading = ref(false);

const versionBlocks = ref([]);
const versionLoading = ref(false);
const versionModalOpen = ref(false);
const versionModalBlock = ref(null);
const blockVersions = ref([]);
const blockActiveVersion = ref(0);
const activatingVersion = ref(false);

const options = computed(() =>
  channels.value.map((ch) => ({
    value: ch.id,
    label: ch.name + (ch.base_url ? ` (${ch.base_url})` : ''),
  }))
);

const batchOptions = computed(() =>
  batches.value.map((b) => ({
    value: b.id,
    label: `#${b.id} · ${timestamp2string(b.created_time)} · ${b.diff_count || 0} 项 · ${b.status}`,
  }))
);

const defaultUpstreamOptions = computed(() =>
  (activeBatch.value?.test_results || [])
    .filter((r) => r.status === 'success')
    .map((r) => ({ value: r.name, label: r.name }))
);

const compareSideOptions = computed(() => {
  const opts = [{ value: 'current', label: '当前生效价目' }];
  batches.value
    .filter((b) => b.status === 'completed')
    .forEach((b) => {
      opts.push({ value: `batch:${b.id}`, label: `批次 #${b.id}` });
    });
  versionBlocks.value.forEach((block) => {
    opts.push({
      value: `version:${block.block_id}:${block.active_version || 0}`,
      label: `${block.title} v${block.active_version || '—'}`,
      disabled: !block.active_version,
    });
  });
  return opts;
});

const selectedCount = computed(
  () => Object.values(selectionDraft.value).filter((x) => x.selected).length
);

const sortedBlockVersions = computed(() =>
  [...blockVersions.value].sort((a, b) => b.id - a.id)
);

function statusColor(status) {
  if (status === 'completed') return 'green';
  if (status === 'failed') return 'red';
  return 'gold';
}

const batchColumns = [
  { title: 'ID', key: 'id' },
  { title: t('setting.operation.ratio.upstream_sync.col_time'), key: 'time' },
  { title: t('setting.operation.ratio.upstream_sync.col_status'), key: 'status' },
  { title: t('setting.operation.ratio.upstream_sync.col_diff_count'), key: 'diff' },
  { title: '', key: 'action', width: 120 },
];

const versionColumns = [
  { title: t('setting.operation.ratio.upstream_sync.col_block'), key: 'block' },
  { title: t('setting.operation.ratio.upstream_sync.col_active'), key: 'active' },
  { title: t('setting.operation.ratio.upstream_sync.col_count'), key: 'count' },
  { title: '', key: 'action', width: 140 },
];

const versionHistoryColumns = [
  { title: 'ID', key: 'id' },
  { title: t('setting.operation.ratio.upstream_sync.col_label'), key: 'label' },
  { title: t('setting.operation.ratio.upstream_sync.col_source'), key: 'source' },
  { title: t('setting.operation.ratio.upstream_sync.col_time'), key: 'time' },
  { title: '', key: 'action', width: 120 },
];

async function loadVersionBlocks() {
  versionLoading.value = true;
  try {
    const res = await API.get('/api/ratio_sync/versions');
    if (res.data?.success) versionBlocks.value = res.data.data || [];
  } catch {
    /* ignore */
  }
  versionLoading.value = false;
}

async function loadBatches() {
  try {
    const res = await API.get('/api/ratio_sync/batches?limit=30');
    if (res.data?.success) {
      batches.value = res.data.data || [];
    }
  } catch {
    /* ignore */
  }
}

async function loadBatchDetail(batchId) {
  if (!batchId) return null;
  const res = await API.get(`/api/ratio_sync/batches/${batchId}`);
  if (res.data?.success) {
    activeBatch.value = res.data.data;
    return res.data.data;
  }
  return null;
}

async function loadDiffPage(batchId, page, model) {
  if (!batchId) return;
  diffLoading.value = true;
  try {
    const res = await API.get(`/api/ratio_sync/batches/${batchId}/diffs`, {
      params: { page, page_size: PAGE_SIZE, model: model || undefined },
    });
    if (res.data?.success) {
      const data = res.data.data || {};
      diffRows.value = data.items || [];
      const total = data.total || 0;
      diffTotal.value = total;
      diffTotalPages.value = Math.max(
        1,
        data.total_pages || Math.ceil(total / PAGE_SIZE) || 1
      );
      diffPage.value = data.page || page;
      const draft = {};
      (data.items || []).forEach((row) => {
        const key = rowKey(row.model, row.field);
        draft[key] = {
          model: row.model,
          field: row.field,
          upstream_name:
            row.upstream_name ||
            pickDefaultUpstream(row, defaultUpstream.value, upstreamNamesFromRow(row)),
          selected: row.selected ?? false,
        };
      });
      selectionDraft.value = { ...selectionDraft.value, ...draft };
    } else {
      showError(res.data?.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
    }
  } catch (e) {
    showError(e.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
  } finally {
    diffLoading.value = false;
  }
}

onMounted(() => {
  API.get('/api/ratio_sync/channels')
    .then((res) => {
      if (res.data?.success) channels.value = res.data.data || [];
    })
    .catch(() => {});
  loadVersionBlocks();
  loadBatches();
  API.get('/api/ratio_sync/batches/latest')
    .then((res) => {
      if (res.data?.success && res.data.data?.id) {
        activeBatchId.value = res.data.data.id;
        activeBatch.value = res.data.data;
      }
    })
    .catch(() => {});
});

async function pollBatchUntilDone(batchId) {
  for (let i = 0; i < 120; i++) {
    const detail = await loadBatchDetail(batchId);
    if (!detail) break;
    if (detail.status === 'completed' || detail.status === 'failed') {
      return detail;
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  return null;
}

async function fetchUpstream() {
  if (selected.value.length === 0) {
    showError(t('setting.operation.ratio.upstream_sync.select_upstream'));
    return;
  }
  loading.value = true;
  try {
    const res = await API.post('/api/ratio_sync/fetch', {
      channel_ids: selected.value,
      timeout: 15,
    });
    const batchId = res.data?.data?.batch_id;
    if (!batchId) {
      showError(res.data?.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
      loading.value = false;
      return;
    }
    activeBatchId.value = batchId;
    const detail = await pollBatchUntilDone(batchId);
    await loadBatches();
    if (!detail) {
      showError(t('setting.operation.ratio.upstream_sync.fetch_timeout'));
      loading.value = false;
      return;
    }
    if (detail.status === 'failed') {
      showError(detail.error_message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
      loading.value = false;
      return;
    }
    const pref =
      (detail.test_results || []).find((r) => r.status === 'success')?.name || '';
    defaultUpstream.value = pref;
    applySource.value = pref;
    selectionDraft.value = {};
    modelSearch.value = '';
    modelQuery.value = '';
    diffPage.value = 1;
    await loadDiffPage(batchId, 1, '');
    reviewOpen.value = true;
    const hasError = (detail.test_results || []).some((r) => r.status === 'error');
    showSuccess(
      hasError
        ? t('setting.operation.ratio.upstream_sync.fetch_partial')
        : t('setting.operation.ratio.upstream_sync.fetch_ok')
    );
  } catch (e) {
    showError(e.message || t('setting.operation.ratio.upstream_sync.fetch_fail'));
  }
  loading.value = false;
}

async function openBatchReview(batchId) {
  if (!batchId) return;
  activeBatchId.value = batchId;
  const batch = await loadBatchDetail(batchId);
  const count = batch?.diff_count || 0;
  if (count > 0) {
    diffTotal.value = count;
    diffTotalPages.value = Math.max(1, Math.ceil(count / PAGE_SIZE));
  }
  diffPage.value = 1;
  modelQuery.value = '';
  modelSearch.value = '';
  await loadDiffPage(batchId, 1, '');
  reviewOpen.value = true;
}

function onModelSearchEnter() {
  modelQuery.value = modelSearch.value.trim();
  loadDiffPage(activeBatchId.value, 1, modelSearch.value.trim());
}

function applyDefaultUpstreamToPage(name) {
  defaultUpstream.value = name;
  const next = { ...selectionDraft.value };
  diffRows.value.forEach((row) => {
    const key = rowKey(row.model, row.field);
    next[key] = {
      model: row.model,
      field: row.field,
      upstream_name: pickDefaultUpstream(row, name, upstreamNamesFromRow(row)),
      selected: next[key]?.selected ?? false,
    };
  });
  selectionDraft.value = next;
}

function toggleRowSelection(row, checked) {
  const key = rowKey(row.model, row.field);
  const prev = selectionDraft.value[key];
  selectionDraft.value = {
    ...selectionDraft.value,
    [key]: {
      model: row.model,
      field: row.field,
      upstream_name:
        prev?.upstream_name ||
        pickDefaultUpstream(row, defaultUpstream.value, upstreamNamesFromRow(row)),
      selected: checked,
    },
  };
}

function setRowUpstream(row, value) {
  const key = rowKey(row.model, row.field);
  const prev = selectionDraft.value[key];
  selectionDraft.value = {
    ...selectionDraft.value,
    [key]: {
      model: row.model,
      field: row.field,
      upstream_name: value,
      selected: prev?.selected ?? false,
    },
  };
}

async function selectAllBatch() {
  if (!activeBatchId.value) return;
  savingSelections.value = true;
  try {
    const res = await API.post(
      `/api/ratio_sync/batches/${activeBatchId.value}/selections/select_all`,
      { upstream_name: defaultUpstream.value, selected: true }
    );
    if (res.data?.success) {
      showSuccess(t('setting.operation.ratio.upstream_sync.select_all_ok'));
      await loadBatchDetail(activeBatchId.value);
      await loadDiffPage(activeBatchId.value, diffPage.value, modelQuery.value);
    } else {
      showError(res.data?.message);
    }
  } catch (e) {
    showError(e.message);
  }
  savingSelections.value = false;
}

async function saveSelections() {
  if (!activeBatchId.value) return;
  const items = Object.values(selectionDraft.value).filter((x) => x.selected);
  if (items.length === 0) {
    showError(t('setting.operation.ratio.upstream_sync.apply_none'));
    return;
  }
  savingSelections.value = true;
  try {
    const res = await API.put(`/api/ratio_sync/batches/${activeBatchId.value}/selections`, {
      items,
    });
    if (res.data?.success) {
      showSuccess(res.data.message || t('setting.operation.ratio.upstream_sync.selections_saved'));
      await loadBatchDetail(activeBatchId.value);
    } else {
      showError(res.data?.message);
    }
  } catch (e) {
    showError(e.message);
  }
  savingSelections.value = false;
}

async function applySelected() {
  if (!activeBatchId.value) return;
  const savedCount = activeBatch.value?.selection_count ?? 0;
  if (savedCount === 0) {
    showError(t('setting.operation.ratio.upstream_sync.apply_no_saved'));
    return;
  }
  if (diffTotal.value === 0) {
    showError(t('setting.operation.ratio.upstream_sync.no_diff'));
    return;
  }
  applying.value = true;
  try {
    const res = await API.post(`/api/ratio_sync/batches/${activeBatchId.value}/apply`, {
      activate: activateOnApply.value,
      source: applySource.value || defaultUpstream.value,
      label: t('setting.operation.ratio.upstream_sync.apply_label_default'),
    });
    if (res.data?.success) {
      showSuccess(res.data.message || t('setting.operation.ratio.upstream_sync.apply_ok'));
      window.dispatchEvent(new CustomEvent(PRICING_APPLIED_EVENT));
      await loadVersionBlocks();
      reviewOpen.value = false;
    } else {
      showError(res.data?.message || t('setting.operation.ratio.upstream_sync.apply_fail'));
    }
  } catch (e) {
    showError(e.message || t('setting.operation.ratio.upstream_sync.apply_fail'));
  }
  applying.value = false;
}

async function loadComparePage(page = 1, model = compareModelQuery.value) {
  if (!compareLeft.value || !compareRight.value) return;
  compareLoading.value = true;
  try {
    const res = await API.get('/api/ratio_sync/compare/diffs', {
      params: {
        left: compareLeft.value,
        right: compareRight.value,
        page,
        page_size: PAGE_SIZE,
        model: model || undefined,
      },
    });
    if (res.data?.success) {
      const data = res.data.data || {};
      compareRows.value = data.items || [];
      const total = data.total || 0;
      compareTotal.value = total;
      compareTotalPages.value = Math.max(
        1,
        data.total_pages || Math.ceil(total / PAGE_SIZE) || 1
      );
      comparePage.value = data.page || page;
    } else {
      showError(res.data?.message);
    }
  } catch (e) {
    showError(e.message);
  }
  compareLoading.value = false;
}

async function openBlockVersions(block) {
  versionModalBlock.value = block;
  versionModalOpen.value = true;
  blockVersions.value = [];
  try {
    const res = await API.get(`/api/ratio_sync/versions/${block.block_id}`);
    if (res.data?.success) {
      blockVersions.value = res.data.data?.versions || [];
      blockActiveVersion.value = res.data.data?.active_version || 0;
    } else {
      showError(res.data?.message);
    }
  } catch (e) {
    showError(e.message);
  }
}

async function activateVersion(versionId) {
  if (!versionModalBlock.value) return;
  activatingVersion.value = true;
  try {
    const res = await API.post('/api/ratio_sync/versions/activate', {
      block_id: versionModalBlock.value.block_id,
      version_id: versionId,
    });
    if (res.data?.success) {
      showSuccess(res.data.message || t('setting.operation.ratio.upstream_sync.version_activated'));
      blockActiveVersion.value = versionId;
      window.dispatchEvent(new CustomEvent(PRICING_APPLIED_EVENT));
      await loadVersionBlocks();
      await openBlockVersions(versionModalBlock.value);
    } else {
      showError(res.data?.message);
    }
  } catch (e) {
    showError(e.message);
  }
  activatingVersion.value = false;
}

// Functional pager bar component.
const PagerBar = (props, { emit }) => {
  const { page, totalPages, total, loading: pgLoading } = props;
  if (!totalPages || totalPages <= 1) return null;

  const pageItems = [];
  const addPage = (p) => {
    if (p >= 1 && p <= totalPages && !pageItems.includes(p)) pageItems.push(p);
  };
  addPage(1);
  for (let p = page - 2; p <= page + 2; p += 1) addPage(p);
  addPage(totalPages);
  pageItems.sort((a, b) => a - b);

  const buttons = [];
  let prev = 0;
  pageItems.forEach((p) => {
    if (p - prev > 1) buttons.push({ type: 'ellipsis', key: `ellipsis-${p}` });
    buttons.push({ type: 'page', value: p, key: `page-${p}` });
    prev = p;
  });

  const go = (nextPage) => {
    if (pgLoading || nextPage < 1 || nextPage > totalPages || nextPage === page) return;
    emit('pageChange', nextPage);
  };

  return h('div', { class: 'upstream-ratio-sync__pager-bar' }, [
    h(
      'span',
      { class: 'upstream-ratio-sync__pager-info' },
      t('setting.operation.ratio.upstream_sync.page_info', { page, totalPages, total })
    ),
    h(
      AButton,
      {
        size: 'small',
        disabled: pgLoading || page <= 1,
        'aria-label': t('setting.operation.ratio.upstream_sync.page_prev'),
        onClick: () => go(page - 1),
      },
      () => h(LeftOutlined)
    ),
    ...buttons.map((item) =>
      item.type === 'ellipsis'
        ? h('span', { key: item.key, class: 'upstream-ratio-sync__pager-ellipsis' }, '…')
        : h(
            AButton,
            {
              key: item.key,
              size: 'small',
              type: item.value === page ? 'primary' : 'default',
              disabled: pgLoading,
              onClick: () => go(item.value),
            },
            () => String(item.value)
          )
    ),
    h(
      AButton,
      {
        size: 'small',
        disabled: pgLoading || page >= totalPages,
        'aria-label': t('setting.operation.ratio.upstream_sync.page_next'),
        onClick: () => go(page + 1),
      },
      () => h(RightOutlined)
    ),
  ]);
};
PagerBar.props = ['page', 'totalPages', 'total', 'loading'];
PagerBar.emits = ['pageChange'];

// Functional diff table component (shared by review + compare).
const DiffTable = (props) => {
  const { rows, readOnly } = props;
  const header = h('thead', {}, [
    h('tr', {}, [
      !readOnly ? h('th', { style: 'width:1%' }) : null,
      h('th', {}, t('setting.operation.ratio.editor.col_model')),
      h('th', {}, t('setting.operation.ratio.upstream_sync.col_field')),
      h('th', {}, t('setting.operation.ratio.upstream_sync.col_change')),
      !readOnly ? h('th', {}, t('setting.operation.ratio.upstream_sync.col_pick')) : null,
      readOnly ? h('th', {}, t('setting.operation.ratio.upstream_sync.col_upstream')) : null,
    ]),
  ]);

  const body = h(
    'tbody',
    {},
    rows.map((row) => {
      const key = rowKey(row.model, row.field);
      const names = upstreamNamesFromRow(row);
      const draft = selectionDraft.value[key];
      const upstream =
        draft?.upstream_name || pickDefaultUpstream(row, defaultUpstream.value, names);
      const upVal = row.upstreams?.[upstream];
      const lowConf = row.confidence?.[upstream] === false;
      const targetName = names.find((n) => row.upstreams[n] !== 'same') || names[0];
      const compareVal = targetName ? row.upstreams[targetName] : null;
      return h(
        'tr',
        { key, class: lowConf ? 'upstream-ratio-sync__row-warning' : '' },
        [
          !readOnly
            ? h('td', { style: 'width:1%' }, [
                h('input', {
                  type: 'checkbox',
                  checked: !!draft?.selected,
                  onChange: (e) => toggleRowSelection(row, !!e.target.checked),
                }),
              ])
            : null,
          h('td', { class: 'upstream-ratio-sync__model-cell', title: row.model }, row.model),
          h(
            'td',
            { class: 'upstream-ratio-sync__field-cell' },
            FIELD_LABELS[row.field] || row.field
          ),
          h('td', { class: 'upstream-ratio-sync__change-cell' }, [
            h('span', {}, formatValue(row.current)),
            h('span', { class: 'upstream-ratio-sync__change-arrow' }, '→'),
            h(
              'span',
              { class: 'upstream-ratio-sync__change-new' },
              formatValue(readOnly ? compareVal : upVal)
            ),
            lowConf
              ? h(
                  'span',
                  { class: 'upstream-ratio-sync__low-conf-label' },
                  t('setting.operation.ratio.upstream_sync.low_confidence')
                )
              : null,
          ]),
          !readOnly
            ? h('td', { class: 'upstream-ratio-sync__pick-cell' }, [
                h(
                  'select',
                  {
                    class: 'upstream-ratio-sync__native-select',
                    value: upstream,
                    onChange: (e) => setRowUpstream(row, e.target.value),
                  },
                  names.map((n) => h('option', { key: n, value: n }, n))
                ),
              ])
            : null,
          readOnly ? h('td', {}, targetName || '—') : null,
        ]
      );
    })
  );

  return h('table', { class: 'upstream-ratio-sync__diff-table' }, [header, body]);
};
DiffTable.props = ['rows', 'readOnly'];
</script>

<style>
.upstream-ratio-sync__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.upstream-ratio-sync__section-title {
  margin-top: 1.5rem;
  padding-bottom: 0.4rem;
  border-bottom: 1px solid rgba(34, 36, 38, 0.12);
  font-weight: 600;
}

.upstream-ratio-sync__fetch-status {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.upstream-ratio-sync__toolbar {
  background: #f8f9fa;
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 6px;
  padding: 0.85rem 1rem 0.75rem;
  margin-bottom: 0.85rem;
}

.upstream-ratio-sync__toolbar-row {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(220px, 1fr) auto;
  gap: 0.75rem 1rem;
  align-items: end;
}

.upstream-ratio-sync__toolbar-row + .upstream-ratio-sync__toolbar-row {
  margin-top: 0.75rem;
}

.upstream-ratio-sync__toolbar-row--review {
  grid-template-columns: repeat(3, minmax(180px, 1fr));
}

.upstream-ratio-sync__toolbar-row--compare {
  grid-template-columns: repeat(4, minmax(160px, 1fr));
}

.upstream-ratio-sync__toolbar-field label {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 0.35rem;
}

.upstream-ratio-sync__toolbar-check {
  display: flex;
  align-items: center;
  min-height: 38px;
  padding-bottom: 0.1rem;
  white-space: nowrap;
}

.upstream-ratio-sync__summary {
  font-size: 0.85rem;
  color: rgba(0, 0, 0, 0.55);
}

.upstream-ratio-sync__summary-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.65rem;
  margin-bottom: 0.65rem;
}

.upstream-ratio-sync__pager-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.upstream-ratio-sync__pager-info {
  font-size: 0.85rem;
  white-space: nowrap;
  color: rgba(0, 0, 0, 0.55);
}

.upstream-ratio-sync__pager-ellipsis {
  display: inline-flex;
  align-items: center;
  padding: 0 0.15rem;
  color: rgba(0, 0, 0, 0.45);
  font-size: 0.85rem;
  user-select: none;
}

.upstream-ratio-sync__table-wrap {
  position: relative;
  min-height: 180px;
  max-height: 52vh;
  overflow: auto;
  overscroll-behavior: contain;
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 6px;
}

.upstream-ratio-sync__diff-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.upstream-ratio-sync__diff-table th,
.upstream-ratio-sync__diff-table td {
  border: 1px solid rgba(34, 36, 38, 0.1);
  padding: 0.4rem 0.55rem;
  text-align: left;
  vertical-align: top;
}

.upstream-ratio-sync__diff-table thead th {
  position: sticky;
  top: 0;
  z-index: 2;
  background: #f9fafb;
}

.upstream-ratio-sync__row-warning {
  background: #fffaf3;
}

.upstream-ratio-sync__pager-footer {
  display: flex;
  justify-content: center;
  padding-top: 0.65rem;
  border-top: 1px solid rgba(34, 36, 38, 0.08);
  margin-top: 0.65rem;
}

.upstream-ratio-sync__model-cell {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.82rem;
  line-height: 1.35;
  word-break: break-all;
  max-width: 280px;
}

.upstream-ratio-sync__field-cell,
.upstream-ratio-sync__change-cell {
  white-space: nowrap;
}

.upstream-ratio-sync__change-arrow {
  margin: 0 0.35rem;
  color: rgba(0, 0, 0, 0.35);
}

.upstream-ratio-sync__change-new {
  font-weight: 600;
  color: #2185d0;
}

.upstream-ratio-sync__low-conf-label {
  margin-left: 4px;
  padding: 0 6px;
  border-radius: 8px;
  font-size: 0.7rem;
  background: #f2711c;
  color: #fff;
}

.upstream-ratio-sync__pick-cell {
  min-width: 180px;
}

.upstream-ratio-sync__native-select {
  width: 100%;
  height: 32px;
  padding: 0.35rem 0.5rem;
  border: 1px solid rgba(34, 36, 38, 0.15);
  border-radius: 4px;
  background: #fff;
  color: rgba(0, 0, 0, 0.87);
  font-size: 0.85rem;
}

.ant-table-row-positive > td {
  background: #f6fff6;
}

@media (max-width: 900px) {
  .upstream-ratio-sync__toolbar-row,
  .upstream-ratio-sync__toolbar-row--review,
  .upstream-ratio-sync__toolbar-row--compare {
    grid-template-columns: 1fr;
  }

  .upstream-ratio-sync__toolbar-check {
    min-height: 0;
    padding-bottom: 0;
  }
}

html.dark .upstream-ratio-sync__toolbar {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.12);
}

html.dark .upstream-ratio-sync__toolbar-field label,
html.dark .upstream-ratio-sync__summary,
html.dark .upstream-ratio-sync__pager-info {
  color: rgba(255, 255, 255, 0.65);
}

html.dark .upstream-ratio-sync__diff-table thead th {
  background: #1b1c1d;
}

html.dark .upstream-ratio-sync__change-arrow {
  color: rgba(255, 255, 255, 0.35);
}

html.dark .upstream-ratio-sync__change-new {
  color: #54c8ff;
}

html.dark .upstream-ratio-sync__native-select {
  background: #1b1c1d;
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.87);
}
</style>
