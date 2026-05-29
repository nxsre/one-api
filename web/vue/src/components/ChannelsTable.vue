<template>
  <div>
    <a-form layout="inline" class="channels-table-search" @submit.prevent="searchChannels">
      <a-input-search
        v-model:value="searchKeyword"
        :placeholder="t('channel.search')"
        :loading="searching"
        allow-clear
        style="width: 100%"
        @search="searchChannels"
      />
    </a-form>

    <a-alert
      v-if="showPrompt"
      class="channels-table-notice"
      type="info"
      closable
      style="margin: 12px 0"
      @close="dismissPrompt"
    >
      <template #description>
        <p>{{ t('channel.balance_notice') }}</p>
        <p>{{ t('channel.test_notice') }}</p>
        <p>{{ t('channel.detail_notice') }}</p>
      </template>
    </a-alert>

    <a-table
      :columns="columns"
      :data-source="pagedChannels"
      :loading="loading"
      :pagination="false"
      row-key="id"
      size="small"
      style="margin-top: 12px"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'id'">
          {{ record.id }}
        </template>
        <template v-else-if="column.key === 'name'">
          {{ record.name ? record.name : t('channel.table.no_name') }}
        </template>
        <template v-else-if="column.key === 'group'">
          <component :is="renderGroup(record.group)" />
        </template>
        <template v-else-if="column.key === 'type'">
          <component :is="renderType(record.type)" />
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tooltip v-if="record.status === 2" :title="t('channel.table.status_disabled_tip')">
            <a-tag color="red">{{ t('channel.table.status_disabled') }}</a-tag>
          </a-tooltip>
          <a-tooltip v-else-if="record.status === 3" :title="t('channel.table.status_auto_disabled_tip')">
            <a-tag color="gold">{{ t('channel.table.status_auto_disabled') }}</a-tag>
          </a-tooltip>
          <a-tag v-else-if="record.status === 1" color="green">{{ t('channel.table.status_enabled') }}</a-tag>
          <a-tag v-else>{{ t('channel.table.status_unknown') }}</a-tag>
        </template>
        <template v-else-if="column.key === 'response_time'">
          <a-tooltip
            :title="record.test_time ? timestamp2string(record.test_time) : t('channel.table.not_tested')"
          >
            <a-tag :color="responseTimeColor(record.response_time)">
              {{ responseTimeText(record.response_time) }}
            </a-tag>
          </a-tooltip>
        </template>
        <template v-else-if="column.key === 'balance'">
          <a-tooltip :title="t('channel.table.click_to_update')">
            <span
              style="cursor: pointer"
              @click="updateChannelBalance(record.id, record.name)"
            >
              {{ t('channel.table.balance_not_supported') }}
            </span>
          </a-tooltip>
        </template>
        <template v-else-if="column.key === 'priority'">
          <a-tooltip :title="t('channel.table.priority_tip')">
            <a-input-number
              :default-value="record.priority"
              :controls="false"
              style="max-width: 70px"
              @blur="(e) => manageChannel(record.id, 'priority', e.target.value)"
            />
          </a-tooltip>
        </template>
        <template v-else-if="column.key === 'test_model'">
          <a-select
            :placeholder="t('channel.table.select_test_model')"
            :options="record.model_options"
            :default-value="record.test_model"
            :field-names="{ label: 'text', value: 'value' }"
            style="width: 220px"
            @change="(v) => switchTestModel(record.id, v)"
          />
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="channels-table-actions">
            <a-button
              size="small"
              type="primary"
              @click="testChannel(record.id, record.name, record.test_model)"
            >
              {{ t('channel.buttons.test') }}
            </a-button>
            <a-popconfirm
              :title="`${t('channel.buttons.confirm_delete')} ${record.name}`"
              @confirm="manageChannel(record.id, 'delete')"
            >
              <a-button size="small" danger>{{ t('channel.buttons.delete') }}</a-button>
            </a-popconfirm>
            <a-button
              size="small"
              @click="manageChannel(record.id, record.status === 1 ? 'disable' : 'enable')"
            >
              {{ record.status === 1 ? t('channel.buttons.disable') : t('channel.buttons.enable') }}
            </a-button>
            <a-button size="small" @click="router.push('/channel/edit/' + record.id)">
              {{ t('channel.buttons.edit') }}
            </a-button>
          </div>
        </template>
      </template>
    </a-table>

    <div class="channels-table-footer">
      <a-button size="small" :loading="loading" @click="router.push('/channel/add')">
        {{ t('channel.buttons.add') }}
      </a-button>
      <a-button size="small" :loading="loading" @click="testChannels('all')">
        {{ t('channel.buttons.test_all') }}
      </a-button>
      <a-button size="small" :loading="loading" @click="testChannels('disabled')">
        {{ t('channel.buttons.test_disabled') }}
      </a-button>
      <a-popconfirm
        :title="t('channel.buttons.confirm_delete_disabled')"
        @confirm="deleteAllDisabledChannels"
      >
        <a-button size="small" :loading="loading">{{ t('channel.buttons.delete_disabled') }}</a-button>
      </a-popconfirm>
      <a-button size="small" :loading="loading" @click="refresh">
        {{ t('channel.buttons.refresh') }}
      </a-button>
      <a-button size="small" @click="toggleShowDetail">
        {{ showDetail ? t('channel.buttons.hide_detail') : t('channel.buttons.show_detail') }}
      </a-button>
      <a-pagination
        class="channels-table-pagination"
        :current="activePage"
        :page-size="ITEMS_PER_PAGE"
        :total="paginationTotal"
        size="small"
        :show-size-changer="false"
        @change="onPaginationChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, h } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { Tag } from 'ant-design-vue';
