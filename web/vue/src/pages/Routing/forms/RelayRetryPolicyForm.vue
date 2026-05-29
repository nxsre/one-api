<template>
  <div class="routing-policy-form-segment">
    <a-form layout="vertical">
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item :label="t('routing.form_max_retries_override')">
            <a-input
              type="number"
              :value="p.max_retries_override === null || p.max_retries_override === undefined ? '' : p.max_retries_override"
              :disabled="disabled"
              :placeholder="t('routing.form_placeholder_inherit_system')"
              @update:value="onMaxRetriesChange"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_base_backoff_ms')">
            <a-input
              type="number"
              :value="p.base_backoff_ms"
              :disabled="disabled"
              @update:value="(v) => nfi('base_backoff_ms', v)"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_max_backoff_ms')">
            <a-input
              type="number"
              :value="p.max_backoff_ms"
              :disabled="disabled"
              @update:value="(v) => nfi('max_backoff_ms', v)"
            />
          </a-form-item>
        </a-col>
      </a-row>

      <a-form-item :label="t('routing.form_retry_allowlist')">
        <a-select
          mode="multiple"
          show-search
          :options="HTTP_RETRY_CODE_OPTIONS"
          :value="p.retry_http_status_allowlist || []"
          :disabled="disabled"
          :placeholder="t('routing.form_pick_http_codes')"
          style="width: 100%"
          @update:value="(v) => setField('retry_http_status_allowlist', v)"
        />
      </a-form-item>

      <a-form-item :label="t('routing.form_retry_denylist')">
        <a-select
          mode="multiple"
          show-search
          :options="HTTP_RETRY_CODE_OPTIONS"
          :value="p.retry_http_status_denylist || []"
          :disabled="disabled"
          :placeholder="t('routing.form_pick_http_codes')"
          style="width: 100%"
          @update:value="(v) => setField('retry_http_status_denylist', v)"
        />
      </a-form-item>

      <a-form-item>
        <a-checkbox
          :checked="!!p.force_different_channel_each_attempt"
          :disabled="disabled"
          @change="(e) => setField('force_different_channel_each_attempt', e.target.checked)"
        >
          {{ t('routing.form_force_different_channel') }}
        </a-checkbox>
      </a-form-item>

      <PolicyFormResetRow :disabled="disabled" @reset="resetDefaults" />
    </a-form>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  EMPTY_RELAY_RETRY,
  HTTP_RETRY_CODE_OPTIONS,
  safeParseObject,
  stringifyPolicy,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow.vue';

const props = defineProps({
  value: { type: String, default: '{}' },
  disabled: { type: Boolean, default: false },
});
const emit = defineEmits(['change']);

const { t } = useI18n();

function parseRelay(o) {
  return {
    max_retries_override:
      o.max_retries_override === undefined ? null : o.max_retries_override,
    base_backoff_ms: toInt(o.base_backoff_ms, 150),
    max_backoff_ms: toInt(o.max_backoff_ms, 4000),
    retry_http_status_allowlist: Array.isArray(o.retry_http_status_allowlist)
      ? o.retry_http_status_allowlist.map((x) => toInt(x, 0))
      : [...EMPTY_RELAY_RETRY.retry_http_status_allowlist],
    retry_http_status_denylist: Array.isArray(o.retry_http_status_denylist)
      ? o.retry_http_status_denylist.map((x) => toInt(x, 0))
      : [],
    force_different_channel_each_attempt: !!o.force_different_channel_each_attempt,
  };
}

const p = ref(parseRelay(safeParseObject(props.value, EMPTY_RELAY_RETRY)));

watch(
  () => props.value,
  (v) => {
    p.value = parseRelay(safeParseObject(v, EMPTY_RELAY_RETRY));
  }
);

function commit(next) {
  p.value = next;
  const o = { ...next };
  if (
    o.max_retries_override === null ||
    o.max_retries_override === undefined ||
    o.max_retries_override === ''
  ) {
    delete o.max_retries_override;
  }
  emit('change', stringifyPolicy(o));
}

function nfi(name, v) {
  commit({ ...p.value, [name]: toInt(v, p.value[name] ?? 0) });
}

function setField(name, v) {
  commit({ ...p.value, [name]: v });
}

function onMaxRetriesChange(v) {
  const s = String(v).trim();
  if (s === '') commit({ ...p.value, max_retries_override: null });
  else commit({ ...p.value, max_retries_override: toInt(s, 0) });
}

function resetDefaults() {
  commit({ ...EMPTY_RELAY_RETRY });
}
</script>
