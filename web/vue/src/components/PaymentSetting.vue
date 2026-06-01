<template>
  <div class="payment-setting">
    <a-alert type="info" show-icon :message="t('payment.note')" style="margin-bottom: 16px" />

    <!-- 微信支付全局配置 -->
    <a-card :title="t('payment.wechat.title')" size="small" style="margin-bottom: 16px">
      <a-form layout="vertical" :autocomplete="noAutofillFormProps.autocomplete">
        <a-form-item>
          <a-switch v-model:checked="wxEnabled" @change="saveEnabled" />
          <span style="margin-left: 8px">{{ t('payment.wechat.enabled') }}</span>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.appid')">
              <a-input
                v-model:value="inputs.WeChatPayAppId"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillTextProps"
              />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.mchid')">
              <a-input
                v-model:value="inputs.WeChatPayMchId"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillTextProps"
              />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.serial')">
              <a-input
                v-model:value="inputs.WeChatPayCertSerialNo"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillTextProps"
              />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.apiv3')">
              <a-input-password
                v-model:value="inputs.WeChatPayApiV3Key"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillSecretProps"
              />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.notify_domain')">
              <a-input
                v-model:value="inputs.WeChatPayNotifyDomain"
                placeholder="https://pay.example.com"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillTextProps"
              />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.quota_per_yuan')">
              <a-input
                v-model:value="inputs.WeChatPayQuotaPerYuan"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillTextProps"
              />
            </a-form-item>
          </a-col>
          <a-col :span="24">
            <a-form-item :label="t('payment.wechat.private_key')">
              <a-textarea
                v-model:value="inputs.WeChatPayPrivateKey"
                :rows="6"
                placeholder="-----BEGIN PRIVATE KEY-----"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillSecretProps"
              />
            </a-form-item>
          </a-col>
        </a-row>
        <a-button type="primary" :loading="saving" @click="saveWechat">{{ t('payment.wechat.save') }}</a-button>
      </a-form>
    </a-card>

    <!-- 渠道授权（默认全关） -->
    <a-card :title="t('payment.access.title')" size="small">
      <p class="payment-hint">{{ t('payment.access.hint') }}</p>
      <a-space style="margin-bottom: 12px" wrap>
        <a-select v-model:value="grant.scope_type" style="width: 120px">
          <a-select-option value="user">{{ t('payment.access.scope_user') }}</a-select-option>
          <a-select-option value="tenant">{{ t('payment.access.scope_tenant') }}</a-select-option>
        </a-select>
        <a-input-number v-model:value="grant.scope_id" :min="1" :precision="0" :placeholder="t('payment.access.scope_id')" style="width: 140px" />
        <a-checkbox-group v-model:value="grant.channels" :options="channelOptions" />
        <a-button type="primary" :loading="granting" @click="saveGrant">{{ t('payment.access.save') }}</a-button>
      </a-space>

      <a-table
        :data-source="grants"
        :columns="columns"
        :pagination="false"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'scope_type'">
            {{ record.scope_type === 'tenant' ? t('payment.access.scope_tenant') : t('payment.access.scope_user') }}
          </template>
          <template v-else-if="column.key === 'channels'">
            <a-tag v-for="ch in (record.channels ? record.channels.split(',') : [])" :key="ch" color="green">{{ ch }}</a-tag>
            <span v-if="!record.channels" style="color: rgba(0,0,0,0.35)">{{ t('payment.access.none') }}</span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a @click="editGrant(record)">{{ t('payment.access.edit') }}</a>
              <a-popconfirm :title="t('payment.access.delete_confirm')" @confirm="deleteGrant(record)">
                <a style="color: #ff4d4f">{{ t('payment.access.delete') }}</a>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  API,
  showError,
  showSuccess,
  noAutofillFormProps,
  noAutofillTextProps,
  noAutofillSecretProps,
} from '@/helpers';

const { t } = useI18n();

