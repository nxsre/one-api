<template>
  <div>
    <a-form layout="inline" class="mb-4" @submit.prevent="searchUsers">
      <a-form-item>
        <a-select v-model:value="scope" style="min-width: 140px" :options="scopeOptions" />
      </a-form-item>
      <a-form-item v-if="scope === 'tenant'">
        <a-select
          v-model:value="tenantId"
          allow-clear
          show-search
          style="min-width: 180px"
          :placeholder="t('user.table.select_tenant')"
          :options="tenantOptions"
          :filter-option="filterTenantOption"
        />
      </a-form-item>
      <a-form-item>
        <a-select
          v-model:value="roles"
          mode="multiple"
          allow-clear
          style="min-width: 200px"
          :placeholder="t('user.table.filter_role')"
          :options="roleFilterOptions"
        />
      </a-form-item>
      <a-form-item>
        <a-input v-model:value="searchKeyword" :placeholder="t('user.search')">
          <template #prefix><SearchOutlined /></template>
        </a-input>
      </a-form-item>
      <a-form-item>
        <a-button type="primary" html-type="submit" :loading="loading" @click="searchUsers">
          {{ t('user.search') }}
        </a-button>
      </a-form-item>
    </a-form>

    <a-table
      :columns="columns"
      :data-source="users"
      :loading="loading"
      :pagination="false"
      row-key="user_id"
      size="small"
    >
      <template #headerCell="{ column }">
        <span
          v-if="column.sortable"
          style="cursor: pointer"
          @click="sortUser(column.key)"
        >{{ column.title }}</span>
        <span v-else>{{ column.title }}</span>
      </template>

      <template #bodyCell="{ column, record, index }">
        <template v-if="column.key === 'username'">
          <a-popover :title="record.display_name ? record.display_name : record.username">
            <template #content>{{ record.email ? record.email : '未绑定邮箱地址' }}</template>
            <span>{{ renderText(record.username, 15) }}</span>
          </a-popover>
        </template>

        <template v-else-if="column.key === 'group'">
          <component :is="renderGroup(record.group)" />
        </template>

        <template v-else-if="column.key === 'tenant'">
          <a-tag v-if="record.tenant_id" color="blue">{{ getTenantName(record.tenant_id) }}</a-tag>
          <span v-else style="color: gray">独立用户</span>
        </template>

        <template v-else-if="column.key === 'quota'">
          <a-tooltip :title="t('user.table.remaining_quota')">
            <a-tag><component :is="renderQuotaNode(record.quota)" /></a-tag>
          </a-tooltip>
          <a-tooltip :title="t('user.table.used_quota')">
            <a-tag><component :is="renderQuotaNode(record.used_quota)" /></a-tag>
          </a-tooltip>
          <a-tooltip :title="t('user.table.request_count')">
            <a-tag>{{ renderNumber(record.request_count) }}</a-tag>
          </a-tooltip>
        </template>

        <template v-else-if="column.key === 'role'">
          <a-tag :color="roleColor(record.role)">{{ roleText(record.role) }}</a-tag>
        </template>

        <template v-else-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">{{ statusText(record.status) }}</a-tag>
        </template>

        <template v-else-if="column.key === 'actions'">
          <a-space>
            <a-popconfirm
              :title="`${t('user.buttons.delete_user')} ${record.username}`"
              :disabled="record.role >= 50"
              @confirm="manageUser(record.username, 'delete', index)"
            >
              <a-button size="small" danger :disabled="record.role >= 50">
                {{ t('user.buttons.delete') }}
              </a-button>
            </a-popconfirm>
            <a-button
              size="small"
              :disabled="record.role >= 50"
              @click="manageUser(record.username, record.status === 1 ? 'disable' : 'enable', index)"
            >
              {{ record.status === 1 ? t('user.buttons.disable') : t('user.buttons.enable') }}
            </a-button>
            <router-link :to="'/user/edit/' + record.user_id">
              <a-button size="small">{{ t('user.buttons.edit') }}</a-button>
            </router-link>
          </a-space>
        </template>
      </template>
    </a-table>

    <div class="flex items-center mt-4" style="gap: 10px">
      <router-link to="/user/add">
        <a-button size="small" :loading="loading">{{ t('user.buttons.add') }}</a-button>
      </router-link>
      <a-select
        v-model:value="orderBy"
        style="min-width: 160px"
        :placeholder="t('user.table.sort_by')"
        :options="sortOptions"
        @change="handleOrderByChange"
      />
      <a-pagination
        class="ml-auto"
        :current="activePage"
        :total="totalCount"
        :page-size="ITEMS_PER_PAGE"
        :show-size-changer="false"
        size="small"
        @change="onPaginationChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, h } from 'vue';
import { useI18n } from 'vue-i18n';
import { SearchOutlined } from '@ant-design/icons-vue';
import { API, showError, showSuccess } from '@/helpers';
import { renderGroup, renderNumber, renderQuota, renderText } from '@/helpers/render';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();

const users = ref([]);
const loading = ref(true);
const activePage = ref(1);
const totalCount = ref(0);
const searchKeyword = ref('');
const orderBy = ref('');
const scope = ref('independent');
const tenantId = ref('');
const roles = ref([]);
const tenantOptions = ref([]);

