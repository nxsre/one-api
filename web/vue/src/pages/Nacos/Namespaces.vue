<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <div
        style="display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-bottom: 12px"
      >
        <h3 class="header" style="margin-bottom: 0">{{ t('nacos.ns_native_title') }}</h3>
        <div>
          <a-button :disabled="loading" @click="load">
            <template #icon><ReloadOutlined /></template>
            {{ t('nacos.ns_refresh') }}
          </a-button>
          <a-button
            type="primary"
            style="margin-left: 8px"
            @click="() => { resetForm(); createOpen = true; }"
          >
            <template #icon><PlusOutlined /></template>
            {{ t('nacos.ns_add') }}
          </a-button>
        </div>
      </div>

      <a-alert type="info" :message="t('nacos.ns_manage_hint')" show-icon style="margin-bottom: 12px" />

      <div v-if="loading" style="text-align: center; padding: 24px">
        <a-spin />
      </div>
      <div
        v-else-if="rows.length === 0"
        style="text-align: center; padding: 48px 16px; color: #888"
      >
        <GlobalOutlined style="font-size: 48px; opacity: 0.35; margin-bottom: 12px" />
        <h4 style="color: #888">{{ t('nacos.ns_no_data') }}</h4>
      </div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => r.namespace || 'row'"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <strong>{{ record.namespaceShowName || record.namespace }}</strong>
            <a-tag v-if="isPublicNs(record)" style="margin-left: 8px">
              {{ t('nacos.ns_public_badge') }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'id'">
            <code style="font-size: 12px">{{ record.namespace || '(public)' }}</code>
          </template>
          <template v-else-if="column.key === 'tenantId'">
            <code
              v-if="record.ownerTenantId != null && record.ownerTenantId > 0"
              style="font-size: 12px"
            >{{ record.ownerTenantId }}</code>
            <span v-else>—</span>
          </template>
          <template v-else-if="column.key === 'tenantName'">
            <span :title="record.ownerTenantName || ''">{{ record.ownerTenantName || '—' }}</span>
          </template>
          <template v-else-if="column.key === 'desc'">
            <span
              :title="descCell(record)"
              style="display: inline-block; max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: bottom"
            >{{ descCell(record) }}</span>
          </template>
          <template v-else-if="column.key === 'configCount'">
            <a-tag>{{ record.configCount ?? 0 }}</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openDetail(record)">{{ t('nacos.ns_detail') }}</a-button>
            <a-button size="small" :disabled="isPublicNs(record)" @click="openEdit(record)">
              {{ t('nacos.ns_edit') }}
            </a-button>
            <a-button size="small" danger :disabled="isPublicNs(record)" @click="openDelete(record)">
              {{ t('nacos.skills_action_delete') }}
            </a-button>
          </template>
        </template>
      </a-table>

      <div
        v-if="!loading && rows.length > 0"
        style="margin-top: 12px; font-size: 13px; color: #64748b"
      >
        {{ t('nacos.ns_total', { total: rows.length }) }}
      </div>
    </a-card>

    <!-- Create -->
    <a-modal
      v-model:open="createOpen"
      :title="t('nacos.ns_add')"
      :confirm-loading="saving"
      :ok-text="t('nacos.ns_confirm')"
      :cancel-text="t('nacos.skills_close')"
      @ok="handleCreate"
      @cancel="() => { createOpen = false; resetForm(); }"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.ns_col_name')"
          :hint="t('nacos.ns_placeholder_name')"
          :value="formName"
          :origin-value="formBaseline.name"
          :height="88"
          @update:value="(v) => (formName = v)"
        />
        <SettingMonacoField
          :label="t('nacos.ns_col_id')"
          :hint="t('nacos.ns_placeholder_id')"
          :value="formId"
          :origin-value="formBaseline.id"
          :height="88"
          @update:value="(v) => (formId = v)"
        />
        <SettingMonacoField
          :label="t('nacos.ns_col_desc')"
          :hint="t('nacos.ns_placeholder_desc')"
          :value="formDesc"
          :origin-value="formBaseline.desc"
          :height="120"
          @update:value="(v) => (formDesc = v)"
        />
      </a-form>
    </a-modal>

    <!-- Edit -->
    <a-modal
      v-model:open="editOpen"
      :title="t('nacos.ns_edit_modal_title')"
      :confirm-loading="saving"
      :ok-text="t('nacos.ns_confirm')"
      :cancel-text="t('nacos.skills_close')"
      @ok="handleEdit"
      @cancel="() => { editOpen = false; resetForm(); }"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.ns_col_id')"
          :value="selected?.namespace || ''"
          :origin-value="formBaseline.id"
          read-only
          :enable-diff="false"
          :height="72"
        />
        <SettingMonacoField
          :label="t('nacos.ns_col_name')"
          :hint="t('nacos.ns_placeholder_name')"
          :value="formName"
          :origin-value="formBaseline.name"
          :height="88"
          @update:value="(v) => (formName = v)"
        />
        <SettingMonacoField
          :label="t('nacos.ns_col_desc')"
          :hint="t('nacos.ns_placeholder_desc')"
          :value="formDesc"
          :origin-value="formBaseline.desc"
          :height="120"
          @update:value="(v) => (formDesc = v)"
        />
      </a-form>
    </a-modal>

    <!-- Delete -->
    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.skills_action_delete')"
      :ok-text="t('nacos.ns_confirm')"
      :cancel-text="t('nacos.skills_close')"
      :ok-button-props="{ danger: true }"
      @ok="handleDelete"
      @cancel="deleteOpen = false"
    >
      <p>{{ t('nacos.ns_delete_confirm_named', { name: selected?.namespaceShowName || selected?.namespace }) }}</p>
    </a-modal>

    <!-- Detail -->
    <a-modal
      v-model:open="detailOpen"
      :title="t('nacos.ns_detail')"
      :footer="null"
      @cancel="detailOpen = false"
    >
      <div v-if="detailLoading" style="text-align: center; padding: 16px">
        <a-spin />
      </div>
      <a-descriptions v-else-if="detailData" :column="1" bordered size="small">
        <a-descriptions-item :label="t('nacos.ns_col_name')">
          {{ detailData.namespaceShowName }}
        </a-descriptions-item>
        <a-descriptions-item :label="t('nacos.ns_col_id')">
          <code>{{ detailData.namespace || '(public)' }}</code>
        </a-descriptions-item>
        <a-descriptions-item :label="t('nacos.ns_col_tenant_id')">
          {{ detailData.ownerTenantId != null && detailData.ownerTenantId > 0 ? detailData.ownerTenantId : '—' }}
        </a-descriptions-item>
        <a-descriptions-item :label="t('nacos.ns_col_tenant_name')">
          {{ detailData.ownerTenantName || '—' }}
        </a-descriptions-item>
        <a-descriptions-item :label="t('nacos.ns_config_quota_label')">
          {{ t('nacos.ns_config_quota', { used: detailData.configCount ?? 0, quota: detailData.quota ?? 128 }) }}
        </a-descriptions-item>
        <a-descriptions-item :label="t('nacos.ns_col_desc')">
          {{ detailData.namespaceDesc || '—' }}
        </a-descriptions-item>
      </a-descriptions>
      <template #footer>
        <a-button @click="detailOpen = false">{{ t('nacos.skills_close') }}</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { ReloadOutlined, PlusOutlined, GlobalOutlined } from '@ant-design/icons-vue';
import { API, showError, showSuccess } from '@/helpers';
import SettingMonacoField from '@/components/SettingMonacoField.vue';

const { t } = useI18n();

/** 与 Nacos console-ui-next Namespace 及 GET /api/nacos/namespaces 对齐 */
const isPublicNs = (row) => row && row.type === 0;

const loading = ref(true);
const rows = ref([]);
const createOpen = ref(false);
const editOpen = ref(false);
const deleteOpen = ref(false);
const detailOpen = ref(false);
const detailLoading = ref(false);
const detailData = ref(null);
const selected = ref(null);
const saving = ref(false);

const formId = ref('');
const formName = ref('');
const formDesc = ref('');
const formBaseline = ref({ id: '', name: '', desc: '' });

const columns = computed(() => [
  { title: t('nacos.ns_col_name'), key: 'name' },
  { title: t('nacos.ns_col_id'), key: 'id' },
  { title: t('nacos.ns_col_tenant_id'), key: 'tenantId' },
  { title: t('nacos.ns_col_tenant_name'), key: 'tenantName' },
  { title: t('nacos.ns_col_desc'), key: 'desc' },
  { title: t('nacos.ns_col_config_count'), key: 'configCount', align: 'center' },
  { title: t('nacos.ns_col_actions'), key: 'actions', align: 'right' },
]);

const resetForm = () => {
  formId.value = '';
  formName.value = '';
  formDesc.value = '';
  formBaseline.value = { id: '', name: '', desc: '' };
  selected.value = null;
};

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/namespaces');
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      return;
    }
    rows.value = Array.isArray(res.data.data) ? res.data.data : [];
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

