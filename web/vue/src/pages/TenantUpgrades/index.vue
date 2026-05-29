<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">租户升级申请</h2>
      <a-table
        row-key="id"
        :columns="columns"
        :data-source="requests"
        :loading="loading"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'created_time'">
            {{ timestamp2string(record.created_time) }}
          </template>
          <template v-else-if="column.key === 'status'">
            <span v-if="record.status === 0" style="color: orange">待审核</span>
            <span v-else-if="record.status === 1" style="color: green">已通过</span>
            <span v-else-if="record.status === 2" style="color: red">已拒绝</span>
            <span v-else>未知</span>
          </template>
          <template v-else-if="column.key === 'actions'">
            <template v-if="record.status === 0">
              <a-space>
                <a-button size="small" type="primary" @click="approveRequest(record.id)">
                  通过
                </a-button>
                <a-button size="small" danger @click="rejectRequest(record.id)">
                  拒绝
                </a-button>
              </a-space>
            </template>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { API, showError, showSuccess, timestamp2string } from '@/helpers';

const requests = ref([]);
const loading = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '用户 ID', dataIndex: 'user_id', key: 'user_id' },
  { title: '企业名称', dataIndex: 'name', key: 'name' },
  { title: '租户标识 (Slug)', dataIndex: 'slug', key: 'slug' },
  { title: '备注', dataIndex: 'remark', key: 'remark' },
  { title: '状态', key: 'status' },
  { title: '申请时间', key: 'created_time' },
  { title: '操作', key: 'actions' },
];

const loadRequests = async () => {
  loading.value = true;
  const res = await API.get('/api/platform/tenants/upgrades');
  const { success, message, data } = res.data;
  if (success) {
    requests.value = data || [];
  } else {
    showError(message);
  }
  loading.value = false;
};

const approveRequest = async (id) => {
  const res = await API.post(`/api/platform/tenants/upgrades/${id}/approve`);
  if (res.data.success) {
    showSuccess('审批通过，租户已创建');
    loadRequests();
  } else {
    showError(res.data.message);
  }
};

const rejectRequest = async (id) => {
  const res = await API.post(`/api/platform/tenants/upgrades/${id}/reject`);
  if (res.data.success) {
    showSuccess('已拒绝');
    loadRequests();
  } else {
    showError(res.data.message);
  }
};

onMounted(() => {
  loadRequests();
});
</script>
