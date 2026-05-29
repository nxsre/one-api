<template>
  <div ref="el" :style="{ height: typeof height === 'number' ? height + 'px' : height, width: '100%' }"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';
import monaco from '../monaco-setup';

const props = defineProps({
  value: { type: String, default: '' },
  language: { type: String, default: 'plaintext' },
  height: { type: [Number, String], default: 220 },
  readOnly: { type: Boolean, default: false },
  minimap: { type: Boolean, default: false },
  theme: { type: String, default: '' },
  options: { type: Object, default: () => ({}) },
});
const emit = defineEmits(['update:value', 'change']);

const el = ref(null);
let editor = null;
let applyingExternal = false;
let ro = null;

function resolveTheme() {
  if (props.theme) return props.theme;
  return document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs';
}

onMounted(() => {
  editor = monaco.editor.create(el.value, {
    value: props.value,
    language: props.language,
    readOnly: props.readOnly,
    automaticLayout: true,
    minimap: { enabled: props.minimap },
    scrollBeyondLastLine: false,
    fontSize: 13,
    theme: resolveTheme(),
    ...props.options,
  });
  editor.onDidChangeModelContent(() => {
    if (applyingExternal) return;
    const v = editor.getValue();
    emit('update:value', v);
    emit('change', v);
  });

  // When created inside an opening Ant Design modal/drawer, Monaco often measures
  // a too-narrow width before the container reaches full size and never relayouts,
  // leaving a blank strip. Observe the host and force layout; also nudge it a few
  // times across the open animation.
  relayout();
  ro = new ResizeObserver(() => relayout());
  ro.observe(el.value);
  [0, 60, 200, 400].forEach((d) => setTimeout(relayout, d));
});

function relayout() {
  if (editor && el.value && el.value.clientWidth > 0) {
    editor.layout();
  }
}

watch(
  () => props.value,
  (v) => {
    if (!editor) return;
    if (v !== editor.getValue()) {
      applyingExternal = true;
      editor.setValue(v ?? '');
      applyingExternal = false;
    }
  }
);

watch(
  () => props.language,
  (l) => {
    if (editor) monaco.editor.setModelLanguage(editor.getModel(), l);
  }
);

watch(
  () => props.readOnly,
  (ro) => editor && editor.updateOptions({ readOnly: ro })
);

onBeforeUnmount(() => {
  if (ro) {
    ro.disconnect();
    ro = null;
  }
  if (editor) {
    editor.dispose();
    editor = null;
  }
});
</script>
