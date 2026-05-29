<template>
  <div class="nacos-monaco-host" :style="{ height: typeof height === 'number' ? height + 'px' : height }">
    <MonacoEditor
      :value="value ?? ''"
      :language="language"
      :read-only="readOnly"
      :minimap="minimap"
      :theme="theme || undefined"
      :height="'100%'"
      :options="editorOptions"
      @update:value="onUpdate"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue';
import MonacoEditor from '@/components/MonacoEditor.vue';
import './nacos-monaco.css';

const props = defineProps({
  value: { type: String, default: '' },
  language: { type: String, default: 'plaintext' },
  readOnly: { type: Boolean, default: false },
  height: { type: [Number, String], default: 380 },
  // Minimap shown with a divider border (see nacos-monaco.css) so it reads as an
  // intentional panel rather than an unstyled blank strip.
  minimap: { type: Boolean, default: true },
  theme: { type: String, default: 'vs' },
});

const emit = defineEmits(['update:value', 'change']);

const editorOptions = computed(() => ({
  fontSize: 13,
  lineHeight: 22,
  padding: { top: 10, bottom: 10 },
  wordWrap: 'on',
  scrollBeyondLastLine: false,
  automaticLayout: true,
  tabSize: 2,
  formatOnPaste: !props.readOnly,
  formatOnType: !props.readOnly,
  folding: true,
  lineNumbers: 'on',
  renderWhitespace: 'selection',
  smoothScrolling: true,
  cursorBlinking: 'smooth',
  renderLineHighlight: 'all',
  overviewRulerBorder: false,
  hideCursorInOverviewRuler: true,
  minimap: { enabled: props.minimap },
  overviewRulerLanes: 0,
  scrollbar: { verticalScrollbarSize: 10, horizontalScrollbarSize: 10 },
}));

function onUpdate(v) {
  emit('update:value', v ?? '');
  emit('change', v ?? '');
}
</script>
