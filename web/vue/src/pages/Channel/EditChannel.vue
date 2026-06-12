<template>
  <div class="dashboard-container p-6 channel-edit-page">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">
        {{ isEdit ? t('channel.edit.title_edit') : t('channel.edit.title_create') }}
      </h2>
      <a-spin :spinning="loading || channelTypesLoading">
        <a-form layout="vertical" autocomplete="off">
          <!-- Basic section -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title">{{ t('channel.edit.section_basic') }}</h4>
            <a-form-item :label="t('channel.edit.type')" required>
              <a-select
                show-search
                :options="channelTypeOptions"
                :field-names="{ label: 'text', value: 'value' }"
                :placeholder="channelTypesLoading ? t('channel.edit.types_loading') : undefined"
                :value="inputs.type"
                :filter-option="filterTypeOption"
                @change="(v) => handleTypeChange(v)"
              />
            </a-form-item>
            <a-form-item :label="t('channel.edit.name')" required>
              <a-input
                :placeholder="t('channel.edit.name_placeholder')"
                v-model:value="inputs.name"
                autocomplete="off"
              />
            </a-form-item>
            <a-form-item :label="t('channel.edit.group')" required>
              <a-select
                mode="tags"
                :placeholder="t('channel.edit.group_placeholder')"
                v-model:value="inputs.groups"
                :options="groupOptions"
                :field-names="{ label: 'text', value: 'value' }"
              />
            </a-form-item>
            <component :is="renderChannelTip(inputs.type)" />
          </div>

          <!-- Upstream section -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title">{{ t('channel.edit.section_upstream') }}</h4>
            <a-alert
              v-if="currentTypeDescription"
              type="info"
              :message="currentTypeDescription"
              style="margin-bottom: 12px"
            />
            <a-form-item v-if="inputs.type === 14 || inputs.type === 42" label="Gemini API 版本">
              <a-input
                placeholder="例如：v1"
                :value="config.api_version || ''"
                @update:value="(v) => handleConfigChange('api_version', v)"
                autocomplete="off"
              />
            </a-form-item>
            <div class="channel-edit-upstream-row">
              <a-form-item :label="t('channel.edit.base_url')" class="channel-edit-upstream-field">
                <p class="channel-edit-field-hint">{{ t('channel.edit.base_url_all_hint') }}</p>
                <a-input v-model:value="inputs.base_url" autocomplete="off" />
              </a-form-item>
              <a-form-item
                v-if="!batch"
                :label="t('channel.edit.key')"
                class="channel-edit-upstream-field"
                :required="!isEdit"
              >
                <p class="channel-edit-field-hint">{{ type2secretPrompt(inputs.type) }}</p>
                <div
                  v-if="isEdit && keyMasked"
                  class="channel-edit-key-preview"
                  style="margin-bottom: 8px"
                >
                  <a-input readonly :value="keyRevealed ? revealedKey : keyMasked" autocomplete="off">
                    <template #suffix>
                      <a-tooltip :title="keyRevealed ? t('channel.edit.key_hide') : t('channel.edit.key_show')">
                        <component
                          :is="keyRevealed ? EyeInvisibleOutlined : EyeOutlined"
                          style="cursor: pointer"
                          @click="toggleRevealKey"
                        />
                      </a-tooltip>
                      <a-tooltip :title="t('channel.edit.key_copy')">
                        <CopyOutlined style="cursor: pointer; margin-left: 10px" @click="copyConfiguredKey" />
                      </a-tooltip>
                    </template>
                  </a-input>
                  <p class="channel-edit-field-hint">{{ t('channel.edit.key_masked_hint') }}</p>
                </div>
                <a-input
                  v-model:value="inputs.key"
                  :placeholder="isEdit ? t('channel.edit.key_edit_placeholder') : type2secretPrompt(inputs.type)"
                  autocomplete="off"
                />
              </a-form-item>
            </div>
            <a-form-item v-if="batch" :label="t('channel.edit.key')">
              <p class="channel-edit-field-hint">{{ t('channel.edit.batch_placeholder') }}</p>
              <a-textarea v-model:value="inputs.key" :rows="10" autocomplete="off" />
            </a-form-item>
            <a-form-item v-if="!isEdit">
              <a-checkbox v-model:checked="batch">{{ t('channel.edit.batch') }}</a-checkbox>
            </a-form-item>
          </div>

          <!-- Models section -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title">{{ t('channel.edit.section_models') }}</h4>
            <a-form-item :label="t('channel.edit.models')" required>
              <a-select
                mode="tags"
                show-search
                :token-separators="[',', ' ', '\n']"
                :placeholder="t('channel.edit.models_placeholder')"
                :value="inputs.models"
                :options="modelOptions"
                :field-names="{ label: 'text', value: 'value' }"
                :filter-option="filterModelOption"
                @change="(v) => handleInputChange('models', v)"
              >
                <template #tagRender="{ value, closable, onClose }">
                  <a-tag
                    :class="modelTagClass(value)"
                    :closable="closable"
                    :title="modelTagTitle(value)"
                    style="margin-right: 3px"
                    @close="onClose"
                    @click="copy(value)"
                  >
                    {{ value }}
                  </a-tag>
                </template>
              </a-select>
            </a-form-item>

            <div
              v-if="modelTestSummary || ((testModelsBusy || testModelsProgress.running) && testModelsProgress.total > 0)"
              class="channel-edit-model-test-block"
            >
              <p v-if="modelTestSummary" class="channel-edit-model-test-summary">
                {{ modelTestSummary }}
              </p>
              <a-progress
                v-if="(testModelsBusy || testModelsProgress.running) && testModelsProgress.total > 0"
                class="channel-edit-model-test-progress"
                :percent="testProgressPercent"
                :status="testModelsProgress.running && !testModelsProgress.paused ? 'active' : 'normal'"
              />
            </div>

            <div class="channel-edit-models-toolbar">
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
              <div class="channel-edit-model-test-concurrency">
                <label class="channel-edit-model-test-concurrency__label" for="channel-test-models-concurrency">
                  {{ t('channel.edit.buttons.test_models_concurrency') }}
                </label>
                <a-input-number
                  id="channel-test-models-concurrency"
                  :min="1"
                  :max="10"
                  :step="1"
                  :value="testModelsConcurrency"
                  :disabled="testModelsBusy || testModelsProgress.running"
                  @change="onConcurrencyChange"
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
              <a-button @click="handleInputChange('models', [])">
                {{ t('channel.edit.buttons.clear') }}
              </a-button>
            </div>

            <a-form-item :label="t('channel.edit.custom_models_bulk')">
              <p class="channel-edit-field-hint">{{ t('channel.edit.buttons.custom_placeholder') }}</p>
              <a-textarea :rows="4" v-model:value="customModel" />
              <a-button style="margin-top: 8px" @click="addCustomModel">
                {{ t('channel.edit.buttons.add_custom') }}
              </a-button>
            </a-form-item>

            <a-form-item :label="t('channel.edit.model_mapping')">
              <ChannelModelMappingEditor
                :value="inputs.model_mapping"
                :upstream-model-options="inputs.models"
                @change="(v) => handleInputChange('model_mapping', v)"
              />
            </a-form-item>

            <a-form-item :label="t('channel.edit.system_prompt')">
              <p class="channel-edit-field-hint">{{ t('channel.edit.system_prompt_placeholder') }}</p>
              <a-textarea :rows="10" v-model:value="inputs.system_prompt" />
            </a-form-item>

            <div class="channel-edit-section channel-edit-section--extended">
              <h4 class="channel-edit-section__title">{{ t('channel.edit.newapi_section') }}</h4>
              <a-form-item :label="t('channel.edit.openai_organization')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.openai_organization_hint') }}</p>
                <a-input
                  :placeholder="t('channel.edit.openai_organization_placeholder')"
                  v-model:value="inputs.openai_organization"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.test_model')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.test_model_hint') }}</p>
                <a-input
                  :placeholder="t('channel.edit.test_model_placeholder')"
                  v-model:value="inputs.test_model"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item>
                <p class="channel-edit-field-hint">{{ t('channel.edit.auto_ban_hint') }}</p>
                <a-switch :checked="inputs.auto_ban !== 0" @change="toggleAutoBan" />
                <span style="margin-left: 8px">{{ t('channel.edit.auto_ban') }}</span>
              </a-form-item>
              <a-form-item :label="t('channel.edit.remark')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.remark_hint') }}</p>
                <a-input
                  :placeholder="t('channel.edit.remark_placeholder')"
                  v-model:value="inputs.remark"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.tag')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.tag_hint') }}</p>
                <a-input
                  :placeholder="t('channel.edit.tag_placeholder')"
                  v-model:value="inputs.tag"
                  autocomplete="off"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.status_code_mapping')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.status_code_mapping_hint') }}</p>
                <a-textarea
                  :rows="4"
                  :placeholder="t('channel.edit.status_code_mapping_placeholder')"
                  v-model:value="inputs.status_code_mapping"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.param_override')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.param_override_hint') }}</p>
                <a-textarea
                  :rows="6"
                  :placeholder="t('channel.edit.param_override_placeholder')"
                  v-model:value="inputs.param_override"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.header_override')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.header_override_hint') }}</p>
                <a-textarea
                  :rows="4"
                  :placeholder="t('channel.edit.header_override_placeholder')"
                  v-model:value="inputs.header_override"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.setting_json')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.setting_json_hint') }}</p>
                <a-textarea
                  :rows="4"
                  :placeholder="t('channel.edit.setting_json_placeholder')"
                  v-model:value="inputs.setting"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.settings_json')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.settings_json_hint') }}</p>
                <a-textarea
                  :rows="4"
                  :placeholder="t('channel.edit.settings_json_placeholder')"
                  v-model:value="inputs.settings"
                />
              </a-form-item>
              <a-form-item :label="t('channel.edit.other_info')">
                <p class="channel-edit-field-hint">{{ t('channel.edit.other_info_hint') }}</p>
                <a-textarea
                  :rows="3"
                  :placeholder="t('channel.edit.other_info_placeholder')"
                  v-model:value="inputs.other_info"
                />
              </a-form-item>
            </div>
          </div>

          <!-- Routing advanced section -->
          <div class="channel-edit-section">
            <h4 class="channel-edit-section__title">{{ t('channel.edit.routing_advanced') }}</h4>
            <a-form-item :label="t('channel.edit.routing_provider')">
              <a-input
                :placeholder="t('channel.edit.routing_provider_placeholder')"
                :value="config.routing_provider"
                @update:value="(v) => handleConfigChange('routing_provider', v)"
              />
            </a-form-item>
            <a-form-item>
              <a-switch
                :checked="!!config.routing_skip_adaptive"
                @change="(checked) => handleConfigChange('routing_skip_adaptive', checked)"
              />
              <span style="margin-left: 8px">{{ t('channel.edit.routing_skip_adaptive') }}</span>
            </a-form-item>
          </div>

          <a-alert v-if="isEdit && !batch" type="info" style="margin-bottom: 12px">
            <template #description>
              <p>{{ t('channel.edit.load_key_hint') }}</p>
              <a-button size="small" @click="openLoadKeyModal('edit')">
                {{ t('channel.edit.load_key_button') }}
              </a-button>
            </template>
          </a-alert>

          <div class="channel-edit-actions">
            <a-button @click="handleCancel">{{ t('channel.edit.buttons.cancel') }}</a-button>
            <a-button type="primary" @click="submit">{{ t('channel.edit.buttons.submit') }}</a-button>
          </div>
        </a-form>
      </a-spin>
    </a-card>

    <a-modal
      :open="loadKeyOpen"
      width="480px"
      :title="t('channel.edit.load_key_modal_title')"
      @cancel="() => !loadKeyBusy && (loadKeyOpen = false)"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('channel.edit.load_key_modal_code')">
          <a-input v-model:value="loadKeyCode" autocomplete="off" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button :disabled="loadKeyBusy" @click="loadKeyOpen = false">
          {{ t('channel.edit.buttons.cancel') }}
        </a-button>
        <a-button type="primary" :loading="loadKeyBusy" @click="submitLoadKey">
          {{ t('channel.edit.load_key_modal_submit') }}
        </a-button>
      </template>
    </a-modal>

    <FillCatalogModelsModal
      :open="fillCatalogOpen"
      @close="fillCatalogOpen = false"
      @confirm="onFillCatalogConfirm"
    />

    <ChannelModelTestReportModal
      :open="modelTestReportOpen"
      :rows="modelTestReportRows"
      :total-models="(inputs.models || []).length"
      :channel-type-label="currentTypeLabel"
      :base-url="reportBaseUrl"
      @close="modelTestReportOpen = false"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import {
  API,
  copy,
  splitModelNameList,
  showError,
  showInfo,
  showSuccess,
  verifyJSON,
  verifyStepUp2FA,
  fetchChannelKeyAfterVerify,
  renderChannelTip,
} from '@/helpers';
import { EyeOutlined, EyeInvisibleOutlined, CopyOutlined } from '@ant-design/icons-vue';
import { setChannelTypeOptionsCache } from '@/helpers/helper';
import { fetchModelCatalogModelIds } from '@/helpers/modelCatalog';
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
  normalizeChannelTestBaseUrl,
  normalizeModelTestReportRows,
  renderChannelModelLabel,
  runChannelModelTestJob,
} from '@/helpers/channelModelTest';
import ChannelModelMappingEditor from './ChannelModelMappingEditor.vue';
import FillCatalogModelsModal from './FillCatalogModelsModal.vue';
import ChannelModelTestReportModal from './ChannelModelTestReportModal.vue';
import './ChannelEdit.css';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const channelId = route.params.id;
const isEdit = channelId !== undefined;

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

