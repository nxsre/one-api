<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">{{ t('tenant_console.tokens.title') }}</h2>
      <div class="mb-3 flex gap-2">
        <router-link to="/tenant-console/users">
          <a-button>{{ t('tenant_console.tokens.back_users') }}</a-button>
        </router-link>
        <router-link :to="`/tenant-console/users/${encodeURIComponent(ownerId)}/tokens/add`">
          <a-button type="primary">{{ t('tenant_console.tokens.add') }}</a-button>
        </router-link>
      </div>

      <a-table
        row-key="id"
        :columns="columns"
        :data-source="tokens"
        :loading="loading"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="statusColor(record.status)">{{ statusText(record.status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'created_time'">
            {{ timestamp2string(record.created_time) }}
          </template>
          <template v-else-if="column.key === 'key'">
            <a-button size="small" @click="copyKey(record)">
              {{ t('tenant_console.tokens.copy_sk') }}
            </a-button>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button size="small" @click="toggleStatus(record)">
                {{ record.status === 1 ? t('tenant_console.tokens.disable') : t('tenant_console.tokens.enable') }}
              </a-button>
              <router-link
                :to="`/tenant-console/users/${encodeURIComponent(ownerId)}/tokens/edit/${record.id}`"
              >
                <a-button size="small">{{ t('tenant_console.tokens.edit') }}</a-button>
              </router-link>
            </a-space>
          </template>
        </template>
      </a-table>

      <div class="mt-3 flex gap-2">
        <a-button :disabled="pageIdx <= 0 || loading" @click="load(pageIdx - 1)">
          {{ t('tenant_console.users.prev') }}
        </a-button>
        <a-button
          :disabled="tokens.length < ITEMS_PER_PAGE || loading"
          @click="load(pageIdx + 1)"
        >
          {{ t('tenant_console.users.next') }}
        </a-button>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { API, copy, showError, showSuccess, timestamp2string } from '@/helpers';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();
const route = useRoute();
const ownerId = computed(() => route.params.ownerId);

const tokens = ref([]);
const loading = ref(true);
const pageIdx = ref(0);

const columns = [
  { title: t('tenant_console.tokens.col_name'), dataIndex: 'name', key: 'name' },
  { title: t('tenant_console.tokens.col_status'), key: 'status' },
  { title: t('tenant_console.tokens.col_created'), key: 'created_time' },
  { title: t('tenant_console.tokens.col_key'), key: 'key' },
  { title: t('tenant_console.tokens.col_actions'), key: 'actions' },
];

function statusColor(status) {
  switch (status) {
    case 1:
      return 'green';
    case 2:
      return 'red';
    case 3:
      return 'gold';
    case 4:
      return 'default';
    default:
      return 'default';
  }
}

function statusText(status) {
  switch (status) {
    case 1:
      return t('token.table.status_enabled');
    case 2:
      return t('token.table.status_disabled');
    case 3:
      return t('token.table.status_expired');
    case 4:
      return t('token.table.status_depleted');
    default:
      return t('token.table.status_unknown');
  }
}

const load = async (p) => {
  loading.value = true;
  try {
    const res = await API.get(
      `/api/tenant_console/users/${encodeURIComponent(ownerId.value)}/tokens?p=${p}`
    );
    const { success, message, data } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    tokens.value = Array.isArray(data) ? data : [];
    pageIdx.value = p;
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

const toggleStatus = async (row) => {
  const next = row.status === 1 ? 2 : 1;
  try {
    const res = await API.put(
      `/api/tenant_console/users/${encodeURIComponent(ownerId.value)}/tokens?status_only=1`,
      { id: row.id, status: next }
    );
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    showSuccess(t('tenant_console.tokens.status_updated'));
    load(pageIdx.value);
  } catch (e) {
    showError(e.message);
  }
};

const copyKey = (row) => {
  copy(row.key ? `sk-${row.key}` : '').then(() =>
    showSuccess(t('tenant_console.tokens.copied'))
  );
};

watch(
  () => ownerId.value,
  () => {
    load(0);
  },
  { immediate: true }
);
</script>
