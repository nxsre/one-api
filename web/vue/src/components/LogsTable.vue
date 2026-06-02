<template>
  <div class="logs-page-body">
    <div class="logs-stat-segment" style="margin-bottom: 1rem">
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; gap: 0.75rem 1.25rem"
      >
        <h3 style="margin: 0; font-weight: 600; font-size: 1.05rem">
          {{ t('log.usage_details') }}
        </h3>
        <div style="display: flex; align-items: center; gap: 0.5rem">
          <span style="color: rgba(0, 0, 0, 0.55)">{{ t('log.total_quota') }}：</span>
          <a-spin v-if="statLoading" size="small" />
          <strong v-else>{{ renderQuota(stat.quota, t) }}</strong>
          <span style="color: rgba(0, 0, 0, 0.45); font-size: 0.92rem">
            RPM {{ stat.rpm ?? 0 }} · TPM {{ stat.tpm ?? 0 }}
          </span>
        </div>
      </div>
    </div>

    <div class="logs-quickbar">
      <a-radio-group v-model:value="quickRange" button-style="solid" size="small" @change="applyQuickRange">
        <a-radio-button :value="5">{{ t('log.quick.last5') }}</a-radio-button>
        <a-radio-button :value="15">{{ t('log.quick.last15') }}</a-radio-button>
        <a-radio-button :value="30">{{ t('log.quick.last30') }}</a-radio-button>
      </a-radio-group>
      <span class="logs-quickbar__auto">
        <a-switch size="small" :checked="autoRefresh" @change="toggleAutoRefresh" />
        {{ t('log.quick.auto_refresh') }}
      </span>
      <a-button type="primary" size="small" @click="refresh">{{ t('log.buttons.submit') }}</a-button>
      <a-button size="small" @click="showFilters = !showFilters">
        {{ showFilters ? t('log.quick.hide_filters') : t('log.quick.show_filters') }}
      </a-button>
    </div>

    <a-form v-show="showFilters" layout="vertical" class="logs-filter-form">
      <a-row :gutter="16">
        <a-col :span="5">
          <a-form-item :label="t('log.table.token_name')">
            <a-input
              v-model:value="inputs.token_name"
              :placeholder="t('log.table.token_name_placeholder')"
            />
          </a-form-item>
        </a-col>
        <a-col :span="5">
          <a-form-item :label="t('log.table.model_name')">
            <a-input
              v-model:value="inputs.model_name"
              :placeholder="t('log.table.model_name_placeholder')"
            />
          </a-form-item>
        </a-col>
        <a-col :span="9">
          <a-form-item :label="t('log.table.start_time') + ' ~ ' + t('log.table.end_time')">
            <a-range-picker
              v-model:value="dateRange"
              show-time
              format="YYYY-MM-DD HH:mm:ss"
              style="width: 100%"
              @change="onDateRangeChange"
            />
          </a-form-item>
        </a-col>
        <a-col :span="5">
          <a-form-item :label="t('log.buttons.query')">
            <a-button type="primary" block @click="refresh">
              {{ t('log.buttons.submit') }}
            </a-button>
          </a-form-item>
        </a-col>
      </a-row>

      <a-row v-if="isAdminUser" :gutter="16">
        <a-col :span="5">
          <a-form-item :label="t('log.table.channel_id')">
            <a-input
              v-model:value="inputs.channel"
              :placeholder="t('log.table.channel_id_placeholder')"
            />
          </a-form-item>
        </a-col>
        <a-col :span="5">
          <a-form-item :label="t('log.table.username')">
            <a-input
              v-model:value="inputs.username"
              :placeholder="t('log.table.username_placeholder')"
              autocomplete="off"
            />
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :span="5">
          <a-form-item label="分组">
            <a-input v-model:value="inputs.group" placeholder="user group" />
          </a-form-item>
        </a-col>
        <a-col :span="7">
          <a-form-item label="请求 ID">
            <a-input
              v-model:value="inputs.request_id"
              placeholder="request_id 精确匹配"
            />
          </a-form-item>
        </a-col>
      </a-row>
    </a-form>

    <div class="logs-table-wrap">
      <a-table
        class="logs-data-table"
        size="small"
        row-key="id"
        :columns="columns"
        :data-source="visibleLogs"
        :loading="loading"
        :pagination="false"
        :scroll="{ x: 'max-content' }"
      >
        <template #headerCell="{ column }">
          <span
            v-if="column.sortField"
            style="cursor: pointer"
            :title="column.headerTitle"
            @click="sortLog(column.sortField)"
          >
            {{ column.title }}
          </span>
          <span v-else>{{ column.title }}</span>
        </template>

        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'time'">
            <span
              class="log-time-cell"
              :title="record.request_id ? `${timestamp2string(record.created_at)}\n请求 ID：${record.request_id}（点击复制）` : timestamp2string(record.created_at)"
              :role="record.request_id ? 'button' : undefined"
              :tabindex="record.request_id ? 0 : undefined"
              @click="copyRequestId(record.request_id)"
            >
              {{ timestamp2string(record.created_at) }}
            </span>
          </template>

          <template v-else-if="column.key === 'channel'">
            <div v-if="record.channel || record.channel_name" class="log-channel-cell">
              <router-link
                v-if="record.channel"
                class="log-channel-id"
                :to="`/channel/edit/${record.channel}`"
              >
                #{{ record.channel }}
              </router-link>
              <span
                v-if="record.channel_name"
                class="log-channel-name"
                :title="record.channel_name"
              >
                {{ record.channel_name }}
              </span>
            </div>
          </template>

          <template v-else-if="column.key === 'group'">
            <a-tag v-if="record.group">{{ record.group }}</a-tag>
          </template>

          <template v-else-if="column.key === 'type'">
            <a-tag :color="typeColor(record.type)">{{ typeText(record.type) }}</a-tag>
          </template>

          <template v-else-if="column.key === 'model'">
            <span v-if="record.model_name" class="log-model-ellipsis" :title="record.model_name">
              <a-tag :color="colorForLabel(record.model_name)">{{ record.model_name }}</a-tag>
            </span>
          </template>

          <template v-else-if="column.key === 'user'">
            <router-link
              v-if="record.username"
              class="log-user-link"
              :to="`/user/edit/${record.user_id}`"
            >
              {{ record.username }}
            </router-link>
          </template>

          <template v-else-if="column.key === 'token_name'">
            <a-tag v-if="record.token_name" :color="colorForLabel(record.token_name)">
              {{ record.token_name }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'token_id'">
            {{ record.token_id ? record.token_id : '' }}
          </template>

          <template v-else-if="column.key === 'ip'">
            <code v-if="record.ip" class="log-ip">{{ record.ip }}</code>
          </template>

          <template v-else-if="column.key === 'prompt_tokens'">
            <span v-if="record.prompt_tokens == null" style="color: #999">—</span>
            <span v-else>{{ record.prompt_tokens }}</span>
          </template>

          <template v-else-if="column.key === 'completion_tokens'">
            <span v-if="record.completion_tokens == null" style="color: #999">—</span>
            <span v-else>{{ record.completion_tokens }}</span>
          </template>
          <template v-else-if="column.key === 'cached_tokens'">
            <span v-if="!record.cached_tokens" style="color: #999">—</span>
            <span v-else>{{ record.cached_tokens }}</span>
          </template>
          <template v-else-if="column.key === 'reasoning_tokens'">
            <span v-if="!record.reasoning_tokens" style="color: #999">—</span>
            <span v-else>{{ record.reasoning_tokens }}</span>
          </template>

          <template v-else-if="column.key === 'quota'">
            <span v-if="record.quota == null" style="color: #999">—</span>
            <span v-else>{{ renderQuota(record.quota, t, 4) }}</span>
          </template>

          <template v-else-if="column.key === 'use_time'">
            <span v-if="record.use_time == null" style="color: #999">—</span>
            <span v-else>{{ record.use_time }}</span>
          </template>

          <template v-else-if="column.key === 'detail'">
            <div v-if="hasDetail(record)" class="log-detail-preview">
              <div class="log-detail-body" :title="detailPreviewTitle(record)">
                <span
                  v-if="buildLogDetailRatioLine(record)"
                  class="log-detail-line log-detail-line-ratio"
                >{{ buildLogDetailRatioLine(record) }}</span>
                <span
                  v-if="buildLogDetailMetaLine(record, t)"
                  class="log-detail-line log-detail-line-meta"
                >{{ buildLogDetailMetaLine(record, t) }}</span>
              </div>
              <a-button
                class="log-detail-expand-btn"
                size="small"
                :title="t('log.table.detail_expand')"
                :aria-label="t('log.table.detail_expand')"
                @click.stop="openFull(buildLogDetailFullText(record, t))"
              >
                <template #icon><ExpandOutlined /></template>
              </a-button>
            </div>
            <span v-else class="log-detail-empty">—</span>
          </template>
        </template>
      </a-table>

      <div class="logs-table-toolbar">
        <div class="logs-table-toolbar-left">
          <a-select
            class="logs-type-select"
            :placeholder="t('log.type.select')"
            :options="LOG_OPTIONS"
            v-model:value="logType"
            style="min-width: 8rem"
          />
          <a-checkbox v-model:checked="includeErrors" class="logs-include-errors">
            {{ t('log.show_failed') }}
          </a-checkbox>
          <a-button size="small" :loading="loading" @click="refresh">
            {{ t('log.buttons.refresh') }}
          </a-button>
          <a-button
            v-if="isAdminUser"
            size="small"
            :loading="exporting"
            @click="exportLogs"
          >
            导出 CSV
          </a-button>
        </div>
        <a-pagination
          class="logs-table-pagination"
          v-model:current="activePage"
          size="small"
          :total="total"
          :page-size="ITEMS_PER_PAGE"
          :show-size-changer="false"
          @change="onPaginationChange"
        />
      </div>
    </div>

    <a-modal
      v-model:open="logDetailModal.open"
      :title="t('log.table.detail')"
      width="800px"
      @cancel="closeModal"
    >
      <pre class="log-detail-modal-body">{{ logDetailModal.body }}</pre>
      <template #footer>
        <a-button @click="copyDetail">{{ t('log.table.detail_copy') }}</a-button>
        <a-button type="primary" @click="closeModal">
          {{ t('log.table.detail_close') }}
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import { ExpandOutlined } from '@ant-design/icons-vue';
import {
  API,
  copy,
  isAdmin,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
  renderQuota,
  colorForLabel,
} from '@/helpers';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();

const logs = ref([]);
const total = ref(0);
const loading = ref(true);
const exporting = ref(false);
const activePage = ref(1);
const logType = ref(0);
const includeErrors = ref(false);
const isAdminUser = isAdmin();

// 快捷时间范围（分钟）+ 自动刷新（5 秒）+ 高级筛选折叠
const quickRange = ref(5);
const autoRefresh = ref(false);
const showFilters = ref(false);
const AUTO_REFRESH_MS = 5000;
let autoTimer = null;

const now = new Date();
const inputs = reactive({
  username: '',
  token_name: '',
  model_name: '',
  start_timestamp: timestamp2string(now.getTime() / 1000 - 300),
  end_timestamp: timestamp2string(now.getTime() / 1000),
  channel: '',
  group: '',
  request_id: '',
});

const dateRange = ref([
  dayjs(inputs.start_timestamp),
  dayjs(inputs.end_timestamp),
]);

const onDateRangeChange = (vals) => {
  if (vals && vals.length === 2 && vals[0] && vals[1]) {
    inputs.start_timestamp = vals[0].format('YYYY-MM-DD HH:mm:ss');
    inputs.end_timestamp = vals[1].format('YYYY-MM-DD HH:mm:ss');
  } else {
    inputs.start_timestamp = '';
    inputs.end_timestamp = '';
  }
};

const stat = reactive({ quota: 0, rpm: 0, tpm: 0 });
const statLoading = ref(false);

const logDetailModal = reactive({ open: false, body: '' });

const includeErrorsQuery = () => (includeErrors.value ? '&include_errors=1' : '');

const LOG_OPTIONS = computed(() => [
  { label: t('log.type.all'), value: 0 },
  { label: t('log.type.topup'), value: 1 },
  { label: t('log.type.usage'), value: 2 },
  { label: t('log.type.admin'), value: 3 },
  { label: t('log.type.system'), value: 4 },
  { label: t('log.type.test'), value: 5 },
  { label: t('log.type.error'), value: 6 },
  { label: t('log.type.refund'), value: 7 },
]);

const typeText = (type) => {
  switch (type) {
    case 1: return '充值';
    case 2: return '消费';
    case 3: return '管理';
    case 4: return '系统';
    case 5: return '测试';
    case 6: return '错误';
    case 7: return '退款';
    default: return '未知';
  }
};

const typeColor = (type) => {
  switch (type) {
    case 1: return 'green';
    case 2: return 'lime';
    case 3: return 'orange';
    case 4: return 'purple';
    case 5: return 'purple';
    case 6: return 'red';
    case 7: return 'cyan';
    default: return 'default';
  }
};

const columns = computed(() => {
  const cols = [
    { title: t('log.table.time'), key: 'time', sortField: 'created_at', width: '12rem' },
  ];
  if (isAdminUser) {
    cols.push({ title: t('log.table.channel'), key: 'channel', sortField: 'channel', width: '9.5rem' });
  }
  cols.push(
    { title: '分组', key: 'group', width: '4.5rem' },
    { title: t('log.table.type'), key: 'type', sortField: 'type', width: '4.5rem' },
    { title: t('log.table.model'), key: 'model', sortField: 'model_name', width: '13.5rem' },
  );
  if (isAdminUser) {
    cols.push({ title: t('log.table.username'), key: 'user', sortField: 'username', width: '5.5rem' });
  }
  cols.push(
    { title: t('log.table.token_name'), key: 'token_name', sortField: 'token_name', width: '6rem' },
    { title: 'ID', key: 'token_id', sortField: 'token_id', width: '4.5rem' },
    { title: 'IP', key: 'ip', sortField: 'ip', width: '7.5rem' },
    { title: t('log.table.prompt_tokens_short'), key: 'prompt_tokens', sortField: 'prompt_tokens', headerTitle: t('log.table.prompt_tokens'), width: '4.75rem', align: 'right' },
    { title: t('log.table.completion_tokens_short'), key: 'completion_tokens', sortField: 'completion_tokens', headerTitle: t('log.table.completion_tokens'), width: '4.75rem', align: 'right' },
    { title: '缓存', key: 'cached_tokens', headerTitle: '缓存命中 token', width: '4.5rem', align: 'right' },
    { title: '推理', key: 'reasoning_tokens', headerTitle: '推理 token', width: '4.5rem', align: 'right' },
    { title: t('log.table.quota'), key: 'quota', sortField: 'quota', width: '5.5rem', align: 'right' },
    { title: t('log.table.use_time'), key: 'use_time', sortField: 'use_time', width: '3.75rem', align: 'right' },
    { title: t('log.table.detail'), key: 'detail', class: 'log-col-detail' },
  );
  return cols;
});

const visibleLogs = computed(() => logs.value.filter((l) => !l.deleted));

const copyRequestId = async (requestId) => {
  if (!requestId) return;
  if (await copy(requestId)) {
    showSuccess(`已复制请求 ID：${requestId}`);
  } else {
    showWarning(`请求 ID 复制失败：${requestId}`);
  }
};

const getLogSelfStat = async () => {
  const localStartTimestamp = Date.parse(inputs.start_timestamp) / 1000;
  const localEndTimestamp = Date.parse(inputs.end_timestamp) / 1000;
  const g = encodeURIComponent(inputs.group || '');
  const rid = encodeURIComponent(inputs.request_id || '');
  const res = await API.get(
    `/api/log/self/stat?type=${logType.value}&token_name=${encodeURIComponent(inputs.token_name)}&model_name=${encodeURIComponent(inputs.model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}${includeErrorsQuery()}`
  );
  const { success, message, data } = res.data;
  if (success) {
    Object.assign(stat, data);
  } else {
    showError(message);
  }
};

const getLogStat = async () => {
  const localStartTimestamp = Date.parse(inputs.start_timestamp) / 1000;
  const localEndTimestamp = Date.parse(inputs.end_timestamp) / 1000;
  const g = encodeURIComponent(inputs.group || '');
  const rid = encodeURIComponent(inputs.request_id || '');
  const res = await API.get(
    `/api/log/stat?type=${logType.value}&username=${encodeURIComponent(inputs.username)}&token_name=${encodeURIComponent(inputs.token_name)}&model_name=${encodeURIComponent(inputs.model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${inputs.channel}&group=${g}&request_id=${rid}${includeErrorsQuery()}`
  );
  const { success, message, data } = res.data;
  if (success) {
    Object.assign(stat, data);
  } else {
    showError(message);
  }
};

const fetchUsageStat = async () => {
  statLoading.value = true;
  try {
    if (isAdminUser) {
      await getLogStat();
    } else {
      await getLogSelfStat();
    }
  } finally {
    statLoading.value = false;
  }
};

const loadLogs = async (page) => {
  let url = '';
  const localStartTimestamp = Date.parse(inputs.start_timestamp) / 1000;
  const localEndTimestamp = Date.parse(inputs.end_timestamp) / 1000;
  const g = encodeURIComponent(inputs.group || '');
  const rid = encodeURIComponent(inputs.request_id || '');
  if (isAdminUser) {
    url = `/api/log/?p=${page}&page_size=${ITEMS_PER_PAGE}&type=${logType.value}&username=${encodeURIComponent(inputs.username)}&token_name=${encodeURIComponent(inputs.token_name)}&model_name=${encodeURIComponent(inputs.model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${inputs.channel}&group=${g}&request_id=${rid}${includeErrorsQuery()}`;
  } else {
    url = `/api/log/self/?p=${page}&page_size=${ITEMS_PER_PAGE}&type=${logType.value}&token_name=${encodeURIComponent(inputs.token_name)}&model_name=${encodeURIComponent(inputs.model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}${includeErrorsQuery()}`;
  }
  const res = await API.get(url);
  const { success, message, data } = res.data;
  if (success) {
    const items = data && data.items !== undefined ? data.items : data;
    const pageTotal =
      data && typeof data.total === 'number'
        ? data.total
        : Array.isArray(items)
          ? items.length
          : 0;
    logs.value = Array.isArray(items) ? items : [];
    total.value = pageTotal;
  } else {
    showError(message);
  }
  loading.value = false;
};

const exportLogs = async () => {
  exporting.value = true;
  try {
    const localStartTimestamp = Date.parse(inputs.start_timestamp) / 1000;
    const localEndTimestamp = Date.parse(inputs.end_timestamp) / 1000;
    const g = encodeURIComponent(inputs.group || '');
    const rid = encodeURIComponent(inputs.request_id || '');
    const url = `/api/log/export?type=${logType.value}&username=${encodeURIComponent(inputs.username)}&token_name=${encodeURIComponent(inputs.token_name)}&model_name=${encodeURIComponent(inputs.model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${inputs.channel}&group=${g}&request_id=${rid}${includeErrorsQuery()}`;
    const res = await API.get(url, { responseType: 'blob' });
    const blob = new Blob([res.data], { type: 'text/csv;charset=utf-8' });
    const dl = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = dl;
    a.download = `logs_export_${localStartTimestamp}_${localEndTimestamp}.csv`;
    a.click();
    window.URL.revokeObjectURL(dl);
    showSuccess('账单 CSV 导出完成');
  } catch (e) {
    showError(e.message || '导出失败');
  } finally {
    exporting.value = false;
  }
};

const onPaginationChange = async (page) => {
  loading.value = true;
  await loadLogs(page);
  activePage.value = page;
};

const refresh = async () => {
  loading.value = true;
  activePage.value = 1;
  await loadLogs(1);
  await fetchUsageStat();
};

// 按当前 quickRange 把时间窗口设为「最近 N 分钟」（滑动窗口），再查询。
const applyQuickRange = () => {
  const n = Number(quickRange.value);
  if (!n || n <= 0) return;
  const end = new Date();
  const start = new Date(end.getTime() - n * 60 * 1000);
  inputs.start_timestamp = timestamp2string(start.getTime() / 1000);
  inputs.end_timestamp = timestamp2string(end.getTime() / 1000);
  dateRange.value = [dayjs(inputs.start_timestamp), dayjs(inputs.end_timestamp)];
  refresh();
};

const stopAutoRefresh = () => {
  if (autoTimer) {
    clearInterval(autoTimer);
    autoTimer = null;
  }
};

const toggleAutoRefresh = (checked) => {
  autoRefresh.value = checked;
  stopAutoRefresh();
  if (checked) {
    applyQuickRange(); // 立即刷新一次
    autoTimer = setInterval(applyQuickRange, AUTO_REFRESH_MS);
  }
};

watch([logType, includeErrors], () => {
  refresh();
});

onMounted(() => {
  refresh();
});

onUnmounted(stopAutoRefresh);

const sortLog = (key) => {
  if (logs.value.length === 0) return;
  loading.value = true;
  const sortedLogs = [...logs.value];
  if (typeof sortedLogs[0][key] === 'string') {
    sortedLogs.sort((a, b) => ('' + a[key]).localeCompare(b[key]));
  } else {
    sortedLogs.sort((a, b) => {
      if (a[key] === b[key]) return 0;
      if (a[key] > b[key]) return -1;
      if (a[key] < b[key]) return 1;
      return 0;
    });
  }
  if (sortedLogs[0].id === logs.value[0].id) {
    sortedLogs.reverse();
  }
  logs.value = sortedLogs;
  loading.value = false;
};

// ---- Detail building helpers (ported verbatim) ----

function parseLogOther(other) {
  const s = other != null ? String(other).trim() : '';
  if (!s) return null;
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      return o;
    }
  } catch {
    /* 非 JSON，按纯文本展示 */
  }
  return null;
}

