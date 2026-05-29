<template>
  <a-modal
    :open="open"
    :title="title || t('nacos.cs_diff_title')"
    width="80%"
    :footer="null"
    @cancel="$emit('close')"
  >
    <div class="nacos-monaco-diff-wrap" style="height: min(72vh, 680px)">
      <div ref="el" style="width: 100%; height: 100%"></div>
    </div>
    <template #footer>
      <a-button @click="$emit('close')">{{ t('nacos.skills_close') }}</a-button>
    </template>
  </a-modal>
</template>

<script setup>
import { ref, watch, onBeforeUnmount, nextTick } from 'vue';
import { useI18n } from 'vue-i18n';
import monaco from '@/monaco-setup';
import './nacos-monaco.css';

const { t } = useI18n();

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '' },
  original: { type: String, default: '' },
  modified: { type: String, default: '' },
  language: { type: String, default: 'plaintext' },
  theme: { type: String, default: 'vs' },
});

defineEmits(['close']);

const el = ref(null);
let diffEditor = null;

function resolveTheme() {
  if (props.theme) return props.theme;
  return document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs';
}

function createEditor() {
  if (!el.value || diffEditor) return;
  diffEditor = monaco.editor.createDiffEditor(el.value, {
    readOnly: true,
    renderSideBySide: true,
    scrollBeyondLastLine: false,
    minimap: { enabled: false },
    fontSize: 13,
    automaticLayout: true,
    theme: resolveTheme(),
  });
  setModels();
}

function setModels() {
  if (!diffEditor) return;
  diffEditor.setModel({
    original: monaco.editor.createModel(props.original ?? '', props.language),
    modified: monaco.editor.createModel(props.modified ?? '', props.language),
  });
}

function disposeEditor() {
  if (diffEditor) {
    const model = diffEditor.getModel();
    diffEditor.dispose();
    diffEditor = null;
    if (model) {
      model.original && model.original.dispose();
      model.modified && model.modified.dispose();
    }
  }
}

watch(
  () => props.open,
  async (isOpen) => {
    if (isOpen) {
      await nextTick();
      createEditor();
    } else {
      disposeEditor();
    }
  }
);

watch(
  () => [props.original, props.modified, props.language],
  () => {
    if (diffEditor) {
      const old = diffEditor.getModel();
      setModels();
      if (old) {
        old.original && old.original.dispose();
        old.modified && old.modified.dispose();
      }
    }
  }
);

onBeforeUnmount(disposeEditor);
</script>
