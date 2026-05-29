<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.a2a_title') }}</h3>
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button size="small" type="primary" @click="openCreate">{{ t('nacos.a2a_new') }}</a-button>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => r.agentName"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">{{ record.agentName }}</template>
          <template v-else-if="column.key === 'desc'">{{ record.description }}</template>
          <template v-else-if="column.key === 'scope'">{{ record.scope || 'PUBLIC' }}</template>
          <template v-else-if="column.key === 'enable'">{{ record.enable ? '✓' : '—' }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openEdit(record.agentName)">{{ t('nacos.a2a_edit') }}</a-button>
            <a-button
              size="small"
              danger
              @click="() => { deleteName = record.agentName; deleteOpen = true; }"
            >
              {{ t('nacos.a2a_delete') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="editOpen"
      :title="t('nacos.a2a_edit_title')"
      width="80%"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
      @ok="save"
      @cancel="editOpen = false"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('nacos.a2a_col_name')">
          <a-input v-model:value="editName" :disabled="nameLocked" allow-clear />
        </a-form-item>
        <a-form-item :label="t('nacos.a2a_col_desc')">
          <a-textarea v-model:value="editDesc" :auto-size="{ minRows: 3, maxRows: 8 }" allow-clear />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item label="bizTags">
              <a-input v-model:value="editBiz" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('nacos.a2a_col_scope')">
              <a-select v-model:value="editScope" style="width: 100%">
                <a-select-option value="PUBLIC">PUBLIC</a-select-option>
                <a-select-option value="PRIVATE">PRIVATE</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item>
          <a-checkbox v-model:checked="editEnable">{{ t('nacos.a2a_col_enable') }}</a-checkbox>
        </a-form-item>
        <SettingMonacoField
          label="card (JSON)"
          language="json"
          enable-json-format
          minimap
          :value="editCard"
          :origin-value="editBaseline.card"
          :height="400"
          @update:value="(v) => (editCard = v)"
        />
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.a2a_delete_confirm')"
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

const defaultCard = () =>
  JSON.stringify({ name: 'example-agent', version: '1.0.0' }, null, 2);

const loading = ref(true);
const rows = ref([]);
const namespace = ref(getStoredNacosNamespace());
const editOpen = ref(false);
const editName = ref('');
const editDesc = ref('');
const editCard = ref(defaultCard());
const editBiz = ref('');
const editScope = ref('PUBLIC');
const editEnable = ref(true);
const editBaseline = ref({
  name: '',
  desc: '',
  card: defaultCard(),
  biz: '',
  scope: 'PUBLIC',
});
const deleteOpen = ref(false);
const deleteName = ref('');
const nameLocked = ref(false);

const columns = computed(() => [
  { title: t('nacos.a2a_col_name'), key: 'name' },
  { title: t('nacos.a2a_col_desc'), key: 'desc' },
  { title: t('nacos.a2a_col_scope'), key: 'scope' },
  { title: t('nacos.a2a_col_enable'), key: 'enable' },
  { title: t('nacos.a2a_col_actions'), key: 'actions' },
]);

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/a2a', {
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
  const card = defaultCard();
  editName.value = '';
  editDesc.value = '';
  editCard.value = card;
  editBiz.value = '';
  editScope.value = 'PUBLIC';
  editEnable.value = true;
  editBaseline.value = { name: '', desc: '', card, biz: '', scope: 'PUBLIC' };
  editOpen.value = true;
};

const openEdit = async (name) => {
  try {
    const res = await API.get('/api/nacos/a2a/detail', {
      params: { namespace: namespace.value, agentName: name },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load detail failed');
      return;
    }
    const d = res.data.data;
    const cardStr = JSON.stringify(d.card || {}, null, 2);
    editName.value = d.agentName;
    editDesc.value = d.description || '';
    editCard.value = cardStr;
    editBiz.value = d.bizTags || '';
    editScope.value = d.scope || 'PUBLIC';
    editEnable.value = !!d.enable;
    editBaseline.value = {
      name: d.agentName,
      desc: d.description || '',
      card: cardStr,
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
  let card;
  try {
    card = JSON.parse(editCard.value || '{}');
  } catch {
    showError(t('nacos.a2a_invalid_json'));
    return;
  }
  const name = editName.value.trim();
  if (!name) {
    showError(t('nacos.a2a_name_required'));
    return;
  }
  try {
    const res = await API.post(
      `/api/nacos/a2a?namespace=${encodeURIComponent(namespace.value)}`,
      {
        agentName: name,
        description: editDesc.value,
        card,
        bizTags: editBiz.value,
        scope: editScope.value,
        enable: editEnable.value,
      }
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('nacos.a2a_saved'));
    editOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doDelete = async () => {
  try {
    const res = await API.delete('/api/nacos/a2a', {
      params: { namespace: namespace.value, agentName: deleteName.value },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.a2a_deleted'));
    deleteOpen.value = false;
    deleteName.value = '';
    load();
  } catch (e) {
    showError(e.message);
  }
};
</script>