import {
  API,
  loadChannelModels,
  setPromptShown,
  shouldShowPrompt,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
  renderGroup,
} from '@/helpers';
import { setChannelTypeOptionsCache } from '@/helpers/helper';
import { ITEMS_PER_PAGE } from '@/constants';

const { t } = useI18n();
const router = useRouter();

const promptID = 'detail';

function isShowDetail() {
  return localStorage.getItem('show_detail') === 'true';
}

/** API / 库表使用 MUI 系 color 名，需映射到 Ant Design Tag 预设色 */
function antChannelLabelColor(apiColor) {
  const m = {
    success: 'green',
    primary: 'blue',
    warning: 'orange',
    info: 'cyan',
    error: 'red',
    secondary: 'purple',
  };
  return m[apiColor] || apiColor || 'default';
}

const channelTypeOpts = ref([]);
const channels = ref([]);
const loading = ref(true);
const activePage = ref(1);
const searchKeyword = ref('');
const searching = ref(false);
const showPrompt = ref(shouldShowPrompt(promptID));
const showDetail = ref(isShowDetail());

// Header cells sortable by click (mirrors React sortChannel-on-header behaviour).
function sortableHeader(sortKey) {
  return () => ({ style: { cursor: 'pointer' }, onClick: () => sortChannel(sortKey) });
}

const columns = computed(() => {
  const cols = [
    { title: t('channel.table.id'), dataIndex: 'id', key: 'id', customHeaderCell: sortableHeader('id') },
    { title: t('channel.table.name'), key: 'name', dataIndex: 'name', customHeaderCell: sortableHeader('name') },
    { title: t('channel.table.group'), key: 'group', customHeaderCell: sortableHeader('group') },
    { title: t('channel.table.type'), key: 'type', customHeaderCell: sortableHeader('type') },
    { title: t('channel.table.status'), key: 'status', customHeaderCell: sortableHeader('status') },
    { title: t('channel.table.response_time'), key: 'response_time', customHeaderCell: sortableHeader('response_time') },
    { title: t('channel.table.balance'), key: 'balance', customHeaderCell: sortableHeader('balance') },
  ];
  if (showDetail.value) {
    cols.push({ title: t('channel.table.priority'), key: 'priority', customHeaderCell: sortableHeader('priority') });
    cols.push({ title: t('channel.table.test_model'), key: 'test_model', width: 240 });
  }
  cols.push({ title: t('channel.table.actions'), key: 'actions' });
  return cols;
});

