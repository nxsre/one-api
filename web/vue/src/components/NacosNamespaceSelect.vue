<template>
  <a-select
    :value="value || undefined"
    :options="selectOptions"
    :loading="loading"
    :disabled="disabled"
    :placeholder="t('nacos.ns_select_placeholder')"
    :style="fluid ? { width: '100%' } : undefined"
    show-search
    :filter-option="false"
    @change="(v) => emit('change', v)"
    @search="onSearch"
  >
    <template v-if="customOption" #dropdownRender="{ menuNode }">
      <component :is="menuNode" />
      <div
        style="padding: 6px 12px; cursor: pointer; border-top: 1px solid rgba(0,0,0,.06)"
        @mousedown.prevent="addCustom"
      >
        {{ t('nacos.ns_custom') }}: {{ searchQuery }}
      </div>
    </template>
  </a-select>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { API, showError } from '@/helpers';

const { t } = useI18n();

const props = defineProps({
  value: { type: String, default: '' },
  fluid: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
});

const emit = defineEmits(['change']);

const options = ref([]);
const loading = ref(false);
const searchQuery = ref('');
let mounted = true;
let timer = null;

onUnmounted(() => {
  mounted = false;
  if (timer) clearTimeout(timer);
});

// Ensure the current value is always present as an option.
watch(
  () => props.value,
  (v) => {
    if (!v) return;
    if (!options.value.some((o) => o.value === v)) {
      options.value = [...options.value, { key: v, value: v, label: v }].sort(
        (a, b) => a.label.localeCompare(b.label)
      );
    }
  },
  { immediate: true }
);

const fetchOptions = (q) => {
  loading.value = true;
  API.get('/api/nacos/namespaces/options', { params: { q: q || '' } })
    .then((res) => {
      if (!mounted) return;
      if (!res.data?.success) {
        showError(res.data?.message || 'namespace options');
        return;
      }
      const ns = res.data.data?.namespaces || [];
      options.value = ns.map((id) => ({ key: id, value: id, label: id }));
    })
    .catch((e) => {
      if (mounted) showError(e.message);
    })
    .finally(() => {
      if (mounted) loading.value = false;
    });
};

// Initial load.
fetchOptions('');

const onSearch = (q) => {
  searchQuery.value = q;
  if (timer) clearTimeout(timer);
  const delay = q ? 280 : 0;
  timer = setTimeout(() => fetchOptions(q), delay);
};

const selectOptions = computed(() => options.value);

const customOption = computed(() => {
  const v = String(searchQuery.value || '').trim();
  return !!v && !options.value.some((o) => o.value === v);
});

const addCustom = () => {
  const v = String(searchQuery.value || '').trim();
  if (!v) return;
  emit('change', v);
  if (!options.value.some((o) => o.value === v)) {
    options.value = [...options.value, { key: v, value: v, label: v }].sort(
      (a, b) => a.label.localeCompare(b.label)
    );
  }
  searchQuery.value = '';
};
</script>