function userInputLabel() {
  const key = t('log.table.user_input');
  return key !== 'log.table.user_input' ? key : '用户输入';
}

const LOG_OTHER_KNOWN_KEYS = new Set([
  'user_input',
  'http_status',
  'http_method',
  'http_path',
  'client_request_headers',
  'client_response_headers',
  'upstream_request_headers',
  'upstream_response_headers',
  'upstream_request_meta',
  'upstream_response_meta',
  'client_ip',
  'xff',
  'request_body_bytes',
  'response_body_bytes',
  'system_prompt_reset',
]);

function detailLabel(key, fallback) {
  const v = t(key);
  return v !== key ? v : fallback;
}

function appendDetailSection(parts, title, body) {
  const text = body != null ? String(body).trim() : '';
  if (!text) return;
  parts.push(`${title}\n${text}`);
}

function normalizeHeaderMap(headers) {
  if (!headers || typeof headers !== 'object') return {};
  const out = {};
  Object.entries(headers).forEach(([k, v]) => {
    if (v == null) return;
    out[k] = Array.isArray(v) ? v.map(String) : [String(v)];
  });
  return out;
}

function formatCurlHeaders(prefix, headers, statusLine) {
  const lines = [];
  if (statusLine) {
    lines.push(`${prefix} ${statusLine}`);
  }
  Object.keys(headers)
    .sort((a, b) => a.localeCompare(b))
    .forEach((k) => {
      headers[k].forEach((v) => {
        lines.push(`${prefix} ${k}: ${v}`);
      });
    });
  lines.push(prefix);
  return lines.join('\n');
}