const pagedChannels = computed(() =>
  channels.value
    .filter((c) => !c.deleted)
    .slice((activePage.value - 1) * ITEMS_PER_PAGE, activePage.value * ITEMS_PER_PAGE)
);

const paginationTotal = computed(() => {
  const len = channels.value.filter((c) => !c.deleted).length;
  // mirror the React total-pages logic (allow loading one more page)
  const pages = Math.ceil(len / ITEMS_PER_PAGE) + (len % ITEMS_PER_PAGE === 0 ? 1 : 0);
  return pages * ITEMS_PER_PAGE;
});

function renderType(type) {
  if (type === 0 || type === '0') {
    return h(Tag, {}, () => t('channel.table.status_unknown'));
  }
  const opt = channelTypeOpts.value.find((o) => o.value === type);
  return h(Tag, { color: antChannelLabelColor(opt?.color) }, () => (opt ? opt.text : type));
}

function responseTimeText(responseTime) {
  if (responseTime === 0 || responseTime == null) {
    return t('channel.table.not_tested');
  }
  return (responseTime / 1000).toFixed(2) + 's';
}

function responseTimeColor(responseTime) {
  if (responseTime === 0 || responseTime == null) return 'default';
  if (responseTime <= 1000) return 'green';
  if (responseTime <= 3000) return 'lime';
  if (responseTime <= 5000) return 'gold';
  return 'red';
}

function processChannelData(channel) {
  if (channel.models === '' || channel.models == null) {
    channel.models = [];
    channel.test_model = '';
  } else if (typeof channel.models === 'string') {
    channel.models = channel.models.split(',');
    if (channel.models.length > 0) {
      channel.test_model = channel.models[0];
    }
    channel.model_options = channel.models.map((model) => ({
      key: model,
      text: model,
      value: model,
    }));
    const tm =
      channel.test_model && String(channel.test_model).trim()
        ? String(channel.test_model).trim()
        : '';
    channel.test_model = tm || (channel.models.length > 0 ? channel.models[0] : '');
  }
  return channel;
}

async function loadChannels(startIdx) {
  const res = await API.get(`/api/channel/?p=${startIdx}`);
  const { success, message, data } = res.data;
  if (success) {
    const localChannels = data.map(processChannelData);
    if (startIdx === 0) {
      channels.value = localChannels;
    } else {
      const newChannels = [...channels.value];
      newChannels.splice(startIdx * ITEMS_PER_PAGE, data.length, ...localChannels);
      channels.value = newChannels;
    }
  } else {
    showError(message);
  }
  loading.value = false;
}

function onPaginationChange(page) {
  (async () => {
    if (page === Math.ceil(channels.value.length / ITEMS_PER_PAGE) + 1) {
      await loadChannels(page - 1);
    }
    activePage.value = page;
  })();
}

async function refresh() {
  loading.value = true;
  await loadChannels(activePage.value - 1);
}

function toggleShowDetail() {
  showDetail.value = !showDetail.value;
  localStorage.setItem('show_detail', showDetail.value.toString());
}

function dismissPrompt() {
  showPrompt.value = false;
  setPromptShown(promptID);
}

onMounted(() => {
  loadChannels(0).catch((reason) => showError(reason));
  loadChannelModels();
  (async () => {
    try {
      const res = await API.get('/api/model_catalog/editor_options');
      const opts = res.data?.data?.channel_types;
      if (Array.isArray(opts)) {
        channelTypeOpts.value = opts;
        setChannelTypeOptionsCache(opts);
      }
    } catch (_) {
      /* 未登录或非管理员时任由类型显示为数字 */
    }
  })();
});