// 抑制 Chrome「文本框+密码框=登录表单」的自动填充（如把账号/密码塞进证书序列号/APIv3 密钥）。
const autofillUnlocked = ref(false);

const WECHAT_KEYS = [
  'WeChatPayAppId',
  'WeChatPayMchId',
  'WeChatPayCertSerialNo',
  'WeChatPayApiV3Key',
  'WeChatPayNotifyDomain',
  'WeChatPayQuotaPerYuan',
  'WeChatPayPrivateKey',
];

const inputs = reactive({
  WeChatPayAppId: '',
  WeChatPayMchId: '',
  WeChatPayCertSerialNo: '',
  WeChatPayApiV3Key: '',
  WeChatPayNotifyDomain: '',
  WeChatPayQuotaPerYuan: '500000',
  WeChatPayPrivateKey: '',
});
const wxEnabled = ref(false);
const saving = ref(false);

const allChannels = ref(['wxpay']);
const channelOptions = computed(() => allChannels.value.map((c) => ({ label: c, value: c })));

const grants = ref([]);
const granting = ref(false);
const grant = reactive({ scope_type: 'user', scope_id: null, channels: [] });

const columns = [
  { title: t('payment.access.col_scope'), key: 'scope_type' },
  { title: 'ID', dataIndex: 'scope_id', key: 'scope_id' },
  { title: t('payment.access.col_channels'), key: 'channels' },
  { title: t('payment.access.col_action'), key: 'action', width: 140 },
];

const getOptions = async () => {
  const res = await API.get('/api/option/');
  const { success, message, data } = res.data;
  if (!success) {
    showError(message);
    return;
  }
  data.forEach((item) => {
    if (item.key === 'WeChatPayEnabled') {
      wxEnabled.value = item.value === 'true';
    } else if (item.key in inputs) {
      inputs[item.key] = item.value;
    }
  });
};

const updateOption = async (key, value) => {
  const res = await API.put('/api/option/', { key, value });
  const { success, message } = res.data;
  if (!success) showError(message);
  return success;
};

const saveEnabled = async (checked) => {
  await updateOption('WeChatPayEnabled', checked ? 'true' : 'false');
};

const saveWechat = async () => {
  saving.value = true;
  try {
    for (const key of WECHAT_KEYS) {
      await updateOption(key, inputs[key] ?? '');
    }
    showSuccess(t('payment.wechat.saved'));
  } finally {
    saving.value = false;
  }
};

const loadGrants = async () => {
  const res = await API.get('/api/payment/access');
  const { success, message, data } = res.data;
  if (!success) {
    showError(message);
    return;
  }
  grants.value = (data && data.items) || [];
  if (data && data.all_channels && data.all_channels.length) {
    allChannels.value = data.all_channels;
  }
};

const saveGrant = async () => {
  if (!grant.scope_id || grant.scope_id <= 0) {
    showError(t('payment.access.scope_id_required'));
    return;
  }
  granting.value = true;
  try {
    const res = await API.put('/api/payment/access', {
      scope_type: grant.scope_type,
      scope_id: grant.scope_id,
      channels: grant.channels,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('payment.access.saved'));
      grant.channels = [];
      grant.scope_id = null;
      await loadGrants();
    } else {
      showError(message);
    }
  } finally {
    granting.value = false;
  }
};

const editGrant = (record) => {
  grant.scope_type = record.scope_type;
  grant.scope_id = record.scope_id;
  grant.channels = record.channels ? record.channels.split(',') : [];
};

const deleteGrant = async (record) => {
  const res = await API.delete('/api/payment/access', {
    params: { scope_type: record.scope_type, scope_id: record.scope_id },
  });
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('payment.access.deleted'));
    await loadGrants();
  } else {
    showError(message);
  }
};

onMounted(() => {
  getOptions();
  loadGrants();
});
</script>

<style scoped>
.payment-hint {
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 12px;
}
</style>
