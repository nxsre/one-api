<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">平台财务报表</h2>
      <a-form layout="inline" class="mb-4">
        <a-form-item label="开始日期">
          <a-input v-model:value="startTime" type="date" />
        </a-form-item>
        <a-form-item label="结束日期">
          <a-input v-model:value="endTime" type="date" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" :loading="loading" @click="loadAll">查询</a-button>
        </a-form-item>
        <a-form-item>
          <a-button @click="exportCSV">导出 CSV 账单</a-button>
        </a-form-item>
      </a-form>
      <a-table
        row-key="_rowKey"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :pagination="false"
        size="middle"
      />
    </a-card>

    <a-card class="chart-card mt-4">
      <div class="flex items-center justify-between mb-4 flex-wrap gap-2">
        <h2 class="text-xl font-semibold">充值与兑换记录</h2>
        <div class="flex items-center gap-3 flex-wrap">
          <span class="text-sm text-gray-500">
            共 {{ topupTotal }} 条 · 本页合计 {{ renderQuota(pageTopupSum, t) }}
          </span>
          <a-button size="small" @click="exportTopupCSV">导出 CSV</a-button>
        </div>
      </div>
      <a-table
        row-key="id"
        :columns="topupColumns"
        :data-source="topupRows"
        :loading="topupLoading"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'created_at'">
            {{ timestamp2string(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'source'">
            <a-tag :color="sourceColor(record)">{{ sourceText(record) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'quota'">
            {{ renderQuota(record.quota, t) }}
          </template>
        </template>
      </a-table>
      <div class="flex justify-end mt-4">
        <a-pagination
          size="small"
          :current="topupPage"
          :total="topupTotal"
          :page-size="ITEMS_PER_PAGE"
          :show-size-changer="false"
          @change="onTopupPageChange"
        />
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { API, showError, renderQuota, timestamp2string } from '@/helpers';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();

const reports = ref([]);
const loading = ref(false);

// default to last 30 days
const startTime = ref(
  new Date(new Date().setDate(new Date().getDate() - 30)).toISOString().slice(0, 10)
);
const endTime = ref(new Date().toISOString().slice(0, 10));

const columns = [
  { title: '租户 ID', dataIndex: 'tenant_id', key: 'tenant_id' },
  { title: '企业名称', dataIndex: 'tenant_name', key: 'tenant_name' },
  { title: '消费额度 (Quota)', dataIndex: 'quota', key: 'quota' },
  { title: '请求次数', dataIndex: 'request_count', key: 'request_count' },
  { title: '提示 Token', dataIndex: 'prompt_tokens', key: 'prompt_tokens' },
  { title: '补全 Token', dataIndex: 'completion_tokens', key: 'completion_tokens' },
];

const rows = computed(() => reports.value.map((r, i) => ({ ...r, _rowKey: i })));

// ---- 充值与兑换记录（充值类日志 type=1，涵盖兑换码、微信支付、管理员充值等）----
const topupRows = ref([]);
const topupLoading = ref(false);
const topupPage = ref(1);
const topupTotal = ref(0);

const topupColumns = [
  { title: '时间', key: 'created_at' },
  { title: '用户', dataIndex: 'username', key: 'username' },
  { title: '来源', key: 'source' },
  { title: '金额', key: 'quota' },
  { title: '说明', dataIndex: 'content', key: 'content' },
];

const pageTopupSum = computed(() =>
  topupRows.value.reduce((sum, r) => sum + (Number(r.quota) || 0), 0)
);

const sourceText = (record) => {
  const c = record.content || '';
  if (c.includes('兑换码')) return '兑换码';
  if (c.includes('微信') || c.includes('wechat')) return '微信支付';
  if (c.includes('管理') || c.includes('admin')) return '管理员充值';
  return '充值';
};

const sourceColor = (record) => {
  switch (sourceText(record)) {
    case '兑换码':
      return 'green';
    case '微信支付':
      return 'cyan';
    case '管理员充值':
      return 'orange';
    default:
      return 'blue';
  }
};

const rangeTs = () => {
  const startTs = Math.floor(new Date(startTime.value).getTime() / 1000);
  // Include the whole end day
  const endTs = Math.floor(new Date(endTime.value).getTime() / 1000) + 86400;
  return { startTs, endTs };
};

const loadReports = async () => {
  loading.value = true;
  const { startTs, endTs } = rangeTs();
  const res = await API.get(
    `/api/platform/reports/billing?start_time=${startTs}&end_time=${endTs}`
  );
  if (res.data.success) {
    reports.value = res.data.data || [];
  } else {
    showError(res.data.message);
  }
  loading.value = false;
};

const loadTopups = async () => {
  topupLoading.value = true;
  const { startTs, endTs } = rangeTs();
  const res = await API.get(
    `/api/log/?p=${topupPage.value}&page_size=${ITEMS_PER_PAGE}&type=1&start_timestamp=${startTs}&end_timestamp=${endTs}`
  );
  if (res.data.success) {
    const data = res.data.data || {};
    topupRows.value = data.items || [];
    topupTotal.value = data.total || 0;
  } else {
    showError(res.data.message);
  }
  topupLoading.value = false;
};

const onTopupPageChange = (page) => {
  topupPage.value = page;
  loadTopups();
};

const loadAll = () => {
  topupPage.value = 1;
  loadReports();
  loadTopups();
};

const exportCSV = () => {
  const { startTs, endTs } = rangeTs();
  window.open(
    `/api/platform/reports/billing/export?start_time=${startTs}&end_time=${endTs}`,
    '_blank'
  );
};

const exportTopupCSV = () => {
  const { startTs, endTs } = rangeTs();
  window.open(
    `/api/log/export?type=1&start_timestamp=${startTs}&end_timestamp=${endTs}`,
    '_blank'
  );
};

onMounted(() => {
  loadAll();
});
</script>
