<template>
  <div class="model-catalog-provider-search">
    <label v-if="label" class="model-catalog-provider-search__label">{{ label }}</label>
    <a-select
      :value="value || undefined"
      :options="mergedOptions"
      :loading="loading"
      :placeholder="placeholder || t('model_catalog.filter_ph_provider')"
      :style="fluid ? { width: '100%' } : undefined"
      show-search
      allow-clear
      :filter-option="false"
      :not-found-content="t('model_catalog.provider_no_results')"
      @search="handleSearchChange"
      @change="handleChange"
      @blur="handleBlur"
    >
      <template v-if="searchText && !optionExists(searchText)" #dropdownRender="{ menuNode }">
        <component :is="menuNode" />
        <a-divider style="margin: 4px 0" />
        <div style="padding: 5px 12px; cursor: pointer" @mousedown.prevent="addCurrent">
          {{ searchText }}
        </div>
      </template>
    </a-select>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  fetchModelCatalogProviders,
  providerOptionsToDropdown,
} from '@/helpers/modelCatalog';

const props = defineProps({
  value: { type: String, default: '' },
  label: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  tenantConsole: { type: Boolean, default: false },
  fluid: { type: Boolean, default: true },
});
const emit = defineEmits(['change']);

const { t } = useI18n();

const options = ref([]);
const loading = ref(false);
const searchText = ref('');
let timer = null;

async function loadProviders(query) {
  loading.value = true;
  try {
    const rows = await fetchModelCatalogProviders(query, 40, props.tenantConsole);
    options.value = providerOptionsToDropdown(rows).map((o) => ({
      value: o.value,
      label: o.text,
    }));
  } catch {
    options.value = [];
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadProviders(props.value);
});

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer);
});

const mergedOptions = computed(() => {
  const current = String(props.value || '').trim();
  if (!current) return options.value;
  if (options.value.some((o) => o.value === current)) return options.value;
  return [{ value: current, label: current }, ...options.value];
});

function optionExists(v) {
  const s = String(v ?? '').trim();
  return options.value.some((o) => String(o.value) === s);
}

function handleSearchChange(query) {
  searchText.value = query;
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => {
    loadProviders(query);
  }, 220);
}

// emit('change', value, exact)：exact 表示 value 命中某个已知 provider 选项（应走精确 provider_key 过滤），
// 否则为手工输入（走模糊 filter_provider）。
function handleChange(v) {
  searchText.value = '';
  const val = v || '';
  emit('change', val, optionExists(val));
}

function addCurrent() {
  const v = String(searchText.value ?? '').trim();
  if (!v) return;
  searchText.value = '';
  emit('change', v, optionExists(v));
}

function handleBlur() {
  if (!String(props.value || '').trim() && mergedOptions.value.length === 1) {
    emit('change', mergedOptions.value[0].value, true);
  }
}
</script>

<style scoped>
.model-catalog-provider-search__label {
  display: block;
  margin-bottom: 4px;
  font-weight: 700;
  font-size: 0.92857143em;
}
</style>
