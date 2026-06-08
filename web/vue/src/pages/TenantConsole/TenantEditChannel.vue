<template>
  <div class="dashboard-container channel-edit-page p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">
        {{ isEdit ? t('channel.edit.title_edit') : t('channel.edit.title_create') }}
      </h2>
      <a-spin :spinning="loading || channelTypesLoading">
        <a-form layout="vertical" autocomplete="off">
          <!-- 基础信息 -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title text-base font-medium mb-2">
              {{ t('channel.edit.section_basic') }}
            </h4>
            <a-form-item :label="t('channel.edit.type')" required>
              <a-select
                :value="inputs.type"
                show-search
                :options="channelTypeOptions"
                :field-names="{ label: 'text', value: 'value' }"
                option-filter-prop="text"
                :placeholder="channelTypesLoading ? t('channel.edit.types_loading') : undefined"
                @change="onTypeChange"
              />
            </a-form-item>
            <a-form-item :label="t('channel.edit.name')" required>
              <a-input
                v-model:value="inputs.name"
                :placeholder="t('channel.edit.name_placeholder')"
                autocomplete="off"
              />
            </a-form-item>
            <a-form-item :label="t('channel.edit.group')" required>
              <a-select
                v-model:value="inputs.groups"
                mode="tags"
                :options="groupOptions"
                :field-names="{ label: 'text', value: 'value' }"
                option-filter-prop="text"
                :placeholder="t('channel.edit.group_placeholder')"
              />
            </a-form-item>
            <ChannelTip :channel-type="inputs.type" />
          </div>

          <!-- 上游配置 -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title text-base font-medium mb-2">
              {{ t('channel.edit.section_upstream') }}
            </h4>
            <a-alert v-if="currentTypeDescription" type="info" :message="currentTypeDescription" class="mb-3" />
            <a-form-item v-if="inputs.type === 14 || inputs.type === 42" label="Gemini API 版本">
              <a-input v-model:value="config.api_version" placeholder="例如：v1" autocomplete="off" />
            </a-form-item>
            <div class="channel-edit-upstream-row">
              <a-form-item :label="t('channel.edit.base_url')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.base_url_all_hint') }}</p>
                <a-input v-model:value="inputs.base_url" autocomplete="off" />
              </a-form-item>
              <a-form-item v-if="!batch" :label="t('channel.edit.key')" required>
                <p class="channel-edit-field-hint">{{ type2secretPrompt(inputs.type) }}</p>
                <a-input
                  v-model:value="inputs.key"
                  :placeholder="type2secretPrompt(inputs.type)"
                  autocomplete="off"
                />
              </a-form-item>
            </div>
            <a-form-item v-if="batch" :label="t('channel.edit.key')">
              <p class="channel-edit-field-hint">{{ t('channel.edit.batch_placeholder') }}</p>
              <a-textarea v-model:value="inputs.key" :rows="10" autocomplete="off" />
            </a-form-item>
            <a-checkbox v-if="!isEdit" v-model:checked="batch">
              {{ t('channel.edit.batch') }}
            </a-checkbox>
          </div>

          <!-- 模型 -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title text-base font-medium mb-2">
              {{ t('channel.edit.section_models') }}
            </h4>
            <a-form-item :label="t('channel.edit.models')" required>
              <a-select
                :value="inputs.models"
                mode="multiple"
                show-search
                :options="modelOptions"
                :field-names="{ label: 'text', value: 'value' }"
                option-filter-prop="text"
                :placeholder="t('channel.edit.models_placeholder')"
                @change="onModelsChange"
              >
                <template #tagRender="{ value: tagValue, label, closable, onClose }">
                  <a-tag
                    :class="modelTagClass(tagValue)"
                    :title="modelTagTitle(tagValue)"
                    :closable="closable"
                    style="margin-right: 3px"
                    @close="onClose"
                    @click="copyModel(tagValue)"
                  >
                    {{ label }}
                  </a-tag>
                </template>
              </a-select>
            </a-form-item>

            <div
              v-if="modelTestSummary || ((testModelsBusy || testModelsProgress.running) && testModelsProgress.total > 0)"
              class="channel-edit-model-test-block"
            >
              <p v-if="modelTestSummary" class="channel-edit-model-test-summary">{{ modelTestSummary }}</p>
              <a-progress
                v-if="(testModelsBusy || testModelsProgress.running) && testModelsProgress.total > 0"
                class="channel-edit-model-test-progress"
                :percent="testProgressPercent"
                :status="testModelsProgress.running && !testModelsProgress.paused ? 'active' : 'normal'"
              />
            </div>

            <div class="channel-edit-models-toolbar flex flex-wrap gap-2 items-center mb-3">
              <a-button
                :loading="testModelsBusy"
                :disabled="testModelsBusy || testModelsProgress.running || fetchUpstreamBusy"
                @click="testAllModels"
              >
                {{ t('channel.edit.buttons.test_models') }}
              </a-button>
              <a-button
                :loading="testModelsControlBusy"
                :disabled="!testModelsProgress.running || testModelsProgress.paused || testModelsProgress.cancelReq"
                @click="controlRunningTestJob('pause')"
              >
                {{ t('channel.edit.buttons.pause_test_models') }}
              </a-button>
              <a-button
                :loading="testModelsControlBusy"
                :disabled="!testModelsProgress.running || !testModelsProgress.paused"
                @click="controlRunningTestJob('resume')"
              >
                {{ t('channel.edit.buttons.resume_test_models') }}
              </a-button>
              <a-button
                :loading="testModelsControlBusy"
                :disabled="!testModelsProgress.running || testModelsProgress.cancelReq"
                @click="controlRunningTestJob('cancel')"
              >
                {{ t('channel.edit.buttons.cancel_test_models') }}
              </a-button>
              <a-button :disabled="modelTestReportRows.length === 0" @click="modelTestReportOpen = true">
                {{ t('channel.edit.buttons.view_test_report') }}
              </a-button>
              <div class="channel-edit-model-test-concurrency flex items-center gap-1">
                <label class="channel-edit-model-test-concurrency__label">
                  {{ t('channel.edit.buttons.test_models_concurrency') }}
                </label>
                <a-input-number
                  v-model:value="testModelsConcurrency"
                  :min="1"
                  :max="10"
                  :step="1"
                  :disabled="testModelsBusy || testModelsProgress.running"
                  style="width: 80px"
                />
              </div>
              <a-button :loading="fetchUpstreamBusy" :disabled="fetchUpstreamBusy" @click="fetchUpstreamModelsList">
                {{ t('channel.edit.buttons.fetch_upstream_models') }}
              </a-button>
              <a-button @click="fillCatalogOpen = true">
                {{ t('channel.edit.buttons.fill_models') }}
              </a-button>
              <a-button :loading="fillAllBusy" :disabled="fillAllBusy" @click="fillAllCatalogModels">
                {{ t('channel.edit.buttons.fill_all') }}
              </a-button>
              <a-button
                :disabled="testModelsBusy || testModelsProgress.running || failedModelCount === 0"
                @click="clearInvalidModels"
              >
                {{ t('channel.edit.buttons.clear_failed') }}
              </a-button>
              <a-button @click="setModels([])">
                {{ t('channel.edit.buttons.clear') }}
              </a-button>
            </div>

            <a-form-item :label="t('channel.edit.custom_models_bulk')">
              <p class="channel-edit-field-hint">{{ t('channel.edit.buttons.custom_placeholder') }}</p>
              <a-textarea v-model:value="customModel" :rows="4" />
              <a-button class="mt-2" @click="addCustomModel">
                {{ t('channel.edit.buttons.add_custom') }}
              </a-button>
            </a-form-item>

            <!-- 模型重定向编辑器 -->
            <a-form-item :label="t('channel.edit.model_mapping')">
              <div class="channel-model-mapping-editor">
                <p class="channel-edit-field-hint" style="white-space: pre-line">{{ mappingHint }}</p>
                <template v-if="!mappingJsonMode">
                  <a-table
                    :columns="mappingColumns"
                    :data-source="mappingRowsData"
                    :pagination="false"
                    size="small"
                    row-key="_idx"
                    class="channel-model-mapping-table"
                  >
                    <template #bodyCell="{ column, record }">
                      <template v-if="column.key === 'from'">
                        <a-input
                          :value="record.from"
                          :placeholder="t('channel.edit.model_mapping_ph_request')"
                          @update:value="(v) => updateMappingRow(record._idx, 'from', v)"
                        />
                      </template>
                      <template v-else-if="column.key === 'to'">
                        <a-select
                          v-if="mappingUpstreamOptions.length > 0"
                          :value="record.to || undefined"
                          show-search
                          allow-clear
                          :options="mappingToOptions(record.to)"
                          :field-names="{ label: 'text', value: 'value' }"
                          option-filter-prop="text"
                          :placeholder="t('channel.edit.model_mapping_ph_upstream')"
                          style="width: 100%"
                          @change="(v) => updateMappingRow(record._idx, 'to', v || '')"
                        />
                        <a-input
                          v-else
                          :value="record.to"
                          :placeholder="t('channel.edit.model_mapping_ph_upstream')"
                          @update:value="(v) => updateMappingRow(record._idx, 'to', v)"
                        />
                      </template>
                      <template v-else-if="column.key === 'op'">
                        <a-button
                          size="small"
                          :disabled="mappingRows.length <= 1"
                          :title="t('channel.edit.model_mapping_remove')"
                          @click="removeMappingRow(record._idx)"
                        >
                          <template #icon><DeleteOutlined /></template>
                        </a-button>
                      </template>
                    </template>
                  </a-table>
                  <div class="channel-model-mapping-editor__toolbar mt-2 flex gap-2">
                    <a-button size="small" @click="addMappingRow">
                      <template #icon><PlusOutlined /></template>
                      {{ t('channel.edit.model_mapping_add') }}
                    </a-button>
                    <a-button size="small" @click="enterMappingJsonMode">
                      {{ t('channel.edit.model_mapping_json_toggle') }}
                    </a-button>
                  </div>
                </template>
                <div v-else class="channel-model-mapping-editor__json">
                  <a-textarea
                    v-model:value="mappingJsonRaw"
                    :rows="12"
                    :placeholder="t('channel.edit.model_mapping_json_placeholder')"
                  />
                  <div class="channel-model-mapping-editor__toolbar mt-2 flex gap-2">
                    <a-button size="small" @click="applyMappingJsonPaste">
                      {{ t('channel.edit.model_mapping_json_apply') }}
                    </a-button>
                    <a-button size="small" @click="cancelMappingJsonMode">
                      {{ t('channel.edit.model_mapping_json_cancel') }}
                    </a-button>
                  </div>
                </div>
              </div>
            </a-form-item>

            <a-form-item :label="t('channel.edit.system_prompt')">
              <p class="channel-edit-field-hint">{{ t('channel.edit.system_prompt_placeholder') }}</p>
              <a-textarea v-model:value="inputs.system_prompt" :rows="10" />
            </a-form-item>

            <!-- 扩展字段 -->
            <div class="channel-edit-section channel-edit-section--extended">
              <h4 class="channel-edit-section__title text-base font-medium mb-2">
                {{ t('channel.edit.newapi_section') }}
              </h4>
              <a-form-item :label="t('channel.edit.openai_organization')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.openai_organization_hint') }}</p>
                <a-input
                  v-model:value="inputs.openai_organization"
                  :placeholder="t('channel.edit.openai_organization_placeholder')"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.test_model')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.test_model_hint') }}</p>
                <a-input
                  v-model:value="inputs.test_model"
                  :placeholder="t('channel.edit.test_model_placeholder')"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item>
                <p class="channel-edit-field-hint">{{ t('channel.edit.auto_ban_hint') }}</p>
                <a-switch :checked="inputs.auto_ban !== 0" @change="toggleAutoBan" />
                <span class="ml-2">{{ t('channel.edit.auto_ban') }}</span>
              </a-form-item>
              <a-form-item :label="t('channel.edit.remark')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.remark_hint') }}</p>
                <a-input
                  v-model:value="inputs.remark"
                  :placeholder="t('channel.edit.remark_placeholder')"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.tag')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.tag_hint') }}</p>
                <a-input
                  v-model:value="inputs.tag"
                  :placeholder="t('channel.edit.tag_placeholder')"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.status_code_mapping')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.status_code_mapping_hint') }}</p>
                <a-textarea
                  v-model:value="inputs.status_code_mapping"
                  :rows="4"
                  :placeholder="t('channel.edit.status_code_mapping_placeholder')"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.param_override')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.param_override_hint') }}</p>
                <a-textarea
                  v-model:value="inputs.param_override"
                  :rows="6"
                  :placeholder="t('channel.edit.param_override_placeholder')"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.header_override')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.header_override_hint') }}</p>
                <a-textarea
                  v-model:value="inputs.header_override"
                  :rows="4"
                  :placeholder="t('channel.edit.header_override_placeholder')"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.setting_json')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.setting_json_hint') }}</p>
                <a-textarea
                  v-model:value="inputs.setting"
                  :rows="4"
                  :placeholder="t('channel.edit.setting_json_placeholder')"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.settings_json')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.settings_json_hint') }}</p>
                <a-textarea
                  v-model:value="inputs.settings"
                  :rows="4"
                  :placeholder="t('channel.edit.settings_json_placeholder')"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.other_info')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.other_info_hint') }}</p>
                <a-textarea
                  v-model:value="inputs.other_info"
                  :rows="3"
                  :placeholder="t('channel.edit.other_info_placeholder')"
                />
              </a-form-item>
            </div>
          </div>

          <!-- 路由高级 -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title text-base font-medium mb-2">
              {{ t('channel.edit.routing_advanced') }}
            </h4>
            <a-form-item :label="t('channel.edit.routing_provider')">
              <a-input
                v-model:value="config.routing_provider"
                :placeholder="t('channel.edit.routing_provider_placeholder')"
              />
            </a-form-item>
            <a-form-item>
              <a-switch v-model:checked="config.routing_skip_adaptive" />
              <span class="ml-2">{{ t('channel.edit.routing_skip_adaptive') }}</span>
            </a-form-item>
          </div>

          <div class="channel-edit-actions flex gap-2">
            <a-button @click="handleCancel">{{ t('channel.edit.buttons.cancel') }}</a-button>
            <a-button type="primary" @click="submit">{{ t('channel.edit.buttons.submit') }}</a-button>
          </div>
        </a-form>
      </a-spin>
    </a-card>

    <!-- 从模型目录填充 -->
    <a-modal v-model:open="fillCatalogOpen" :title="t('channel.edit.fill_catalog_title')" :footer="null" width="520px">
      <p class="channel-edit-field-hint">{{ t('channel.edit.fill_catalog_hint') }}</p>
      <a-form-item :label="t('model_catalog.filter_provider_label')">
        <a-select
          v-model:value="fillProvider"
          show-search
          allow-clear
          :options="fillProviderOptions"
          :field-names="{ label: 'text', value: 'value' }"
          option-filter-prop="text"
          :placeholder="t('channel.edit.fill_catalog_provider_ph')"
          style="width: 100%"
        />
      </a-form-item>
      <a-form-item v-if="fillPreviewCount != null && fillProvider">
        <label>{{ t('channel.edit.fill_catalog_preview', { count: fillPreviewCount }) }}</label>
      </a-form-item>
      <div class="flex justify-end gap-2 mt-2">
        <a-button @click="fillCatalogOpen = false">{{ t('channel.edit.fill_catalog_cancel') }}</a-button>
        <a-button type="primary" :loading="fillLoading" :disabled="fillLoading" @click="submitFillCatalog">
          {{ t('channel.edit.fill_catalog_confirm') }}
        </a-button>
      </div>
    </a-modal>

    <!-- 测试报告 -->
    <a-modal
      v-model:open="modelTestReportOpen"
      :title="t('channel.edit.test_report.title')"
      :footer="null"
      width="1000px"
      class="channel-model-test-report-modal"
    >
      <div class="channel-model-test-report__meta mb-2">
        <span v-if="reportChannelTypeLabel">{{ t('channel.edit.test_report.channel_type') }}: {{ reportChannelTypeLabel }}</span>
        <span v-if="reportBaseUrl"> {{ t('channel.edit.test_report.base_url') }}: {{ reportBaseUrl }}</span>
      </div>
      <div class="channel-model-test-report__filter-bar flex gap-2 mb-3">
        <a-button
          v-for="item in reportFilterItems"
          :key="item.key"
          size="small"
          :type="reportFilter === item.key ? 'primary' : 'default'"
          @click="reportFilter = item.key"
        >
          {{ reportFilterLabel(item) }}
        </a-button>
      </div>
      <a-table
        :columns="reportColumns"
        :data-source="reportFilteredRows"
        :pagination="false"
        size="small"
        row-key="model"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'kind'">{{ record.test_protocol || record.test_kind || '—' }}</template>
          <template v-else-if="column.key === 'status'">
            <span
              :class="`channel-model-test-report__status channel-model-test-report__status--${reportStatusKey(record)}`"
              :title="reportStatusTooltip(record)"
            >{{ reportStatusKey(record) }}</span>
          </template>
          <template v-else-if="column.key === 'started_at'">{{ reportFormatStartedAt(record) }}</template>
          <template v-else-if="column.key === 'time'">{{ reportFormatTime(record) }}</template>
          <template v-else-if="column.key === 'message'">{{ record.message || '—' }}</template>
        </template>
      </a-table>
      <div class="flex justify-end mt-3">
        <a-button @click="modelTestReportOpen = false">{{ t('channel.edit.test_report.close') }}</a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter, useRoute } from 'vue-router';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons-vue';
