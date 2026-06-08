<template>
  <div class="routing-policy-form-segment">
    <h5>{{ t('routing.form_routing_core') }}</h5>
    <a-form layout="vertical">
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item :label="t('routing.form_selection_mode')">
            <a-select
              :options="selModeOpts"
              :value="p.selection_mode || 'weighted_random'"
              :disabled="disabled"
              @update:value="(v) => setField('selection_mode', v)"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_consistent_hash_seed')">
            <a-input
              :value="p.consistent_hash_seed ?? ''"
              :disabled="disabled"
              placeholder="consistent_hash"
              @update:value="(v) => setField('consistent_hash_seed', v)"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_sticky_source')">
            <a-select
              :options="stickyOpts"
              :value="p.sticky_source || 'token_id'"
              :disabled="disabled"
              @update:value="(v) => setField('sticky_source', v)"
            />
          </a-form-item>
        </a-col>
      </a-row>

      <h5>{{ t('routing.form_direction') }}</h5>
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item>
            <a-checkbox
              :checked="!!p.direction_enabled"
              :disabled="disabled"
              @change="(e) => setField('direction_enabled', e.target.checked)"
            >
              {{ t('routing.form_direction_enabled') }}
            </a-checkbox>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_direction_probe_ratio')">
            <a-input type="number" step="0.01" :value="p.direction_probe_ratio" :disabled="disabled"
              @update:value="(v) => nff('direction_probe_ratio', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_direction_min_probe_ratio')">
            <a-input type="number" step="0.01" :value="p.direction_min_probe_ratio" :disabled="disabled"
              @update:value="(v) => nff('direction_min_probe_ratio', v)" />
          </a-form-item>
        </a-col>
      </a-row>
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item :label="t('routing.form_direction_latency_weight')">
            <a-input type="number" step="0.1" :value="p.direction_latency_weight" :disabled="disabled"
              @update:value="(v) => nff('direction_latency_weight', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_direction_error_weight')">
            <a-input type="number" step="0.1" :value="p.direction_error_weight" :disabled="disabled"
              @update:value="(v) => nff('direction_error_weight', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_probe_pick_strategy')">
            <a-select
              :options="probeOpts"
              :value="p.probe_pick_strategy || 'secondary_uniform'"
              :disabled="disabled"
              @update:value="(v) => setField('probe_pick_strategy', v)"
            />
          </a-form-item>
        </a-col>
      </a-row>

      <h5>{{ t('routing.form_adaptive_circuit') }}</h5>
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item>
            <a-checkbox
              :checked="!!p.auto_adaptive_enabled"
              :disabled="disabled"
              @change="(e) => setField('auto_adaptive_enabled', e.target.checked)"
            >
              {{ t('routing.form_auto_adaptive_enabled') }}
            </a-checkbox>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_adaptive_interval_sec')">
            <a-input type="number" :value="p.adaptive_interval_sec" :disabled="disabled"
              @update:value="(v) => nfi('adaptive_interval_sec', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_circuit_fail_threshold')">
            <a-input type="number" :value="p.circuit_fail_threshold" :disabled="disabled"
              @update:value="(v) => nfi('circuit_fail_threshold', v)" />
          </a-form-item>
        </a-col>
      </a-row>
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item :label="t('routing.form_circuit_cooldown_sec')">
            <a-input type="number" :value="p.circuit_cooldown_sec" :disabled="disabled"
              @update:value="(v) => nfi('circuit_cooldown_sec', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_half_open_probe_ms')">
            <a-input type="number" :value="p.half_open_probe_ms" :disabled="disabled"
              @update:value="(v) => nfi('half_open_probe_ms', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_aggregation_interval_sec')">
            <a-input type="number" :value="p.aggregation_interval_sec" :disabled="disabled"
              @update:value="(v) => nfi('aggregation_interval_sec', v)" />
          </a-form-item>
        </a-col>
      </a-row>
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item :label="t('routing.form_adaptive_aggressive_penalty')">
            <a-input type="number" step="0.01" :value="p.adaptive_aggressive_penalty" :disabled="disabled"
              @update:value="(v) => nff('adaptive_aggressive_penalty', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_adaptive_gentle_boost')">
            <a-input type="number" step="0.01" :value="p.adaptive_gentle_boost" :disabled="disabled"
              @update:value="(v) => nff('adaptive_gentle_boost', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_adaptive_err_ratio_threshold')">
            <a-input type="number" step="0.01" :value="p.adaptive_err_ratio_threshold" :disabled="disabled"
              @update:value="(v) => nff('adaptive_err_ratio_threshold', v)" />
          </a-form-item>
        </a-col>
      </a-row>
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item :label="t('routing.form_auto_weight_min')">
            <a-input type="number" step="0.01" :value="p.auto_weight_min" :disabled="disabled"
              @update:value="(v) => nff('auto_weight_min', v)" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item :label="t('routing.form_auto_weight_max')">
            <a-input type="number" step="0.01" :value="p.auto_weight_max" :disabled="disabled"
              @update:value="(v) => nff('auto_weight_max', v)" />
          </a-form-item>
        </a-col>
      </a-row>
      <PolicyFormResetRow :disabled="disabled" @reset="resetDefaults" />
    </a-form>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  EMPTY_ROUTING_POLICY,
  safeParseObject,
  stringifyPolicy,
  toFloat,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow.vue';

const props = defineProps({
  value: { type: String, default: '{}' },
  disabled: { type: Boolean, default: false },
});
const emit = defineEmits(['change']);

const { t } = useI18n();

const p = ref(safeParseObject(props.value, EMPTY_ROUTING_POLICY));

watch(
  () => props.value,
  (v) => {
    p.value = safeParseObject(v, EMPTY_ROUTING_POLICY);
  }
);

function commit(next) {
  p.value = next;
  emit('change', stringifyPolicy(next));
}

function setField(name, v) {
  commit({ ...p.value, [name]: v });
}

function nfi(name, v) {
  commit({ ...p.value, [name]: toInt(v, p.value[name] ?? 0) });
}

function nff(name, v) {
  commit({ ...p.value, [name]: toFloat(v, p.value[name] ?? 0) });
}

function resetDefaults() {
  commit({ ...EMPTY_ROUTING_POLICY });
}

const selModeOpts = computed(() => [
  { value: 'weighted_random', label: t('routing.form_sel_weighted_random') },
  { value: 'consistent_hash', label: t('routing.form_sel_consistent_hash') },
]);

const stickyOpts = [
  { value: 'token_id', label: 'token_id' },
  { value: 'user_id', label: 'user_id' },
  { value: 'none', label: 'none' },
];

const probeOpts = [
  { value: 'secondary_uniform', label: 'secondary_uniform' },
  { value: 'worst_first', label: 'worst_first' },
];
</script>
