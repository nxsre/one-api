<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <div class="header" style="font-size: 1.28em; font-weight: 700; margin-bottom: 0.75rem">
        {{ isEdit ? t('redemption.edit.title_edit') : t('redemption.edit.title_create') }}
      </div>
      <a-spin :spinning="loading">
        <a-form layout="vertical" autocomplete="off">
          <a-form-item :label="t('redemption.edit.name')">
            <a-input
              v-model:value="inputs.name"
              :placeholder="t('redemption.edit.name_placeholder')"
            />
          </a-form-item>

          <a-form-item :label="`${t('redemption.edit.quota')}${renderQuotaWithPrompt(inputs.quota, t)}`">
            <a-input-number
              v-model:value="inputs.quota"
              :placeholder="t('redemption.edit.quota_placeholder')"
              :min="0"
              :step="1"
              style="width: 100%"
            />
          </a-form-item>

          <a-form-item v-if="!isEdit" :label="t('redemption.edit.count')">
            <a-input-number
              v-model:value="inputs.count"
              :placeholder="t('redemption.edit.count_placeholder')"
              :min="1"
              :step="1"
              style="width: 100%"
            />
          </a-form-item>

          <div class="flex gap-2">
            <a-button type="primary" @click="submit">
              {{ t('redemption.edit.buttons.submit') }}
            </a-button>
            <a-button @click="handleCancel">
              {{ t('redemption.edit.buttons.cancel') }}
            </a-button>
          </div>
        </a-form>
      </a-spin>
    </a-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import {
  API,
  downloadTextAsFile,
  showError,
  showSuccess,
  renderQuotaWithPrompt,
} from '@/helpers';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const redemptionId = route.params.id;
const isEdit = redemptionId !== undefined;
const loading = ref(isEdit);

const originInputs = {
  name: '',
  quota: 100000,
  count: 1,
};

const inputs = reactive({ ...originInputs });

const handleCancel = () => {
  router.push('/redemption');
};

const loadRedemption = async () => {
  const res = await API.get(`/api/redemption/${redemptionId}`);
  const { success, message, data } = res.data;
  if (success) {
    Object.assign(inputs, data);
  } else {
    showError(message);
  }
  loading.value = false;
};

if (isEdit) {
  loadRedemption();
}

const submit = async () => {
  if (!isEdit && inputs.name === '') return;
  const localInputs = {
    ...inputs,
    count: parseInt(inputs.count),
    quota: parseInt(inputs.quota),
  };
  let res;
  if (isEdit) {
    res = await API.put('/api/redemption/', {
      ...localInputs,
      id: parseInt(redemptionId),
    });
  } else {
    res = await API.post('/api/redemption/', {
      ...localInputs,
    });
  }
  const { success, message, data } = res.data;
  if (success) {
    if (isEdit) {
      showSuccess(t('redemption.messages.update_success'));
    } else {
      showSuccess(t('redemption.messages.create_success'));
      Object.assign(inputs, originInputs);
    }
  } else {
    showError(message);
  }
  if (!isEdit && data) {
    let text = '';
    for (let i = 0; i < data.length; i++) {
      text += data[i] + '\n';
    }
    downloadTextAsFile(text, `${inputs.name}.txt`);
  }
};
</script>
