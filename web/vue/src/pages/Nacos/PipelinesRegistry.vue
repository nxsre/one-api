<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.pipelines_title') }}</h3>
      <a-alert type="info" :message="t('nacos.pipelines_hint')" show-icon style="margin: 8px 0 12px" />
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button size="small" type="primary" @click="runScan">{{ t('nacos.pipelines_run_scan') }}</a-button>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <a-table
        v-else
        size="small"
        :columns="columns"
        :data-source="rows"
        :row-key="(r) => r.id"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'id'">{{ record.id }}</template>
          <template v-else-if="column.key === 'type'">{{ record.jobType }}</template>
          <template v-else-if="column.key === 'status'">{{ record.status }}</template>
          <template v-else-if="column.key === 'message'">{{ record.message || '—' }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" @click="openDetail(record.id)">{{ t('nacos.pipelines_detail') }}</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="detailOpen"
      :title="t('nacos.pipelines_detail_title')"
      width="80%"
      :footer="null"
      @cancel="detailOpen = false"
    >
      <div v-if="detail" class="ant-form-item" style="display: block">
        <label style="display: block; margin-bottom: 6px; font-weight: 500">{{ t('nacos.pipelines_json_view') }}</label>
        <NacosMonacoEditor
          read-only
          language="json"
          :value="detailJson"
          :height="detailHeight"
        />
      </div>
      <template #footer>
        <a-button @click="detailOpen = false">{{ t('nacos.skills_close') }}</a-button>
      </template>
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

const { t } = useI18n();

const loading = ref(true);
const rows = ref([]);
const namespace = ref(getStoredNacosNamespace());
const detailOpen = ref(false);
const detail = ref(null);
const detailHeight = ref(Math.min(560, Math.round(window.innerHeight * 0.5)));

const detailJson = computed(() =>
  detail.value ? JSON.stringify(detail.value, null, 2) : ''
);

const columns = computed(() => [
  { title: 'id', key: 'id' },
  { title: t('nacos.pipelines_col_type'), key: 'type' },
  { title: t('nacos.pipelines_col_status'), key: 'status' },
  { title: 'message', key: 'message' },
  { title: t('nacos.pipelines_col_actions'), key: 'actions' },
]);

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/pipelines', {
      params: { namespace: namespace.value, page: 1, size: 100 },
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

const runScan = async () => {
  try {
    const res = await API.post(
      `/api/nacos/pipelines/run-scan?namespace=${encodeURIComponent(namespace.value)}`
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'run failed');
      return;
    }
    showSuccess(t('nacos.pipelines_scan_ok'));
    load();
  } catch (e) {
    showError(e.message);
  }
};

const openDetail = async (id) => {
  try {
    const res = await API.get('/api/nacos/pipelines/detail', {
      params: { namespace: namespace.value, id },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load detail failed');
      return;
    }
    detail.value = res.data.data;
    detailOpen.value = true;
  } catch (e) {
    showError(e.message);
  }
};
</script>
