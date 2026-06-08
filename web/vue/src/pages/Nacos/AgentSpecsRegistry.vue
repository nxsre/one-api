<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.agentspecs_title') }}</h3>
      <a-alert
        v-if="info"
        type="info"
        show-icon
        style="margin: 8px 0 12px"
        :message="infoText"
      />
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button
          size="small"
          type="primary"
          @click="() => { uploadFile = null; uploadOverwrite = false; uploadOpen = true; }"
        >
          {{ t('nacos.agentspecs_upload') }}
        </a-button>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => r.name"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">{{ record.name }}</template>
          <template v-else-if="column.key === 'desc'">{{ record.description == null ? '' : String(record.description) }}</template>
          <template v-else-if="column.key === 'enable'">{{ record.enable ? '✓' : '—' }}</template>
          <template v-else-if="column.key === 'scope'">{{ record.scope || 'PUBLIC' }}</template>
          <template v-else-if="column.key === 'editing'">{{ record.editingVersion }}</template>
          <template v-else-if="column.key === 'reviewing'">{{ record.reviewingVersion }}</template>
          <template v-else-if="column.key === 'online'">{{ record.onlineCnt != null ? record.onlineCnt : '-' }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openDetail(record.name)">{{ t('nacos.agentspecs_action_detail') }}</a-button>
            <a-button size="small" @click="openEdit(record)">{{ t('nacos.agentspecs_action_edit') }}</a-button>
            <a-button
              size="small"
              @click="() => { submitName = record.name; submitVersion = ''; submitVersionBaseline = ''; submitOpen = true; }"
            >
              {{ t('nacos.agentspecs_action_submit') }}
            </a-button>
            <a-button size="small" @click="openPublish(record.name, record.reviewingVersion)">
              {{ t('nacos.agentspecs_action_publish') }}
            </a-button>
            <a-button size="small" @click="openLabels(record)">{{ t('nacos.agentspecs_action_labels') }}</a-button>
            <a-button
              size="small"
              danger
              @click="() => { deleteName = record.name; deleteOpen = true; }"
            >
              {{ t('nacos.agentspecs_action_delete') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Detail -->
    <a-modal
      v-model:open="detailOpen"
      :title="t('nacos.agentspecs_detail_title')"
      width="80%"
      :footer="null"
      @cancel="detailOpen = false"
    >
      <div v-if="detailLoading" style="text-align: center; padding: 24px"><a-spin /></div>
      <div v-else-if="detail">
        <p><strong>{{ detail.name }}</strong> — {{ detail.description }}</p>
        <p>
          enable: {{ detail.enable ? 'yes' : 'no' }} | scope:
          {{ detail.scope || 'PUBLIC' }} | bizTags: {{ detail.bizTags || '—' }}
        </p>
        <p v-if="detail.labels && Object.keys(detail.labels).length > 0">
          labels: {{ JSON.stringify(detail.labels) }}
        </p>
        <a-table
          size="small"
          :columns="versionColumns"
          :data-source="detail.versions || []"
          :row-key="(v) => v.version"
          :pagination="false"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'version'">{{ record.version }}</template>
            <template v-else-if="column.key === 'status'">{{ record.status }}</template>
            <template v-else-if="column.key === 'commit'">{{ record.commitMsg || '—' }}</template>
            <template v-else-if="column.key === 'actions'">
              <a-button size="small" @click="downloadVersionZip(detail.name, record.version)">
                {{ t('nacos.agentspecs_download') }}
              </a-button>
            </template>
          </template>
        </a-table>
      </div>
      <template #footer>
        <a-button @click="detailOpen = false">{{ t('nacos.agentspecs_close') }}</a-button>
      </template>
    </a-modal>

    <!-- Edit -->
    <a-modal
      v-model:open="editOpen"
      :title="`${t('nacos.agentspecs_edit_title')}: ${editName}`"
      :ok-text="t('nacos.agentspecs_save')"
      :cancel-text="t('nacos.agentspecs_close')"
      @ok="saveMetadata"
      @cancel="editOpen = false"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('nacos.agentspecs_col_desc')">
          <a-textarea v-model:value="editDesc" :auto-size="{ minRows: 3, maxRows: 8 }" allow-clear />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item label="bizTags">
              <a-input v-model:value="editBiz" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('nacos.agentspecs_col_scope')">
              <a-select v-model:value="editScope" style="width: 100%">
                <a-select-option value="PUBLIC">PUBLIC</a-select-option>
                <a-select-option value="PRIVATE">PRIVATE</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item>
          <a-checkbox v-model:checked="editEnable">{{ t('nacos.agentspecs_col_enable') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Upload -->
    <a-modal
      v-model:open="uploadOpen"
      :title="t('nacos.agentspecs_upload_title')"
      :ok-text="t('nacos.agentspecs_upload')"
      :cancel-text="t('nacos.agentspecs_close')"
      @ok="doUpload"
      @cancel="uploadOpen = false"
    >
      <p>{{ t('nacos.agentspecs_upload_hint') }}</p>
      <input type="file" accept=".zip,application/zip" @change="onFileChange" />
      <a-form-item style="margin-top: 12px">
        <a-checkbox v-model:checked="uploadOverwrite">{{ t('nacos.agentspecs_upload_overwrite') }}</a-checkbox>
      </a-form-item>
    </a-modal>

    <!-- Submit -->
    <a-modal
      v-model:open="submitOpen"
      :title="`${t('nacos.agentspecs_submit_title')}: ${submitName}`"
      :ok-text="t('nacos.agentspecs_action_submit')"
      :cancel-text="t('nacos.agentspecs_close')"
      @ok="doSubmit"
      @cancel="submitOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.agentspecs_version_optional')"
          :hint="t('nacos.agentspecs_submit_hint')"
          :value="submitVersion"
          :origin-value="submitVersionBaseline"
          :height="88"
          @update:value="(v) => (submitVersion = v)"
        />
      </a-form>
    </a-modal>

    <!-- Publish -->
    <a-modal
      v-model:open="publishOpen"
      :title="`${t('nacos.agentspecs_publish_title')}: ${publishName}`"
      :ok-text="t('nacos.agentspecs_action_publish')"
      :cancel-text="t('nacos.agentspecs_close')"
      :ok-button-props="{ type: 'primary' }"
      @ok="doPublish"
      @cancel="publishOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.agentspecs_publish_version')"
          :hint="`${t('nacos.agentspecs_publish_hint')} · ${publishCandidates.length ? publishCandidates.join(', ') : '—'}`"
          :value="publishVersion"
          :origin-value="publishVersionBaseline"
          :height="96"
          @update:value="(v) => (publishVersion = v)"
        />
        <a-form-item>
          <a-checkbox v-model:checked="publishUpdateLatest">{{ t('nacos.agentspecs_update_latest') }}</a-checkbox>
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="publishForce">{{ t('nacos.agentspecs_force_publish') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Labels -->
    <a-modal
      v-model:open="labelsOpen"
      :title="`${t('nacos.agentspecs_labels_title')}: ${labelsName}`"
      width="80%"
      :ok-text="t('nacos.agentspecs_save')"
      :cancel-text="t('nacos.agentspecs_close')"
      @ok="saveLabels"
      @cancel="labelsOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          label="labels (JSON object)"
          language="json"
          enable-json-format
          minimap
          :value="labelsText"
          :origin-value="labelsBaseline"
          :height="360"
          @update:value="(v) => (labelsText = v)"
        />
        <a-form-item>
          <a-checkbox v-model:checked="labelsReplace">{{ t('nacos.agentspecs_labels_replace') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Delete -->
    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.agentspecs_delete_confirm')"
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

const fmtBytes = (n) => {
  if (n == null || !Number.isFinite(Number(n))) return '';
  const v = Number(n);
  if (v < 1024) return `${v} B`;
  if (v < 1024 * 1024) return `${(v / 1024).toFixed(1)} KiB`;
  return `${(v / (1024 * 1024)).toFixed(1)} MiB`;
};

const loading = ref(true);
const info = ref(null);
const rows = ref([]);
const namespace = ref(getStoredNacosNamespace());

const detailOpen = ref(false);
const detailLoading = ref(false);
const detail = ref(null);

const editOpen = ref(false);
const editName = ref('');
const editDesc = ref('');
const editBiz = ref('');
const editScope = ref('PUBLIC');
const editEnable = ref(true);
const editBaseline = ref({ desc: '', biz: '', scope: 'PUBLIC' });

const uploadOpen = ref(false);
const uploadFile = ref(null);
const uploadOverwrite = ref(false);

const deleteOpen = ref(false);
const deleteName = ref('');

const submitOpen = ref(false);
const submitName = ref('');
const submitVersion = ref('');
const submitVersionBaseline = ref('');

const publishOpen = ref(false);
const publishName = ref('');
const publishVersion = ref('');
const publishVersionBaseline = ref('');
const publishUpdateLatest = ref(true);
const publishForce = ref(false);
const publishCandidates = ref([]);

const labelsOpen = ref(false);
const labelsName = ref('');
const labelsText = ref('{}');
const labelsReplace = ref(false);
const labelsBaseline = ref('{}');

const columns = computed(() => [
  { title: t('nacos.agentspecs_col_name'), key: 'name' },
  { title: t('nacos.agentspecs_col_desc'), key: 'desc' },
  { title: t('nacos.agentspecs_col_enable'), key: 'enable' },
  { title: t('nacos.agentspecs_col_scope'), key: 'scope' },
  { title: 'editing', key: 'editing' },
  { title: 'reviewing', key: 'reviewing' },
  { title: 'online', key: 'online' },
  { title: t('nacos.agentspecs_col_actions'), key: 'actions' },
]);

const versionColumns = computed(() => [
  { title: 'version', key: 'version' },
  { title: 'status', key: 'status' },
  { title: 'commit', key: 'commit' },
  { title: t('nacos.agentspecs_col_actions'), key: 'actions' },
]);

const infoText = computed(() => {
  const i = info.value;
  if (!i) return '';
  let s = `${t('nacos.registry_storage')}: ${i.zip_storage}`;
  if (i.zip_local_dir) s += ` | 本地: ${i.zip_local_dir}`;
  if (i.s3_remote_configured) s += ' | S3 已配置';
  if (i.max_upload_bytes)
    s += ` | ${t('nacos.agentspecs_max_upload')}: ${fmtBytes(i.max_upload_bytes)}`;
  return s;
});

const openPublish = async (name, reviewingFromList) => {
  publishName.value = name;
  publishUpdateLatest.value = true;
  publishForce.value = false;
  let candidates = [];
  if (reviewingFromList) {
    candidates = [reviewingFromList];
  }
  try {
    const res = await API.get('/api/nacos/agentspecs/detail', {
      params: { namespace: namespace.value, name },
    });
    if (res.data?.success) {
      const rev = (res.data.data?.versions || [])
        .filter((v) => v.status === 'reviewing')
        .map((v) => v.version);
      if (rev.length) {
        candidates = rev;
      }
    }
  } catch {
    /* ignore */
  }
  publishCandidates.value = candidates;
  const initial = candidates[0] || '';
  publishVersion.value = initial;
  publishVersionBaseline.value = initial;
  publishOpen.value = true;
};

const load = async () => {
  loading.value = true;
  try {
    const ir = await API.get('/api/nacos/registry/info');
    if (!ir.data?.success) {
      showError(ir.data?.message || 'load info failed');
      return;
    }
    info.value = ir.data.data;
    const sr = await API.get('/api/nacos/agentspecs', {
      params: { namespace: namespace.value, page: 1, size: 100 },
    });
    if (!sr.data?.success) {
      showError(sr.data?.message || 'load agentspecs failed');
      return;
    }
    rows.value = sr.data.data?.pageItems || [];
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

const openDetail = async (name) => {
  detailOpen.value = true;
  detailLoading.value = true;
  detail.value = null;
  try {
    const res = await API.get('/api/nacos/agentspecs/detail', {
      params: { namespace: namespace.value, name },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load detail failed');
      detailOpen.value = false;
      return;
    }
    detail.value = res.data.data;
  } catch (e) {
    showError(e.message);
    detailOpen.value = false;
  } finally {
    detailLoading.value = false;
  }
};

const openEdit = (r) => {
  editName.value = r.name;
  const desc = r.description == null ? '' : String(r.description);
  const biz = r.bizTags == null ? '' : String(r.bizTags);
  const scope = r.scope || 'PUBLIC';
  editDesc.value = desc;
  editBiz.value = biz;
  editScope.value = scope;
  editEnable.value = !!r.enable;
  editBaseline.value = { desc, biz, scope };
  editOpen.value = true;
};

const openLabels = (r) => {
  const lt = JSON.stringify(r.labels || {}, null, 2);
  labelsName.value = r.name;
  labelsText.value = lt;
  labelsBaseline.value = lt;
  labelsReplace.value = false;
  labelsOpen.value = true;
};

const onFileChange = (e) => {
  uploadFile.value = e.target.files && e.target.files[0];
};

const saveMetadata = async () => {
  try {
    const res = await API.put('/api/nacos/agentspecs/metadata', {
      namespace: namespace.value,
      name: editName.value,
      description: editDesc.value,
      bizTags: editBiz.value,
      scope: editScope.value,
      enable: editEnable.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('nacos.agentspecs_saved'));
    editOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doUpload = async () => {
  if (!uploadFile.value) {
    showError(t('nacos.agentspecs_pick_file'));
    return;
  }
  const fd = new FormData();
  fd.append('file', uploadFile.value);
  try {
    const res = await API.post(
      `/api/nacos/agentspecs/upload?namespace=${encodeURIComponent(
        namespace.value
      )}&overwrite=${uploadOverwrite.value ? 'true' : 'false'}`,
      fd
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'upload failed');
      return;
    }
    showSuccess(t('nacos.agentspecs_upload_ok'));
    uploadOpen.value = false;
    uploadFile.value = null;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doDelete = async () => {
  try {
    const res = await API.delete('/api/nacos/agentspecs/item', {
      params: { namespace: namespace.value, name: deleteName.value },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.agentspecs_deleted'));
    deleteOpen.value = false;
    deleteName.value = '';
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doSubmit = async () => {
  try {
    const res = await API.post('/api/nacos/agentspecs/submit', {
      namespace: namespace.value,
      name: submitName.value,
      version: submitVersion.value || undefined,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'submit failed');
      return;
    }
    showSuccess(t('nacos.agentspecs_submit_ok'));
    submitOpen.value = false;
    submitVersion.value = '';
    submitVersionBaseline.value = '';
    load();
    if (detailOpen.value && detail.value?.name === submitName.value) {
      openDetail(submitName.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const doPublish = async () => {
  if (!publishVersion.value.trim()) {
    showError(t('nacos.agentspecs_version_required'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/agentspecs/publish', {
      namespace: namespace.value,
      name: publishName.value,
      version: publishVersion.value.trim(),
      updateLatest: publishUpdateLatest.value,
      forcePublish: publishForce.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'publish failed');
      return;
    }
    showSuccess(t('nacos.agentspecs_publish_ok'));
    publishOpen.value = false;
    publishVersion.value = '';
    load();
    if (detailOpen.value && detail.value?.name === publishName.value) {
      openDetail(publishName.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const downloadVersionZip = async (name, ver) => {
  try {
    const res = await API.get('/api/nacos/agentspecs/download', {
      params: { namespace: namespace.value, name, version: ver },
      responseType: 'blob',
    });
    const blob = new Blob([res.data], { type: 'application/zip' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${name}-${ver}.zip`;
    a.click();
    window.URL.revokeObjectURL(url);
  } catch (e) {
    showError(e.message);
  }
};

const saveLabels = async () => {
  let labels;
  try {
    labels = JSON.parse(labelsText.value || '{}');
  } catch {
    showError(t('nacos.agentspecs_labels_invalid'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/agentspecs/labels', {
      namespace: namespace.value,
      name: labelsName.value,
      labels,
      replace: labelsReplace.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'save labels failed');
      return;
    }
    showSuccess(t('nacos.agentspecs_labels_saved'));
    labelsOpen.value = false;
    load();
    if (detailOpen.value && detail.value?.name === labelsName.value) {
      openDetail(labelsName.value);
    }
  } catch (e) {
    showError(e.message);
  }
};
</script>
