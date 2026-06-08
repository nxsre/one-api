<template>
  <div class="dashboard-container routing-page">
    <a-card class="chart-card">
      <div class="header" style="font-size: 1.2em; font-weight: 600; margin-bottom: 12px;">
        {{ t('routing.title') }}
      </div>
      <a-tabs>
        <!-- Policies -->
        <a-tab-pane key="policies" :tab="t('routing.tab_policies')">
          <a-alert type="info" :message="t('routing.policy_hint')" style="margin-bottom: 12px" />

          <div class="routing-policy-card" style="margin-bottom: 1rem">
            <a-form layout="vertical">
              <h4 class="routing-policy-card__title">{{ t('routing.protocol_bridge_title') }}</h4>
              <p style="margin-bottom: 0.75rem; opacity: 0.85">{{ t('routing.protocol_bridge_hint') }}</p>
              <a-checkbox
                :checked="policies.RelayProtocolBridgeEnabled === 'true'"
                @change="(e) => setPolicy('RelayProtocolBridgeEnabled', e.target.checked ? 'true' : 'false')"
              >
                {{ t('routing.protocol_bridge_label') }}
              </a-checkbox>
            </a-form>
            <div class="routing-policy-save-footer" style="margin-top: 0.75rem">
              <span v-if="!isRoot()" class="routing-policy-save-footer__hint">{{ t('routing.need_root_save') }}</span>
              <a-button type="primary" size="small" :disabled="!isRoot()" @click="saveOption('RelayProtocolBridgeEnabled')">
                <template #icon><SaveOutlined /></template>
                {{ t('routing.save') }}
              </a-button>
            </div>
          </div>

          <div v-for="k in POLICY_KEYS_BASIC" :key="k" class="routing-policy-card">
            <a-form layout="vertical">
              <h4 class="routing-policy-card__title">{{ k }}</h4>
              <a-alert
                v-if="k === 'RelayRetryPolicy'"
                type="info"
                :message="t('routing.policy_example_note_relay')"
                style="margin-bottom: 0.75rem"
              />
              <details style="margin-bottom: 0.85rem">
                <summary style="cursor: pointer; user-select: none; font-weight: 500">
                  {{ t('routing.policy_example_toggle') }}
                </summary>
                <pre class="routing-policy-example-pre">{{ ROUTING_POLICY_JSON_SAMPLES[k] }}</pre>
              </details>
              <RoutingPolicyForm
                v-if="k === 'RoutingPolicy'"
                :value="policies[k] ?? '{}'"
                :disabled="false"
                @change="(v) => setPolicy(k, v)"
              />
              <RelayRetryPolicyForm
                v-else-if="k === 'RelayRetryPolicy'"
                :value="policies[k] ?? '{}'"
                :disabled="false"
                @change="(v) => setPolicy(k, v)"
              />
              <ModelRateLimitPolicyForm
                v-else-if="k === 'ModelRateLimitPolicy'"
                :value="policies[k] ?? '{}'"
                :disabled="false"
                :model-opts="modelOpts"
                :group-opts="groupOpts"
                :lookup-ready="lookupReady"
                @change="(v) => setPolicy(k, v)"
                @add-model="addModelOpt"
                @add-group="addGroupOpt"
              />
            </a-form>
            <div class="routing-policy-save-footer">
              <span v-if="!isRoot()" class="routing-policy-save-footer__hint">{{ t('routing.need_root_save') }}</span>
              <a-button type="primary" size="small" :disabled="!isRoot()" @click="saveOption(k)">
                <template #icon><SaveOutlined /></template>
                {{ t('routing.save') }}
              </a-button>
            </div>
          </div>

          <div class="routing-policy-card">
            <a-form layout="vertical">
              <h4 class="routing-policy-card__title">ModelAliasPolicy</h4>
              <a-alert type="info" :message="t('routing.alias_hint')" style="margin-bottom: 0.75rem" />
              <details style="margin-bottom: 0.85rem">
                <summary style="cursor: pointer; user-select: none; font-weight: 500">
                  {{ t('routing.policy_example_toggle') }}
                </summary>
                <pre class="routing-policy-example-pre">{{ ROUTING_POLICY_JSON_SAMPLES.ModelAliasPolicy }}</pre>
              </details>
              <ModelAliasPolicyForm
                :value="policies.ModelAliasPolicy ?? '{}'"
                :disabled="false"
                :model-opts="modelOpts"
                :group-opts="groupOpts"
                :lookup-ready="lookupReady"
                @change="(v) => setPolicy('ModelAliasPolicy', v)"
                @add-model="addModelOpt"
                @add-group="addGroupOpt"
              />
            </a-form>
            <div class="routing-policy-save-footer routing-policy-save-footer--split">
              <span v-if="!isRoot()" class="routing-policy-save-footer__hint">{{ t('routing.need_root_save') }}</span>
              <div class="routing-policy-save-footer__buttons">
                <a-button type="primary" size="small" :disabled="!isRoot()" @click="saveOption('ModelAliasPolicy')">
                  <template #icon><SaveOutlined /></template>
                  {{ t('routing.save') }}
                </a-button>
                <a-button size="small" @click="validateAlias">
                  <template #icon><CheckCircleOutlined /></template>
                  {{ t('routing.validate_alias') }}
                </a-button>
              </div>
            </div>
          </div>
        </a-tab-pane>

        <!-- Observe -->
        <a-tab-pane key="observe" :tab="t('routing.tab_observe')">
          <h4>{{ t('routing.preview_title') }}</h4>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('routing.group')">
                  <AddableSelect :value="previewGroup" :options="groupOpts" :disabled="!lookupReady"
                    :placeholder="t('routing.combo_search_placeholder')" :addition-label="addCustomLabel"
                    @add="addGroupOpt" @update:value="(v) => (previewGroup = v ?? '')" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('routing.model')">
                  <AddableSelect :value="previewModel" :options="modelOpts" :disabled="!lookupReady"
                    :placeholder="t('routing.combo_search_placeholder')" :addition-label="addCustomLabel"
                    @add="addModelOpt" @update:value="(v) => (previewModel = v ?? '')" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-button @click="loadPreview">{{ t('routing.refresh') }}</a-button>
          </a-form>

          <template v-if="previewMeta">
            <a-alert type="info" style="margin: 12px 0"
              :message="`${t('routing.candidate_count')}: ${previewMeta.candidate_count}`" />
            <a-table
              :columns="previewColumns"
              :data-source="previewMeta.channels || []"
              :pagination="false"
              row-key="id"
              size="small"
              bordered
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'adaptive_enabled'">{{ record.adaptive_enabled ? 'Y' : 'N' }}</template>
                <template v-else-if="column.key === 'circuit_open'">{{ record.circuit_open ? 'OPEN' : '-' }}</template>
              </template>
            </a-table>
          </template>

          <h4 style="margin-top: 1.5rem; border-bottom: 1px solid rgba(34,36,38,0.1); padding-bottom: 6px;">
            {{ t('routing.manual_weight') }}
          </h4>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('routing.channel_id')">
                  <AddableSelect :value="mwCh" :options="channelOpts" :disabled="!lookupReady"
                    :placeholder="t('routing.combo_channel_placeholder')" :addition-label="addCustomLabel"
                    @add="addChannelOpt" @update:value="(v) => (mwCh = v ?? '')" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('routing.multiplier')">
                  <a-input v-model:value="mwMul" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-button @click="postManualWeight">{{ t('routing.apply') }}</a-button>
          </a-form>

          <h4 style="margin-top: 1.5rem; border-bottom: 1px solid rgba(34,36,38,0.1); padding-bottom: 6px;">
            {{ t('routing.reset_auto') }}
          </h4>
          <a-form layout="vertical">
            <a-form-item :label="t('routing.channel_id')">
              <AddableSelect :value="rwCh" :options="channelOpts" :disabled="!lookupReady"
                :placeholder="t('routing.combo_channel_placeholder')" :addition-label="addCustomLabel"
                @add="addChannelOpt" @update:value="(v) => (rwCh = v ?? '')" />
            </a-form-item>
            <a-button @click="resetAuto">{{ t('routing.reset_auto_btn') }}</a-button>
          </a-form>
        </a-tab-pane>

        <!-- Metrics -->
        <a-tab-pane key="metrics" :tab="t('routing.tab_metrics')">
          <div class="routing-tab-toolbar" style="display:flex;align-items:flex-end;gap:12px;margin-bottom:12px;">
            <a-form layout="vertical" style="margin:0">
              <a-form-item :label="t('routing.metrics_day')" style="margin:0">
                <a-input v-model:value="metricsDay" style="width: 220px" />
              </a-form-item>
            </a-form>
            <a-button @click="loadMetrics">
              <template #icon><ReloadOutlined /></template>
              {{ t('routing.refresh') }}
            </a-button>
          </div>
          <a-table :columns="metricsColumns" :data-source="metrics" :pagination="false"
            :row-key="(r, i) => i" size="small" bordered />
        </a-tab-pane>

        <!-- Fuse -->
        <a-tab-pane key="fuse" :tab="t('routing.tab_fuse')">
          <div class="routing-tab-toolbar" style="margin-bottom:12px;">
            <a-button @click="loadFuse">
              <template #icon><ReloadOutlined /></template>
              {{ t('routing.refresh') }}
            </a-button>
          </div>
          <a-table :columns="fuseColumns" :data-source="fuse" :pagination="false"
            :row-key="(r, i) => i" size="small" bordered>
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'time'">{{ formatMs(record.unix_ms) }}</template>
            </template>
          </a-table>
        </a-tab-pane>

        <!-- Chart -->
        <a-tab-pane key="chart" :tab="t('routing.tab_chart')">
          <div class="routing-chart-toolbar">
            <a-form layout="vertical">
              <a-row :gutter="16">
                <a-col :span="8">
                  <a-form-item :label="t('routing.channel_id')">
                    <AddableSelect :value="tsCh" :options="channelOpts" :disabled="!lookupReady"
                      :placeholder="t('routing.combo_channel_placeholder')" :addition-label="addCustomLabel"
                      @add="addChannelOpt" @update:value="(v) => (tsCh = v ?? '')" />
                  </a-form-item>
                </a-col>
                <a-col :span="8">
                  <a-form-item :label="t('routing.model')">
                    <AddableSelect :value="tsModel" :options="chartModelOpts" :disabled="!lookupReady"
                      :placeholder="t('routing.combo_search_placeholder')" :addition-label="addCustomLabel"
                      @add="addModelOpt" @update:value="(v) => (tsModel = v ?? '')" />
                  </a-form-item>
                </a-col>
                <a-col :span="8">
                  <a-form-item :label="t('routing.hours')">
                    <a-input type="number" :min="1" :max="168" :value="tsHours"
                      @update:value="(v) => (tsHours = parseInt(v, 10) || 24)" />
                  </a-form-item>
                </a-col>
              </a-row>
            </a-form>
            <div class="routing-chart-toolbar__actions" style="display:flex;align-items:center;gap:12px;">
              <a-checkbox :checked="tsShowAllModels" :disabled="!String(tsCh).trim()"
                @change="(e) => (tsShowAllModels = !!e.target.checked)">
                {{ t('routing.chart_show_all_models') }}
              </a-checkbox>
              <a-button type="primary" :loading="tsLoading" :disabled="tsLoading" @click="loadTs">
                <template #icon><ReloadOutlined /></template>
                {{ t('routing.refresh') }}
              </a-button>
            </div>
            <p class="routing-chart-toolbar__hint" style="opacity:0.7;margin-top:8px;">{{ t('routing.chart_models_hint') }}</p>
          </div>
          <div class="routing-chart-panel">
            <div v-if="tsData.length === 0" class="routing-chart-empty"
              style="text-align:center;padding:48px 0;opacity:0.6;">
              <LineChartOutlined style="font-size: 48px" />
              <p>{{ t('routing.chart_empty') }}</p>
            </div>
            <VChart v-else :option="chartOption" autoresize style="height: 360px; width: 100%" />
          </div>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import {
  SaveOutlined,
  CheckCircleOutlined,
  ReloadOutlined,
  LineChartOutlined,
} from '@ant-design/icons-vue';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
} from 'echarts/components';
import VChart from 'vue-echarts';
import { API, isAdmin, isRoot, showError, showSuccess } from '@/helpers';
import { ROUTING_POLICY_JSON_SAMPLES } from './policyExamples';
import RelayRetryPolicyForm from './forms/RelayRetryPolicyForm.vue';
import RoutingPolicyForm from './forms/RoutingPolicyForm.vue';
import ModelAliasPolicyForm from './forms/ModelAliasPolicyForm.vue';
import ModelRateLimitPolicyForm from './forms/ModelRateLimitPolicyForm.vue';
import AddableSelect from './forms/AddableSelect.vue';
import {
  fetchRoutingChannels,
  fetchRoutingGroups,
  fetchRoutingModelCatalog,
  parseChannelModelsField,
} from './routingLookupLoaders';

