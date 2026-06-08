<template>
  <div v-if="loading" style="padding: 24px; text-align: center">
    <a-spin />
  </div>

  <div v-else-if="!serving" style="padding: 16px; max-width: 896px">
    <h3 style="margin-top: 0">{{ t('nacos.native_console_title') }}</h3>
    <a-alert type="warning" :message="t('nacos.native_console_unconfigured')" show-icon />
    <p style="color: #64748b; line-height: 1.6">{{ t('nacos.native_console_hint') }}</p>
  </div>

  <div v-else style="padding: 16px; max-width: 896px">
    <h3 style="margin-top: 0">{{ t('nacos.native_console_title') }}</h3>
    <a-alert type="success" :message="t('nacos.native_console_tab_body')" show-icon />
    <p>
      <code style="font-size: 12px; word-break: break-all">{{ uiUrl }}</code>
    </p>
    <a-button type="primary" @click="openTab">
      {{ t('nacos.native_console_open_again') }}
    </a-button>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { API, showError } from '@/helpers';

const { t } = useI18n();

const loading = ref(true);
const serving = ref(false);

const sameOriginNacosUiUrl = () => `${window.location.origin}/nacos-ui/`;
const uiUrl = computed(() => sameOriginNacosUiUrl());

const openTab = () => {
  window.open(sameOriginNacosUiUrl(), '_blank', 'noopener,noreferrer');
};

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/nacos/registry/info');
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      serving.value = false;
      return;
    }
    const ok = !!res.data.data?.native_console_serving;
    serving.value = ok;
    if (ok) {
      window.open(sameOriginNacosUiUrl(), '_blank', 'noopener,noreferrer');
    }
  } catch (e) {
    showError(e.message);
    serving.value = false;
  } finally {
    loading.value = false;
  }
};

onMounted(load);
</script>
