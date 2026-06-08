<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">{{ t('tenant_console.channels.title') }}</h2>
      <div class="mb-3">
        <router-link to="/tenant-console/channels/add">
          <a-button type="primary">{{ t('tenant_console.channels.add') }}</a-button>
        </router-link>
      </div>

      <a-table
        row-key="id"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">{{ renderNumber(record.type) }}</template>
          <template v-else-if="column.key === 'priority'">{{ record.priority ?? '—' }}</template>
          <template v-else-if="column.key === 'api_call_count'">{{ record.api_call_count ?? 0 }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="goEdit(record)">
              {{ t('tenant_console.channels.edit') }}
            </a-button>
          </template>
        </template>
      </a-table>

      <div class="mt-3 flex gap-2">
        <a-button :disabled="pageIdx <= 0 || loading" @click="load(pageIdx - 1)">
          {{ t('tenant_console.users.prev') }}
        </a-button>
        <a-button
          :disabled="rows.length < ITEMS_PER_PAGE || loading"
          @click="load(pageIdx + 1)"
        >
          {{ t('tenant_console.users.next') }}
        </a-button>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { API, showError, renderNumber } from '@/helpers';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();
const router = useRouter();
const rows = ref([]);
const loading = ref(true);
const pageIdx = ref(0);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: t('tenant_console.channels.col_name'), dataIndex: 'name', key: 'name' },
  { title: t('tenant_console.channels.col_type'), key: 'type' },
  { title: t('tenant_console.channels.col_status'), dataIndex: 'status', key: 'status' },
  { title: t('tenant_console.channels.col_priority'), key: 'priority' },
  { title: 'API 调用次数', key: 'api_call_count' },
  { title: t('tenant_console.channels.col_actions'), key: 'actions' },
];

const goEdit = (ch) => {
  router.push({ path: `/tenant-console/channels/edit/${ch.id}`, state: { channel: ch } });
};

const load = async (p) => {
  loading.value = true;
  try {
    const res = await API.get(
      `/api/tenant_console/channels?p=${p}&page_size=${ITEMS_PER_PAGE}`
    );
    const { success, message, data } = res.data;
    if (!success) {
      if (String(message).includes('请先在页面顶部选择')) {
        rows.value = [];
      } else {
        showError(message);
      }
      return;
    }
    rows.value = Array.isArray(data) ? data : [];
    pageIdx.value = p;
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  load(0);
});
</script>