dayjs.extend(utc);
use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent]);

const POLICY_KEYS_BASIC = ['RoutingPolicy', 'RelayRetryPolicy', 'ModelRateLimitPolicy'];

const { t } = useI18n();
const router = useRouter();

const policies = reactive({});
const previewGroup = ref('default');
const previewModel = ref('');
const previewMeta = ref(null);
const metricsDay = ref(dayjs().utc().format('YYYYMMDD'));
const metrics = ref([]);
const fuse = ref([]);
const tsCh = ref('');
const tsModel = ref('');
const tsHours = ref(24);
const tsData = ref([]);
const mwCh = ref('');
const mwMul = ref('1');
const rwCh = ref('');
const groupOpts = ref([]);
const fullModelCatalog = ref([]);
const modelOpts = ref([]);
const routingChannels = ref([]);
const channelOpts = ref([]);
const lookupReady = ref(false);
const tsShowAllModels = ref(false);
const tsResolvedModels = ref([]);
const tsLoading = ref(false);

const addCustomLabel = computed(() => `${t('routing.combo_add_custom')} `);

let tsCancelToken = 0;

function setPolicy(key, value) {
  policies[key] = value;
}

const previewColumns = computed(() => [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: t('routing.col_name'), dataIndex: 'name', key: 'name' },
  { title: t('routing.col_provider'), dataIndex: 'provider', key: 'provider' },
  { title: 'W', dataIndex: 'base_weight', key: 'base_weight' },
  { title: '×m', dataIndex: 'manual_multiplier', key: 'manual_multiplier' },
  { title: '×a', dataIndex: 'auto_multiplier', key: 'auto_multiplier' },
  { title: t('routing.adaptive'), key: 'adaptive_enabled' },
  { title: 'Eff', dataIndex: 'effective_weight', key: 'effective_weight' },
  { title: 'Pri', dataIndex: 'priority', key: 'priority' },
  { title: 'Fuse', key: 'circuit_open' },
]);