function defaultChannelInputs() {
  return {
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
  };
}

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

const loading = ref(isEdit);
const channelTypeOptions = ref([]);
const channelTypesLoading = ref(true);
const batch = ref(false);
const loadKeyOpen = ref(false);
const loadKeyCode = ref('');
const loadKeyBusy = ref(false);
const loadKeyIntent = ref('edit'); // 'edit' 载入输入框 | 'reveal' 查看 | 'copy' 复制
const keyMasked = ref(''); // 后端返回的脱敏预览串
const keyRevealed = ref(false); // 当前是否显示完整密钥
const revealedKey = ref(''); // 经 2FA 取回并缓存的完整密钥
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
const modelTestStatus = ref({});
const modelTestMessages = ref({});
const modelTestMeta = ref({});
const modelTestSummary = ref('');
const modelTestReportOpen = ref(false);
const modelTestReportRows = ref([]);
let testModelsPolling = false;

const inputs = reactive(defaultChannelInputs());
const originModelOptions = ref([]);
const modelOptions = ref([]);
const groupOptions = ref([]);
const customModel = ref('');
const config = reactive({ ...DEFAULT_CHANNEL_CONFIG });

const currentTypeOption = computed(() =>
  channelTypeOptions.value.find((o) => o.value === inputs.type)
);
const currentTypeDescription = computed(() => currentTypeOption.value?.description || '');
const currentTypeLabel = computed(() => currentTypeOption.value?.text || '');
// 当前渠道类型的内置模型（如高德 amap、AiPPT、深知 deep-research）：仅存在于适配器，不入模型目录表
const currentTypeBuiltinModels = computed(() =>
  (currentTypeOption.value?.builtin_models || [])
    .map((m) => String(m).trim())
    .filter(Boolean)
);
const reportBaseUrl = computed(() =>
  normalizeChannelTestBaseUrl(inputs.base_url, inputs.type, channelTypeOptions.value)
);