import {
  API,
  copy,
  splitModelNameList,
  showError,
  showInfo,
  showSuccess,
  verifyJSON,
  renderChannelTip,
  setChannelTypeOptionsCache,
} from '@/helpers';
import { fetchModelCatalogModelIds } from '@/helpers/modelCatalog';
import {
  fetchModelCatalogProviders,
  providerCatalogModelFilters,
  providerOptionsToDropdown,
} from '@/helpers/modelCatalog';
import {
  applyJobSnapshotToModelState,
  applyStoredModelTestResults,
  buildChannelModelTestPayload,
  buildJobProgressSummary,
  buildStoredModelTestSummary,
  controlChannelModelTestJob,
  fetchChannelModelTestResults,
  filterOutFailedModels,
  MODEL_TEST_FAIL,
  MODEL_TEST_OK,
  MODEL_TEST_SKIP,
  MODEL_TEST_TESTING,
  normalizeChannelTestBaseUrl,
  normalizeModelTestReportRows,
  runChannelModelTestJob,
  summarizeModelTestReport,
} from '@/helpers/channelModelTest';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const DEFAULT_CHANNEL_CONFIG = {
  region: '',
  sk: '',
  ak: '',
  user_id: '',
  vertex_ai_project_id: '',
  vertex_ai_adc: '',
  api_version: '',
  library_id: '',
  plugin: '',
  routing_provider: '',
  routing_skip_adaptive: false,
  deep_research_mode: '',
};