function upstreamRequestPath(meta) {
  const url = meta?.url != null ? String(meta.url) : '';
  if (!url) return '/';
  try {
    const u = new URL(url);
    return u.pathname + (u.search || '') || '/';
  } catch {
    return url;
  }
}

function buildClientRequestCurl(parsed) {
  if (!parsed?.client_request_headers) return '';
  const method = parsed.http_method || 'GET';
  const path = parsed.http_path || '/';
  return formatCurlHeaders(
    '>',
    normalizeHeaderMap(parsed.client_request_headers),
    `${method} ${path} HTTP/1.1`,
  );
}

function buildClientResponseCurl(parsed) {
  if (!parsed?.client_response_headers) return '';
  const code = parsed.http_status;
  const statusLine =
    code != null && code !== '' ? `HTTP/1.1 ${code}` : 'HTTP/1.1';
  return formatCurlHeaders(
    '<',
    normalizeHeaderMap(parsed.client_response_headers),
    statusLine,
  );
}

function buildUpstreamRequestCurl(parsed) {
  if (!parsed?.upstream_request_headers) return '';
  const meta = parsed.upstream_request_meta || {};
  const method = meta.method || 'GET';
  const path = upstreamRequestPath(meta);
  return formatCurlHeaders(
    '>',
    normalizeHeaderMap(parsed.upstream_request_headers),
    `${method} ${path} HTTP/1.1`,
  );
}

