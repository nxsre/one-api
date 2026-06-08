<template>
  <a-alert v-if="visible" type="info" show-icon style="margin-bottom: 1rem">
    <template #message>
      <div class="flex flex-wrap items-center gap-3">
        <span>{{ t('tenant_console.impersonate.banner') }}</span>
        <a-select
          :value="value || undefined"
          show-search
          :loading="loading"
          :options="options"
          :field-names="{ label: 'text', value: 'value' }"
          option-filter-prop="text"
          :placeholder="t('tenant_console.impersonate.pick_tenant')"
          style="min-width: 280px"
          @change="onChange"
        />
      </div>
    </template>
    <template v-if="!loading && tenants.length > 0 && !value" #description>
      {{ t('tenant_console.impersonate.need_select') }}
    </template>
  </a-alert>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  API,
  isAdmin,
  isTenantAdmin,
  showError,
  getTenantConsoleActingTenantId,
  setTenantConsoleActingTenantId,
} from '@/helpers';

const { t } = useI18n();
const tenants = ref([]);
const loading = ref(true);
const value = ref(getTenantConsoleActingTenantId());

const visible = computed(() => isAdmin() && !isTenantAdmin());

const options = computed(() =>
  tenants.value.map((row) => ({
    key: String(row.id),
    value: String(row.id),
    text: `${row.name} (#${row.id})`,
  }))
);

const loadTenants = async (silent = false) => {
  if (!silent) loading.value = true;
  try {
    const res = await API.get('/api/platform/tenants');
    const { success, data, message } = res.data;
    if (!success) {
      showError(message || t('tenant_console.impersonate.load_failed'));
      return;
    }
    const list = Array.isArray(data) ? data : [];
    tenants.value = list;
    let cur = getTenantConsoleActingTenantId();
    if (!cur && list.length > 0) {
      setTenantConsoleActingTenantId(String(list[0].id));
      cur = getTenantConsoleActingTenantId();
    }
    value.value = cur;
  } finally {
    loading.value = false;
  }
};

const onChange = (v) => {
  setTenantConsoleActingTenantId(v);
  value.value = getTenantConsoleActingTenantId();
  window.location.reload();
};

onMounted(() => {
  if (!visible.value) return;
  loadTenants();
});
</script>