const defaultChannelInputs = () => ({
  name: '',
  type: 41,
  key: '',
  base_url: '',
  model_mapping: '',
  system_prompt: '',
  models: [],
  groups: ['default'],
  openai_organization: '',
  test_model: '',
  auto_ban: 1,
  remark: '',
  tag: '',
  status_code_mapping: '',
  param_override: '',
  header_override: '',
  setting: '',
  settings: '',
  other_info: '',
});

function type2secretPrompt(type) {
  switch (type) {
    case 43:
      return t('channel.edit.key_prompts.aippt');
    case 44:
      return t('channel.edit.key_prompts.amap_poi');
    case 45:
      return t('channel.edit.key_prompts.deep_research');
    case 5:
    case 46:
      return t('channel.edit.key_prompts.anthropic');
    case 14:
    case 15:
    case 42:
      return t('channel.edit.key_prompts.gemini');
    default:
      return t('channel.edit.key_prompts.default');
  }
}

const channelId = computed(() => route.params.id);
const isEdit = computed(() => channelId.value !== undefined);

const loading = ref(isEdit.value);
const channelTypeOptions = ref([]);
const channelTypesLoading = ref(true);
const batch = ref(false);
const fetchUpstreamBusy = ref(false);
const fillCatalogOpen = ref(false);
const fillAllBusy = ref(false);
const testModelsBusy = ref(false);
const testModelsControlBusy = ref(false);
const testModelsProgress = reactive({
  total: 0,
  completed: 0,
  currentModel: '',
  concurrency: 3,
  running: false,
  paused: false,
  cancelReq: false,
  state: '',
});
const testModelsConcurrency = ref(3);
let testModelsPolling = false;
const modelTestStatus = ref({});
const modelTestMessages = ref({});
const modelTestMeta = ref({});
const modelTestSummary = ref('');
const modelTestReportOpen = ref(false);
const modelTestReportRows = ref([]);
const inputs = reactive(defaultChannelInputs());
const originModelOptions = ref([]);
const modelOptions = ref([]);
const groupOptions = ref([]);
const customModel = ref('');
const config = reactive({ ...DEFAULT_CHANNEL_CONFIG });

