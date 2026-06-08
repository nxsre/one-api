<template>
  <div class="setting-monaco-field field">
    <div class="setting-monaco-field__head">
      <div class="setting-monaco-field__label-block">
        <label v-if="label != null"><slot name="label">{{ label }}</slot></label>
        <slot v-else name="label" />
        <div v-if="hint" class="setting-monaco-field__hint">
          <slot name="hint">{{ hint }}</slot>
        </div>
      </div>
      <div v-if="isCode" class="setting-monaco-field__actions">
        <a-button
          v-if="!readOnly && enableJsonFormat"
          size="small"
          @click="formatJson"
        >
          {{ t('setting.monaco.format_json') }}
        </a-button>
        <a-button v-if="enableDiff" size="small" @click="diffOpen = true">
          {{ t('setting.monaco.compare_saved') }}
        </a-button>
      </div>
    </div>

    <!-- Code fields (JSON / markdown / etc.) keep the Monaco editor. -->
    <div v-if="isCode" class="setting-monaco-field__surface">
      <MonacoEditor
        :value="strVal"
        :language="language"
        :height="height"
        :minimap="minimap"
        :theme="monacoTheme"
        :read-only="readOnly"
        @update:value="onEditorChange"
      />
    </div>
    <!-- Plain scalar fields render as ordinary inputs (no code editor chrome). -->
    <a-textarea
      v-else-if="isMultiline"
      :value="strVal"
      :auto-size="{ minRows: 3, maxRows: 10 }"
      :readonly="readOnly"
      allow-clear
      @update:value="onEditorChange"
    />
    <a-input
      v-else
      :value="strVal"
      :readonly="readOnly"
      allow-clear
      @update:value="onEditorChange"
    />

    <a-modal
      v-if="isCode && enableDiff"
      v-model:open="diffOpen"
      :title="t('setting.monaco.diff_title')"
      width="min(96vw, 1080px)"
      :footer="null"
      destroy-on-close
      @after-open-change="onDiffOpenChange"
    >
      <div ref="diffEl" class="setting-monaco-field__diff"></div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import MonacoEditor from '@/components/MonacoEditor.vue';
import monaco from '@/monaco-setup';
import { showError } from '@/helpers';

const { t } = useI18n();

const props = defineProps({
  label: { type: [String, Number], default: null },
  hint: { type: String, default: '' },
  value: { type: [String, Number], default: '' },
  originValue: { type: [String, Number], default: '' },
  language: { type: String, default: 'plaintext' },
  height: { type: [Number, String], default: 220 },
  minimap: { type: Boolean, default: false },
  enableJsonFormat: { type: Boolean, default: false },
  enableDiff: { type: Boolean, default: true },
  readOnly: { type: Boolean, default: false },
});

const emit = defineEmits(['update:value', 'change']);

const diffOpen = ref(false);
const diffEl = ref(null);
let diffEditor = null;

const isDark = ref(document.documentElement.classList.contains('dark'));
let observer = null;
onMounted(() => {
  observer = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark');
  });
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  });
});
onBeforeUnmount(() => {
  if (observer) observer.disconnect();
  disposeDiff();
});

const monacoTheme = computed(() => (isDark.value ? 'vs-dark' : 'vs'));

const diffLanguage = computed(() => {
  if (props.language === 'json') return 'json';
  if (props.language === 'markdown' || props.language === 'html')
    return props.language;
  return 'plaintext';
});

const strVal = computed(() => (props.value == null ? '' : String(props.value)));
const strOrigin = computed(() =>
  props.originValue == null ? '' : String(props.originValue)
);

// Only use the Monaco code editor for real code fields. A caller signals "code"
// by passing a non-plaintext language (json/markdown/yaml/...) or enabling JSON
// format. Plain scalar fields (name/desc/bizTags/scope/...) render as inputs.
const isCode = computed(() => {
  if (props.enableJsonFormat) return true;
  const lang = String(props.language || '').toLowerCase();
  return lang !== '' && lang !== 'plaintext' && lang !== 'text';
});

// Plain-text fields: use a textarea when the value spans lines or the caller
// asked for a taller surface; otherwise a single-line input.
const isMultiline = computed(() => {
  if (strVal.value.includes('\n')) return true;
  const h = Number(props.height);
  return Number.isFinite(h) && h > 110;
});

function onEditorChange(v) {
  if (props.readOnly) return;
  emit('update:value', v ?? '');
  emit('change', v ?? '');
}

function formatJson() {
  try {
    const trimmed = strVal.value.trim();
    if (!trimmed) {
      emit('update:value', '');
      emit('change', '');
      return;
    }
    const parsed = JSON.parse(trimmed);
    const out = JSON.stringify(parsed, null, 2);
    emit('update:value', out);
    emit('change', out);
  } catch (e) {
    showError((e && e.message) || t('setting.monaco.json_invalid'));
  }
}

function disposeDiff() {
  if (diffEditor) {
    const models = diffEditor.getModel();
    diffEditor.dispose();
    diffEditor = null;
    if (models) {
      models.original?.dispose();
      models.modified?.dispose();
    }
  }
}

function onDiffOpenChange(opened) {
  if (opened) {
    requestAnimationFrame(() => {
      if (!diffEl.value) return;
      disposeDiff();
      diffEditor = monaco.editor.createDiffEditor(diffEl.value, {
        automaticLayout: true,
        readOnly: true,
        renderSideBySide: true,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 13,
        theme: monacoTheme.value,
      });
      diffEditor.setModel({
        original: monaco.editor.createModel(strOrigin.value, diffLanguage.value),
        modified: monaco.editor.createModel(strVal.value, diffLanguage.value),
      });
    });
  } else {
    disposeDiff();
  }
}
</script>

<style>
.setting-monaco-field.field {
  margin-bottom: 1.35rem !important;
}

.setting-monaco-field__head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px 12px;
  margin-bottom: 8px;
}

.setting-monaco-field__label-block {
  flex: 1 1 auto;
  min-width: 0;
}

.setting-monaco-field__label-block > label {
  display: block;
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--app-chrome-text, #030d12);
  line-height: 1.35;
}

.setting-monaco-field__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--app-chrome-text-muted, #5c6b76);
  margin-top: 4px;
  font-weight: normal;
  white-space: pre-wrap;
  word-break: break-word;
}

.setting-monaco-field__actions {
  flex: 0 0 auto;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 2px;
}

.setting-monaco-field__surface {
  border-radius: 10px;
  transition:
    box-shadow 0.15s ease,
    border-color 0.15s ease;
}

.setting-monaco-field__surface:focus-within {
  box-shadow:
    0 0 0 1px var(--app-accent, #0079ce),
    0 0 0 4px color-mix(in srgb, var(--app-accent, #0079ce) 22%, transparent);
}

html.dark .setting-monaco-field__surface:focus-within {
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--app-accent, #3d9fdf) 85%, transparent),
    0 0 0 4px color-mix(in srgb, var(--app-accent, #3d9fdf) 28%, transparent);
}

.setting-monaco-field__diff {
  width: 100%;
  height: min(60vh, 520px);
}
</style>
