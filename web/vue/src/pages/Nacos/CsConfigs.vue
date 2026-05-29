<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.cs_title') }}</h3>
      <a-alert type="info" :message="t('nacos.cs_hint')" show-icon style="margin: 8px 0 12px" />
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button size="small" type="primary" @click="openCreate">{{ t('nacos.cs_new') }}</a-button>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => `${r.dataId}@@${r.groupName || r.group}`"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'dataId'">{{ record.dataId }}</template>
          <template v-else-if="column.key === 'group'">{{ record.groupName || record.group }}</template>
          <template v-else-if="column.key === 'type'">{{ record.type || '—' }}</template>
          <template v-else-if="column.key === 'updated'">{{ fmtTime(record.updateTime) }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openEdit(record.dataId, record.groupName || record.group)">
              {{ t('nacos.cs_edit') }}
            </a-button>
            <a-button
              size="small"
              @click="() => { viewContent = record.content || ''; viewLang = monacoLanguageFromNacosType(record.type); viewOpen = true; }"
            >
              {{ t('nacos.cs_preview') }}
            </a-button>
            <a-button
              size="small"
              @click="loadHistory(record.dataId, record.groupName || record.group, record.type)"
            >
              {{ t('nacos.cs_history') }}
            </a-button>
            <a-button
              size="small"
              danger
              @click="() => { deleteKey = { dataId: record.dataId, group: record.groupName || record.group }; deleteOpen = true; }"
            >
              {{ t('nacos.cs_delete') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Edit / publish -->
    <a-modal
      v-model:open="editOpen"
      :title="t('nacos.cs_edit_title')"
      width="80%"
      :ok-text="t('nacos.cs_publish')"
      :cancel-text="t('nacos.skills_close')"
      @ok="savePublish"
      @cancel="editOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.cs_col_dataid')"
          :value="editDataId"
          :origin-value="editKeysBaseline.dataId"
          :read-only="lockKeys"
          :height="88"
          @update:value="(v) => (editDataId = v)"
        />
        <SettingMonacoField
          :label="t('nacos.cs_col_group')"
          :value="editGroup"
          :origin-value="editKeysBaseline.group"
          :read-only="lockKeys"
          :height="88"
          @update:value="(v) => (editGroup = v)"
        />
        <SettingMonacoField
          :label="t('nacos.cs_col_type')"
          hint="yaml / json / text"
          :value="editType"
          :origin-value="editKeysBaseline.type"
          :height="88"
          @update:value="(v) => (editType = v)"
        />
        <div class="ant-form-item" style="display: block">
          <label style="display: block; margin-bottom: 6px; font-weight: 500">{{ t('nacos.cs_content') }}</label>
          <NacosMonacoEditor
            :language="monacoLanguageFromNacosType(editType)"
            :value="editContent"
            :height="420"
            @update:value="(v) => (editContent = v)"
          />
        </div>
      </a-form>
    </a-modal>

    <!-- History -->
    <a-modal
      v-model:open="histOpen"
      :title="`${t('nacos.cs_history_title')}: ${histKey.dataId} / ${histKey.group}`"
      width="80%"
      :footer="null"
      @cancel="histOpen = false"
    >
      <div v-if="histLoading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="histColumns"
        :data-source="histRows"
        :row-key="(h) => h.id"
        :pagination="false"
        :scroll="{ y: 400 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'id'">{{ record.id }}</template>
          <template v-else-if="column.key === 'action'">{{ record.action }}</template>
          <template v-else-if="column.key === 'operator'">
            {{ record.operatorName || (record.operatorId ? `#${record.operatorId}` : '—') }}
          </template>
          <template v-else-if="column.key === 'time'">{{ fmtTime(record.createdAt) }}</template>
          <template v-else-if="column.key === 'preview'">
            <code style="font-size: 12px">{{ preview(record.content) }}</code>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button
              size="small"
              @click="() => { viewContent = record.content || ''; viewLang = monacoLanguageFromNacosType(histConfigType); viewOpen = true; }"
            >
              {{ t('nacos.cs_preview') }}
            </a-button>
            <a-button size="small" @click="openDiffVsCurrent(record)">{{ t('nacos.cs_compare') }}</a-button>
            <a-button size="small" type="primary" @click="doRollback(record.id)">
              {{ t('nacos.cs_rollback') }}
            </a-button>
          </template>
        </template>
      </a-table>
      <template #footer>
        <a-button @click="histOpen = false">{{ t('nacos.skills_close') }}</a-button>
      </template>
    </a-modal>

    <!-- View -->
    <a-modal
      v-model:open="viewOpen"
      :title="t('nacos.cs_content')"
      width="80%"
      :footer="null"
      @cancel="viewOpen = false"
    >
      <NacosMonacoEditor
        read-only
        :language="viewLang"
        :value="viewContent"
        :height="viewHeight"
      />
      <template #footer>
        <a-button @click="viewOpen = false">{{ t('nacos.skills_close') }}</a-button>
      </template>
    </a-modal>

    <NacosMonacoDiffModal
      :open="diffOpen"
      :original="diffOriginal"
      :modified="diffModified"
      :language="diffLang"
      @close="diffOpen = false"
    />

    <!-- Delete confirm -->
    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.cs_delete_confirm')"
      :ok-button-props="{ danger: true }"
      @ok="doDelete"
      @cancel="() => { deleteOpen = false; deleteKey = { dataId: '', group: '' }; }"
    >
      <p>{{ `${deleteKey.dataId} @ ${deleteKey.group} / ${namespace}` }}</p>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  API,
  getStoredNacosNamespace,
  setStoredNacosNamespace,
  showError,
  showSuccess,
} from '@/helpers';
import NacosNamespaceSelect from '@/components/NacosNamespaceSelect.vue';
import NacosMonacoEditor from '@/components/nacos/NacosMonacoEditor.vue';
import NacosMonacoDiffModal from '@/components/nacos/NacosMonacoDiffModal.vue';
import SettingMonacoField from '@/components/SettingMonacoField.vue';
import {
  monacoLanguageFromNacosType,
  normalizeConfigEOL,
} from '@/components/nacos/monacoLanguage';

const { t } = useI18n();

const fmtTime = (ms) => {
  if (!ms) return '—';
  try {
    return new Date(ms).toLocaleString();
  } catch {
    return String(ms);
  }
};

const preview = (s, n = 96) => {
  if (s == null) return '';
  const str = String(s);
  return str.length <= n ? str : `${str.slice(0, n)}…`;
};

const loading = ref(true);
const rows = ref([]);
const namespace = ref(getStoredNacosNamespace());

const editOpen = ref(false);
const editDataId = ref('');
const editGroup = ref('');
const editType = ref('');
const editContent = ref('');
const lockKeys = ref(false);
const editKeysBaseline = ref({ dataId: '', group: '', type: '' });

const histOpen = ref(false);
const histLoading = ref(false);
const histRows = ref([]);
const histKey = ref({ dataId: '', group: '' });
const histConfigType = ref('');

const viewOpen = ref(false);
const viewContent = ref('');
const viewLang = ref('plaintext');
const viewHeight = ref(Math.min(520, Math.round(window.innerHeight * 0.55)));

const diffOpen = ref(false);
const diffOriginal = ref('');
const diffModified = ref('');
const diffLang = ref('plaintext');

const deleteOpen = ref(false);
const deleteKey = ref({ dataId: '', group: '' });

const columns = computed(() => [
  { title: t('nacos.cs_col_dataid'), key: 'dataId' },
  { title: t('nacos.cs_col_group'), key: 'group' },
  { title: t('nacos.cs_col_type'), key: 'type' },
  { title: t('nacos.cs_col_updated'), key: 'updated' },
  { title: t('nacos.cs_col_actions'), key: 'actions' },
]);

const histColumns = computed(() => [
  { title: 'id', key: 'id' },
  { title: t('nacos.cs_hist_action'), key: 'action' },
  { title: t('nacos.cs_hist_operator'), key: 'operator' },
  { title: t('nacos.cs_hist_time'), key: 'time' },
  { title: t('nacos.cs_hist_preview'), key: 'preview' },
  { title: t('nacos.cs_col_actions'), key: 'actions' },
]);

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/cs/configs', {
      params: { namespace: namespace.value, page: 1, size: 200 },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      return;
    }
    rows.value = res.data.data?.pageItems || [];
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

watch(
  namespace,
  () => {
    setStoredNacosNamespace(namespace.value);
    load();
  },
  { immediate: true }
);

const onNamespaceChange = (v) => {
  namespace.value = v || 'public';
};

const openCreate = () => {
  lockKeys.value = false;
  editDataId.value = '';
  editGroup.value = 'DEFAULT_GROUP';
  editType.value = 'yaml';
  editContent.value = '# example\nkey: value\n';
  editKeysBaseline.value = { dataId: '', group: 'DEFAULT_GROUP', type: 'yaml' };
  editOpen.value = true;
};

const openEdit = async (dataId, group) => {
  histKey.value = { dataId, group };
  try {
    const res = await API.get('/api/nacos/cs/configs/detail', {
      params: { namespace: namespace.value, dataId, groupName: group },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      return;
    }
    const d = res.data.data;
    editDataId.value = d.dataId || '';
    editGroup.value = d.groupName || d.group || '';
    editType.value = d.type || '';
    editContent.value = d.content ?? '';
    lockKeys.value = true;
    editKeysBaseline.value = {
      dataId: d.dataId || '',
      group: d.groupName || d.group || '',
      type: d.type || '',
    };
    editOpen.value = true;
  } catch (e) {
    showError(e.message);
  }
};

const savePublish = async () => {
  const dataId = editDataId.value.trim();
  const group = editGroup.value.trim();
  if (!dataId || !group) {
    showError(t('nacos.cs_keys_required'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/cs/configs/publish', {
      namespace: namespace.value,
      dataId,
      groupName: group,
      content: editContent.value,
      type: editType.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'publish failed');
      return;
    }
    showSuccess(t('nacos.cs_published'));
    editOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const loadHistory = async (dataId, group, configType) => {
  histKey.value = { dataId, group };
  histConfigType.value = configType || '';
  histOpen.value = true;
  histLoading.value = true;
  histRows.value = [];
  try {
    const res = await API.get('/api/nacos/cs/configs/history', {
      params: { namespace: namespace.value, dataId, groupName: group, page: 1, size: 50 },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'history failed');
      histOpen.value = false;
      return;
    }
    histRows.value = res.data.data?.pageItems || [];
  } catch (e) {
    showError(e.message);
    histOpen.value = false;
  } finally {
    histLoading.value = false;
  }
};

const doRollback = async (historyId) => {
  try {
    const res = await API.post('/api/nacos/cs/configs/rollback', {
      namespace: namespace.value,
      dataId: histKey.value.dataId,
      groupName: histKey.value.group,
      historyId,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'rollback failed');
      return;
    }
    showSuccess(t('nacos.cs_rollback_ok'));
    loadHistory(histKey.value.dataId, histKey.value.group, histConfigType.value);
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doDelete = async () => {
  try {
    const res = await API.delete('/api/nacos/cs/configs/item', {
      params: {
        namespace: namespace.value,
        dataId: deleteKey.value.dataId,
        groupName: deleteKey.value.group,
      },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.cs_deleted'));
    deleteOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const openDiffVsCurrent = async (histRow) => {
  try {
    const res = await API.get('/api/nacos/cs/configs/detail', {
      params: {
        namespace: namespace.value,
        dataId: histKey.value.dataId,
        groupName: histKey.value.group,
      },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      return;
    }
    const d = res.data.data;
    const lang = monacoLanguageFromNacosType(
      d.type || histConfigType.value || histRow?.type
    );
    diffLang.value = lang;
    diffOriginal.value = normalizeConfigEOL(histRow?.content ?? '');
    diffModified.value = normalizeConfigEOL(d.content ?? '');
    diffOpen.value = true;
  } catch (e) {
    showError(e.message);
  }
};
</script>
