<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h2>{{ t('topup.title') }}</h2>

      <a-row :gutter="16">
        <a-col :xs="24" :md="12">
          <a-card :style="{ height: '100%', boxShadow: '0 1px 3px rgba(0,0,0,0.1)' }">
            <div style="height: 100%; display: flex; flex-direction: column;">
              <h3 style="color: #2185d0; margin: 1em;">
                <CreditCardOutlined style="margin-right: 0.5em;" />{{ t('topup.get_code.title') }}
              </h3>
              <div style="flex: 1; display: flex; flex-direction: column; justify-content: space-between;">
                <div style="text-align: center; padding-top: 1em;">
                  <a-statistic
                    :value="renderQuota(userQuota, t)"
                    :title="t('topup.get_code.current_quota')"
                    :value-style="{ color: '#2185d0' }"
                  />
                </div>
                <div style="text-align: center; padding-bottom: 1em;">
                  <a-button type="primary" size="large" style="width: 80%;" @click="openTopUpLink">
                    {{ t('topup.get_code.button') }}
                  </a-button>
                </div>
              </div>
            </div>
          </a-card>
        </a-col>

        <a-col :xs="24" :md="12">
          <a-card :style="{ height: '100%', boxShadow: '0 1px 3px rgba(0,0,0,0.1)' }">
            <div style="height: 100%; display: flex; flex-direction: column;">
              <h3 style="color: #21ba45; margin: 1em;">
                <TagOutlined style="margin-right: 0.5em;" />{{ t('topup.redeem_code.title') }}
              </h3>
              <div style="flex: 1; display: flex; flex-direction: column; justify-content: space-between;">
                <a-input
                  v-model:value="redemptionCode"
                  :placeholder="t('topup.redeem_code.placeholder')"
                  @paste="onPaste"
                >
                  <template #prefix>
                    <KeyOutlined />
                  </template>
                  <template #addonAfter>
                    <a-button type="text" size="small" @click="pasteFromClipboard">
                      <template #icon><SnippetsOutlined /></template>
                      {{ t('topup.redeem_code.paste') }}
                    </a-button>
                  </template>
                </a-input>

                <div style="padding-bottom: 1em; margin-top: 1em;">
                  <a-button
                    block
                    size="large"
                    style="background-color: #21ba45; border-color: #21ba45; color: #fff;"
                    :loading="isSubmitting"
                    :disabled="isSubmitting"
                    @click="topUp"
                  >
                    {{ isSubmitting ? t('topup.redeem_code.submitting') : t('topup.redeem_code.submit') }}
                  </a-button>
                </div>
              </div>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  CreditCardOutlined,
  TagOutlined,
  KeyOutlined,
  SnippetsOutlined,
} from '@ant-design/icons-vue';
import { API, showError, showInfo, showSuccess, renderQuota } from '@/helpers';

const { t } = useI18n();

const redemptionCode = ref('');
const topUpLink = ref('');
const userQuota = ref(0);
const isSubmitting = ref(false);
const user = ref({});

const topUp = async () => {
  if (redemptionCode.value === '') {
    showInfo(t('topup.redeem_code.empty_code'));
    return;
  }
  isSubmitting.value = true;
  try {
    const res = await API.post('/api/user/topup', {
      key: redemptionCode.value,
    });
    const { success, message, data } = res.data;
    if (success) {
      showSuccess(t('topup.redeem_code.success'));
      userQuota.value = userQuota.value + data;
      redemptionCode.value = '';
    } else {
      showError(message);
    }
  } catch (err) {
    showError(t('topup.redeem_code.request_failed'));
  } finally {
    isSubmitting.value = false;
  }
};

const openTopUpLink = () => {
  if (!topUpLink.value) {
    showError(t('topup.redeem_code.no_link'));
    return;
  }
  let url = new URL(topUpLink.value);
  let username = user.value.username;
  let user_id = user.value.user_id;
  url.searchParams.append('username', username);
  url.searchParams.append('user_id', user_id);
  url.searchParams.append('transaction_id', crypto.randomUUID());
  window.open(url.toString(), '_blank');
};

const onPaste = (e) => {
  e.preventDefault();
  const pastedText = e.clipboardData.getData('text');
  redemptionCode.value = pastedText.trim();
};

const pasteFromClipboard = async () => {
  try {
    const text = await navigator.clipboard.readText();
    redemptionCode.value = text.trim();
  } catch (err) {
    showError(t('topup.redeem_code.paste_error'));
  }
};

const getUserQuota = async () => {
  let res = await API.get(`/api/user/self`);
  const { success, message, data } = res.data;
  if (success) {
    userQuota.value = data.quota;
    user.value = data;
  } else {
    showError(message);
  }
};

onMounted(() => {
  let status = localStorage.getItem('status');
  if (status) {
    status = JSON.parse(status);
    if (status.top_up_link) {
      topUpLink.value = status.top_up_link;
    }
  }
  getUserQuota();
});
</script>
