<template>
  <div class="flex justify-center px-3" style="margin-top: 48px">
    <a-card class="w-full" style="max-width: 450px; box-shadow: 0 1px 3px rgba(0,0,0,0.12)">
      <div class="text-center mb-6">
        <img :src="logo" class="mx-auto mb-2 h-12" alt="logo" />
        <h2 class="text-xl font-semibold">{{ t('auth.reset.confirm.title') }}</h2>
      </div>

      <a-form layout="vertical">
        <a-form-item>
          <a-input v-model:value="inputs.email" :placeholder="t('auth.reset.email')" readonly>
            <template #prefix><MailOutlined /></template>
          </a-input>
        </a-form-item>

        <a-form-item v-if="newPassword">
          <a-input
            :value="newPassword"
            :placeholder="t('auth.reset.confirm.new_password')"
            readonly
            style="cursor: pointer; background-color: #f8f9fa"
            @click="onNewPasswordClick"
          >
            <template #prefix><LockOutlined /></template>
          </a-input>
        </a-form-item>

        <a-button
          block
          size="large"
          type="primary"
          :loading="loading"
          :disabled="disableButton"
          class="mb-4"
          @click="handleSubmit"
        >
          {{ disableButton
            ? t('auth.reset.confirm.button_disabled')
            : t('auth.reset.confirm.button') }}
        </a-button>
      </a-form>

      <p v-if="newPassword" class="text-center text-sm text-gray-500">
        {{ t('auth.reset.confirm.notice') }}
      </p>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { MailOutlined, LockOutlined } from '@ant-design/icons-vue';
import { API, copy, getLogo, showError, showNotice } from '../helpers';

const { t } = useI18n();
const route = useRoute();
const logo = getLogo();

const inputs = reactive({ email: '', token: '' });
const loading = ref(false);
const disableButton = ref(false);
const newPassword = ref('');
const countdown = ref(30);

let countdownTimer = null;

onMounted(() => {
  inputs.token = route.query.token || '';
  inputs.email = route.query.email || '';
});

watch([disableButton, countdown], () => {
  clearInterval(countdownTimer);
  if (disableButton.value && countdown.value > 0) {
    countdownTimer = setInterval(() => {
      countdown.value -= 1;
    }, 1000);
  } else if (countdown.value === 0) {
    disableButton.value = false;
    countdown.value = 30;
  }
});

function onNewPasswordClick(e) {
  e.target.select();
  navigator.clipboard.writeText(newPassword.value);
  showNotice(t('auth.reset.confirm.notice'));
}

async function handleSubmit() {
  disableButton.value = true;
  if (!inputs.email) return;
  loading.value = true;
  const res = await API.post('/api/user/reset', {
    email: inputs.email,
    token: inputs.token,
  });
  const { success, message } = res.data;
  if (success) {
    const password = res.data.data;
    newPassword.value = password;
    await copy(password);
    showNotice(t('messages.notice.password_copied', { password }));
  } else {
    showError(message);
  }
  loading.value = false;
}
</script>