function buildUpstreamResponseCurl(parsed) {
  if (!parsed?.upstream_response_headers) return '';
  const meta = parsed.upstream_response_meta || {};
  let statusLine = 'HTTP/1.1';
  if (meta.status) {
    statusLine = String(meta.status);
    if (!statusLine.startsWith('HTTP')) {
      statusLine = `HTTP/1.1 ${statusLine}`;
    }
  } else if (meta.status_code != null) {
    statusLine = `HTTP/1.1 ${meta.status_code}`;
  }
  return formatCurlHeaders(
    '<',
    normalizeHeaderMap(parsed.upstream_response_headers),
    statusLine,
  );
}

function buildLogDetailMetaBlock(log, parsed) {
  const lines = [];
  const clientIp = parsed?.client_ip || log.ip;
  if (clientIp) {
    lines.push(`${detailLabel('log.detail.client_ip', '客户端 IP')}: ${clientIp}`);
  }
  if (parsed?.xff) {
    lines.push(`X-Forwarded-For: ${parsed.xff}`);
  }
  if (parsed?.request_body_bytes != null) {
    lines.push(
      `${detailLabel('log.detail.request_body_bytes', '请求体大小')}: ${parsed.request_body_bytes} bytes`,
    );
  }
  if (parsed?.response_body_bytes != null) {
    lines.push(
      `${detailLabel('log.detail.response_body_bytes', '响应体大小')}: ${parsed.response_body_bytes} bytes`,
    );
  }
  return lines.join('\n');
}