async function manageChannel(id, action, value) {
  const data = { id };
  let res;
  switch (action) {
    case 'delete':
      res = await API.delete(`/api/channel/${id}/`);
      break;
    case 'enable':
      data.status = 1;
      res = await API.put('/api/channel/', data);
      break;
    case 'disable':
      data.status = 2;
      res = await API.put('/api/channel/', data);
      break;
    case 'priority':
      if (value === '' || value == null) {
        return;
      }
      data.priority = parseInt(value);
      res = await API.put('/api/channel/', data);
      break;
    case 'weight':
      if (value === '' || value == null) {
        return;
      }
      data.weight = parseInt(value);
      if (data.weight < 0) {
        data.weight = 0;
      }
      res = await API.put('/api/channel/', data);
      break;
    default:
      return;
  }
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('channel.messages.operation_success'));
    const channel = res.data.data;
    const newChannels = [...channels.value];
    const realIdx = newChannels.findIndex((c) => c.id === id);
    if (realIdx !== -1) {
      if (action === 'delete') {
        newChannels[realIdx].deleted = true;
      } else {
        newChannels[realIdx].status = channel.status;
      }
    }
    channels.value = newChannels;
  } else {
    showError(message);
  }
}

async function searchChannels() {
  if (searchKeyword.value === '') {
    await loadChannels(0);
    activePage.value = 1;
    return;
  }
  searching.value = true;
  const res = await API.get(`/api/channel/search?keyword=${searchKeyword.value}`);
  const { success, message, data } = res.data;
  if (success) {
    channels.value = data.map(processChannelData);
    activePage.value = 1;
  } else {
    showError(message);
  }
  searching.value = false;
}

function switchTestModel(id, model) {
  const newChannels = [...channels.value];
  const realIdx = newChannels.findIndex((c) => c.id === id);
  if (realIdx !== -1) {
    newChannels[realIdx].test_model = model;
  }
  channels.value = newChannels;
}

async function testChannel(id, name, m) {
  const res = await API.get(`/api/channel/test/${id}?model=${m}`);
  const { success, message, time, model } = res.data;
  if (success) {
    showSuccess(t('channel.messages.test_success', { name, model, time, message }));
  } else {
    showError(message);
  }
  const newChannels = [...channels.value];
  const realIdx = newChannels.findIndex((c) => c.id === id);
  if (realIdx !== -1) {
    newChannels[realIdx].response_time = time * 1000;
    newChannels[realIdx].test_time = Date.now() / 1000;
  }
  channels.value = newChannels;
}

async function testChannels(scope) {
  const res = await API.get(`/api/channel/test?scope=${scope}`);
  const { success, message } = res.data;
  if (success) {
    showInfo(t('channel.messages.test_all_started'));
  } else {
    showError(message);
  }
}

async function deleteAllDisabledChannels() {
  const res = await API.delete(`/api/channel/disabled`);
  const { success, message, data } = res.data;
  if (success) {
    showSuccess(t('channel.messages.delete_disabled_success', { count: data }));
    await refresh();
  } else {
    showError(message);
  }
}

async function updateChannelBalance(id, name) {
  const res = await API.get(`/api/channel/update_balance/${id}/`);
  const { success, message, balance } = res.data;
  if (success) {
    const newChannels = [...channels.value];
    const realIdx = newChannels.findIndex((c) => c.id === id);
    if (realIdx !== -1) {
      newChannels[realIdx].balance = balance;
      newChannels[realIdx].balance_updated_time = Date.now() / 1000;
    }
    channels.value = newChannels;
    showSuccess(t('channel.messages.balance_update_success', { name }));
  } else {
    showError(message);
  }
}

function sortChannel(key) {
  if (channels.value.length === 0) return;
  loading.value = true;
  const sortedChannels = [...channels.value];
  sortedChannels.sort((a, b) => {
    if (!isNaN(a[key])) {
      return a[key] - b[key];
    }
    return ('' + a[key]).localeCompare(b[key]);
  });
  if (sortedChannels[0].id === channels.value[0].id) {
    sortedChannels.reverse();
  }
  channels.value = sortedChannels;
  loading.value = false;
}
</script>

<style scoped>
.channels-table-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  row-gap: 6px;
}

.channels-table-footer {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.channels-table-pagination {
  margin-left: auto;
}
</style>