const metricsColumns = [
  { title: 'Key', dataIndex: 'redis_key', key: 'redis_key' },
  { title: 'ok', dataIndex: 'ok', key: 'ok' },
  { title: 'fail', dataIndex: 'fail', key: 'fail' },
  { title: 'lat_n', dataIndex: 'lat_n', key: 'lat_n' },
];

const fuseColumns = [
  { title: 'time', key: 'time' },
  { title: 'ch', dataIndex: 'channel_id', key: 'channel_id' },
  { title: 'state', dataIndex: 'state', key: 'state' },
  { title: 'reason', dataIndex: 'reason', key: 'reason' },
];

function formatMs(ms) {
  return dayjs(ms).format('YYYY-MM-DD HH:mm:ss');
}

const chartModelOpts = computed(() => {
  const toOpts = (ids) => ids.map((m) => ({ key: m, text: m, value: m }));
  const full = toOpts(fullModelCatalog.value);
  const idStr = String(tsCh.value).trim();
  if (!idStr || tsShowAllModels.value || tsResolvedModels.value.length === 0) {
    return full;
  }
  return toOpts(tsResolvedModels.value);
});

const chartOption = computed(() => ({
  grid: { top: 24, right: 56, bottom: 24, left: 56 },
  tooltip: {
    trigger: 'axis',
    formatter: (params) => {
      const lines = params.map((p) => {
        if (p.seriesName === 'err_ratio') {
          return `${t('routing.chart_legend_err')}: ${(Number(p.value) * 100).toFixed(2)}%`;
        }
        return `${t('routing.chart_legend_latency')}: ${p.value} ms`;
      });
      return `${params[0]?.axisValue || ''}<br/>${lines.join('<br/>')}`;
    },
  },
  xAxis: {
    type: 'category',
    data: tsData.value.map((r) => r.time),
    axisLabel: { fontSize: 11 },
  },
  yAxis: [
    {
      type: 'value',
      name: t('routing.chart_axis_latency'),
      nameTextStyle: { fontSize: 11 },
      axisLabel: { fontSize: 11 },
    },
    {
      type: 'value',
      min: 0,
      max: 1,
      position: 'right',
      name: t('routing.chart_axis_error'),
      nameTextStyle: { fontSize: 11 },
      axisLabel: { fontSize: 11, formatter: (v) => `${Math.round(v * 100)}%` },
    },
  ],
  series: [
    {
      name: 'avg_latency_ms',
      type: 'line',
      yAxisIndex: 0,
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#5b6ee1' },
      itemStyle: { color: '#5b6ee1' },
      data: tsData.value.map((r) => r.avg_latency_ms),
    },
    {
      name: 'err_ratio',
      type: 'line',
      yAxisIndex: 1,
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#21ba45' },
      itemStyle: { color: '#21ba45' },
      data: tsData.value.map((r) => r.err_ratio),
    },
  ],
}));

