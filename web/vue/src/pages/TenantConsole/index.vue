<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">
        {{ t('tenant_console.home.title') }}
      </h2>
      <a-alert
        v-if="tenantMeta"
        type="info"
        show-icon
        class="mb-4"
      >
        <template #message>租户 ID（子账号登录「租户登录」页时请填写此项）</template>
        <template #description>
          <strong>{{ tenantMeta.tenant_id }}</strong>
          <template v-if="tenantMeta.name"> — {{ tenantMeta.name }}</template>
          <template v-if="tenantMeta.slug"> ({{ tenantMeta.slug }})</template>
        </template>
      </a-alert>
      <h4 class="text-base font-medium border-b pb-2 mb-4">
        {{ t('tenant_console.home.subtitle') }}
      </h4>
      <a-space>
        <router-link to="/tenant-console/users">
          <a-button type="primary">{{ t('header.tenant_subusers') }}</a-button>
        </router-link>
        <router-link to="/tenant-console/channels">
          <a-button>{{ t('header.tenant_channels') }}</a-button>
        </router-link>
      </a-space>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { API, showError } from '@/helpers';

const { t } = useI18n();
const tenantMeta = ref(null);

onMounted(async () => {
  try {
    const res = await API.get('/api/tenant_console/meta/tenant');
    const { success, data, message } = res.data || {};
    if (!success || !data) {
      if (message) showError(message);
      return;
    }
    tenantMeta.value = data;
  } catch (e) {
    showError(e.message);
  }
});
</script>