const failedModelCount = computed(
  () =>
    (inputs.models || []).filter((m) => modelTestStatus.value[String(m)] === MODEL_TEST_FAIL).length
);

const testProgressPercent = computed(() =>
  testModelsProgress.total
    ? Math.min(100, Math.round((testModelsProgress.completed / testModelsProgress.total) * 100))
    : 0
);

function filterTypeOption(input, option) {
  return String(option.text || '').toLowerCase().includes(String(input).toLowerCase());
}

function filterModelOption(input, option) {
  return String(option.value || '').toLowerCase().includes(String(input).toLowerCase());
}

function modelTagClass(value) {
  return renderChannelModelLabel(
    { value, text: value },
    modelTestStatus.value,
    modelTestMessages.value,
    modelTestMeta.value
  ).className;
}

function modelTagTitle(value) {
  return renderChannelModelLabel(
    { value, text: value },
    modelTestStatus.value,
    modelTestMessages.value,
    modelTestMeta.value
  ).title;
}

function handleCancel() {
  router.push('/channel');
}

function handleInputChange(name, value) {
  if (name === 'models') {
    const allowed = new Set((value || []).map((m) => String(m)));
    const keep = (obj) =>
      Object.fromEntries(Object.entries(obj || {}).filter(([key]) => allowed.has(key)));
    modelTestStatus.value = keep(modelTestStatus.value);
    modelTestMessages.value = keep(modelTestMessages.value);
    modelTestMeta.value = keep(modelTestMeta.value);
  }
  inputs[name] = value;
}