async function loadBundle() {
  try {
    const res = await API.get('/api/routing/policy-bundle');
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      return;
    }
    const data = res.data.data || {};
    Object.keys(policies).forEach((k) => delete policies[k]);
    Object.assign(policies, {
      ...data,
      RelayProtocolBridgeEnabled: data.RelayProtocolBridgeEnabled === 'true' ? 'true' : 'false',
    });
  } catch (e) {
    showError(e.message);
  }
}

function addGroupOpt(raw) {
  const v = String(raw ?? '').trim();
  if (!v) return;
  if (!groupOpts.value.some((o) => o.value === v)) {
    groupOpts.value = [...groupOpts.value, { key: v, text: v, value: v }];
  }
}

function addModelOpt(raw) {
  const v = String(raw ?? '').trim();
  if (!v) return;
  if (!fullModelCatalog.value.some((x) => x === v)) {
    fullModelCatalog.value = [...fullModelCatalog.value, v].sort((a, b) => a.localeCompare(b));
  }
  if (!modelOpts.value.some((o) => o.value === v)) {
    modelOpts.value = [...modelOpts.value, { key: v, text: v, value: v }];
  }
}

function addChannelOpt(raw) {
  const v = String(raw ?? '').trim();
  if (!v) return;
  if (!channelOpts.value.some((o) => o.value === v)) {
    channelOpts.value = [...channelOpts.value, { key: v, text: v, value: v }];
  }
}