const handleCancel = () => {
  router.push('/tenant-console/channels');
};

const currentTypeDescription = computed(() => {
  const cur = channelTypeOptions.value.find((o) => o.value === inputs.type);
  return cur?.description || '';
});

// Functional component that renders the channel tip VNode (returned by renderChannelTip).
const ChannelTip = (props) => renderChannelTip(props.channelType);
ChannelTip.props = ['channelType'];

const testProgressPercent = computed(() =>
  Math.min(
    100,
    Math.round((testModelsProgress.completed / (testModelsProgress.total || 1)) * 100)
  )
);

const failedModelCount = computed(
  () => (inputs.models || []).filter((m) => modelTestStatus.value[String(m)] === MODEL_TEST_FAIL).length
);

function setModels(value) {
  // mirror React handleInputChange('models') — prune stale test state
  const allowed = new Set((value || []).map((m) => String(m)));
  const keep = (obj) =>
    Object.fromEntries(Object.entries(obj || {}).filter(([key]) => allowed.has(key)));
  modelTestStatus.value = keep(modelTestStatus.value);
  modelTestMessages.value = keep(modelTestMessages.value);
  modelTestMeta.value = keep(modelTestMeta.value);
  inputs.models = value;
}

const onModelsChange = (value) => {
  setModels(value);
};

const onTypeChange = (value) => {
  const opt = channelTypeOptions.value.find((o) => o.value === value);
  const defaultUrl = opt && opt.default_base_url ? String(opt.default_base_url).trim() : '';
  const prevType = inputs.type;
  let base_url = inputs.base_url;
  if (prevType !== value && defaultUrl) {
    const prevTrim = String(inputs.base_url || '').trim();
    const prevOpt = channelTypeOptions.value.find((o) => o.value === prevType);
    const prevDefault =
      prevOpt && prevOpt.default_base_url ? String(prevOpt.default_base_url).trim() : '';
    if (!prevTrim || prevTrim === prevDefault) {
      base_url = defaultUrl;
    }
  }
  inputs.type = value;
  inputs.base_url = base_url;
};

const toggleAutoBan = () => {
  inputs.auto_ban = inputs.auto_ban === 0 ? 1 : 0;
};

const copyModel = (value) => {
  copy(value);
};

// ---- model tag status styling ----
function modelTagClass(model) {
  const status = modelTestStatus.value[String(model)];
  const cls = ['model-tag'];
  if (status === MODEL_TEST_OK) cls.push('model-tag--ok');
  if (status === MODEL_TEST_FAIL) cls.push('model-tag--fail');
  if (status === MODEL_TEST_SKIP) cls.push('model-tag--skip');
  if (status === MODEL_TEST_TESTING) cls.push('model-tag--testing');
  return cls.join(' ');
}
function modelTagTitle(model) {
  const status = modelTestStatus.value[String(model)];
  const meta = modelTestMeta.value[String(model)];
  const failMsg = modelTestMessages.value[String(model)];
  const parts = [];
  if (meta?.testedAt) {
    const d = new Date(meta.testedAt * 1000);
    if (!Number.isNaN(d.getTime())) parts.push(d.toLocaleString());
  }
  if (status === MODEL_TEST_FAIL && failMsg) {
    parts.push(failMsg);
  } else if (status === MODEL_TEST_OK && meta?.message) {
    const msg = String(meta.message);
    parts.push(msg.length > 120 ? `${msg.slice(0, 120)}…` : msg);
  }
  return parts.length ? parts.join(' · ') : undefined;
}