function handleTypeChange(value) {
  const opt = channelTypeOptions.value.find((o) => o.value === value);
  const defaultUrl = opt && opt.default_base_url ? String(opt.default_base_url).trim() : '';
  const typeChanged = inputs.type !== value;
  let base_url = inputs.base_url;
  if (typeChanged && defaultUrl) {
    const prevTrim = String(inputs.base_url || '').trim();
    const prevOpt = channelTypeOptions.value.find((o) => o.value === inputs.type);
    const prevDefault =
      prevOpt && prevOpt.default_base_url ? String(prevOpt.default_base_url).trim() : '';
    if (!prevTrim || prevTrim === prevDefault) {
      base_url = defaultUrl;
    }
  }
  inputs.type = value;
  inputs.base_url = base_url;
  // 切换到带内置模型的类型（高德/AiPPT/深知等）时，若模型为空则自动填入内置模型，省去手动输入
  if (typeChanged) {
    const builtins = (opt?.builtin_models || []).map((m) => String(m).trim()).filter(Boolean);
    if (builtins.length && (inputs.models || []).length === 0) {
      handleInputChange('models', [...new Set(builtins)]);
    }
  }
}

function handleConfigChange(name, value) {
  config[name] = value;
}

function toggleAutoBan() {
  inputs.auto_ban = inputs.auto_ban === 0 ? 1 : 0;
}

