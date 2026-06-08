<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.mcp_title') }}</h3>
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button size="small" type="primary" @click="openCreate">{{ t('nacos.mcp_new') }}</a-button>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => r.serverName"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">{{ record.serverName }}</template>
          <template v-else-if="column.key === 'desc'">{{ record.description }}</template>
          <template v-else-if="column.key === 'scope'">{{ record.scope || 'PUBLIC' }}</template>
          <template v-else-if="column.key === 'enable'">{{ record.enable ? '✓' : '—' }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openEdit(record.serverName)">{{ t('nacos.mcp_edit') }}</a-button>
            <a-button
              size="small"
              danger
              @click="() => { deleteName = record.serverName; deleteOpen = true; }"
            >
              {{ t('nacos.mcp_delete') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="editOpen"
      :title="t('nacos.mcp_edit_title')"
      width="80%"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
      @ok="save"
      @cancel="editOpen = false"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('nacos.mcp_col_name')">
          <a-input v-model:value="editName" :disabled="nameLocked" allow-clear />
        </a-form-item>
        <a-form-item :label="t('nacos.mcp_col_desc')">
          <a-textarea v-model:value="editDesc" :auto-size="{ minRows: 3, maxRows: 8 }" allow-clear />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item label="bizTags">
              <a-input v-model:value="editBiz" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('nacos.mcp_col_scope')">
              <a-select v-model:value="editScope" style="width: 100%">
                <a-select-option value="PUBLIC">PUBLIC</a-select-option>
                <a-select-option value="PRIVATE">PRIVATE</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item>
          <a-checkbox v-model:checked="editEnable">{{ t('nacos.mcp_col_enable') }}</a-checkbox>
        </a-form-item>
        <SettingMonacoField
          label="spec (JSON)"
          language="json"
          enable-json-format
          minimap
          :value="editSpec"
          :origin-value="editBaseline.spec"
          :height="400"
          @update:value="(v) => (editSpec = v)"
        />
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.mcp_delete_confirm')"
      :ok-button-props="{ danger: true }"
      @ok="doDelete"
      @cancel="() => { deleteOpen = false; deleteName = ''; }"
    >
      <p>{{ `${deleteName} @ ${namespace}` }}</p>
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
import SettingMonacoField from '@/components/SettingMonacoField.vue';

const { t } = useI18n();

const defaultSpec = () =>
  JSON.stringify({ name: '', version: '1', transport: { type: 'stdio' } }, null, 2);

const loading = ref(true);
const rows = ref([]);
const namespace = ref(getStoredNacosNamespace());
const editOpen = ref(false);
const editName = ref('');
const editDesc = ref('');
const editSpec = ref(defaultSpec());
const editBiz = ref('');
const editScope = ref('PUBLIC');
const editEnable = ref(true);
const editBaseline = ref({
  name: '',
  desc: '',
  spec: defaultSpec(),
  biz: '',
  scope: 'PUBLIC',
});
const deleteOpen = ref(false);
const deleteName = ref('');
const nameLocked = ref(false);

const columns = computed(() => [
  { title: t('nacos.mcp_col_name'), key: 'name' },
  { title: t('nacos.mcp_col_desc'), key: 'desc' },
  { title: t('nacos.mcp_col_scope'), key: 'scope' },
  { title: t('nacos.mcp_col_enable'), key: 'enable' },
  { title: t('nacos.mcp_col_actions'), key: 'actions' },
]);

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/mcp', {
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
  nameLocked.value = false;
  const spec = defaultSpec();
  editName.value = '';
  editDesc.value = '';
  editSpec.value = spec;
  editBiz.value = '';
  editScope.value = 'PUBLIC';
  editEnable.value = true;
  editBaseline.value = { name: '', desc: '', spec, biz: '', scope: 'PUBLIC' };
  editOpen.value = true;
};

const openEdit = async (name) => {
  try {
    const res = await API.get('/api/nacos/mcp/detail', {
      params: { namespace: namespace.value, serverName: name },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load detail failed');
      return;
    }
    const d = res.data.data;
    const specStr = JSON.stringify(d.spec || {}, null, 2);
    editName.value = d.serverName;
    editDesc.value = d.description || '';
    editSpec.value = specStr;
    editBiz.value = d.bizTags || '';
    editScope.value = d.scope || 'PUBLIC';
    editEnable.value = !!d.enable;
    editBaseline.value = {
      name: d.serverName,
      desc: d.description || '',
      spec: specStr,
      biz: d.bizTags || '',
      scope: d.scope || 'PUBLIC',
    };
    editOpen.value = true;
    nameLocked.value = true;
  } catch (e) {
    showError(e.message);
  }
};

const save = async () => {
  let spec;
  try {
    spec = JSON.parse(editSpec.value || '{}');
  } catch {
    showError(t('nacos.mcp_invalid_json'));
    return;
  }
  const name = editName.value.trim();
  if (!name) {
    showError(t('nacos.mcp_name_required'));
    return;
  }
  try {
    const res = await API.post(
      `/api/nacos/mcp?namespace=${encodeURIComponent(namespace.value)}`,
      {
        serverName: name,
        description: editDesc.value,
        spec,
        bizTags: editBiz.value,
        scope: editScope.value,
        enable: editEnable.value,
      }
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('nacos.mcp_saved'));
    editOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doDelete = async () => {
  try {
    const res = await API.delete('/api/nacos/mcp', {
      params: { namespace: namespace.value, serverName: deleteName.value },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.mcp_deleted'));
    deleteOpen.value = false;
    deleteName.value = '';
    load();
  } catch (e) {
    showError(e.message);
  }
};
</script>