// ---- model mapping editor ----
const MODEL_MAP_EXAMPLE_LINES = `{
  "gpt-3.5-turbo-0301": "gpt-3.5-turbo",
  "gpt-4-0314": "gpt-4"
}`;
const mappingHint = computed(
  () => `${t('channel.edit.model_mapping_placeholder')}\n${MODEL_MAP_EXAMPLE_LINES}`
);
const mappingRows = ref([{ from: '', to: '' }]);
const mappingJsonMode = ref(false);
const mappingJsonRaw = ref('');
let mappingSyncingFromValue = false;

const mappingColumns = [
  { title: t('channel.edit.model_mapping_col_request'), key: 'from' },
  { title: t('channel.edit.model_mapping_col_upstream'), key: 'to' },
  { title: '', key: 'op', width: 60 },
];
const mappingRowsData = computed(() =>
  mappingRows.value.map((r, i) => ({ ...r, _idx: i }))
);
const mappingUpstreamOptions = computed(() => {
  const seen = new Set();
  const out = [];
  (inputs.models || []).forEach((id) => {
    const s = String(id || '').trim();
    if (!s || seen.has(s)) return;
    seen.add(s);
    out.push({ key: s, value: s, text: s });
  });
  return out;
});
const mappingToOptions = (current) => {
  const cur = String(current || '').trim();
  if (!cur || mappingUpstreamOptions.value.some((o) => o.value === cur)) {
    return mappingUpstreamOptions.value;
  }
  return [{ key: `custom-${cur}`, value: cur, text: cur }, ...mappingUpstreamOptions.value];
};