async function saveOption(key) {
  if (!isRoot()) {
    showError(t('routing.need_root_save'));
    return;
  }
  try {
    let raw;
    if (key === 'RelayProtocolBridgeEnabled') {
      raw = policies[key] === 'true' ? 'true' : 'false';
    } else {
      raw = String(policies[key] ?? '').trim() || '{}';
    }
    const res = await API.put('/api/option', { key, value: raw });
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('routing.saved'));
  } catch (e) {
    showError(e.message);
  }
}

async function loadPreview() {
  try {
    const res = await API.get(
      `/api/routing/channel-preview?group=${encodeURIComponent(previewGroup.value)}&model=${encodeURIComponent(previewModel.value)}`
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'preview failed');
      return;
    }
    previewMeta.value = res.data.data;
  } catch (e) {
    showError(e.message);
  }
}

async function loadMetrics() {
  try {
    const res = await API.get(`/api/routing/metrics-day?day=${encodeURIComponent(metricsDay.value)}`);
    if (!res.data?.success) {
      showError(res.data?.message || 'metrics failed');
      return;
    }
    metrics.value = res.data.data || [];
  } catch (e) {
    showError(e.message);
  }
}

async function loadFuse() {
  try {
    const res = await API.get('/api/routing/fuse-events?limit=80');
    if (!res.data?.success) {
      showError(res.data?.message || 'fuse failed');
      return;
    }
    fuse.value = res.data.data || [];
  } catch (e) {
    showError(e.message);
  }
}