function onConcurrencyChange(value) {
  const n = parseInt(String(value), 10);
  testModelsConcurrency.value = Number.isFinite(n) ? Math.min(10, Math.max(1, n)) : 3;
}

async function loadChannel() {
  const res = await API.get(`/api/channel/${channelId}`);
  const { success, message, data } = res.data;
  if (success) {
    if (data.models === '') {
      data.models = [];
    } else {
      data.models = [
        ...new Set(
          data.models
            .split(',')
            .map((id) => id.trim())
            .filter(Boolean)
        ),
      ];
    }
    if (data.group === '') {
      data.groups = [];
    } else {
      data.groups = data.group.split(',');
    }
    if (data.model_mapping !== '') {
      data.model_mapping = JSON.stringify(JSON.parse(data.model_mapping), null, 2);
    }
    const { other: _legacyOther, key: maskedKey, ...channelData } = data;
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
    // 编辑时密钥字段始终留空（留空=不修改）；脱敏串只用于只读预览，绝不回填到可提交的输入框
    inputs.key = '';
    keyMasked.value = maskedKey || '';
    revealedKey.value = '';
    keyRevealed.value = false;
    if (data.config !== '' && data.config != null) {
      const cfg = JSON.parse(data.config);
      Object.assign(config, DEFAULT_CHANNEL_CONFIG, cfg);
    } else {
      Object.assign(config, DEFAULT_CHANNEL_CONFIG);
    }
  } else {
    showError(message);
  }
  loading.value = false;
}

function applyJobSnapshot(jobData, models) {
  const applied = applyJobSnapshotToModelState(jobData, models);
  modelTestStatus.value = applied.status;
  modelTestMessages.value = applied.messages;
  modelTestMeta.value = applied.meta;
  Object.assign(testModelsProgress, applied.progress);
  modelTestSummary.value = buildJobProgressSummary(t, jobData);
  modelTestReportRows.value = normalizeModelTestReportRows(jobData, []);
}

async function pollRunningJob(payload, models) {
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
}

