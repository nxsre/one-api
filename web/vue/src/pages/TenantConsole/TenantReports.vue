<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">账单与报表</h2>
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

const startTime = ref(
  new Date(new Date().setDate(new Date().getDate() - 30)).toISOString().slice(0, 10)
);
const endTime = ref(new Date().toISOString().slice(0, 10));

const columns = [
  { title: '用户 ID', dataIndex: 'user_id', key: 'user_id' },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '渠道 ID', dataIndex: 'channel_id', key: 'channel_id' },
  { title: '渠道名', dataIndex: 'channel_name', key: 'channel_name' },
  { title: '模型名称', dataIndex: 'model_name', key: 'model_name' },
  { title: '消费额度 (Quota)', dataIndex: 'quota', key: 'quota' },
  { title: '请求次数', dataIndex: 'request_count', key: 'request_count' },
  { title: '提示 Token', dataIndex: 'prompt_tokens', key: 'prompt_tokens' },
  { title: '补全 Token', dataIndex: 'completion_tokens', key: 'completion_tokens' },
];

const rows = computed(() => reports.value.map((r, i) => ({ ...r, _rowKey: i })));

const loadReports = async () => {
  loading.value = true;
  const startTs = Math.floor(new Date(startTime.value).getTime() / 1000);
  const endTs = Math.floor(new Date(endTime.value).getTime() / 1000) + 86400;

  const res = await API.get(
    `/api/tenant_console/reports/billing?start_time=${startTs}&end_time=${endTs}`
  );
  if (res.data.success) {
    reports.value = res.data.data || [];
  } else {
    // 避免代管模式下未选择租户时弹红框报错，如果报错包含“请先在页面顶部选择”，则静默处理
    if (String(res.data.message).includes('请先在页面顶部选择')) {
      reports.value = [];
    } else {
      showError(res.data.message);
    }
  }
  loading.value = false;
};

const exportCSV = () => {
  const startTs = Math.floor(new Date(startTime.value).getTime() / 1000);
  const endTs = Math.floor(new Date(endTime.value).getTime() / 1000) + 86400;
  window.open(
    `/api/tenant_console/reports/billing/export?start_time=${startTs}&end_time=${endTs}`,
    '_blank'
  );
};

onMounted(() => {
  loadReports();
});
</script>
