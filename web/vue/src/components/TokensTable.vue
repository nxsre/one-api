<template>
  <div>
    <a-form @submit.prevent="searchTokens">
      <a-input
        v-model:value="searchKeyword"
        allow-clear
        :placeholder="t('token.search')"
        @change="onKeywordChange"
        @pressEnter="searchTokens"
      >
        <template #prefix><SearchOutlined /></template>
      </a-input>
    </a-form>

    <a-table
      class="mt-4"
      size="small"
      row-key="id"
      :columns="columns"
      :data-source="pagedTokens"
      :loading="loading"
      :pagination="false"
    >
      <template #headerCell="{ column }">
        <span
          v-if="column.sortKey"
          style="cursor: pointer"
          @click="sortToken(column.sortKey)"
        >
          {{ column.title }}
        </span>
        <span v-else>{{ column.title }}</span>
      </template>

      <template #bodyCell="{ column, record, index }">
        <template v-if="column.key === 'name'">
          {{ record.name ? record.name : t('token.table.no_name') }}
        </template>

        <template v-else-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">
            {{ statusText(record.status) }}
          </a-tag>
        </template>

        <template v-else-if="column.key === 'used_quota'">
          {{ renderQuota(record.used_quota, t) }}
        </template>

        <template v-else-if="column.key === 'remain_quota'">
          {{ record.unlimited_quota ? t('token.table.unlimited') : renderQuota(record.remain_quota, t, 2) }}
        </template>

        <template v-else-if="column.key === 'created_time'">
          {{ timestamp2string(record.created_time) }}
        </template>

        <template v-else-if="column.key === 'expired_time'">
          {{ record.expired_time === -1 ? t('token.table.never_expire') : timestamp2string(record.expired_time) }}
        </template>

        <template v-else-if="column.key === 'actions'">
          <div class="flex items-center flex-wrap gap-2">
            <a-button-group size="small">
              <a-button type="primary" @click="onCopy('', record.key)">
                {{ t('token.buttons.copy') }}
              </a-button>
              <a-dropdown>
                <a-button type="primary"><DownOutlined /></a-button>
                <template #overlay>
                  <a-menu @click="({ key }) => onCopy(key, record.key)">
                    <a-menu-item v-for="opt in COPY_OPTIONS" :key="opt.value">
                      {{ opt.text }}
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </a-button-group>

            <a-button-group size="small">
              <a-button type="primary" @click="onOpenLink('', record.key)">
                {{ t('token.buttons.chat') }}
              </a-button>
              <a-dropdown>
                <a-button type="primary"><DownOutlined /></a-button>
                <template #overlay>
                  <a-menu @click="({ key }) => onOpenLink(key, record.key)">
                    <a-menu-item v-for="opt in OPEN_LINK_OPTIONS" :key="opt.value">
                      {{ opt.text }}
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </a-button-group>

            <a-popconfirm
              :title="`${t('token.buttons.confirm_delete')} ${record.name}`"
              @confirm="manageToken(record.id, 'delete', index)"
            >
              <a-button size="small" danger>{{ t('token.buttons.delete') }}</a-button>
            </a-popconfirm>

            <a-button
              size="small"
              @click="manageToken(record.id, record.status === 1 ? 'disable' : 'enable', index)"
            >
              {{ record.status === 1 ? t('token.buttons.disable') : t('token.buttons.enable') }}
            </a-button>

            <a-button size="small" @click="router.push('/token/edit/' + record.id)">
              {{ t('token.buttons.edit') }}
            </a-button>
          </div>
        </template>
      </template>
    </a-table>

    <div class="flex items-center flex-wrap gap-2 mt-4">
      <a-button size="small" :loading="loading" @click="router.push('/token/add')">
        {{ t('token.buttons.add') }}
      </a-button>
      <a-button size="small" :loading="loading" @click="refresh">
        {{ t('token.buttons.refresh') }}
      </a-button>
      <a-select
        v-model:value="orderBy"
        style="width: 200px; margin-left: 10px"
        :placeholder="t('token.sort.placeholder')"
        :options="orderByOptions"
        @change="handleOrderByChange"
      />
      <a-pagination
        class="ml-auto"
        size="small"
        :current="activePage"
        :total="totalItems"
        :page-size="ITEMS_PER_PAGE"
        :show-size-changer="false"
        @change="onPaginationChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { SearchOutlined, DownOutlined } from '@ant-design/icons-vue';
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

