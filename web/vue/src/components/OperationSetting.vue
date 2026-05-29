<template>
  <a-spin :spinning="loading">
    <a-form layout="vertical" @submit.prevent>
      <h3>{{ t('setting.operation.quota.title') }}</h3>
      <div class="settings-grid">
        <a-form-item :label="t('setting.operation.quota.new_user')">
          <a-input
            v-model:value="inputs.QuotaForNewUser"
            type="number"
            min="0"
            :placeholder="t('setting.operation.quota.new_user_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.quota.pre_consume')">
          <a-input
            v-model:value="inputs.PreConsumedQuota"
            type="number"
            min="0"
            :placeholder="t('setting.operation.quota.pre_consume_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.quota.inviter_reward')">
          <a-input
            v-model:value="inputs.QuotaForInviter"
            type="number"
            min="0"
            :placeholder="t('setting.operation.quota.inviter_reward_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.quota.invitee_reward')">
          <a-input
            v-model:value="inputs.QuotaForInvitee"
            type="number"
            min="0"
            :placeholder="t('setting.operation.quota.invitee_reward_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
      </div>
      <a-button @click="submitConfig('quota')">
        {{ t('setting.operation.quota.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.operation.ratio.title') }}</h3>
      <div class="pricing-entry-blocks-toolbar">
        <a-button size="small" @click="expandAllBlocks">
          {{ t('setting.operation.ratio.editor.expand_all') }}
        </a-button>
        <a-button size="small" @click="collapseAllBlocks">
          {{ t('setting.operation.ratio.editor.collapse_all') }}
        </a-button>
      </div>
      <PricingEntryBlockPanel
        v-for="block in RATIO_BLOCK_DEFS"
        :key="block.blockId"
        :title="t(block.titleKey)"
        :expanded="!!expandedBlocks[block.blockId]"
        :entry-count="blockCounts[block.blockId] ?? null"
        @toggle="toggleBlock(block.blockId)"
      >
        <template v-if="block.helpContent" #help>
          <BlockHelpTrigger :content="block.helpContent" />
        </template>
        <PricingEntryEditor
          :block-id="block.blockId"
          :value-kind="block.valueKind"
          :key-column-label="t(block.keyLabelKey)"
          :sub-key-column-label="block.subKeyLabelKey ? t(block.subKeyLabelKey) : undefined"
          :key-options="block.keyOptions === 'group' ? mergedGroupOptions : mergedModelOptions"
          :key-dropdown="lookupReady"
          :on-mutate="loadBlockCounts"
        />
      </PricingEntryBlockPanel>
      <a-form-item>
        <a-checkbox
          :checked="inputs.ExposeRatioEnabled === 'true'"
          @change="updateOption('ExposeRatioEnabled')"
        >
          {{ t('setting.operation.ratio.expose_enabled') }}
        </a-checkbox>
      </a-form-item>
      <a-button @click="submitConfig('ratio')">
        {{ t('setting.operation.ratio.buttons.save_expose') }}
      </a-button>
      <UpstreamRatioSync />

      <a-divider />
      <h3>{{ t('setting.operation.log.title') }}</h3>
      <a-form-item>
        <a-checkbox
          :checked="inputs.LogConsumeEnabled === 'true'"
          @change="updateOption('LogConsumeEnabled')"
        >
          {{ t('setting.operation.log.enable_consume') }}
        </a-checkbox>
        <a-checkbox
          :checked="inputs.ErrorLogEnabled === 'true'"
          style="margin-left: 1rem"
          @change="updateOption('ErrorLogEnabled')"
        >
          {{ t('setting.operation.log.enable_error') }}
        </a-checkbox>
      </a-form-item>
      <div class="settings-grid">
        <a-form-item :label="t('setting.operation.log.target_time')">
          <a-input v-model:value="historyTimestamp" type="datetime-local" />
        </a-form-item>
      </div>
      <a-button @click="deleteHistoryLogs">
        {{ t('setting.operation.log.buttons.clean') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.operation.monitor.title') }}</h3>
      <div class="settings-grid">
        <a-form-item :label="t('setting.operation.monitor.max_response_time')">
          <a-input
            v-model:value="inputs.ChannelDisableThreshold"
            type="number"
            min="0"
            :placeholder="t('setting.operation.monitor.max_response_time_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.monitor.quota_reminder')">
          <a-input
            v-model:value="inputs.QuotaRemindThreshold"
            type="number"
            min="0"
            :placeholder="t('setting.operation.monitor.quota_reminder_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
      </div>
      <a-form-item>
        <a-checkbox
          :checked="inputs.AutomaticDisableChannelEnabled === 'true'"
          @change="updateOption('AutomaticDisableChannelEnabled')"
        >
          {{ t('setting.operation.monitor.auto_disable') }}
        </a-checkbox>
        <a-checkbox
          :checked="inputs.AutomaticEnableChannelEnabled === 'true'"
          style="margin-left: 1rem"
          @change="updateOption('AutomaticEnableChannelEnabled')"
        >
          {{ t('setting.operation.monitor.auto_enable') }}
        </a-checkbox>
      </a-form-item>
      <a-button @click="submitConfig('monitor')">
        {{ t('setting.operation.monitor.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.operation.general.title') }}</h3>
      <div class="settings-grid">
        <a-form-item :label="t('setting.operation.general.topup_link')">
          <a-input
            v-model:value="inputs.TopUpLink"
            :placeholder="t('setting.operation.general.topup_link_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.general.chat_link')">
          <a-input
            v-model:value="inputs.ChatLink"
            :placeholder="t('setting.operation.general.chat_link_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.general.quota_per_unit')">
          <a-input
            v-model:value="inputs.QuotaPerUnit"
            type="number"
            step="0.01"
            :placeholder="t('setting.operation.general.quota_per_unit_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
        <a-form-item label="租户私有渠道默认千次调用单价">
          <a-input
            v-model:value="inputs.DefaultTenantChannelPricePer1k"
            type="number"
            step="0.001"
            placeholder="未配置专属单价时，系统默认千次调用单价"
          />
        </a-form-item>
        <a-form-item :label="t('setting.operation.general.retry_times')">
          <a-input
            v-model:value="inputs.RetryTimes"
            type="number"
            step="1"
            min="0"
            :placeholder="t('setting.operation.general.retry_times_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
      </div>
      <a-form-item>
        <a-checkbox
          :checked="inputs.DisplayInCurrencyEnabled === 'true'"
          @change="updateOption('DisplayInCurrencyEnabled')"
        >
          {{ t('setting.operation.general.display_in_currency') }}
        </a-checkbox>
        <a-checkbox
          :checked="inputs.DisplayTokenStatEnabled === 'true'"
          style="margin-left: 1rem"
          @change="updateOption('DisplayTokenStatEnabled')"
        >
          {{ t('setting.operation.general.display_token_stat') }}
        </a-checkbox>
        <a-checkbox
          :checked="inputs.ApproximateTokenEnabled === 'true'"
          style="margin-left: 1rem"
          @change="updateOption('ApproximateTokenEnabled')"
        >
          {{ t('setting.operation.general.approximate_token') }}
        </a-checkbox>
      </a-form-item>
      <a-button @click="submitConfig('general')">
        {{ t('setting.operation.general.buttons.save') }}
      </a-button>
    </a-form>
  </a-spin>
</template>

<script setup>
import { computed, h, onMounted, onBeforeUnmount, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Popover } from 'ant-design-vue';
import { QuestionCircleOutlined } from '@ant-design/icons-vue';
import {
  API,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
  noAutofillSecretProps,
  noAutofillTextProps,
  fetchOperationGroupOptions,
  fetchOperationModelOptions,
} from '@/helpers';
import PricingEntryEditor from '@/components/PricingEntryEditor.vue';
import PricingEntryBlockPanel from '@/components/PricingEntryBlockPanel.vue';
import UpstreamRatioSync from '@/components/UpstreamRatioSync.vue';
import {
  BillingExprHelpContent,
  BillingModeHelpContent,
} from '@/components/OperationFieldHelp.vue';

const { t } = useI18n();

const RATIO_BLOCK_DEFS = [
  {
    blockId: 'model_ratio',
    titleKey: 'setting.operation.ratio.model.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'completion_ratio',
    titleKey: 'setting.operation.ratio.completion.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'model_price',
    titleKey: 'setting.operation.ratio.model_price.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'cache_ratio',
    titleKey: 'setting.operation.ratio.cache.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'create_cache_ratio',
    titleKey: 'setting.operation.ratio.create_cache.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'image_ratio',
    titleKey: 'setting.operation.ratio.image.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'audio_ratio',
    titleKey: 'setting.operation.ratio.audio.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'audio_completion_ratio',
    titleKey: 'setting.operation.ratio.audio_completion.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'billing_mode',
    titleKey: 'setting.operation.ratio.billing_mode.title',
    valueKind: 'string',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
    helpContent: BillingModeHelpContent,
  },
  {
    blockId: 'billing_expr',
    titleKey: 'setting.operation.ratio.billing_expr.title',
    valueKind: 'string',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
    helpContent: BillingExprHelpContent,
  },
  {
    blockId: 'group_ratio',
    titleKey: 'setting.operation.ratio.group.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_group',
    keyOptions: 'group',
  },
  {
    blockId: 'group_group_ratio',
    titleKey: 'setting.operation.ratio.group_group.title',
    valueKind: 'group_group',
    keyLabelKey: 'setting.operation.ratio.group_group.col_user_group',
    subKeyLabelKey: 'setting.operation.ratio.group_group.col_use_group',
    keyOptions: 'group',
  },
  {
    blockId: 'topup_group_ratio',
    titleKey: 'setting.operation.ratio.topup_group.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_group',
    keyOptions: 'group',
  },
];

// Help-trigger as a functional component using a-popover.
const BlockHelpTrigger = (props) => {
  if (!props.content) return null;
  return h(
    Popover,
    {
      placement: 'topLeft',
      overlayClassName: 'operation-billing-help-popup-wrap',
    },
    {
      content: () =>
        h('div', { class: 'operation-billing-help-popup' }, [h(props.content)]),
      default: () =>
        h(
          'span',
          {
            class: 'operation-field-help-trigger',
            role: 'button',
            tabindex: 0,
            onClick: (e) => e.stopPropagation(),
            onKeydown: (e) => {
              if (e.key === 'Enter' || e.key === ' ') e.stopPropagation();
            },
          },
          [h(QuestionCircleOutlined, { class: 'operation-field-help-icon', 'aria-label': 'help' })]
        ),
    }
  );
};
BlockHelpTrigger.props = ['content'];

const now = new Date();
const inputs = reactive({
  QuotaForNewUser: 0,
  QuotaForInviter: 0,
  QuotaForInvitee: 0,
  QuotaRemindThreshold: 0,
  PreConsumedQuota: 0,
  ModelRatio: '',
  ModelPrice: '',
  CompletionRatio: '',
  CacheRatio: '',
  CreateCacheRatio: '',
  ImageRatio: '',
  AudioRatio: '',
  AudioCompletionRatio: '',
  BillingMode: '',
  BillingExpr: '',
  GroupRatio: '',
  GroupGroupRatio: '',
  TopupGroupRatio: '',
  ExposeRatioEnabled: '',
  TopUpLink: '',
  ChatLink: '',
  QuotaPerUnit: 0,
  DefaultTenantChannelPricePer1k: 0,
  AutomaticDisableChannelEnabled: '',
  AutomaticEnableChannelEnabled: '',
  ChannelDisableThreshold: 0,
  LogConsumeEnabled: '',
  ErrorLogEnabled: 'true',
  DisplayInCurrencyEnabled: '',
  DisplayTokenStatEnabled: '',
  ApproximateTokenEnabled: '',
  RetryTimes: 0,
});
const originInputs = reactive({});
const modelOptions = ref([]);
const groupOptions = ref([]);
const lookupReady = ref(false);
const expandedBlocks = ref({});
const blockCounts = ref({});
const loading = ref(false);
const historyTimestamp = ref(
  timestamp2string(now.getTime() / 1000 - 30 * 24 * 3600)
);

const mergedModelOptions = computed(() => modelOptions.value);
const mergedGroupOptions = computed(() => groupOptions.value);
const ratioBlockIds = RATIO_BLOCK_DEFS.map((b) => b.blockId);

const loadBlockCounts = async () => {
  try {
    const res = await API.get('/api/pricing_entries/blocks');
    const body = res.data || {};
    if (!body.success) return;
    const rows = Array.isArray(body.data) ? body.data : [];
    const next = {};
    rows.forEach((row) => {
      if (row?.block_id != null) {
        next[row.block_id] = Number(row.entry_count) || 0;
      }
    });
    blockCounts.value = next;
  } catch {
    /* ignore */
  }
};

const toggleBlock = (blockId) => {
  expandedBlocks.value = {
    ...expandedBlocks.value,
    [blockId]: !expandedBlocks.value[blockId],
  };
};

const expandAllBlocks = () => {
  const next = {};
  ratioBlockIds.forEach((id) => {
    next[id] = true;
  });
  expandedBlocks.value = next;
};

const collapseAllBlocks = () => {
  expandedBlocks.value = {};
};

const getOptions = async () => {
  const res = await API.get('/api/option/');
  const { success, message, data } = res.data;
  if (success) {
    const newInputs = {};
    data.forEach((item) => {
      let value = item.value;
      if (value === '{}') value = '';
      newInputs[item.key] = value;
    });
    const ratioDefaults = { ExposeRatioEnabled: 'false' };
    const merged = { ...ratioDefaults, ...newInputs };
    Object.assign(inputs, merged);
    Object.keys(originInputs).forEach((k) => delete originInputs[k]);
    Object.assign(originInputs, merged);
  } else {
    showError(message);
  }
};

const updateOption = async (key, value) => {
  loading.value = true;
  if (key.endsWith('Enabled')) {
    value = inputs[key] === 'true' ? 'false' : 'true';
  }
  try {
    const res = await API.put('/api/option/', { key, value });
    const { success, message } = res.data;
    if (success) {
      inputs[key] = value;
      originInputs[key] = value;
      return true;
    }
    showError(message);
    return false;
  } catch (e) {
    showError(e.message || '保存失败');
    return false;
  } finally {
    loading.value = false;
  }
};

const submitConfig = async (group) => {
  switch (group) {
    case 'monitor':
      if (originInputs.ChannelDisableThreshold !== inputs.ChannelDisableThreshold) {
        await updateOption('ChannelDisableThreshold', inputs.ChannelDisableThreshold);
      }
      if (originInputs.QuotaRemindThreshold !== inputs.QuotaRemindThreshold) {
        await updateOption('QuotaRemindThreshold', inputs.QuotaRemindThreshold);
      }
      break;
    case 'ratio': {
      let savedCount = 0;
      if (originInputs.ExposeRatioEnabled !== inputs.ExposeRatioEnabled) {
        const ok = await updateOption(
          'ExposeRatioEnabled',
          inputs.ExposeRatioEnabled === 'true' ? 'true' : 'false'
        );
        if (!ok) return;
        savedCount += 1;
      }
      if (savedCount > 0) {
        showSuccess(t('setting.operation.ratio.save_success'));
      } else {
        showWarning(t('setting.operation.ratio.save_no_changes'));
      }
      break;
    }
    case 'quota':
      if (originInputs.QuotaForNewUser !== inputs.QuotaForNewUser) {
        await updateOption('QuotaForNewUser', inputs.QuotaForNewUser);
      }
      if (originInputs.QuotaForInvitee !== inputs.QuotaForInvitee) {
        await updateOption('QuotaForInvitee', inputs.QuotaForInvitee);
      }
      if (originInputs.QuotaForInviter !== inputs.QuotaForInviter) {
        await updateOption('QuotaForInviter', inputs.QuotaForInviter);
      }
      if (originInputs.PreConsumedQuota !== inputs.PreConsumedQuota) {
        await updateOption('PreConsumedQuota', inputs.PreConsumedQuota);
      }
      break;
    case 'general':
      if (originInputs.TopUpLink !== inputs.TopUpLink) {
        await updateOption('TopUpLink', inputs.TopUpLink);
      }
      if (originInputs.ChatLink !== inputs.ChatLink) {
        await updateOption('ChatLink', inputs.ChatLink);
      }
      if (originInputs.QuotaPerUnit !== inputs.QuotaPerUnit) {
        await updateOption('QuotaPerUnit', inputs.QuotaPerUnit);
      }
      if (
        originInputs.DefaultTenantChannelPricePer1k !==
        inputs.DefaultTenantChannelPricePer1k
      ) {
        await updateOption(
          'DefaultTenantChannelPricePer1k',
          inputs.DefaultTenantChannelPricePer1k
        );
      }
      if (originInputs.RetryTimes !== inputs.RetryTimes) {
        await updateOption('RetryTimes', inputs.RetryTimes);
      }
      break;
    default:
      break;
  }
};

const deleteHistoryLogs = async () => {
  const res = await API.delete(
    `/api/log/?target_timestamp=${Date.parse(historyTimestamp.value) / 1000}`
  );
  const { success, message, data } = res.data;
  if (success) {
    showSuccess(`${data} 条日志已清理！`);
    return;
  }
  showError('日志清理失败：' + message);
};

const onPricingApplied = () => {
  loadBlockCounts();
};

onMounted(() => {
  (async () => {
    try {
      const [models, groups] = await Promise.all([
        fetchOperationModelOptions(),
        fetchOperationGroupOptions(),
      ]);
      modelOptions.value = models;
      groupOptions.value = groups;
      lookupReady.value = true;
    } catch (e) {
      showError(e.message || 'failed to load model/group options');
    }
  })();
  getOptions();
  loadBlockCounts();
  window.addEventListener('one-api-pricing-applied', onPricingApplied);
});

onBeforeUnmount(() => {
  window.removeEventListener('one-api-pricing-applied', onPricingApplied);
});
</script>

<style scoped>
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0 1rem;
}

h3 {
  margin-top: 0.5rem;
  margin-bottom: 0.75rem;
}
</style>
