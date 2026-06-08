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
          <a-col :xs="24" :md="12">
            <a-form-item :label="t('payment.wechat.discount')">
              <a-input
                v-model:value="inputs.WeChatPayDiscount"
                :readonly="!autofillUnlocked"
                @focus="autofillUnlocked = true"
                v-bind="noAutofillTextProps"
              />
              <div class="payment-hint" style="margin-top: 4px">{{ t('payment.wechat.discount_hint') }}</div>
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

    <!-- 支付折扣规则 -->
    <a-card :title="t('payment.discount.title')" size="small" style="margin-top: 16px">
      <p class="payment-hint">
        {{ t('payment.discount.hint') }}
        <template v-if="discountGlobal !== null"> · {{ t('payment.discount.global') }}: {{ discountGlobal }}</template>
      </p>
      <a-space style="margin-bottom: 12px" wrap>
        <a-select v-model:value="drule.match_type" style="width: 130px">
          <a-select-option value="user">{{ t('payment.discount.by_user') }}</a-select-option>
          <a-select-option value="group">{{ t('payment.discount.by_group') }}</a-select-option>
          <a-select-option value="tag">{{ t('payment.discount.by_tag') }}</a-select-option>
        </a-select>
        <a-input v-model:value="drule.match_value" :placeholder="t('payment.discount.value_ph')" style="width: 200px" />
        <a-input-number v-model:value="drule.discount" :min="0.0001" :max="1" :step="0.01" :placeholder="t('payment.discount.discount_ph')" style="width: 140px" />
        <a-button type="primary" :loading="dsaving" @click="saveDiscountRule">{{ t('payment.discount.save') }}</a-button>
      </a-space>

      <a-table :data-source="discountRules" :columns="dcolumns" :pagination="false" row-key="id" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'match_type'">
            {{ record.match_type === 'user' ? t('payment.discount.by_user') : record.match_type === 'group' ? t('payment.discount.by_group') : t('payment.discount.by_tag') }}
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-tag :color="record.enabled ? 'green' : 'default'">{{ record.enabled ? t('payment.discount.on') : t('payment.discount.off') }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a @click="editDiscountRule(record)">{{ t('payment.access.edit') }}</a>
              <a-popconfirm :title="t('payment.access.delete_confirm')" @confirm="deleteDiscountRule(record)">
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
  'WeChatPayDiscount',
  'WeChatPayPrivateKey',
];

const inputs = reactive({
  WeChatPayAppId: '',
  WeChatPayMchId: '',
  WeChatPayCertSerialNo: '',
  WeChatPayApiV3Key: '',
  WeChatPayNotifyDomain: '',
  WeChatPayQuotaPerYuan: '500000',
  WeChatPayDiscount: '1',
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

// 折扣规则
const discountRules = ref([]);
const discountGlobal = ref(null);
const dsaving = ref(false);
const drule = reactive({ match_type: 'user', match_value: '', discount: null });
const dcolumns = [
  { title: t('payment.discount.col_type'), key: 'match_type' },
  { title: t('payment.discount.col_value'), dataIndex: 'match_value', key: 'match_value' },
  { title: t('payment.discount.col_discount'), dataIndex: 'discount', key: 'discount' },
  { title: t('payment.discount.col_enabled'), key: 'enabled' },
  { title: t('payment.access.col_action'), key: 'action', width: 140 },
];

const loadDiscount = async () => {
  const res = await API.get('/api/payment/discount');
  const { success, message, data } = res.data;
  if (!success) {
    showError(message);
    return;
  }
  discountRules.value = (data && data.items) || [];
  if (data && typeof data.global === 'number') discountGlobal.value = data.global;
};

const saveDiscountRule = async () => {
  if (!drule.match_value || !drule.match_value.trim()) {
    showError(t('payment.discount.value_required'));
    return;
  }
  if (!drule.discount || drule.discount <= 0 || drule.discount > 1) {
    showError(t('payment.discount.discount_required'));
    return;
  }
  dsaving.value = true;
  try {
    const res = await API.put('/api/payment/discount', {
      match_type: drule.match_type,
      match_value: drule.match_value.trim(),
      discount: drule.discount,
      enabled: true,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('payment.discount.saved'));
      drule.match_value = '';
      drule.discount = null;
      await loadDiscount();
    } else {
      showError(message);
    }
  } finally {
    dsaving.value = false;
  }
};

const editDiscountRule = (record) => {
  drule.match_type = record.match_type;
  drule.match_value = record.match_value;
  drule.discount = record.discount;
};

const deleteDiscountRule = async (record) => {
  const res = await API.delete('/api/payment/discount', {
    params: { match_type: record.match_type, match_value: record.match_value },
  });
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('payment.access.deleted'));
    await loadDiscount();
  } else {
    showError(message);
  }
};

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
  loadDiscount();
});
</script>

<style scoped>
.payment-hint {
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 12px;
}
</style>
