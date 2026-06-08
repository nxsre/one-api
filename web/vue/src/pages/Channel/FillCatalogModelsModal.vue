<template>
  <a-modal
    :open="open"
    width="540px"
    :title="t('channel.edit.fill_catalog_title')"
    @cancel="emit('close')"
  >
    <p class="channel-edit-field-hint">{{ t('channel.edit.fill_catalog_hint') }}</p>
    <a-form layout="vertical">
      <a-form-item :label="t('model_catalog.filter_provider_label')">
        <a-select
          show-search
          allow-clear
          :value="provider || undefined"
          :options="mergedOptions"
          :field-names="{ label: 'text', value: 'value' }"
          :placeholder="t('channel.edit.fill_catalog_provider_ph')"
          :loading="providerLoading"
          :filter-option="false"
          :not-found-content="t('model_catalog.provider_no_results')"
          style="width: 100%"
          @search="onProviderSearch"
          @change="(v) => (provider = v || '')"
        />
      </a-form-item>
      <a-form-item v-if="previewCount != null && provider">
        <label>{{ t('channel.edit.fill_catalog_preview', { count: previewCount }) }}</label>
      </a-form-item>
    </a-form>
    <template #footer>
      <a-button @click="emit('close')">{{ t('channel.edit.fill_catalog_cancel') }}</a-button>
      <a-button type="primary" :loading="loading" :disabled="loading" @click="submit">
        {{ t('channel.edit.fill_catalog_confirm') }}
      </a-button>
    </template>
  </a-modal>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  fetchModelCatalogModelIds,
  fetchModelCatalogProviders,
  providerCatalogModelFilters,
  providerOptionsToDropdown,
  showError,
} from '@/helpers';

const props = defineProps({
  open: { type: Boolean, default: false },
  tenantConsole: { type: Boolean, default: false },
});
const emit = defineEmits(['close', 'confirm']);

const { t } = useI18n();

const provider = ref('');
const providerOptions = ref([]);
const providerLoading = ref(false);
const loading = ref(false);
const previewCount = ref(null);

let searchTimer = null;
let previewTimer = null;

const mergedOptions = computed(() => {
  const current = String(provider.value || '').trim();
  if (!current) return providerOptions.value;
  if (providerOptions.value.some((o) => o.value === current)) return providerOptions.value;
  return [{ key: `custom-${current}`, value: current, text: current }, ...providerOptions.value];
});

async function loadProviders(query) {
  providerLoading.value = true;
  try {
    const rows = await fetchModelCatalogProviders(query, 100, props.tenantConsole);
    providerOptions.value = providerOptionsToDropdown(rows);
  } catch {
    providerOptions.value = [];
  } finally {
    providerLoading.value = false;
  }
}

function onProviderSearch(query) {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    loadProviders(query);
  }, 220);
}

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) return;
    provider.value = '';
    previewCount.value = null;
    loadProviders('');
  }
);

watch([() => props.open, provider, providerOptions], () => {
  if (!props.open) return;
  const filters = providerCatalogModelFilters(provider.value, providerOptions.value);
  if (!filters) {
    previewCount.value = null;
    return;
  }
  if (previewTimer) clearTimeout(previewTimer);
  previewTimer = setTimeout(() => {
    fetchModelCatalogModelIds(filters, props.tenantConsole)
      .then((ids) => (previewCount.value = ids.length))
      .catch(() => (previewCount.value = null));
  }, 250);
});

async function submit() {
  const filters = providerCatalogModelFilters(provider.value, providerOptions.value);
  if (!filters) {
    showError(t('channel.edit.fill_catalog_provider_required'));
    return;
  }
  loading.value = true;
  try {
    const ids = await fetchModelCatalogModelIds(filters, props.tenantConsole);
    if (!ids.length) {
      showError(t('channel.edit.fill_catalog_empty'));
      return;
    }
    emit('confirm', ids);
    emit('close');
  } catch (e) {
    showError(e.message || t('channel.edit.fill_catalog_fail'));
  } finally {
    loading.value = false;
  }
}

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer);
  if (previewTimer) clearTimeout(previewTimer);
});
</script>
