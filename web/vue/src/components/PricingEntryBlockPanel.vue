<template>
  <div class="pricing-entry-block-panel" :class="{ 'is-expanded': expanded }">
    <button
      type="button"
      class="pricing-entry-block-panel__header"
      :aria-expanded="expanded"
      :aria-controls="panelId"
      @click="emit('toggle')"
    >
      <DownOutlined v-if="expanded" />
      <RightOutlined v-else />
      <span class="pricing-entry-block-panel__title">{{ title }}</span>
      <span
        v-if="$slots.help"
        class="pricing-entry-block-panel__help"
        @click.stop
      >
        <slot name="help" />
      </span>
      <a-tag
        v-if="entryCount != null"
        class="pricing-entry-block-panel__count"
      >
        {{ t('setting.operation.ratio.editor.block_entry_count', { count: entryCount }) }}
      </a-tag>
    </button>
    <div v-if="expanded" :id="panelId" class="pricing-entry-block-panel__body">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { DownOutlined, RightOutlined } from '@ant-design/icons-vue';

const props = defineProps({
  title: { type: String, default: '' },
  expanded: { type: Boolean, default: false },
  entryCount: { type: Number, default: null },
});
const emit = defineEmits(['toggle']);

const { t } = useI18n();
const panelId = computed(
  () => `pricing-block-${String(props.title).replace(/\s+/g, '-')}`
);
</script>

<style>
.pricing-entry-block-panel {
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 6px;
  margin-bottom: 0.65rem;
  background: #fff;
}

.pricing-entry-block-panel__header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: none;
  background: #f8f9fa;
  cursor: pointer;
  text-align: left;
  border-radius: 6px;
  font: inherit;
  color: inherit;
}

.pricing-entry-block-panel.is-expanded .pricing-entry-block-panel__header {
  border-bottom: 1px solid rgba(34, 36, 38, 0.08);
  border-radius: 6px 6px 0 0;
}

.pricing-entry-block-panel__header:hover {
  background: #f1f3f5;
}

.pricing-entry-block-panel__title {
  font-weight: 600;
  flex: 1 1 auto;
}

.pricing-entry-block-panel__help {
  flex: 0 0 auto;
}

.pricing-entry-block-panel__count {
  margin: 0 !important;
}

.pricing-entry-block-panel__body {
  padding: 0.65rem 0.85rem 0.85rem;
}

.pricing-entry-block-panel__body .pricing-entry-editor {
  margin-bottom: 0;
  border: none;
  padding: 0;
  background: transparent;
}

.pricing-entry-blocks-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

html.dark .pricing-entry-block-panel {
  background: rgba(255, 255, 255, 0.02);
  border-color: rgba(255, 255, 255, 0.12);
}

html.dark .pricing-entry-block-panel__header {
  background: rgba(255, 255, 255, 0.04);
}

html.dark .pricing-entry-block-panel__header:hover {
  background: rgba(255, 255, 255, 0.07);
}
</style>
