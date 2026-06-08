<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('user.add.title') }}</h3>
      <a-form layout="vertical" :autocomplete="noAutofillFormProps.autocomplete">
        <a-form-item :label="t('user.edit.username')" required>
          <a-input
            v-model:value="inputs.username"
            :placeholder="t('user.edit.username_placeholder')"
            :readonly="!autofillUnlocked"
            v-bind="noAutofillTextProps"
            @focus="autofillUnlocked = true"
          />
        </a-form-item>
        <a-form-item :label="t('user.edit.display_name')">
          <a-input
            v-model:value="inputs.display_name"
            :placeholder="t('user.edit.display_name_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('user.edit.password')" required>
          <a-input-password
            v-model:value="inputs.password"
            :placeholder="t('user.edit.password_placeholder')"
            :readonly="!autofillUnlocked"
            v-bind="noAutofillPasswordProps"
            @focus="autofillUnlocked = true"
          />
        </a-form-item>
        <a-button type="primary" html-type="submit" @click="submit">
          {{ t('user.edit.buttons.submit') }}
        </a-button>
      </a-form>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  API,
  showError,
  showSuccess,
  noAutofillFormProps,
  noAutofillPasswordProps,
  noAutofillTextProps,
} from '@/helpers';

const { t } = useI18n();

const originInputs = {
  username: '',
  display_name: '',
  password: '',
};
const inputs = reactive({ ...originInputs });
const autofillUnlocked = ref(false);

const submit = async () => {
  if (inputs.username === '' || inputs.password === '') return;
  const res = await API.post('/api/user/', { ...inputs });
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('user.messages.create_success'));
    Object.assign(inputs, originInputs);
  } else {
    showError(message);
  }
};
</script>