const scopeOptions = computed(() => [
  { value: 'independent', label: t('user.table.scope.independent') },
  { value: 'tenant', label: t('user.table.scope.tenant') },
  { value: 'all', label: t('user.table.scope.all') },
]);

const roleFilterOptions = computed(() => [
  { value: 1, label: t('user.table.role_types.normal') },
  { value: 10, label: t('user.table.role_types.admin') },
  { value: 20, label: t('user.table.role_types.tenant_admin') },
  { value: 50, label: t('user.table.role_types.super_admin') },
  { value: 100, label: t('user.table.role_types.root') },
]);

const sortOptions = computed(() => [
  { value: '', label: t('user.table.sort.default') },
  { value: 'quota', label: t('user.table.sort.quota') },
  { value: 'used_quota', label: t('user.table.sort.used_quota') },
  { value: 'request_count', label: t('user.table.sort.request_count') },
]);

const columns = computed(() => [
  { title: t('user.table.id'), key: 'id', dataIndex: 'user_id', sortable: true },
  { title: t('user.table.username'), key: 'username', sortable: true },
  { title: t('user.table.group'), key: 'group', sortable: true },
  { title: t('user.table.tenant'), key: 'tenant' },
  { title: t('user.table.quota'), key: 'quota', sortable: true },
  { title: t('user.table.role_text'), key: 'role', sortable: true },
  { title: t('user.table.status_text'), key: 'status', sortable: true },
  { title: t('user.table.actions'), key: 'actions' },
]);

// renderQuota returns either a string or a VNode; normalize to a VNode for <component :is>.
const renderQuotaNode = (value) => {
  const rendered = renderQuota(value, t);
  return typeof rendered === 'object' ? rendered : h('span', rendered);
};

const roleText = (role) => {
  switch (role) {
    case 1:
      return t('user.table.role_types.normal');
    case 10:
      return t('user.table.role_types.admin');
    case 20:
      return t('user.table.role_types.tenant_admin');
    case 50:
      return t('user.table.role_types.super_admin');
    case 100:
      return t('user.table.role_types.root');
    default:
      return t('user.table.role_types.unknown');
  }
};

const roleColor = (role) => {
  switch (role) {
    case 1:
      return 'default';
    case 10:
      return 'gold';
    case 20:
      return 'cyan';
    case 50:
      return 'purple';
    case 100:
      return 'orange';
    default:
      return 'red';
  }
};

const statusText = (status) => {
  switch (status) {
    case 1:
      return t('user.table.status_types.activated');
    case 2:
      return t('user.table.status_types.banned');
    default:
      return t('user.table.status_types.unknown');
  }
};

const statusColor = (status) => {
  switch (status) {
    case 1:
      return 'default';
    case 2:
      return 'red';
    default:
      return 'default';
  }
};

const filterTenantOption = (input, option) =>
  String(option.label).toLowerCase().includes(String(input).toLowerCase());

const fetchTenants = async () => {
  try {
    const res = await API.get('/api/platform/tenants/options');
    const { success, data } = res.data;
    if (success) {
      tenantOptions.value = data.map((tenant) => ({
        value: tenant.id,
        label: tenant.name,
      }));
    }
  } catch (e) {
    // eslint-disable-next-line no-console
    console.error(e);
  }
};

const loadUsers = async (page) => {
  loading.value = true;
  try {
    let url = `/api/user/paged?p=${page - 1}&page_size=${ITEMS_PER_PAGE}&order=${orderBy.value}&scope=${scope.value}`;
    if (scope.value === 'tenant' && tenantId.value) {
      url += `&tenant_id=${tenantId.value}`;
    }
    if (roles.value.length > 0) {
      url += `&roles=${roles.value.join(',')}`;
    }
    if (searchKeyword.value) {
      url += `&keyword=${encodeURIComponent(searchKeyword.value)}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      users.value = data.items || [];
      totalCount.value = data.total || 0;
    } else {
      showError(message);
    }
  } catch (err) {
    showError(err.message);
  }
  loading.value = false;
};

const onPaginationChange = (page) => {
  activePage.value = page;
  loadUsers(page);
};

const searchUsers = async () => {
  activePage.value = 1;
  await loadUsers(1);
};

const sortUser = (key) => {
  let newOrder = key;
  if (orderBy.value === key) {
    newOrder = key + '_desc';
  }
  orderBy.value = newOrder;
};

const handleOrderByChange = (value) => {
  orderBy.value = value;
};

const manageUser = async (username, action, idx) => {
  try {
    const res = await API.post('/api/user/manage', { username, action });
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('user.messages.operation_success'));
      const user = res.data.data;
      const newUsers = [...users.value];
      if (action === 'delete') {
        newUsers.splice(idx, 1);
      } else {
        newUsers[idx].status = user.status;
        newUsers[idx].role = user.role;
      }
      users.value = newUsers;
    } else {
      showError(message);
    }
  } catch (e) {
    showError(e.message);
  }
};

const getTenantName = (tid) => {
  const found = tenantOptions.value.find((o) => o.value === tid);
  return found ? found.label : `租户 ${tid}`;
};

onMounted(() => {
  fetchTenants();
});

// Reload (and reset to first page) whenever filters/order change, mirroring the
// React loadUsers dependency array + effect.
watch(
  [orderBy, scope, tenantId, roles, searchKeyword],
  () => {
    activePage.value = 1;
    loadUsers(1);
  },
  { immediate: true }
);
</script>