function parseMappingToRows(str) {
  const s = String(str ?? '').trim();
  if (!s) return [{ from: '', to: '' }];
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      const ent = Object.entries(o);
      if (ent.length === 0) return [{ from: '', to: '' }];
      return ent.map(([from, to]) => ({ from: String(from), to: String(to ?? '') }));
    }
  } catch {
    /* fallthrough */
  }
  return [{ from: '', to: '' }];
}
function rowsToMappingJson(rows) {
  const o = {};
  for (const r of rows) {
    const k = String(r.from ?? '').trim();
    if (!k) continue;
    o[k] = String(r.to ?? '');
  }
  if (Object.keys(o).length === 0) return '';
  return JSON.stringify(o, null, 2);
}
function emitMappingRows(next) {
  mappingRows.value = next;
  mappingSyncingFromValue = true;
  inputs.model_mapping = rowsToMappingJson(next);
  mappingSyncingFromValue = false;
}
function updateMappingRow(i, field, v) {
  const next = mappingRows.value.map((r, j) => (j === i ? { ...r, [field]: v } : r));
  emitMappingRows(next);
}
function addMappingRow() {
  emitMappingRows([...mappingRows.value, { from: '', to: '' }]);
}
function removeMappingRow(i) {
  const next = mappingRows.value.filter((_, j) => j !== i);
  emitMappingRows(next.length ? next : [{ from: '', to: '' }]);
}
function enterMappingJsonMode() {
  mappingJsonRaw.value = inputs.model_mapping?.trim() ? inputs.model_mapping : MODEL_MAP_EXAMPLE_LINES;
  mappingJsonMode.value = true;
}
function cancelMappingJsonMode() {
  mappingJsonMode.value = false;
  mappingJsonRaw.value = '';
}
function applyMappingJsonPaste() {
  const raw = mappingJsonRaw.value.trim();
  if (!raw) {
    emitMappingRows([{ from: '', to: '' }]);
    mappingJsonMode.value = false;
    return;
  }
  if (!verifyJSON(raw)) {
    showError(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  let o;
  try {
    o = JSON.parse(raw);
  } catch {
    showError(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  if (typeof o !== 'object' || o === null || Array.isArray(o)) {
    showError(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  emitMappingRows(parseMappingToRows(raw));
  mappingJsonMode.value = false;
  mappingJsonRaw.value = '';
}
watch(
  () => inputs.model_mapping,
  (v) => {
    if (mappingSyncingFromValue) return;
    mappingRows.value = parseMappingToRows(v);
  }
);

// ---- channel load ----
const loadChannel = async () => {
  let data = null;
  const fromState = window.history.state?.channel;
  if (fromState && String(fromState.id) === String(channelId.value)) {
    data = { ...fromState };
  } else {
    const res = await API.get('/api/tenant_console/channels?p=0&page_size=500');
    const { success, message, data: rows } = res.data;
    if (!success) {
      showError(message);
      loading.value = false;
      return;
    }
    data = (rows || []).find((c) => String(c.id) === String(channelId.value));
    if (!data) {
      showError(t('tenant_console.channel_not_found'));
      loading.value = false;
      return;
    }
  }
  if (data.models === '') {
    data.models = [];
  } else if (typeof data.models === 'string') {
    data.models = [
      ...new Set(data.models.split(',').map((id) => id.trim()).filter(Boolean)),
    ];
  }
  if (!data.groups) {
    if (data.group === '' || data.group == null) {
      data.groups = [];
    } else {
      data.groups = String(data.group).split(',');
    }
  }
  if (data.model_mapping && data.model_mapping !== '') {
    try {
      data.model_mapping = JSON.stringify(JSON.parse(data.model_mapping), null, 2);
    } catch {
      /* keep raw */
    }
  }
  const { other: _legacyOther, ...channelData } = data;
  if (channelData.auto_ban === undefined || channelData.auto_ban === null) {
    channelData.auto_ban = 1;
  }
  channelData.openai_organization = channelData.openai_organization ?? '';
  channelData.test_model = channelData.test_model ?? '';
  channelData.remark = channelData.remark ?? '';
  channelData.tag = channelData.tag ?? '';
  channelData.status_code_mapping = channelData.status_code_mapping ?? '';
  channelData.param_override = channelData.param_override ?? '';
  channelData.header_override = channelData.header_override ?? '';
  channelData.setting = channelData.setting ?? '';
  channelData.settings = channelData.settings ?? '';
  channelData.other_info = channelData.other_info ?? '';
  Object.assign(inputs, defaultChannelInputs(), channelData);
  if (data.config !== '' && data.config != null) {
    const cfg = JSON.parse(data.config);
    Object.assign(config, DEFAULT_CHANNEL_CONFIG, cfg);
  } else {
    Object.assign(config, DEFAULT_CHANNEL_CONFIG);
  }
  loading.value = false;
};

// ---- test job ----
const applyJobSnapshot = (jobData, models) => {
  const applied = applyJobSnapshotToModelState(jobData, models);
  modelTestStatus.value = applied.status;
  modelTestMessages.value = applied.messages;
  modelTestMeta.value = applied.meta;
  Object.assign(testModelsProgress, applied.progress);
  modelTestSummary.value = buildJobProgressSummary(t, jobData);
  modelTestReportRows.value = normalizeModelTestReportRows(jobData, []);
};

const pollRunningJob = async (payload, models) => {
  if (testModelsPolling) return;
  testModelsPolling = true;
  testModelsBusy.value = true;
  try {
    const summary = await runChannelModelTestJob(payload, models, {
      onJobSnapshot: (data) => applyJobSnapshot(data, models),
    });
    if (summary.fail === 0 && summary.ok > 0) {
      showSuccess(t('channel.edit.messages.test_models_all_ok', { count: summary.ok }));
    } else if (summary.fail > 0) {
      showInfo(
        t('channel.edit.messages.test_models_partial', {
          ok: summary.ok,
          fail: summary.fail,
          skip: summary.skip || 0,
        })
      );
    }
    if ((summary.job?.results || []).length > 0) {
      modelTestReportOpen.value = true;
    }
  } catch (e) {
    showError(e.message || t('channel.edit.messages.test_models_fail'));
  } finally {
    testModelsBusy.value = false;
    testModelsProgress.running = false;
    testModelsProgress.currentModel = '';
    testModelsPolling = false;
  }
};

const loadStoredModelTestResults = async () => {
  if (!isEdit.value || !channelId.value) return;
  try {
    const base = normalizeChannelTestBaseUrl(inputs.base_url, inputs.type, channelTypeOptions.value);
    const { results, job } = await fetchChannelModelTestResults({
      channelId: channelId.value,
      baseUrl: base,
      channelType: inputs.type,
      channelTypeOptions: channelTypeOptions.value,
      tenantConsole: true,
    });
    if (job?.running) {
      applyJobSnapshot(job, inputs.models);
      const payload = buildChannelModelTestPayload({
        channelId: channelId.value,
        type: inputs.type,
        baseUrl: inputs.base_url,
        key: '',
        config,
        modelMapping: inputs.model_mapping,
        channelTypeOptions: channelTypeOptions.value,
        concurrency: testModelsConcurrency.value,
        tenantConsole: true,
      });
      pollRunningJob(payload, inputs.models);
      return;
    }
    const applied = applyStoredModelTestResults(results);
    modelTestStatus.value = applied.status;
    modelTestMessages.value = applied.messages;
    modelTestMeta.value = applied.meta;
    Object.assign(testModelsProgress, {
      total: 0,
      completed: 0,
      currentModel: '',
      running: false,
      paused: false,
      cancelReq: false,
      state: '',
    });
    const visible = (inputs.models || []).filter((m) => applied.status[m]);
    const ok = visible.filter((m) => applied.status[m] === MODEL_TEST_OK).length;
    const fail = visible.filter((m) => applied.status[m] === MODEL_TEST_FAIL).length;
    modelTestSummary.value = buildStoredModelTestSummary(t, ok, fail, visible.length);
    modelTestReportRows.value = normalizeModelTestReportRows(null, results);
  } catch {
    // 忽略历史加载失败
  }
};

const fetchModels = async () => {
  try {
    const ids = await fetchModelCatalogModelIds({}, true);
    originModelOptions.value = ids.map((id) => ({ key: id, text: id, value: id }));
  } catch (error) {
    showError(error.message);
  }
};

const fillAllCatalogModels = async () => {
  fillAllBusy.value = true;
  try {
    const ids = await fetchModelCatalogModelIds({}, true);
    if (!ids.length) {
      showInfo(t('channel.edit.fill_catalog_empty'));
      return;
    }
    setModels(ids);
    showSuccess(t('channel.edit.fill_all_ok', { count: ids.length }));
  } catch (e) {
    showError(e.message || t('channel.edit.fill_catalog_fail'));
  } finally {
    fillAllBusy.value = false;
  }
};

const fetchGroups = async () => {
  try {
    const res = await API.get(`/api/tenant_console/meta/groups`);
    groupOptions.value = res.data.data.map((group) => ({ key: group, text: group, value: group }));
  } catch (error) {
    showError(error.message);
  }
};

watch(
  [originModelOptions, () => inputs.models],
  () => {
    const localModelOptions = [...originModelOptions.value];
    (inputs.models || []).forEach((model) => {
      if (!localModelOptions.find((option) => option.key === model)) {
        localModelOptions.push({ key: model, text: model, value: model });
      }
    });
    modelOptions.value = localModelOptions;
  },
  { immediate: true, deep: true }
);

const submit = async () => {
  if (inputs.key === '') {
    if (config.ak !== '' && config.sk !== '' && config.region !== '') {
      inputs.key = `${config.ak}|${config.sk}|${config.region}`;
    }
  }
  if (!isEdit.value && (inputs.name === '' || inputs.key === '')) {
    showInfo(t('channel.edit.messages.name_required'));
    return;
  }
  if (inputs.models.length === 0) {
    showInfo(t('channel.edit.messages.models_required'));
    return;
  }
  if (inputs.model_mapping !== '' && !verifyJSON(inputs.model_mapping)) {
    showInfo(t('channel.edit.messages.model_mapping_invalid'));
    return;
  }
  const jsonFields = [
    ['status_code_mapping', inputs.status_code_mapping],
    ['param_override', inputs.param_override],
    ['header_override', inputs.header_override],
    ['setting', inputs.setting],
    ['settings', inputs.settings],
  ];
  for (const [label, val] of jsonFields) {
    if (val && String(val).trim() && !verifyJSON(val)) {
      showInfo(`${label}: ${t('channel.edit.messages.model_mapping_invalid')}`);
      return;
    }
  }
  let localInputs = { ...inputs };
  if (localInputs.key === 'undefined|undefined|undefined') {
    localInputs.key = '';
  }
  if (localInputs.base_url && localInputs.base_url.endsWith('/')) {
    localInputs.base_url = localInputs.base_url.slice(0, localInputs.base_url.length - 1);
  }
  let cfg = { ...config };
  if ((localInputs.type === 14 || localInputs.type === 42) && !cfg.api_version) {
    cfg.api_version = 'v1';
  }
  let res;
  localInputs.models = localInputs.models.join(',');
  localInputs.group = localInputs.groups.join(',');
  localInputs.config = JSON.stringify(cfg);
  if (isEdit.value) {
    res = await API.put(`/api/tenant_console/channels`, {
      ...localInputs,
      id: parseInt(channelId.value, 10),
    });
  } else {
    res = await API.post(`/api/tenant_console/channels`, localInputs);
  }
  const { success, message } = res.data;
  if (success) {
    if (isEdit.value) {
      showSuccess(t('channel.edit.messages.update_success'));
      await loadChannel();
    } else {
      showSuccess(t('channel.edit.messages.create_success'));
      Object.assign(inputs, defaultChannelInputs());
      customModel.value = '';
      Object.assign(config, DEFAULT_CHANNEL_CONFIG);
    }
  } else {
    showError(message);
  }
};

const addCustomModel = () => {
  const names = splitModelNameList(customModel.value);
  if (names.length === 0) return;
  const localModels = [...inputs.models];
  const newOptions = [];
  for (const name of names) {
    if (localModels.includes(name)) continue;
    localModels.push(name);
    newOptions.push({ key: name, text: name, value: name });
  }
  if (newOptions.length === 0) {
    showInfo(t('channel.edit.messages.custom_models_all_exist'));
    return;
  }
  modelOptions.value = [...modelOptions.value, ...newOptions];
  customModel.value = '';
  setModels(localModels);
};

const resolveKeyForUpstreamFetch = () => {
  if (batch.value) {
    const first = String(inputs.key || '')
      .split(/\r?\n/)
      .map((s) => s.trim())
      .find(Boolean);
    return first || '';
  }
  let k = String(inputs.key || '').trim();
  if (!k && config.ak && config.sk && config.region) {
    k = `${config.ak}|${config.sk}|${config.region}`;
  }
  if (!k && config.region && config.vertex_ai_project_id && config.vertex_ai_adc) {
    k = `${config.region}|${config.vertex_ai_project_id}|${config.vertex_ai_adc}`;
  }
  return k;
};

const fetchUpstreamModelsList = async () => {
  showInfo(t('tenant_console.upstream_fetch_disabled'));
};

const testAllModels = async () => {
  if (testModelsBusy.value || testModelsProgress.running) {
    showInfo(t('channel.edit.messages.test_models_already_running'));
    return;
  }
  const models = (inputs.models || []).map((m) => String(m).trim()).filter(Boolean);
  if (!models.length) {
    showInfo(t('channel.edit.messages.test_models_empty'));
    return;
  }
  const keyFromForm = resolveKeyForUpstreamFetch();
  if (!keyFromForm && !(isEdit.value && channelId.value)) {
    showInfo(t('channel.edit.messages.upstream_fetch_need_key'));
    return;
  }
  modelTestStatus.value = {};
  modelTestMessages.value = {};
  modelTestMeta.value = {};
  Object.assign(testModelsProgress, {
    total: models.length,
    completed: 0,
    currentModel: '',
    running: true,
    paused: false,
    cancelReq: false,
    state: 'running',
  });
  const payload = buildChannelModelTestPayload({
    channelId: isEdit.value ? channelId.value : 0,
    type: inputs.type,
    baseUrl: inputs.base_url,
    key: keyFromForm,
    config,
    modelMapping: inputs.model_mapping,
    channelTypeOptions: channelTypeOptions.value,
    concurrency: testModelsConcurrency.value,
    tenantConsole: true,
  });
  await pollRunningJob(payload, models);
};

const controlRunningTestJob = async (action) => {
  if (!testModelsProgress.running || testModelsControlBusy.value) return;
  testModelsControlBusy.value = true;
  try {
    const payload = buildChannelModelTestPayload({
      channelId: isEdit.value ? channelId.value : 0,
      type: inputs.type,
      baseUrl: inputs.base_url,
      key: '',
      config,
      modelMapping: inputs.model_mapping,
      channelTypeOptions: channelTypeOptions.value,
      concurrency: testModelsConcurrency.value,
      tenantConsole: true,
    });
    const { job } = await controlChannelModelTestJob(payload, action);
    if (job) {
      applyJobSnapshot(job, inputs.models);
    }
  } catch (e) {
    showError(e.message || t('channel.edit.messages.test_models_control_failed'));
  } finally {
    testModelsControlBusy.value = false;
  }
};

const clearInvalidModels = () => {
  const { failed, remaining } = filterOutFailedModels(inputs.models, modelTestStatus.value);
  if (!failed.length) {
    showInfo(t('channel.edit.messages.clear_failed_empty'));
    return;
  }
  setModels(remaining);
  showSuccess(t('channel.edit.messages.clear_failed_done', { count: failed.length }));
};

// ---- fill catalog modal ----
const fillProvider = ref('');
const fillProviderOptions = ref([]);
const fillLoading = ref(false);
const fillPreviewCount = ref(null);
let fillPreviewTimer = null;

watch(fillCatalogOpen, (open) => {
  if (!open) return;
  fillProvider.value = '';
  fillPreviewCount.value = null;
  fetchModelCatalogProviders('', 100, true)
    .then((rows) => {
      fillProviderOptions.value = providerOptionsToDropdown(rows);
    })
    .catch(() => {
      fillProviderOptions.value = [];
    });
});

watch([fillProvider, fillProviderOptions], () => {
  if (!fillCatalogOpen.value) return;
  const filters = providerCatalogModelFilters(fillProvider.value, fillProviderOptions.value);
  if (!filters) {
    fillPreviewCount.value = null;
    return;
  }
  if (fillPreviewTimer) clearTimeout(fillPreviewTimer);
  fillPreviewTimer = setTimeout(() => {
    fetchModelCatalogModelIds(filters, true)
      .then((ids) => {
        fillPreviewCount.value = ids.length;
      })
      .catch(() => {
        fillPreviewCount.value = null;
      });
  }, 250);
});

const submitFillCatalog = async () => {
  const filters = providerCatalogModelFilters(fillProvider.value, fillProviderOptions.value);
  if (!filters) {
    showError(t('channel.edit.fill_catalog_provider_required'));
    return;
  }
  fillLoading.value = true;
  try {
    const ids = await fetchModelCatalogModelIds(filters, true);
    if (!ids.length) {
      showError(t('channel.edit.fill_catalog_empty'));
      return;
    }
    setModels(ids);
    showSuccess(t('channel.edit.fill_catalog_ok', { count: ids.length }));
    fillCatalogOpen.value = false;
  } catch (e) {
    showError(e.message || t('channel.edit.fill_catalog_fail'));
  } finally {
    fillLoading.value = false;
  }
};

// ---- test report modal ----
const reportFilter = ref('all');
const reportFilterItems = [
  { key: 'all', countProp: '', labelKey: 'filter_all' },
  { key: 'done', countProp: 'total', labelKey: 'count_done' },
  { key: 'ok', countProp: 'ok', labelKey: 'count_ok' },
  { key: 'fail', countProp: 'fail', labelKey: 'count_fail' },
  { key: 'skip', countProp: 'skip', labelKey: 'count_skip' },
];
const reportColumns = [
  { title: t('channel.edit.test_report.col_model'), dataIndex: 'model', key: 'model' },
  { title: t('channel.edit.test_report.col_kind'), key: 'kind' },
  { title: t('channel.edit.test_report.col_status'), key: 'status' },
  { title: t('channel.edit.test_report.col_started_at'), key: 'started_at' },
  { title: t('channel.edit.test_report.col_time'), key: 'time' },
  { title: t('channel.edit.test_report.col_message'), key: 'message' },
];
const reportSummary = computed(() => summarizeModelTestReport(modelTestReportRows.value));
const reportAllCount = computed(() =>
  (inputs.models || []).length > 0 ? (inputs.models || []).length : reportSummary.value.total
);
const reportFilteredRows = computed(() => {
  const list = [...(modelTestReportRows.value || [])].sort((a, b) =>
    String(a.model || '').localeCompare(String(b.model || ''))
  );
  if (reportFilter.value === 'ok') return list.filter((r) => r.success && !r.skipped);
  if (reportFilter.value === 'fail') return list.filter((r) => !r.success && !r.skipped);
  if (reportFilter.value === 'skip') return list.filter((r) => r.skipped);
  return list;
});
const reportChannelTypeLabel = computed(
  () => channelTypeOptions.value.find((o) => o.value === inputs.type)?.text || ''
);
const reportBaseUrl = computed(() =>
  normalizeChannelTestBaseUrl(inputs.base_url, inputs.type, channelTypeOptions.value)
);
const reportFilterLabel = (item) => {
  if (item.key === 'all') {
    return `${t('channel.edit.test_report.filter_all')} ${reportAllCount.value}`;
  }
  return t(`channel.edit.test_report.${item.labelKey}`, {
    count: reportSummary.value[item.countProp],
  });
};
function reportStatusKey(row) {
  if (row?.skipped) return 'skip';
  if (row?.timed_out) return 'timeout';
  if (row?.success) return 'ok';
  return 'fail';
}
function reportStatusTooltip(row) {
  const key = reportStatusKey(row);
  if (key === 'timeout') return t('channel.edit.test_report.status_timeout');
  if (key === 'skip') return t('channel.edit.test_report.status_skip');
  if (key === 'ok') return t('channel.edit.test_report.status_ok');
  return t('channel.edit.test_report.status_fail');
}
function reportFormatStartedAt(row) {
  if (row?.started_at) {
    const d = new Date(Number(row.started_at) * 1000);
    if (!Number.isNaN(d.getTime())) return d.toLocaleString();
  }
  if (row?.tested_at && row?.elapsed_ms != null) {
    const d = new Date(Number(row.tested_at) * 1000 - Number(row.elapsed_ms));
    if (!Number.isNaN(d.getTime())) return d.toLocaleString();
  }
  return '—';
}
function reportFormatTime(row) {
  if (row?.elapsed_ms != null) return `${(Number(row.elapsed_ms) / 1000).toFixed(2)}s`;
  if (row?.time != null) return `${Number(row.time).toFixed(2)}s`;
  return '—';
}

// ---- init ----
watch(
  () => loading.value,
  (v) => {
    if (!v && isEdit.value) {
      loadStoredModelTestResults();
    }
  }
);

onMounted(async () => {
  // sync mapping editor with initial value
  mappingRows.value = parseMappingToRows(inputs.model_mapping);
  try {
    const res = await API.get('/api/tenant_console/meta/editor_options');
    const opts = res.data?.data?.channel_types;
    if (Array.isArray(opts) && opts.length) {
      channelTypeOptions.value = opts;
      setChannelTypeOptionsCache(opts);
    }
  } catch (_) {
    /* 忽略 */
  } finally {
    channelTypesLoading.value = false;
  }
  if (isEdit.value) {
    await loadChannel();
  }
  fetchModels();
  fetchGroups();
});
</script>

<style scoped>
.model-tag--ok { border-color: #52c41a; color: #389e0d; }
.model-tag--fail { border-color: #ff4d4f; color: #cf1322; }
.model-tag--skip { border-color: #d9d9d9; color: #8c8c8c; }
.model-tag--testing { border-color: #1677ff; color: #1677ff; }
.channel-edit-field-hint { font-size: 12px; color: rgba(0, 0, 0, 0.45); margin-bottom: 4px; }
.channel-edit-section { margin-bottom: 16px; }
.channel-model-test-report__status--ok { color: #389e0d; }
.channel-model-test-report__status--fail { color: #cf1322; }
.channel-model-test-report__status--skip { color: #8c8c8c; }
.channel-model-test-report__status--timeout { color: #d46b08; }
.channel-model-test-summary { margin-bottom: 8px; }
</style>