const COPY_OPTIONS = [
  { key: 'raw', text: t('token.copy_options.raw'), value: '' },
  { key: 'next', text: t('token.copy_options.next'), value: 'next' },
  { key: 'ama', text: t('token.copy_options.ama'), value: 'ama' },
  { key: 'opencat', text: t('token.copy_options.opencat'), value: 'opencat' },
  { key: 'lobe', text: t('token.copy_options.lobe'), value: 'lobechat' },
];

const OPEN_LINK_OPTIONS = [
  { key: 'next', text: t('token.copy_options.next'), value: 'next' },
  { key: 'ama', text: t('token.copy_options.ama'), value: 'ama' },
  { key: 'opencat', text: t('token.copy_options.opencat'), value: 'opencat' },
  { key: 'lobe', text: t('token.copy_options.lobe'), value: 'lobechat' },
];

const tokens = ref([]);
const loading = ref(true);
const activePage = ref(1);
const searchKeyword = ref('');
const searching = ref(false);
const orderBy = ref('');

const columns = [
  { title: t('token.table.name'), key: 'name', sortKey: 'name' },
  { title: t('token.table.status'), key: 'status', sortKey: 'status' },
  { title: t('token.table.used_quota'), key: 'used_quota', sortKey: 'used_quota' },
  { title: t('token.table.remain_quota'), key: 'remain_quota', sortKey: 'remain_quota' },
  { title: t('token.table.created_time'), key: 'created_time', sortKey: 'created_time' },
  { title: t('token.table.expired_time'), key: 'expired_time', sortKey: 'expired_time' },
  { title: t('token.table.actions'), key: 'actions' },
];

const orderByOptions = [
  { label: t('token.sort.default'), value: '' },
  { label: t('token.sort.by_remain'), value: 'remain_quota' },
  { label: t('token.sort.by_used'), value: 'used_quota' },
];

const visibleTokens = computed(() => tokens.value.filter((token) => !token.deleted));

const pagedTokens = computed(() =>
  visibleTokens.value.slice(
    (activePage.value - 1) * ITEMS_PER_PAGE,
    activePage.value * ITEMS_PER_PAGE
  )
);

// Mirror the source pagination math so an extra page is offered for lazy load-more.
const totalItems = computed(() => {
  const len = tokens.value.length;
  const pages = Math.ceil(len / ITEMS_PER_PAGE) + (len % ITEMS_PER_PAGE === 0 ? 1 : 0);
  return Math.max(pages, 1) * ITEMS_PER_PAGE;
});

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

const loadTokens = async (startIdx) => {
  const res = await API.get(`/api/token/?p=${startIdx}&order=${orderBy.value}`);
  const { success, message, data } = res.data;
  if (success) {
    if (startIdx === 0) {
      tokens.value = data;
    } else {
      let newTokens = [...tokens.value];
      newTokens.splice(startIdx * ITEMS_PER_PAGE, data.length, ...data);
      tokens.value = newTokens;
    }
  } else {
    showError(message);
  }
  loading.value = false;
};

const onPaginationChange = (page) => {
  (async () => {
    if (page === Math.ceil(tokens.value.length / ITEMS_PER_PAGE) + 1) {
      // In this case we have to load more data and then append them.
      await loadTokens(page - 1);
    }
    activePage.value = page;
  })();
};

const refresh = async () => {
  loading.value = true;
  await loadTokens(activePage.value - 1);
};

function resolveServerAddress() {
  let status = localStorage.getItem('status');
  let serverAddress = '';
  if (status) {
    status = JSON.parse(status);
    serverAddress = status.server_address;
  }
  if (serverAddress === '') {
    serverAddress = window.location.origin;
  }
  return serverAddress;
}

