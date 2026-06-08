<template>
  <div>
    <a-form @submit.prevent="searchRedemptions">
      <a-input
        v-model:value="searchKeyword"
        :placeholder="t('redemption.search')"
        allow-clear
        :loading="searching"
        @change="handleKeywordChange"
        @press-enter="searchRedemptions"
      >
        <template #prefix><SearchOutlined /></template>
      </a-input>
    </a-form>

    <a-table
      class="mt-3"
      size="small"
      row-key="id"
      :columns="columns"
      :data-source="pagedRedemptions"
      :loading="loading"
      :pagination="false"
    >
      <template #headerCell="{ column }">
        <span style="cursor: pointer" @click="sortRedemption(column.sortField)">
          {{ column.title }}
        </span>
      </template>

      <template #bodyCell="{ column, record, index }">
        <template v-if="column.key === 'name'">
          {{ record.name ? record.name : t('redemption.table.no_name') }}
        </template>

        <template v-else-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">
            {{ statusText(record.status) }}
          </a-tag>
        </template>

        <template v-else-if="column.key === 'quota'">
          {{ renderQuota(record.quota, t) }}
        </template>

        <template v-else-if="column.key === 'created_time'">
          {{ timestamp2string(record.created_time) }}
        </template>

        <template v-else-if="column.key === 'redeemed_time'">
          {{ record.redeemed_time ? timestamp2string(record.redeemed_time) : t('redemption.table.not_redeemed') }}
        </template>

        <template v-else-if="column.key === 'actions'">
          <div class="flex flex-wrap gap-2">
            <a-button size="small" type="primary" @click="handleCopy(record)">
              {{ t('redemption.buttons.copy') }}
            </a-button>
            <a-popconfirm
              :title="t('redemption.buttons.confirm_delete')"
              :ok-text="t('redemption.buttons.confirm_delete')"
              @confirm="manageRedemption(record.id, 'delete', index)"
            >
              <a-button size="small" danger>
                {{ t('redemption.buttons.delete') }}
              </a-button>
            </a-popconfirm>
            <a-button
              size="small"
              :disabled="record.status === 3"
              @click="manageRedemption(record.id, record.status === 1 ? 'disable' : 'enable', index)"
            >
              {{ record.status === 1 ? t('redemption.buttons.disable') : t('redemption.buttons.enable') }}
            </a-button>
            <a-button size="small" @click="router.push('/redemption/edit/' + record.id)">
              {{ t('redemption.buttons.edit') }}
            </a-button>
          </div>
        </template>
      </template>
    </a-table>

    <div class="flex items-center justify-between mt-3">
      <div class="flex gap-2">
        <a-button size="small" :loading="loading" @click="router.push('/redemption/add')">
          {{ t('redemption.buttons.add') }}
        </a-button>
        <a-button size="small" :loading="loading" @click="refresh">
          {{ t('redemption.buttons.refresh') }}
        </a-button>
      </div>
      <a-pagination
        v-model:current="activePage"
        size="small"
        :total="paginationTotal"
        :page-size="ITEMS_PER_PAGE"
        :show-size-changer="false"
        @change="onPaginationChange"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { SearchOutlined } from '@ant-design/icons-vue';
import {
  API,
  copy,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
  renderQuota,
} from '@/helpers';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();
const router = useRouter();

const redemptions = ref([]);
const loading = ref(true);
const activePage = ref(1);
const searchKeyword = ref('');
const searching = ref(false);

const columns = [
  { title: t('redemption.table.id'), dataIndex: 'id', key: 'id', sortField: 'id' },
  { title: t('redemption.table.name'), dataIndex: 'name', key: 'name', sortField: 'name' },
  { title: t('redemption.table.status'), dataIndex: 'status', key: 'status', sortField: 'status' },
  { title: t('redemption.table.quota'), dataIndex: 'quota', key: 'quota', sortField: 'quota' },
  { title: t('redemption.table.created_time'), dataIndex: 'created_time', key: 'created_time', sortField: 'created_time' },
  { title: t('redemption.table.redeemed_time'), dataIndex: 'redeemed_time', key: 'redeemed_time', sortField: 'redeemed_time' },
  { title: t('redemption.table.actions'), key: 'actions' },
];

