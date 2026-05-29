<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">{{ t('tenant_console.users.title') }}</h2>
      <a-form layout="inline" class="mb-4">
        <a-form-item>
          <a-input
            v-model:value="keyword"
            :placeholder="t('tenant_console.users.search_placeholder')"
            style="width: 320px"
            @press-enter="search"
          />
        </a-form-item>
        <a-form-item>
          <a-button :loading="searching" @click="search">
            {{ t('tenant_console.users.search') }}
          </a-button>
        </a-form-item>
        <a-form-item>
          <router-link to="/tenant-console/users/add">
            <a-button type="primary">{{ t('tenant_console.users.add') }}</a-button>
          </router-link>
        </a-form-item>
      </a-form>

      <a-table
        row-key="user_id"
        :columns="columns"
        :data-source="users"
        :loading="loading"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'username'">{{ renderText(record.username) }}</template>
          <template v-else-if="column.key === 'display_name'">{{ renderText(record.display_name) }}</template>
          <template v-else-if="column.key === 'group'">{{ renderText(record.group) }}</template>
          <template v-else-if="column.key === 'quota'">{{ renderQuota(record.quota, t) }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <router-link :to="`/tenant-console/users/edit/${encodeURIComponent(record.user_id)}`">
                <a-button size="small">{{ t('tenant_console.users.edit') }}</a-button>
              </router-link>
              <router-link :to="`/tenant-console/users/${encodeURIComponent(record.user_id)}/tokens`">
                <a-button size="small">{{ t('tenant_console.users.tokens') }}</a-button>
              </router-link>
              <a-button size="small" danger @click="openDelete(record)">
                {{ t('tenant_console.users.delete') }}
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>

      <div v-if="!keyword.trim()" class="mt-3 flex items-center gap-2">
        <a-button :disabled="pageIdx <= 0 || loading" @click="loadPage(pageIdx - 1)">
          {{ t('tenant_console.users.prev') }}
        </a-button>
        <a-button
          :disabled="users.length < ITEMS_PER_PAGE || loading"
          @click="loadPage(pageIdx + 1)"
        >
          {{ t('tenant_console.users.next') }}
        </a-button>
        <span class="ml-3 opacity-80">
          {{ t('tenant_console.users.page_hint', { n: pageIdx + 1 }) }}
        </span>
      </div>
    </a-card>

    <a-modal
      v-model:open="delOpen"
      :title="t('tenant_console.users.delete_confirm_title')"
      @cancel="delOpen = false"
    >
      <p>
        {{ t('tenant_console.users.delete_confirm_body', { name: delTarget?.username || '' }) }}
      </p>
      <template #footer>
        <a-button @click="delOpen = false">{{ t('tenant_console.users.cancel') }}</a-button>
        <a-button danger type="primary" @click="confirmDelete">
          {{ t('tenant_console.users.delete') }}
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { API, showError, showSuccess, renderQuota } from '@/helpers';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();
const users = ref([]);
const loading = ref(true);
const pageIdx = ref(0);
const keyword = ref('');
const searching = ref(false);
const delOpen = ref(false);
const delTarget = ref(null);

const columns = [
  { title: t('tenant_console.users.col_username'), key: 'username' },
  { title: t('tenant_console.users.col_display_name'), key: 'display_name' },
  { title: t('tenant_console.users.col_group'), key: 'group' },
  { title: t('tenant_console.users.col_quota'), key: 'quota' },
  { title: t('tenant_console.users.col_actions'), key: 'actions' },
];

function renderText(v) {
  if (v === undefined || v === null || v === '') return '—';
  return String(v);
}

const loadPage = async (p) => {
  loading.value = true;
  try {
    const res = await API.get(`/api/tenant_console/users?p=${p}`);
    const { success, message, data } = res.data;
    if (!success) {
      if (String(message).includes('请先在页面顶部选择')) {
        users.value = [];
      } else {
        showError(message);
      }
      return;
    }
    users.value = Array.isArray(data) ? data : [];
    pageIdx.value = p;
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

const search = async () => {
  const kw = keyword.value.trim();
  if (!kw) {
    loadPage(0);
    return;
  }
  searching.value = true;
  try {
    const res = await API.get(
      `/api/tenant_console/users/search?keyword=${encodeURIComponent(kw)}`
    );
    const { success, message, data } = res.data;
    if (!success) {
      if (String(message).includes('请先在页面顶部选择')) {
        users.value = [];
      } else {
        showError(message);
      }
      return;
    }
    users.value = Array.isArray(data) ? data : [];
    pageIdx.value = 0;
  } catch (e) {
    showError(e.message);
  } finally {
    searching.value = false;
  }
};

const openDelete = (u) => {
  delTarget.value = u;
  delOpen.value = true;
};

const confirmDelete = async () => {
  if (!delTarget.value) return;
  try {
    const res = await API.delete(
      `/api/tenant_console/users/${encodeURIComponent(delTarget.value.user_id)}`
    );
    const { success, message } = res.data;
    if (!success) {
      if (String(message).includes('请先在页面顶部选择')) {
        users.value = [];
      } else {
        showError(message);
      }
      return;
    }
    showSuccess(t('tenant_console.users.delete_success'));
    delOpen.value = false;
    delTarget.value = null;
    loadPage(0);
  } catch (e) {
    showError(e.message);
  }
};

onMounted(() => {
  loadPage(0);
});
</script>
