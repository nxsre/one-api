<template>
  <div class="flex justify-center px-3" style="margin-top: 48px">
    <a-card class="w-full" style="max-width: 450px; box-shadow: 0 1px 3px rgba(0,0,0,0.12)">
      <div class="text-center mb-6">
        <img :src="logo" class="mx-auto mb-2 h-12" alt="logo" />
        <h2 class="text-xl font-semibold">{{ t('auth.reset.title') }}</h2>
      </div>

      <a-form layout="vertical" @submit.prevent="handleSubmit">
        <a-form-item>
          <a-input
            v-model:value="inputs.email"
            :placeholder="t('auth.reset.email')"
            @pressEnter="handleSubmit"
          >
            <template #prefix><MailOutlined /></template>
          </a-input>
        </a-form-item>

        <div v-if="turnstileEnabled" class="flex justify-center mb-4">
          <vue-turnstile :site-key="turnstileSiteKey" v-model="turnstileToken" />
        </div>

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
            ? t('auth.register.get_code_retry', { countdown })
            : t('auth.reset.button') }}
        </a-button>
      </a-form>

      <p class="text-center text-sm text-gray-500">{{ t('auth.reset.notice') }}</p>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import VueTurnstile from 'vue-turnstile';
import { MailOutlined } from '@ant-design/icons-vue';
import { API, getLogo, showError, showInfo, showSuccess } from '../helpers';

const { t } = useI18n();
const logo = getLogo();

const inputs = reactive({ email: '' });
const loading = ref(false);
const turnstileEnabled = ref(false);
const turnstileSiteKey = ref('');
const turnstileToken = ref('');
const disableButton = ref(false);
const countdown = ref(30);

let countdownTimer = null;

onMounted(() => {
  let status = localStorage.getItem('status');
  if (status) {
    status = JSON.parse(status);
    if (status.turnstile_check) {
      turnstileEnabled.value = true;
      turnstileSiteKey.value = status.turnstile_site_key;
    }
  }
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

async function handleSubmit() {
  disableButton.value = true;
  if (!inputs.email) return;
  if (turnstileEnabled.value && turnstileToken.value === '') {
    showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
    return;
  }
  loading.value = true;
  const res = await API.get(
    `/api/reset_password?email=${inputs.email}&turnstile=${turnstileToken.value}`
  );
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('auth.reset.notice'));
    inputs.email = '';
  } else {
    showError(message);
    disableButton.value = false;
    countdown.value = 30;
  }
  loading.value = false;
}
</script>
