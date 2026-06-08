<template>
  <div class="routing-policy-form-segment">
    <a-form layout="vertical">
      <a-form-item>
        <a-checkbox
          :checked="!!st.enabled"
          :disabled="disabled"
          @change="(e) => commit({ ...st, enabled: !!e.target.checked })"
        >
          {{ t('routing.form_rl_enabled') }}
        </a-checkbox>
      </a-form-item>
    </a-form>

    <a-tabs>
      <a-tab-pane key="token" :tab="t('routing.form_rl_by_token')">
        <p class="routing-form-hint">{{ t('routing.form_rl_by_token_hint') }}</p>
        <table class="model-rate-limit-policy-table mrl-table">
          <thead>
            <tr>
              <th>{{ t('routing.form_rl_key_model_glob') }}</th>
              <th>QPS</th>
              <th>{{ t('routing.form_rl_burst') }}</th>
              <th>{{ t('routing.form_rl_concurrency') }}</th>
              <th>{{ t('routing.form_rl_daily_quota') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in st.tokenRows" :key="i">
              <td>
                <AddableSelect
                  :value="row.key"
                  :options="[...starOpt, ...modelOpts]"
                  :disabled="!lookupReady || disabled"
                  :addition-label="addCustomLabel"
                  @add="(v) => onAddModel(v)"
                  @update:value="(v) => updateRow('tokenRows', i, { key: v })"
                />
              </td>
              <td>
                <a-input type="number" :value="row.qps" :disabled="disabled"
                  @update:value="(v) => updateRow('tokenRows', i, { qps: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.burst" :disabled="disabled"
                  @update:value="(v) => updateRow('tokenRows', i, { burst: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.concurrency" :disabled="disabled"
                  @update:value="(v) => updateRow('tokenRows', i, { concurrency: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.daily_quota" :disabled="disabled"
                  @update:value="(v) => updateRow('tokenRows', i, { daily_quota: toInt(v, 0) })" />
              </td>
              <td>
                <a-button type="button" :disabled="disabled || st.tokenRows.length <= 1"
                  @click="removeTokenRow(i)">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </td>
            </tr>
          </tbody>
        </table>
        <a-button type="button" size="small" :disabled="disabled"
          @click="commit({ ...st, tokenRows: [...st.tokenRows, { key: '*', ...defaultRule }] })">
          {{ t('routing.form_add_row') }}
        </a-button>
      </a-tab-pane>

      <a-tab-pane key="user" :tab="t('routing.form_rl_by_user')">
        <p class="routing-form-hint">{{ t('routing.form_rl_by_user_hint') }}</p>
        <table class="model-rate-limit-policy-table mrl-table">
          <thead>
            <tr>
              <th>{{ t('routing.form_rl_key_user_id') }}</th>
              <th>QPS</th>
              <th>{{ t('routing.form_rl_burst') }}</th>
              <th>{{ t('routing.form_rl_concurrency') }}</th>
              <th>{{ t('routing.form_rl_daily_quota') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in st.userRows" :key="i">
              <td>
                <a-input :value="row.key" :disabled="disabled" placeholder="user id"
                  @update:value="(v) => updateRow('userRows', i, { key: v })" />
              </td>
              <td>
                <a-input type="number" :value="row.qps" :disabled="disabled"
                  @update:value="(v) => updateRow('userRows', i, { qps: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.burst" :disabled="disabled"
                  @update:value="(v) => updateRow('userRows', i, { burst: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.concurrency" :disabled="disabled"
                  @update:value="(v) => updateRow('userRows', i, { concurrency: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.daily_quota" :disabled="disabled"
                  @update:value="(v) => updateRow('userRows', i, { daily_quota: toInt(v, 0) })" />
              </td>
              <td>
                <a-button type="button" :disabled="disabled"
                  @click="commit({ ...st, userRows: st.userRows.filter((_, j) => j !== i) })">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </td>
            </tr>
          </tbody>
        </table>
        <a-button type="button" size="small" :disabled="disabled"
          @click="commit({ ...st, userRows: [...st.userRows, { key: '', ...defaultRule }] })">
          {{ t('routing.form_add_row') }}
        </a-button>
      </a-tab-pane>

      <a-tab-pane key="group" :tab="t('routing.form_rl_by_group')">
        <p class="routing-form-hint">{{ t('routing.form_rl_by_group_hint') }}</p>
        <table class="model-rate-limit-policy-table mrl-table">
          <thead>
            <tr>
              <th>{{ t('routing.group') }}</th>
              <th>QPS</th>
              <th>{{ t('routing.form_rl_burst') }}</th>
              <th>{{ t('routing.form_rl_concurrency') }}</th>
              <th>{{ t('routing.form_rl_daily_quota') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in st.groupRows" :key="i">
              <td>
                <AddableSelect
                  :value="row.key"
                  :options="groupOpts"
                  :disabled="!lookupReady || disabled"
                  :addition-label="addCustomLabel"
                  @add="(v) => onAddGroup(v)"
                  @update:value="(v) => updateRow('groupRows', i, { key: v })"
                />
              </td>
              <td>
                <a-input type="number" :value="row.qps" :disabled="disabled"
                  @update:value="(v) => updateRow('groupRows', i, { qps: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.burst" :disabled="disabled"
                  @update:value="(v) => updateRow('groupRows', i, { burst: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.concurrency" :disabled="disabled"
                  @update:value="(v) => updateRow('groupRows', i, { concurrency: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="row.daily_quota" :disabled="disabled"
                  @update:value="(v) => updateRow('groupRows', i, { daily_quota: toInt(v, 0) })" />
              </td>
              <td>
                <a-button type="button" :disabled="disabled"
                  @click="commit({ ...st, groupRows: st.groupRows.filter((_, j) => j !== i) })">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </td>
            </tr>
          </tbody>
        </table>
        <a-button type="button" size="small" :disabled="disabled"
          @click="commit({ ...st, groupRows: [...st.groupRows, { key: 'default', ...defaultRule }] })">
          {{ t('routing.form_add_row') }}
        </a-button>
      </a-tab-pane>
    </a-tabs>

    <PolicyFormResetRow :disabled="disabled" @reset="resetDefaults" />
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { DeleteOutlined } from '@ant-design/icons-vue';
import {
  EMPTY_RATE_LIMIT_POLICY,
  safeParseObject,
  stringifyPolicy,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow.vue';
import AddableSelect from './AddableSelect.vue';

const props = defineProps({
  value: { type: String, default: '{}' },
  disabled: { type: Boolean, default: false },
  modelOpts: { type: Array, default: () => [] },
  groupOpts: { type: Array, default: () => [] },
  lookupReady: { type: Boolean, default: false },
});
const emit = defineEmits(['change', 'add-model', 'add-group']);

const { t } = useI18n();

const defaultRule = { qps: 80, burst: 120, concurrency: 20, daily_quota: 0 };
const starOpt = [{ key: '*', text: '*', value: '*' }];
const addCustomLabel = computed(() => `${t('routing.combo_add_custom')} `);

function canonicalJsonString(s) {
  try {
    return JSON.stringify(JSON.parse(String(s ?? '').trim() || '{}'));
  } catch {
    return String(s ?? '');
  }
}

function mapToRows(m) {
  const o = m && typeof m === 'object' ? m : {};
  return Object.entries(o).map(([key, r]) => ({
    key,
    qps: toInt(r?.qps, 0),
    burst: toInt(r?.burst, 0),
    concurrency: toInt(r?.concurrency, 0),
    daily_quota: toInt(r?.daily_quota, 0),
  }));
}

function fromRateJSON(raw) {
  const o = safeParseObject(raw, EMPTY_RATE_LIMIT_POLICY);
  const tokenRows = mapToRows(o.by_token);
  const userRows = mapToRows(o.by_user);
  const groupRows = mapToRows(o.by_group);
  return {
    enabled: o.enabled !== false,
    tokenRows: tokenRows.length ? tokenRows : [{ key: '*', ...defaultRule }],
    userRows,
    groupRows,
  };
}

function toRateJSON(state) {
  const toMap = (rows) => {
    const m = {};
    for (const r of rows) {
      const k = String(r.key ?? '').trim();
      if (!k) continue;
      m[k] = {
        qps: toInt(r.qps, 0),
        burst: toInt(r.burst, 0),
        concurrency: toInt(r.concurrency, 0),
        daily_quota: toInt(r.daily_quota, 0),
      };
    }
    return m;
  };
  return stringifyPolicy({
    enabled: !!state.enabled,
    by_token: toMap(state.tokenRows),
    by_user: toMap(state.userRows),
    by_group: toMap(state.groupRows),
  });
}

const st = ref(fromRateJSON(props.value));

watch(
  () => props.value,
  (v) => {
    try {
      const prevSerialized = toRateJSON(st.value);
      if (canonicalJsonString(prevSerialized) === canonicalJsonString(v)) {
        return;
      }
    } catch {
      /* fall through */
    }
    st.value = fromRateJSON(v);
  }
);

function commit(next) {
  st.value = next;
  const json = toRateJSON(next);
  queueMicrotask(() => {
    emit('change', json);
  });
}

function updateRow(listKey, i, patch) {
  const next = [...st.value[listKey]];
  next[i] = { ...next[i], ...patch };
  commit({ ...st.value, [listKey]: next });
}

function removeTokenRow(i) {
  const next = st.value.tokenRows.filter((_, j) => j !== i);
  commit({
    ...st.value,
    tokenRows: next.length ? next : [{ key: '*', ...defaultRule }],
  });
}

function onAddModel(v) {
  emit('add-model', v);
}
function onAddGroup(v) {
  emit('add-group', v);
}

function resetDefaults() {
  commit(fromRateJSON(stringifyPolicy(EMPTY_RATE_LIMIT_POLICY)));
}
</script>

<style scoped>
.mrl-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 0.75rem;
}
.mrl-table th,
.mrl-table td {
  border: 1px solid rgba(34, 36, 38, 0.1);
  padding: 4px 6px;
  text-align: left;
}
.routing-form-hint {
  opacity: 0.75;
  margin-bottom: 0.5rem;
}
</style>
