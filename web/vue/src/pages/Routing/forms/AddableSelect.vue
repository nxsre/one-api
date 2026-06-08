<template>
  <a-select
    :value="value || undefined"
    :options="mergedOptions"
    :disabled="disabled"
    :placeholder="placeholder"
    :allow-clear="clearable"
    :style="fluid ? { width: '100%' } : undefined"
    show-search
    option-filter-prop="label"
    :filter-option="filterOption"
    @search="onSearch"
    @change="onChange"
    @blur="onBlur"
  >
    <template v-if="addable && searchText && !optionExists(searchText)" #dropdownRender="{ menuNode }">
      <component :is="menuNode" />
      <a-divider style="margin: 4px 0" />
      <div
        style="padding: 5px 12px; cursor: pointer"
        @mousedown.prevent="addCurrent"
      >
        {{ additionLabel }}{{ searchText }}
      </div>
    </template>
  </a-select>
</template>

<script setup>
import { ref, computed } from 'vue';

const props = defineProps({
  value: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  disabled: { type: Boolean, default: false },
  placeholder: { type: String, default: '' },
  clearable: { type: Boolean, default: true },
  fluid: { type: Boolean, default: true },
  addable: { type: Boolean, default: true },
  additionLabel: { type: String, default: '' },
});
const emit = defineEmits(['update:value', 'add']);

const searchText = ref('');

// Normalize incoming options ({key,text,value} or {label,value}) to AntD {label,value}
const normalized = computed(() =>
  (props.options || []).map((o) => ({
    value: o.value,
    label: o.label != null ? o.label : o.text != null ? o.text : String(o.value),
  }))
);

const mergedOptions = computed(() => {
  const current = String(props.value ?? '').trim();
  if (!current) return normalized.value;
  if (normalized.value.some((o) => String(o.value) === current)) return normalized.value;
  return [{ value: current, label: current }, ...normalized.value];
});

function optionExists(v) {
  const s = String(v ?? '').trim();
  return normalized.value.some((o) => String(o.value) === s);
}

function filterOption(input, option) {
  return String(option.label ?? '').toLowerCase().includes(String(input).toLowerCase());
}

function onSearch(v) {
  searchText.value = v;
}

function onChange(v) {
  searchText.value = '';
  emit('update:value', v == null ? '' : v);
}

function addCurrent() {
  const v = String(searchText.value ?? '').trim();
  if (!v) return;
  emit('add', v);
  emit('update:value', v);
  searchText.value = '';
}

function onBlur() {
  searchText.value = '';
}
</script>
