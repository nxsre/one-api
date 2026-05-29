<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.prompts_title') }}</h3>
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button size="small" type="primary" @click="openHeaderCreate">{{ t('nacos.prompts_new_header') }}</a-button>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => r.promptKey"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'key'">{{ record.promptKey }}</template>
          <template v-else-if="column.key === 'desc'">{{ record.description }}</template>
          <template v-else-if="column.key === 'editing'">{{ record.editingVersion }}</template>
          <template v-else-if="column.key === 'reviewing'">{{ record.reviewingVersion }}</template>
          <template v-else-if="column.key === 'online'">{{ record.onlineCnt }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openDetail(record.promptKey)">{{ t('nacos.prompts_detail') }}</a-button>
            <a-button size="small" @click="openHeaderEdit(record)">{{ t('nacos.prompts_edit_header') }}</a-button>
            <a-button size="small" @click="openAddVersion(record)">{{ t('nacos.prompts_add_version') }}</a-button>
            <a-button size="small" @click="openSubmit(record)">{{ t('nacos.prompts_submit') }}</a-button>
            <a-button size="small" @click="openPublish(record)">{{ t('nacos.prompts_publish') }}</a-button>
            <a-button size="small" @click="openLabels(record)">{{ t('nacos.prompts_action_labels') }}</a-button>
            <a-button
              size="small"
              danger
              @click="() => { deleteKey = record.promptKey; deleteOpen = true; }"
            >
              {{ t('nacos.prompts_delete') }}
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Header create/edit -->
    <a-modal
      v-model:open="headerOpen"
      :title="t('nacos.prompts_header_title')"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
      @ok="saveHeader"
      @cancel="headerOpen = false"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('nacos.prompts_col_key')">
          <a-input v-model:value="hk" :disabled="hkLocked" allow-clear />
        </a-form-item>
        <a-form-item :label="t('nacos.prompts_col_desc')">
          <a-textarea v-model:value="hDesc" :auto-size="{ minRows: 3, maxRows: 8 }" allow-clear />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item label="bizTags">
              <a-input v-model:value="hBiz" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('nacos.prompts_col_scope')">
              <a-select v-model:value="hScope" style="width: 100%">
                <a-select-option value="PUBLIC">PUBLIC</a-select-option>
                <a-select-option value="PRIVATE">PRIVATE</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item>
          <a-checkbox v-model:checked="hEnable">{{ t('nacos.prompts_col_enable') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Add version -->
    <a-modal
      v-model:open="verOpen"
      :title="`${t('nacos.prompts_add_version')}: ${verKey}`"
      width="80%"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
      @ok="saveVersion"
      @cancel="verOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          label="content (JSON)"
          language="json"
          enable-json-format
          minimap
          :value="verContent"
          :origin-value="verBaseline"
          :height="400"
          @update:value="(v) => (verContent = v)"
        />
      </a-form>
    </a-modal>

    <!-- Detail -->
    <a-modal
      v-model:open="detailOpen"
      :title="t('nacos.prompts_detail_title')"
      width="80%"
      :footer="null"
      @cancel="detailOpen = false"
    >
      <div v-if="detailLoading" style="text-align: center; padding: 24px"><a-spin /></div>
      <div v-else-if="detail">
        <p><strong>{{ detail.promptKey }}</strong> — {{ detail.description }}</p>
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
          </template>
        </a-table>
      </div>
      <template #footer>
        <a-button @click="detailOpen = false">{{ t('nacos.skills_close') }}</a-button>
      </template>
    </a-modal>

    <!-- Labels -->
    <a-modal
      v-model:open="labelsOpen"
      :title="`${t('nacos.prompts_labels_title')}: ${labelsKey}`"
      width="80%"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
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
          <a-checkbox v-model:checked="labelsReplace">{{ t('nacos.prompts_labels_replace') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Submit -->
    <a-modal
      v-model:open="submitOpen"
      :title="`${t('nacos.prompts_submit')}: ${submitKey}`"
      :ok-text="t('nacos.prompts_submit')"
      :cancel-text="t('nacos.skills_close')"
      @ok="doSubmit"
      @cancel="submitOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.skills_version_optional')"
          :hint="t('nacos.skills_submit_hint')"
          :value="submitVer"
          :origin-value="submitVerBaseline"
          :height="88"
          @update:value="(v) => (submitVer = v)"
        />
      </a-form>
    </a-modal>

    <!-- Publish -->
    <a-modal
      v-model:open="publishOpen"
      :title="`${t('nacos.prompts_publish')}: ${pubKey}`"
      :ok-text="t('nacos.prompts_publish')"
      :cancel-text="t('nacos.skills_close')"
      :ok-button-props="{ type: 'primary' }"
      @ok="doPublish"
      @cancel="publishOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.skills_publish_version')"
          :value="pubVer"
          :origin-value="pubVerBaseline"
          :height="88"
          @update:value="(v) => (pubVer = v)"
        />
        <a-form-item>
          <a-checkbox v-model:checked="pubLatest">{{ t('nacos.skills_update_latest') }}</a-checkbox>
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="pubForce">{{ t('nacos.skills_force_publish') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Delete -->
    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.prompts_delete_confirm')"
      :ok-button-props="{ danger: true }"
      @ok="doDelete"
      @cancel="() => { deleteOpen = false; deleteKey = ''; }"
    >
      <p>{{ `${deleteKey} @ ${namespace}` }}</p>
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

const defaultPromptContent = () =>
  JSON.stringify(
    { messages: [{ role: 'system', content: 'You are a helpful assistant.' }] },
    null,
    2
  );

const loading = ref(true);
const rows = ref([]);
const namespace = ref(getStoredNacosNamespace());

const headerOpen = ref(false);
const hk = ref('');
const hDesc = ref('');
const hBiz = ref('');
const hScope = ref('PUBLIC');
const hEnable = ref(true);
const hkLocked = ref(false);
const headerBaseline = ref({ k: '', desc: '', biz: '', scope: 'PUBLIC' });

const verOpen = ref(false);
const verKey = ref('');
const verContent = ref(defaultPromptContent());
const verBaseline = ref(defaultPromptContent());

const detailOpen = ref(false);
const detail = ref(null);
const detailLoading = ref(false);

const submitOpen = ref(false);
const submitKey = ref('');
const submitVer = ref('');
const submitVerBaseline = ref('');

const publishOpen = ref(false);
const pubKey = ref('');
const pubVer = ref('');
const pubVerBaseline = ref('');
const pubLatest = ref(true);
const pubForce = ref(false);

const deleteOpen = ref(false);
const deleteKey = ref('');

const labelsOpen = ref(false);
const labelsKey = ref('');
const labelsText = ref('{}');
const labelsReplace = ref(false);
const labelsBaseline = ref('{}');

const columns = computed(() => [
  { title: t('nacos.prompts_col_key'), key: 'key' },
  { title: t('nacos.prompts_col_desc'), key: 'desc' },
  { title: 'editing', key: 'editing' },
  { title: 'reviewing', key: 'reviewing' },
  { title: 'online', key: 'online' },
  { title: t('nacos.prompts_col_actions'), key: 'actions' },
]);

const versionColumns = computed(() => [
  { title: 'version', key: 'version' },
  { title: 'status', key: 'status' },
]);

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/prompts', {
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

const openHeaderCreate = () => {
  hkLocked.value = false;
  hk.value = '';
  hDesc.value = '';
  hBiz.value = '';
  hScope.value = 'PUBLIC';
  hEnable.value = true;
  headerBaseline.value = { k: '', desc: '', biz: '', scope: 'PUBLIC' };
  headerOpen.value = true;
};

const openHeaderEdit = (r) => {
  hkLocked.value = true;
  hk.value = r.promptKey;
  hDesc.value = r.description || '';
  hBiz.value = r.bizTags || '';
  hScope.value = r.scope || 'PUBLIC';
  hEnable.value = !!r.enable;
  headerBaseline.value = {
    k: r.promptKey,
    desc: r.description || '',
    biz: r.bizTags || '',
    scope: r.scope || 'PUBLIC',
  };
  headerOpen.value = true;
};

const openAddVersion = (r) => {
  verKey.value = r.promptKey;
  const vc = defaultPromptContent();
  verContent.value = vc;
  verBaseline.value = vc;
  verOpen.value = true;
};

const openSubmit = (r) => {
  submitKey.value = r.promptKey;
  submitVer.value = '';
  submitVerBaseline.value = '';
  submitOpen.value = true;
};

const openPublish = (r) => {
  pubKey.value = r.promptKey;
  const pv = r.reviewingVersion || '';
  pubVer.value = pv;
  pubVerBaseline.value = pv;
  pubLatest.value = true;
  pubForce.value = false;
  publishOpen.value = true;
};

const openLabels = (r) => {
  const lt = JSON.stringify(r.labels || {}, null, 2);
  labelsKey.value = r.promptKey;
  labelsText.value = lt;
  labelsBaseline.value = lt;
  labelsReplace.value = false;
  labelsOpen.value = true;
};

const saveHeader = async () => {
  const key = hk.value.trim();
  if (!key) {
    showError(t('nacos.prompts_key_required'));
    return;
  }
  try {
    const res = await API.post(
      `/api/nacos/prompts/header?namespace=${encodeURIComponent(namespace.value)}`,
      {
        promptKey: key,
        description: hDesc.value,
        bizTags: hBiz.value,
        scope: hScope.value,
        enable: hEnable.value,
      }
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('nacos.prompts_header_saved'));
    headerOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const saveVersion = async () => {
  let parsed;
  try {
    parsed = JSON.parse(verContent.value || '{}');
  } catch {
    showError(t('nacos.prompts_invalid_json'));
    return;
  }
  try {
    const res = await API.post(
      `/api/nacos/prompts/version?namespace=${encodeURIComponent(namespace.value)}`,
      { promptKey: verKey.value, content: parsed }
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'add version failed');
      return;
    }
    showSuccess(t('nacos.prompts_version_added'));
    verOpen.value = false;
    load();
    if (detailOpen.value && detail.value?.promptKey === verKey.value) {
      openDetail(verKey.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const openDetail = async (key) => {
  detailOpen.value = true;
  detailLoading.value = true;
  detail.value = null;
  try {
    const res = await API.get('/api/nacos/prompts/detail', {
      params: { namespace: namespace.value, promptKey: key },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'detail failed');
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

const doSubmit = async () => {
  try {
    const res = await API.post('/api/nacos/prompts/submit', {
      namespace: namespace.value,
      promptKey: submitKey.value,
      version: submitVer.value || undefined,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'submit failed');
      return;
    }
    showSuccess(t('nacos.prompts_submit_ok'));
    submitOpen.value = false;
    submitVer.value = '';
    submitVerBaseline.value = '';
    load();
    if (detailOpen.value && detail.value?.promptKey === submitKey.value) {
      openDetail(submitKey.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const doPublish = async () => {
  if (!pubVer.value.trim()) {
    showError(t('nacos.prompts_version_required'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/prompts/publish', {
      namespace: namespace.value,
      promptKey: pubKey.value,
      version: pubVer.value.trim(),
      updateLatest: pubLatest.value,
      forcePublish: pubForce.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'publish failed');
      return;
    }
    showSuccess(t('nacos.prompts_publish_ok'));
    publishOpen.value = false;
    load();
    if (detailOpen.value && detail.value?.promptKey === pubKey.value) {
      openDetail(pubKey.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const doDelete = async () => {
  try {
    const res = await API.delete('/api/nacos/prompts/item', {
      params: { namespace: namespace.value, promptKey: deleteKey.value },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.prompts_deleted'));
    deleteOpen.value = false;
    deleteKey.value = '';
    load();
  } catch (e) {
    showError(e.message);
  }
};

const saveLabels = async () => {
  let labels;
  try {
    labels = JSON.parse(labelsText.value || '{}');
  } catch {
    showError(t('nacos.prompts_labels_invalid'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/prompts/labels', {
      namespace: namespace.value,
      promptKey: labelsKey.value,
      labels,
      replace: labelsReplace.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'save labels failed');
      return;
    }
    showSuccess(t('nacos.prompts_labels_saved'));
    labelsOpen.value = false;
    load();
    if (detailOpen.value && detail.value?.promptKey === labelsKey.value) {
      openDetail(labelsKey.value);
    }
  } catch (e) {
    showError(e.message);
  }
};
</script>
