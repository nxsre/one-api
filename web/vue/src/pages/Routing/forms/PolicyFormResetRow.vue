<template>
  <div class="routing-policy-form-reset-row">
    <a-button
      type="button"
      size="small"
      class="routing-policy-btn-reset"
      :disabled="disabled"
      @click="emit('reset')"
    >
      <template #icon><UndoOutlined /></template>
      {{ t('routing.form_reset_defaults') }}
    </a-button>
    <span class="routing-policy-reset-hint">
      {{ t('routing.form_reset_local_only_hint') }}
    </span>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n';
import { UndoOutlined } from '@ant-design/icons-vue';

defineProps({
  disabled: { type: Boolean, default: false },
});
const emit = defineEmits(['reset']);

const { t } = useI18n();
</script>

<style scoped>
.routing-policy-form-reset-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}
.routing-policy-reset-hint {
  font-size: 12px;
  opacity: 0.7;
}
</style>