function getLogDetailParts(log) {
  const parsed = parseLogOther(log.other);
  const userInput =
    parsed?.user_input != null ? String(parsed.user_input).trim() : '';
  const content = log.content != null ? String(log.content).trim() : '';
  const badges = [];
  if (log.use_time > 0) badges.push(`${log.use_time}s`);
  if (log.elapsed_time != null && log.elapsed_time !== '')
    badges.push(`${log.elapsed_time}ms`);
  if (log.is_stream) badges.push('Stream');
  if (log.system_prompt_reset) badges.push('System Prompt Reset');
  return { parsed, userInput, content, badges };
}

function buildLogDetailFullText(log) {
  const { parsed, userInput, content, badges } = getLogDetailParts(log);
  const parts = [];
  const label = userInputLabel();

  appendDetailSection(
    parts,
    detailLabel('log.detail.meta', '连接信息'),
    buildLogDetailMetaBlock(log, parsed),
  );
  appendDetailSection(
    parts,
    detailLabel('log.detail.client_request_headers', '客户端请求 Header'),
    buildClientRequestCurl(parsed),
  );
  appendDetailSection(
    parts,
    detailLabel('log.detail.client_response_headers', '响应客户端 Header'),
    buildClientResponseCurl(parsed),
  );
  appendDetailSection(
    parts,
    detailLabel('log.detail.upstream_request_headers', '上游请求 Header'),
    buildUpstreamRequestCurl(parsed),
  );
  appendDetailSection(
    parts,
    detailLabel('log.detail.upstream_response_headers', '上游响应 Header'),
    buildUpstreamResponseCurl(parsed),
  );

  if (userInput) {
    parts.push(`${label}\n${userInput}`);
  }
  if (content) {
    parts.push(content);
  }
  if (badges.length) {
    parts.push(badges.join(' · '));
  }

  if (parsed) {
    const rest = {};
    Object.entries(parsed).forEach(([k, v]) => {
      if (!LOG_OTHER_KNOWN_KEYS.has(k)) {
        rest[k] = v;
      }
    });
    if (Object.keys(rest).length > 0) {
      const hdrLabel = detailLabel('log.table.upstream_meta', '其他元数据');
      parts.push(`${hdrLabel}\n${JSON.stringify(rest, null, 2)}`);
    }
  }
  if (!parsed && log.other != null && String(log.other).trim()) {
    parts.push(String(log.other));
  }
  return parts.join('\n\n');
}