const statusText = (status) => {
  switch (status) {
    case 1:
      return t('redemption.status.unused');
    case 2:
      return t('redemption.status.disabled');
    case 3:
      return t('redemption.status.used');
    default:
      return t('redemption.status.unknown');
  }
};

const statusColor = (status) => {
  switch (status) {
    case 1:
      return 'green';
    case 2:
      return 'red';
    case 3:
      return 'default';
    default:
      return 'default';
  }
};

const visibleRedemptions = computed(() =>
  redemptions.value.filter((r) => !r.deleted)
);

const pagedRedemptions = computed(() =>
  visibleRedemptions.value.slice(
    (activePage.value - 1) * ITEMS_PER_PAGE,
    activePage.value * ITEMS_PER_PAGE
  )
);

// Mirror the React pagination size: an extra page when the last page is full,
// so that loading-more (append) keeps working.
const paginationTotal = computed(() => {
  const len = redemptions.value.length;
  const pages =
    Math.ceil(len / ITEMS_PER_PAGE) +
    (len % ITEMS_PER_PAGE === 0 ? 1 : 0);
  return Math.max(1, pages) * ITEMS_PER_PAGE;
});

const loadRedemptions = async (startIdx) => {
  const res = await API.get(`/api/redemption/?p=${startIdx}`);
  const { success, message, data } = res.data;
  if (success) {
    if (startIdx === 0) {
      redemptions.value = data;
    } else {
      redemptions.value = [...redemptions.value, ...data];
    }
  } else {
    showError(message);
  }
  loading.value = false;
};

const onPaginationChange = async (page) => {
  if (page === Math.ceil(redemptions.value.length / ITEMS_PER_PAGE) + 1) {
    // In this case we have to load more data and then append them.
    await loadRedemptions(page - 1);
  }
  activePage.value = page;
};

const manageRedemption = async (id, action, idx) => {
  const data = { id };
  let res;
  switch (action) {
    case 'delete':
      res = await API.delete(`/api/redemption/${id}/`);
      break;
    case 'enable':
      data.status = 1;
      res = await API.put('/api/redemption/?status_only=true', data);
      break;
    case 'disable':
      data.status = 2;
      res = await API.put('/api/redemption/?status_only=true', data);
      break;
  }
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('token.messages.operation_success'));
    const redemption = res.data.data;
    const newRedemptions = [...redemptions.value];
    const realIdx = (activePage.value - 1) * ITEMS_PER_PAGE + idx;
    if (action === 'delete') {
      newRedemptions[realIdx].deleted = true;
    } else {
      newRedemptions[realIdx].status = redemption.status;
    }
    redemptions.value = newRedemptions;
  } else {
    showError(message);
  }
};

const searchRedemptions = async () => {
  if (searchKeyword.value === '') {
    // if keyword is blank, load files instead.
    await loadRedemptions(0);
    activePage.value = 1;
    return;
  }
  searching.value = true;
  const res = await API.get(`/api/redemption/search?keyword=${searchKeyword.value}`);
  const { success, message, data } = res.data;
  if (success) {
    redemptions.value = data;
    activePage.value = 1;
  } else {
    showError(message);
  }
  searching.value = false;
};

const handleKeywordChange = (e) => {
  searchKeyword.value = (e.target.value || '').trim();
};

const sortRedemption = (key) => {
  if (!key) return;
  if (redemptions.value.length === 0) return;
  loading.value = true;
  const sortedRedemptions = [...redemptions.value];
  sortedRedemptions.sort((a, b) => {
    if (!isNaN(a[key])) {
      return a[key] - b[key];
    }
    return ('' + a[key]).localeCompare(b[key]);
  });
  if (sortedRedemptions[0].id === redemptions.value[0].id) {
    sortedRedemptions.reverse();
  }
  redemptions.value = sortedRedemptions;
  loading.value = false;
};

const handleCopy = async (redemption) => {
  if (await copy(redemption.key)) {
    showSuccess(t('token.messages.copy_success'));
  } else {
    showWarning(t('token.messages.copy_failed'));
    searchKeyword.value = redemption.key;
  }
};

const refresh = async () => {
  loading.value = true;
  await loadRedemptions(0);
  activePage.value = 1;
};

onMounted(() => {
  loadRedemptions(0).catch((reason) => {
    showError(reason);
  });
});
</script>