async function loadStoredModelTestResults() {
  if (!isEdit || !channelId) return;
  try {
    const base = normalizeChannelTestBaseUrl(inputs.base_url, inputs.type, channelTypeOptions.value);
    const { results, job } = await fetchChannelModelTestResults({
      channelId,
      baseUrl: base,
      channelType: inputs.type,
      channelTypeOptions: channelTypeOptions.value,
    });
    if (job?.running) {
      applyJobSnapshot(job, inputs.models);
      const payload = buildChannelModelTestPayload({
        channelId,
        type: inputs.type,
        baseUrl: inputs.base_url,
        key: '',
        config,
        modelMapping: inputs.model_mapping,
        channelTypeOptions: channelTypeOptions.value,
        concurrency: testModelsConcurrency.value,
      });
      void pollRunningJob(payload, inputs.models);
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
}

watch(
  () => loading.value,
  (val) => {
    if (!val && isEdit) {
      void loadStoredModelTestResults();
    }
  }
);

async function fetchModels() {
  try {
    const ids = await fetchModelCatalogModelIds({});
    originModelOptions.value = ids.map((id) => ({ key: id, text: id, value: id }));
  } catch (error) {
    showError(error.message);
  }
}

async function fillAllCatalogModels() {
  fillAllBusy.value = true;
  try {
    const ids = await fetchModelCatalogModelIds({});
    if (!ids.length) {
      showInfo(t('channel.edit.fill_catalog_empty'));
      return;
    }
    handleInputChange('models', ids);
    showSuccess(t('channel.edit.fill_all_ok', { count: ids.length }));
  } catch (e) {
    showError(e.message || t('channel.edit.fill_catalog_fail'));
  } finally {
    fillAllBusy.value = false;
  }
}

async function fetchGroups() {
  try {
    const res = await API.get(`/api/group/`);
    groupOptions.value = res.data.data.map((group) => ({ key: group, text: group, value: group }));
  } catch (error) {
    showError(error.message);
  }
}

watch(
  [originModelOptions, () => inputs.models, currentTypeBuiltinModels],
  () => {
    const localModelOptions = [...originModelOptions.value];
    const pushOption = (model) => {
      if (!model) return;
      if (!localModelOptions.find((option) => option.key === model)) {
        localModelOptions.push({ key: model, text: model, value: model });
      }
    };
    // 内置模型并入下拉，保证选中类型（如高德）后 amap 可见、可选
    currentTypeBuiltinModels.value.forEach(pushOption);
    inputs.models.forEach(pushOption);
    modelOptions.value = localModelOptions;
  },
  { immediate: true }
);

onMounted(() => {
  (async () => {
    try {
      const res = await API.get('/api/model_catalog/editor_options');
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
  })();
  if (isEdit) {
    loadChannel();
  }
  fetchModels();
  fetchGroups();
});

async function submit() {
  if (inputs.key === '') {
    if (config.ak !== '' && config.sk !== '' && config.region !== '') {
      inputs.key = `${config.ak}|${config.sk}|${config.region}`;
    }
  }
  if (!isEdit && (inputs.name === '' || inputs.key === '')) {
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
  const localInputs = { ...inputs };
  if (localInputs.key === 'undefined|undefined|undefined') {
    localInputs.key = '';
  }
  if (localInputs.base_url && localInputs.base_url.endsWith('/')) {
    localInputs.base_url = localInputs.base_url.slice(0, localInputs.base_url.length - 1);
  }
  const cfg = { ...config };
  if ((localInputs.type === 14 || localInputs.type === 42) && !cfg.api_version) {
    cfg.api_version = 'v1';
  }
  let res;
  localInputs.models = localInputs.models.join(',');
  localInputs.group = localInputs.groups.join(',');
  localInputs.config = JSON.stringify(cfg);
  if (isEdit) {
    res = await API.put(`/api/channel/`, { ...localInputs, id: parseInt(channelId) });
  } else {
    res = await API.post(`/api/channel/`, localInputs);
  }
  const { success, message } = res.data;
  if (success) {
    if (isEdit) {
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
}

function addCustomModel() {
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
  handleInputChange('models', localModels);
}

function resolveKeyForUpstreamFetch() {
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
}

function mergeFetchedUpstreamModels(ids) {
  if (!Array.isArray(ids) || ids.length === 0) {
    showInfo(t('channel.edit.messages.upstream_fetch_empty'));
    return;
  }
  const dedup = [...new Set(ids.map((x) => String(x).trim()).filter(Boolean))];
  const localModels = [...new Set([...inputs.models, ...dedup])];
  const optKeys = new Set(originModelOptions.value.map((o) => o.key));
  const newOpts = [...originModelOptions.value];
  for (const id of dedup) {
    if (!optKeys.has(id)) {
      optKeys.add(id);
      newOpts.push({ key: id, text: id, value: id });
    }
  }
  originModelOptions.value = newOpts;
  handleInputChange('models', localModels);
  showSuccess(t('channel.edit.messages.upstream_fetch_success', { count: dedup.length }));
}

async function fetchUpstreamModelsList() {
  const keyFromForm = resolveKeyForUpstreamFetch();
  if (!keyFromForm && !(isEdit && channelId)) {
    showInfo(t('channel.edit.messages.upstream_fetch_need_key'));
    return;
  }
  fetchUpstreamBusy.value = true;
  try {
    const cfg = { ...config };
    if ((inputs.type === 14 || inputs.type === 42) && !cfg.api_version) {
      cfg.api_version = 'v1';
    }
    let baseUrl = String(inputs.base_url || '').trim();
    if (baseUrl.endsWith('/')) {
      baseUrl = baseUrl.slice(0, baseUrl.length - 1);
    }
    const res = await API.post('/api/channel/fetch_upstream_models_preview', {
      type: inputs.type,
      base_url: baseUrl,
      key: keyFromForm,
      config: JSON.stringify(cfg),
      channel_id: isEdit ? Number(channelId) || 0 : 0,
    });
    const { success, message, data } = res.data;
    if (!success) {
      showError(message || '请求失败');
      return;
    }
    mergeFetchedUpstreamModels(data);
  } catch (e) {
    showError(e.message || '请求失败');
  } finally {
    fetchUpstreamBusy.value = false;
  }
}

async function testAllModels() {
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
  if (!keyFromForm && !(isEdit && channelId)) {
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
    channelId: isEdit ? channelId : 0,
    type: inputs.type,
    baseUrl: inputs.base_url,
    key: keyFromForm,
    config,
    modelMapping: inputs.model_mapping,
    channelTypeOptions: channelTypeOptions.value,
    concurrency: testModelsConcurrency.value,
  });
  await pollRunningJob(payload, models);
}

async function controlRunningTestJob(action) {
  if (!testModelsProgress.running || testModelsControlBusy.value) return;
  testModelsControlBusy.value = true;
  try {
    const payload = buildChannelModelTestPayload({
      channelId: isEdit ? channelId : 0,
      type: inputs.type,
      baseUrl: inputs.base_url,
      key: '',
      config,
      modelMapping: inputs.model_mapping,
      channelTypeOptions: channelTypeOptions.value,
      concurrency: testModelsConcurrency.value,
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
}

function clearInvalidModels() {
  const { failed, remaining } = filterOutFailedModels(inputs.models, modelTestStatus.value);
  if (!failed.length) {
    showInfo(t('channel.edit.messages.clear_failed_empty'));
    return;
  }
  handleInputChange('models', remaining);
  showSuccess(t('channel.edit.messages.clear_failed_done', { count: failed.length }));
}

async function openLoadKeyModal(intent = 'edit') {
  try {
    const res = await API.get('/api/user/2fa/status');
    if (!res.data?.success || !res.data?.data?.enabled) {
      showInfo(t('channel.edit.load_key_need_2fa'));
      return;
    }
    loadKeyIntent.value = intent;
    loadKeyCode.value = '';
    loadKeyOpen.value = true;
  } catch (e) {
    showError(e.message || '无法检查两步验证状态');
  }
}

async function submitLoadKey() {
  if (!loadKeyCode.value.trim()) {
    showInfo(t('channel.edit.load_key_code_required'));
    return;
  }
  loadKeyBusy.value = true;
  try {
    await verifyStepUp2FA(loadKeyCode.value);
    const key = await fetchChannelKeyAfterVerify(channelId);
    if (loadKeyIntent.value === 'reveal') {
      revealedKey.value = key;
      keyRevealed.value = true;
      showSuccess(t('channel.edit.key_reveal_success'));
    } else if (loadKeyIntent.value === 'copy') {
      revealedKey.value = key;
      const ok = await copy(key);
      ok ? showSuccess(t('channel.edit.key_copy_success')) : showError(t('channel.edit.key_copy_fail'));
    } else {
      inputs.key = key;
      showSuccess(t('channel.edit.load_key_success'));
    }
    loadKeyOpen.value = false;
    loadKeyCode.value = '';
  } catch (e) {
    showError(e.message || '加载失败');
  } finally {
    loadKeyBusy.value = false;
  }
}

// 查看完整密钥：已缓存则直接切换显隐，否则走 2FA 取回
async function toggleRevealKey() {
  if (keyRevealed.value) {
    keyRevealed.value = false;
    return;
  }
  if (revealedKey.value) {
    keyRevealed.value = true;
    return;
  }
  await openLoadKeyModal('reveal');
}

// 复制完整密钥：已缓存则直接复制，否则走 2FA 取回后复制
async function copyConfiguredKey() {
  if (revealedKey.value) {
    const ok = await copy(revealedKey.value);
    ok ? showSuccess(t('channel.edit.key_copy_success')) : showError(t('channel.edit.key_copy_fail'));
    return;
  }
  await openLoadKeyModal('copy');
}

function onFillCatalogConfirm(ids) {
  handleInputChange('models', ids);
  showSuccess(t('channel.edit.fill_catalog_ok', { count: ids.length }));
}
</script>