onMounted(load);

const handleCreate = async () => {
  if (!formName.value.trim()) {
    showError(t('nacos.ns_name_required'));
    return;
  }
  saving.value = true;
  try {
    const res = await API.post('/api/nacos/namespaces', {
      customNamespaceId: formId.value.trim(),
      namespaceName: formName.value.trim(),
      namespaceDesc: formDesc.value.trim(),
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'create failed');
      return;
    }
    showSuccess(t('nacos.ns_created'));
    createOpen.value = false;
    resetForm();
    load();
  } catch (e) {
    showError(e.message);
  } finally {
    saving.value = false;
  }
};

const openEdit = (ns) => {
  selected.value = ns;
  const name = ns.namespaceShowName || ns.namespace || '';
  const desc = ns.namespaceDesc || '';
  formName.value = name;
  formDesc.value = desc;
  formBaseline.value = { id: ns.namespace || '', name, desc };
  editOpen.value = true;
};

const handleEdit = async () => {
  if (!selected.value || !formName.value.trim()) {
    showError(t('nacos.ns_name_required'));
    return;
  }
  saving.value = true;
  try {
    const res = await API.put('/api/nacos/namespaces', {
      namespace: selected.value.namespace,
      namespaceShowName: formName.value.trim(),
      namespaceDesc: formDesc.value.trim(),
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'update failed');
      return;
    }
    showSuccess(t('nacos.ns_updated'));
    editOpen.value = false;
    resetForm();
    load();
  } catch (e) {
    showError(e.message);
  } finally {
    saving.value = false;
  }
};

const openDelete = (ns) => {
  selected.value = ns;
  deleteOpen.value = true;
};

const handleDelete = async () => {
  if (!selected.value) return;
  try {
    const res = await API.delete(
      `/api/nacos/namespaces/${encodeURIComponent(selected.value.namespace)}`
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.ns_deleted'));
    deleteOpen.value = false;
    selected.value = null;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const openDetail = async (ns) => {
  selected.value = ns;
  detailOpen.value = true;
  detailData.value = null;
  detailLoading.value = true;
  try {
    const res = await API.get('/api/nacos/namespaces/detail', {
      params: { namespaceId: ns.namespace },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'detail failed');
      detailData.value = ns;
      return;
    }
    detailData.value = res.data.data || ns;
  } catch (e) {
    showError(e.message);
    detailData.value = ns;
  } finally {
    detailLoading.value = false;
  }
};

const descCell = (ns) => {
  if (isPublicNs(ns)) {
    return ns.namespaceDesc || t('nacos.ns_public_desc');
  }
  return ns.namespaceDesc || '—';
};
</script>