function buildLogDetailRatioLine(log) {
  return log.content != null ? String(log.content).trim() : '';
}

function extractHttpStatusCode(parsed) {
  if (!parsed || typeof parsed !== 'object') {
    return null;
  }
  for (const key of [
    'http_status',
    'status_code',
    'upstream_http_status',
    'test_http_status',
  ]) {
    const v = parsed[key];
    if (v == null || v === '') {
      continue;
    }
    const n = Number(v);
    if (!Number.isNaN(n) && n > 0) {
      return n;
    }
    return String(v);
  }
  return null;
}

function buildLogDetailMetaLine(log) {
  const { parsed, badges } = getLogDetailParts(log);
  const label =
    t('log.table.http_status') !== 'log.table.http_status'
      ? t('log.table.http_status')
      : 'HTTP状态码';
  const code = extractHttpStatusCode(parsed);
  const tail = badges.join(' · ');

  if (code != null && tail) {
    return `${label} ${code} · ${tail}`;
  }
  if (code != null) {
    return `${label} ${code}`;
  }
  if (tail) {
    return `${label}  ${tail}`;
  }
  return '';
}

const hasDetail = (log) => {
  const full = buildLogDetailFullText(log);
  const ratioLine = buildLogDetailRatioLine(log);
  const metaLine = buildLogDetailMetaLine(log);
  return !!(ratioLine || metaLine || full.trim());
};

const detailPreviewTitle = (log) => {
  const ratioLine = buildLogDetailRatioLine(log);
  const metaLine = buildLogDetailMetaLine(log);
  const full = buildLogDetailFullText(log);
  return [ratioLine, metaLine].filter(Boolean).join('\n') || full;
};

const openFull = (body) => {
  logDetailModal.open = true;
  logDetailModal.body = body;
};

const closeModal = () => {
  logDetailModal.open = false;
  logDetailModal.body = '';
};

const copyDetail = async () => {
  if (await copy(logDetailModal.body)) {
    showSuccess(t('log.table.detail_copied'));
  } else {
    showWarning(t('log.table.detail_copy_failed'));
  }
};
</script>

<style scoped>
.logs-quickbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.logs-quickbar__auto {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: rgba(0, 0, 0, 0.65);
  font-size: 0.9rem;
}
/* 折叠后的高级筛选更紧凑，腾出更多表格空间 */
.logs-filter-form {
  margin-bottom: 8px;
}
.logs-filter-form :deep(.ant-form-item) {
  margin-bottom: 8px;
}
html.dark .logs-quickbar__auto {
  color: rgba(255, 255, 255, 0.65);
}
</style>