const onCopy = async (type, key) => {
  const serverAddress = resolveServerAddress();
  let encodedServerAddress = encodeURIComponent(serverAddress);
  const nextLink = localStorage.getItem('chat_link');
  let nextUrl;

  if (nextLink) {
    nextUrl = nextLink + `/#/?settings={"key":"sk-${key}","url":"${serverAddress}"}`;
  } else {
    nextUrl = `https://app.nextchat.dev/#/?settings={"key":"sk-${key}","url":"${serverAddress}"}`;
  }

  let url;
  switch (type) {
    case 'ama':
      url = `ama://set-api-key?server=${encodedServerAddress}&key=sk-${key}`;
      break;
    case 'opencat':
      url = `opencat://team/join?domain=${encodedServerAddress}&token=sk-${key}`;
      break;
    case 'next':
      url = nextUrl;
      break;
    case 'lobechat':
      url = nextLink + `/?settings={"keyVaults":{"openai":{"apiKey":"sk-${key}","baseURL":"${serverAddress}/v1"}}}`;
      break;
    default:
      url = `sk-${key}`;
  }
  if (await copy(url)) {
    showSuccess(t('token.messages.copy_success'));
  } else {
    showWarning(t('token.messages.copy_failed'));
    searchKeyword.value = url;
  }
};

const onOpenLink = async (type, key) => {
  const serverAddress = resolveServerAddress();
  let encodedServerAddress = encodeURIComponent(serverAddress);
  const chatLink = localStorage.getItem('chat_link');
  let defaultUrl;

  if (chatLink) {
    defaultUrl = chatLink + `/#/?settings={"key":"sk-${key}","url":"${serverAddress}"}`;
  } else {
    defaultUrl = `https://app.nextchat.dev/#/?settings={"key":"sk-${key}","url":"${serverAddress}"}`;
  }
  let url;
  switch (type) {
    case 'ama':
      url = `ama://set-api-key?server=${encodedServerAddress}&key=sk-${key}`;
      break;
    case 'opencat':
      url = `opencat://team/join?domain=${encodedServerAddress}&token=sk-${key}`;
      break;
    case 'lobechat':
      url = chatLink + `/?settings={"keyVaults":{"openai":{"apiKey":"sk-${key}","baseURL":"${serverAddress}/v1"}}}`;
      break;
    default:
      url = defaultUrl;
  }

  window.open(url, '_blank');
};

const manageToken = async (id, action, idx) => {
  let data = { id };
  let res;
  switch (action) {
    case 'delete':
      res = await API.delete(`/api/token/${id}/`);
      break;
    case 'enable':
      data.status = 1;
      res = await API.put('/api/token/?status_only=true', data);
      break;
    case 'disable':
      data.status = 2;
      res = await API.put('/api/token/?status_only=true', data);
      break;
  }
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('token.messages.operation_success'));
    let token = res.data.data;
    let newTokens = [...tokens.value];
    let realIdx = (activePage.value - 1) * ITEMS_PER_PAGE + idx;
    if (action === 'delete') {
      newTokens[realIdx].deleted = true;
    } else {
      newTokens[realIdx].status = token.status;
    }
    tokens.value = newTokens;
  } else {
    showError(message);
  }
};

const searchTokens = async () => {
  if (searchKeyword.value === '') {
    // if keyword is blank, load files instead.
    await loadTokens(0);
    activePage.value = 1;
    orderBy.value = '';
    return;
  }
  searching.value = true;
  const res = await API.get(`/api/token/search?keyword=${searchKeyword.value}`);
  const { success, message, data } = res.data;
  if (success) {
    tokens.value = data;
    activePage.value = 1;
  } else {
    showError(message);
  }
  searching.value = false;
};

const onKeywordChange = (e) => {
  searchKeyword.value = (e.target.value || '').trim();
};

const sortToken = (key) => {
  if (tokens.value.length === 0) return;
  loading.value = true;
  let sortedTokens = [...tokens.value];
  sortedTokens.sort((a, b) => {
    if (!isNaN(a[key])) {
      // If the value is numeric, subtract to sort
      return a[key] - b[key];
    } else {
      // If the value is not numeric, sort as strings
      return ('' + a[key]).localeCompare(b[key]);
    }
  });
  if (sortedTokens[0].id === tokens.value[0].id) {
    sortedTokens.reverse();
  }
  tokens.value = sortedTokens;
  loading.value = false;
};

const handleOrderByChange = (value) => {
  orderBy.value = value;
  activePage.value = 1;
};

watch(orderBy, () => {
  loadTokens(0)
    .then()
    .catch((reason) => {
      showError(reason);
    });
});

onMounted(() => {
  loadTokens(0)
    .then()
    .catch((reason) => {
      showError(reason);
    });
});
</script>