async function loadTs() {
  tsLoading.value = true;
  try {
    const hours = Math.min(168, Math.max(1, parseInt(String(tsHours.value), 10) || 24));
    const res = await API.get(
      `/api/routing/timeseries?channel_id=${encodeURIComponent(tsCh.value)}&model=${encodeURIComponent(tsModel.value)}&hours=${encodeURIComponent(String(hours))}`
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'timeseries failed');
      return;
    }
    tsData.value = (res.data.data || []).map((r) => ({
      ...r,
      time: dayjs.unix(r.minute_unix).utc().format('MM-DD HH:mm'),
    }));
  } catch (e) {
    showError(e.message);
  } finally {
    tsLoading.value = false;
  }
}

async function validateAlias() {
  try {
    const res = await API.post('/api/routing/validate-alias-policy', {
      model_alias_policy_json: policies.ModelAliasPolicy ?? '{}',
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'invalid');
      return;
    }
    showSuccess(t('routing.alias_ok'));
  } catch (e) {
    showError(e.message);
  }
}

async function postManualWeight() {
  try {
    const res = await API.post('/api/routing/manual-weight', {
      channel_id: parseInt(mwCh.value, 10),
      multiplier: parseFloat(mwMul.value),
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'failed');
      return;
    }
    showSuccess(t('routing.saved'));
  } catch (e) {
    showError(e.message);
  }
}

async function resetAuto() {
  try {
    const res = await API.post('/api/routing/reset-auto-weight', {
      channel_id: parseInt(rwCh.value, 10),
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'failed');
      return;
    }
    showSuccess(t('routing.saved'));
  } catch (e) {
    showError(e.message);
  }
}

watch([tsCh, routingChannels], () => {
  const idStr = String(tsCh.value).trim();
  if (!idStr) {
    tsShowAllModels.value = false;
    tsResolvedModels.value = [];
    return;
  }
  const fromList = routingChannels.value.find((c) => String(c.id) === idStr);
  if (fromList && fromList.models.length > 0) {
    tsResolvedModels.value = fromList.models;
    return;
  }
  const num = parseInt(idStr, 10);
  if (!num) {
    tsResolvedModels.value = [];
    return;
  }
  const token = ++tsCancelToken;
  API.get(`/api/channel/${num}`)
    .then((res) => {
      if (token !== tsCancelToken) return;
      if (res.data?.success && res.data.data) {
        tsResolvedModels.value = parseChannelModelsField(res.data.data.models);
      } else {
        tsResolvedModels.value = [];
      }
    })
    .catch(() => {
      if (token === tsCancelToken) tsResolvedModels.value = [];
    });
});

onMounted(async () => {
  if (!isAdmin()) {
    router.push('/');
    return;
  }
  loadBundle();
  try {
    const [groups, catalog, channelRows] = await Promise.all([
      fetchRoutingGroups(API),
      fetchRoutingModelCatalog(API),
      fetchRoutingChannels(API),
    ]);
    groupOpts.value = groups.map((g) => ({ key: g, text: g, value: g }));
    fullModelCatalog.value = catalog;
    modelOpts.value = catalog.map((m) => ({ key: m, text: m, value: m }));
    routingChannels.value = channelRows;
    channelOpts.value = channelRows.map((ch) => ({
      key: String(ch.id),
      value: String(ch.id),
      text: `#${ch.id} ${ch.name || ''}`.trim(),
    }));
  } catch (e) {
    showError(e.message);
  } finally {
    lookupReady.value = true;
  }
});

onBeforeUnmount(() => {
  tsCancelToken++;
});
</script>

<style scoped>
.routing-policy-card {
  border: 1px solid rgba(34, 36, 38, 0.12);
  border-radius: 6px;
  padding: 1rem;
  margin-bottom: 1rem;
}
.routing-policy-card__title {
  margin: 0 0 0.5rem;
}
.routing-policy-save-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}
.routing-policy-save-footer--split {
  justify-content: space-between;
}
.routing-policy-save-footer__buttons {
  display: flex;
  gap: 8px;
}
.routing-policy-save-footer__hint {
  opacity: 0.7;
  font-size: 12px;
}
.routing-policy-example-pre {
  margin-top: 0.5rem;
  padding: 0.75rem;
  background: rgba(34, 36, 38, 0.06);
  border-radius: 4px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.45;
  white-space: pre;
}
</style>
