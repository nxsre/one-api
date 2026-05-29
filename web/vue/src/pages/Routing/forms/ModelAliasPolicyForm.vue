<template>
  <div class="routing-policy-form-segment">
    <a-form layout="vertical">
      <a-form-item :label="t('routing.form_alias_max_chain_steps')">
        <a-input
          type="number"
          :value="st.max_chain_steps"
          :disabled="disabled"
          @update:value="(v) => commit({ ...st, max_chain_steps: toInt(v, 32) })"
        />
      </a-form-item>
    </a-form>

    <h5>{{ t('routing.form_alias_chains') }}</h5>
    <p class="routing-form-hint">{{ t('routing.form_alias_chains_hint') }}</p>
    <table class="alias-table">
      <thead>
        <tr>
          <th>{{ t('routing.form_alias_chain_from') }}</th>
          <th>{{ t('routing.form_alias_chain_to') }}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, i) in st.chainRows" :key="i">
          <td>
            <AddableSelect :value="row.from" :options="modelOpts" :disabled="!lookupReady || disabled"
              :addition-label="addCustomLabel" @add="onAddModel"
              @update:value="(v) => updateChainRow(i, { from: v })" />
          </td>
          <td>
            <AddableSelect :value="row.to" :options="modelOpts" :disabled="!lookupReady || disabled"
              :addition-label="addCustomLabel" @add="onAddModel"
              @update:value="(v) => updateChainRow(i, { to: v })" />
          </td>
          <td>
            <a-button type="button" :disabled="disabled || st.chainRows.length <= 1" @click="removeChainRow(i)">
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </td>
        </tr>
      </tbody>
    </table>
    <a-button type="button" size="small" :disabled="disabled"
      @click="commit({ ...st, chainRows: [...st.chainRows, { from: '', to: '' }] })">
      {{ t('routing.form_add_chain_row') }}
    </a-button>

    <h5 style="margin-top: 1.25rem">{{ t('routing.form_alias_aliases') }}</h5>
    <p class="routing-form-hint">{{ t('routing.form_alias_aliases_hint') }}</p>
    <div v-for="(block, bi) in st.aliasBlocks" :key="bi" class="alias-segment">
      <a-row :gutter="16" align="bottom">
        <a-col :span="18">
          <a-form-item :label="t('routing.form_alias_logical_model')">
            <AddableSelect :value="block.logical" :options="modelOpts" :disabled="!lookupReady || disabled"
              :addition-label="addCustomLabel" @add="onAddModel"
              @update:value="(v) => updateAliasBlock(bi, { logical: v })" />
          </a-form-item>
        </a-col>
        <a-col :span="6">
          <a-form-item label=" ">
            <a-button type="button" danger :disabled="disabled || st.aliasBlocks.length <= 1"
              @click="removeAliasBlock(bi)">
              {{ t('routing.form_remove_alias_block') }}
            </a-button>
          </a-form-item>
        </a-col>
      </a-row>
      <table class="alias-table">
        <thead>
          <tr>
            <th>{{ t('routing.model') }}</th>
            <th>{{ t('routing.multiplier') }}</th>
            <th>{{ t('routing.form_alias_priority') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(tg, ti) in block.targets" :key="ti">
            <td>
              <AddableSelect :value="tg.model" :options="modelOpts" :disabled="!lookupReady || disabled"
                :addition-label="addCustomLabel" @add="onAddModel"
                @update:value="(v) => updateAliasTarget(bi, ti, { model: v })" />
            </td>
            <td>
              <a-input type="number" :value="tg.weight" :disabled="disabled"
                @update:value="(v) => updateAliasTarget(bi, ti, { weight: toInt(v, 0) })" />
            </td>
            <td>
              <a-input type="number" :value="tg.priority" :disabled="disabled"
                @update:value="(v) => updateAliasTarget(bi, ti, { priority: toInt(v, 0) })" />
            </td>
            <td>
              <a-button type="button" :disabled="disabled || block.targets.length <= 1"
                @click="removeAliasTarget(bi, ti)">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </td>
          </tr>
        </tbody>
      </table>
      <a-button type="button" size="small" :disabled="disabled" @click="addAliasTarget(bi)">
        {{ t('routing.form_add_target_row') }}
      </a-button>
    </div>
    <a-button type="button" size="small" :disabled="disabled"
      @click="commit({ ...st, aliasBlocks: [...st.aliasBlocks, { logical: '', targets: [emptyTarget()] }] })">
      {{ t('routing.form_add_alias_block') }}
    </a-button>

    <h5 style="margin-top: 1.5rem">{{ t('routing.form_alias_group_overrides') }}</h5>
    <p class="routing-form-hint">{{ t('routing.form_alias_group_hint') }}</p>
    <div v-for="(gb, gi) in st.groupBlocks" :key="gi" class="alias-segment routing-group-override-segment">
      <h5>{{ t('routing.group') }}: {{ gb.group || '—' }}</h5>
      <a-form-item :label="t('routing.group')">
        <AddableSelect :value="gb.group" :options="groupOpts" :disabled="!lookupReady || disabled"
          :addition-label="addCustomLabel" @add="onAddGroup"
          @update:value="(v) => updateGroupBlock(gi, { group: v })" />
      </a-form-item>

      <h6>{{ t('routing.form_alias_chains') }}</h6>
      <table class="alias-table">
        <tbody>
          <tr v-for="(row, i) in gb.chainRows" :key="i">
            <td>
              <AddableSelect :value="row.from" :options="modelOpts" :disabled="!lookupReady || disabled"
                @add="onAddModel" @update:value="(v) => updateGroupChainRow(gi, i, { from: v })" />
            </td>
            <td>
              <AddableSelect :value="row.to" :options="modelOpts" :disabled="!lookupReady || disabled"
                @add="onAddModel" @update:value="(v) => updateGroupChainRow(gi, i, { to: v })" />
            </td>
            <td>
              <a-button type="button" :disabled="disabled || gb.chainRows.length <= 1"
                @click="removeGroupChainRow(gi, i)">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </td>
          </tr>
        </tbody>
      </table>
      <a-button type="button" size="small" :disabled="disabled" @click="addGroupChainRow(gi)">
        {{ t('routing.form_add_chain_row') }}
      </a-button>

      <h6>{{ t('routing.form_alias_aliases') }}</h6>
      <div v-for="(block, bi) in gb.aliasBlocks" :key="bi" class="alias-segment alias-segment--small">
        <a-form-item :label="t('routing.form_alias_logical_model')">
          <AddableSelect :value="block.logical" :options="modelOpts" :disabled="!lookupReady || disabled"
            @add="onAddModel" @update:value="(v) => updateGroupAliasBlock(gi, bi, { logical: v })" />
        </a-form-item>
        <table class="alias-table">
          <tbody>
            <tr v-for="(tg, ti) in block.targets" :key="ti">
              <td>
                <AddableSelect :value="tg.model" :options="modelOpts" :disabled="!lookupReady || disabled"
                  @add="onAddModel" @update:value="(v) => updateGroupAliasTarget(gi, bi, ti, { model: v })" />
              </td>
              <td>
                <a-input type="number" :value="tg.weight" :disabled="disabled"
                  @update:value="(v) => updateGroupAliasTarget(gi, bi, ti, { weight: toInt(v, 0) })" />
              </td>
              <td>
                <a-input type="number" :value="tg.priority" :disabled="disabled"
                  @update:value="(v) => updateGroupAliasTarget(gi, bi, ti, { priority: toInt(v, 0) })" />
              </td>
              <td>
                <a-button type="button" :disabled="disabled || block.targets.length <= 1"
                  @click="removeGroupAliasTarget(gi, bi, ti)">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </td>
            </tr>
          </tbody>
        </table>
        <a-button type="button" size="small" :disabled="disabled" @click="addGroupAliasTarget(gi, bi)">
          {{ t('routing.form_add_target_row') }}
        </a-button>
        <a-button type="button" size="small" danger :disabled="disabled || gb.aliasBlocks.length <= 1"
          @click="removeGroupAliasBlock(gi, bi)">
          {{ t('routing.form_remove_alias_block') }}
        </a-button>
      </div>
      <a-button type="button" size="small" :disabled="disabled" @click="addGroupAliasBlock(gi)">
        {{ t('routing.form_add_alias_block') }}
      </a-button>

      <a-button type="button" danger size="small" style="margin-top: 0.75rem; display: block"
        :disabled="disabled" @click="removeGroupBlock(gi)">
        {{ t('routing.form_remove_group_override') }}
      </a-button>
    </div>

    <a-button type="button" size="small" style="margin-top: 0.75rem" :disabled="disabled" @click="addGroupBlock">
      {{ t('routing.form_add_group_override') }}
    </a-button>

    <PolicyFormResetRow :disabled="disabled" @reset="resetDefaults" />
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { DeleteOutlined } from '@ant-design/icons-vue';
import {
  EMPTY_ALIAS_POLICY,
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

const addCustomLabel = computed(() => `${t('routing.combo_add_custom')} `);
const emptyTarget = () => ({ model: '', weight: 10, priority: 100 });

function fromAliasJSON(raw) {
  const o = safeParseObject(raw, EMPTY_ALIAS_POLICY);
  const chainRows = Object.entries(o.chains || {}).map(([from, to]) => ({ from, to }));
  const aliasBlocks = Object.entries(o.aliases || {}).map(([logical, targets]) => ({
    logical,
    targets: (Array.isArray(targets) ? targets : []).map((tg) => ({
      model: tg?.model ?? '',
      weight: toInt(tg?.weight, 0),
      priority: Number(tg?.priority) || 0,
    })),
  }));
  const groupBlocks = Object.entries(o.group_overrides || {}).map(([group, frag]) => {
    const f = frag && typeof frag === 'object' ? frag : {};
    const cr = Object.entries(f.chains || {}).map(([from, to]) => ({ from, to }));
    const ab = Object.entries(f.aliases || {}).map(([logical, targets]) => ({
      logical,
      targets: (Array.isArray(targets) ? targets : []).map((tg) => ({
        model: tg?.model ?? '',
        weight: toInt(tg?.weight, 0),
        priority: Number(tg?.priority) || 0,
      })),
    }));
    return {
      group,
      chainRows: cr.length ? cr : [{ from: '', to: '' }],
      aliasBlocks: ab.length ? ab : [{ logical: '', targets: [{ model: '', weight: 10, priority: 100 }] }],
    };
  });
  return {
    max_chain_steps: toInt(o.max_chain_steps, 32),
    chainRows: chainRows.length ? chainRows : [{ from: '', to: '' }],
    aliasBlocks: aliasBlocks.length
      ? aliasBlocks
      : [{ logical: '', targets: [{ model: '', weight: 10, priority: 100 }] }],
    groupBlocks,
  };
}

function toAliasJSON(state) {
  const chains = {};
  for (const r of state.chainRows) {
    const a = String(r.from || '').trim();
    const b = String(r.to || '').trim();
    if (a && b) chains[a] = b;
  }
  const aliases = {};
  for (const b of state.aliasBlocks) {
    const k = String(b.logical || '').trim();
    if (!k) continue;
    const arr = (b.targets || [])
      .filter((tg) => String(tg.model || '').trim())
      .map((tg) => ({
        model: String(tg.model).trim(),
        weight: toInt(tg.weight, 0),
        priority: Number(tg.priority) || 0,
      }));
    if (arr.length) aliases[k] = arr;
  }
  const group_overrides = {};
  for (const g of state.groupBlocks) {
    const gn = String(g.group || '').trim();
    if (!gn) continue;
    const gc = {};
    for (const r of g.chainRows || []) {
      const a = String(r.from || '').trim();
      const b = String(r.to || '').trim();
      if (a && b) gc[a] = b;
    }
    const ga = {};
    for (const b of g.aliasBlocks || []) {
      const k = String(b.logical || '').trim();
      if (!k) continue;
      const arr = (b.targets || [])
        .filter((tg) => String(tg.model || '').trim())
        .map((tg) => ({
          model: String(tg.model).trim(),
          weight: toInt(tg.weight, 0),
          priority: Number(tg.priority) || 0,
        }));
      if (arr.length) ga[k] = arr;
    }
    const frag = {};
    if (Object.keys(gc).length) frag.chains = gc;
    if (Object.keys(ga).length) frag.aliases = ga;
    if (Object.keys(frag).length) group_overrides[gn] = frag;
  }
  return stringifyPolicy({
    max_chain_steps: toInt(state.max_chain_steps, 32),
    chains,
    aliases,
    defaults: {},
    group_overrides,
  });
}

const st = ref(fromAliasJSON(props.value));

watch(
  () => props.value,
  (v) => {
    st.value = fromAliasJSON(v);
  }
);

function commit(next) {
  st.value = next;
  emit('change', toAliasJSON(next));
}

function onAddModel(v) {
  emit('add-model', v);
}
function onAddGroup(v) {
  emit('add-group', v);
}

// chains
function updateChainRow(i, patch) {
  const next = [...st.value.chainRows];
  next[i] = { ...next[i], ...patch };
  commit({ ...st.value, chainRows: next });
}
function removeChainRow(i) {
  const next = st.value.chainRows.filter((_, j) => j !== i);
  commit({ ...st.value, chainRows: next.length ? next : [{ from: '', to: '' }] });
}

// alias blocks
function updateAliasBlock(bi, patch) {
  const blocks = [...st.value.aliasBlocks];
  blocks[bi] = { ...blocks[bi], ...patch };
  commit({ ...st.value, aliasBlocks: blocks });
}
function removeAliasBlock(bi) {
  const blocks = st.value.aliasBlocks.filter((_, j) => j !== bi);
  commit({
    ...st.value,
    aliasBlocks: blocks.length ? blocks : [{ logical: '', targets: [emptyTarget()] }],
  });
}
function updateAliasTarget(bi, ti, patch) {
  const blocks = [...st.value.aliasBlocks];
  const tgs = [...blocks[bi].targets];
  tgs[ti] = { ...tgs[ti], ...patch };
  blocks[bi] = { ...blocks[bi], targets: tgs };
  commit({ ...st.value, aliasBlocks: blocks });
}
function removeAliasTarget(bi, ti) {
  const blocks = [...st.value.aliasBlocks];
  const tgs = blocks[bi].targets.filter((_, j) => j !== ti);
  blocks[bi] = { ...blocks[bi], targets: tgs.length ? tgs : [emptyTarget()] };
  commit({ ...st.value, aliasBlocks: blocks });
}
function addAliasTarget(bi) {
  const blocks = [...st.value.aliasBlocks];
  blocks[bi] = { ...blocks[bi], targets: [...blocks[bi].targets, emptyTarget()] };
  commit({ ...st.value, aliasBlocks: blocks });
}

// group blocks
function updateGroupBlock(gi, patch) {
  const gbs = [...st.value.groupBlocks];
  gbs[gi] = { ...gbs[gi], ...patch };
  commit({ ...st.value, groupBlocks: gbs });
}
function removeGroupBlock(gi) {
  commit({ ...st.value, groupBlocks: st.value.groupBlocks.filter((_, j) => j !== gi) });
}
function addGroupBlock() {
  commit({
    ...st.value,
    groupBlocks: [
      ...st.value.groupBlocks,
      {
        group: '',
        chainRows: [{ from: '', to: '' }],
        aliasBlocks: [{ logical: '', targets: [emptyTarget()] }],
      },
    ],
  });
}

// group chain rows
function updateGroupChainRow(gi, i, patch) {
  const gbs = [...st.value.groupBlocks];
  const rows = [...gbs[gi].chainRows];
  rows[i] = { ...rows[i], ...patch };
  gbs[gi] = { ...gbs[gi], chainRows: rows };
  commit({ ...st.value, groupBlocks: gbs });
}
function removeGroupChainRow(gi, i) {
  const gbs = [...st.value.groupBlocks];
  const rows = gbs[gi].chainRows.filter((_, j) => j !== i);
  gbs[gi] = { ...gbs[gi], chainRows: rows.length ? rows : [{ from: '', to: '' }] };
  commit({ ...st.value, groupBlocks: gbs });
}
function addGroupChainRow(gi) {
  const gbs = [...st.value.groupBlocks];
  gbs[gi] = { ...gbs[gi], chainRows: [...gbs[gi].chainRows, { from: '', to: '' }] };
  commit({ ...st.value, groupBlocks: gbs });
}

// group alias blocks
function updateGroupAliasBlock(gi, bi, patch) {
  const gbs = [...st.value.groupBlocks];
  const ab = [...gbs[gi].aliasBlocks];
  ab[bi] = { ...ab[bi], ...patch };
  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
  commit({ ...st.value, groupBlocks: gbs });
}
function removeGroupAliasBlock(gi, bi) {
  const gbs = [...st.value.groupBlocks];
  const ab = gbs[gi].aliasBlocks.filter((_, j) => j !== bi);
  gbs[gi] = {
    ...gbs[gi],
    aliasBlocks: ab.length ? ab : [{ logical: '', targets: [emptyTarget()] }],
  };
  commit({ ...st.value, groupBlocks: gbs });
}
function addGroupAliasBlock(gi) {
  const gbs = [...st.value.groupBlocks];
  gbs[gi] = {
    ...gbs[gi],
    aliasBlocks: [...gbs[gi].aliasBlocks, { logical: '', targets: [emptyTarget()] }],
  };
  commit({ ...st.value, groupBlocks: gbs });
}
function updateGroupAliasTarget(gi, bi, ti, patch) {
  const gbs = [...st.value.groupBlocks];
  const ab = [...gbs[gi].aliasBlocks];
  const tgs = [...ab[bi].targets];
  tgs[ti] = { ...tgs[ti], ...patch };
  ab[bi] = { ...ab[bi], targets: tgs };
  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
  commit({ ...st.value, groupBlocks: gbs });
}
function removeGroupAliasTarget(gi, bi, ti) {
  const gbs = [...st.value.groupBlocks];
  const ab = [...gbs[gi].aliasBlocks];
  const tgs = ab[bi].targets.filter((_, j) => j !== ti);
  ab[bi] = { ...ab[bi], targets: tgs.length ? tgs : [emptyTarget()] };
  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
  commit({ ...st.value, groupBlocks: gbs });
}
function addGroupAliasTarget(gi, bi) {
  const gbs = [...st.value.groupBlocks];
  const ab = [...gbs[gi].aliasBlocks];
  ab[bi] = { ...ab[bi], targets: [...ab[bi].targets, emptyTarget()] };
  gbs[gi] = { ...gbs[gi], aliasBlocks: ab };
  commit({ ...st.value, groupBlocks: gbs });
}

function resetDefaults() {
  commit(fromAliasJSON(stringifyPolicy(EMPTY_ALIAS_POLICY)));
}
</script>

<style scoped>
.alias-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 0.5rem;
}
.alias-table th,
.alias-table td {
  border: 1px solid rgba(34, 36, 38, 0.1);
  padding: 4px 6px;
  text-align: left;
}
.alias-segment {
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 4px;
  padding: 0.75rem;
  margin-bottom: 1rem;
}
.alias-segment--small {
  padding: 0.5rem;
}
.routing-form-hint {
  opacity: 0.75;
  margin-bottom: 0.5rem;
}
</style>
