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
          <a-button type="primary" :loading="loading" @click="loadReports">查询</a-button>
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { API, showError } from '@/helpers';

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

const loadReports = async () => {
  loading.value = true;
  const startTs = Math.floor(new Date(startTime.value).getTime() / 1000);
  // Include the whole end day
  const endTs = Math.floor(new Date(endTime.value).getTime() / 1000) + 86400;

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

const exportCSV = () => {
  const startTs = Math.floor(new Date(startTime.value).getTime() / 1000);
  const endTs = Math.floor(new Date(endTime.value).getTime() / 1000) + 86400;
  window.open(
    `/api/platform/reports/billing/export?start_time=${startTs}&end_time=${endTs}`,
    '_blank'
  );
};

onMounted(() => {
  loadReports();
});
</script>
