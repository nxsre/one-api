<template>
  <div>
    <div class="app-public-theme-bar flex justify-end gap-2 p-3">
      <a-button
        type="text"
        size="small"
        :title="t('header.language_switch_tooltip')"
        :aria-label="t('header.language_switch_tooltip')"
        @click="toggleLanguage"
      >
        <template #icon><TranslationOutlined /></template>
      </a-button>
      <NacosThemeToggle />
    </div>

    <div class="flex justify-center px-3" style="margin-top: 24px">
      <a-card class="w-full" style="max-width: 450px; box-shadow: 0 1px 3px rgba(0,0,0,0.12)">
        <div class="text-center mb-6">
          <img :src="logo" class="mx-auto mb-2 h-12" alt="logo" />
          <h2 class="text-xl font-semibold">{{ t('auth.register.title') }}</h2>
        </div>

        <a-form layout="vertical" @submit.prevent="handleSubmit">
          <a-form-item>
            <a-input v-model:value="inputs.username" :placeholder="t('auth.register.username')">
              <template #prefix><UserOutlined /></template>
            </a-input>
          </a-form-item>
          <a-form-item>
            <a-input-password v-model:value="inputs.password" :placeholder="t('auth.register.password')">
              <template #prefix><LockOutlined /></template>
            </a-input-password>
          </a-form-item>
          <a-form-item>
            <a-input-password v-model:value="inputs.password2" :placeholder="t('auth.register.confirm_password')">
              <template #prefix><LockOutlined /></template>
            </a-input-password>
          </a-form-item>

          <template v-if="showEmailVerification">
            <a-form-item>
              <a-input-group compact style="display: flex">
                <a-input
                  v-model:value="inputs.email"
                  type="email"
                  :placeholder="t('auth.register.email')"
                  style="flex: 1"
                >
                  <template #prefix><MailOutlined /></template>
                </a-input>
                <a-button :disabled="loading" @click="sendVerificationCode">
                  {{ disableButton
                    ? t('auth.register.get_code_retry', { countdown })
                    : t('auth.register.get_code') }}
                </a-button>
              </a-input-group>
            </a-form-item>
            <a-form-item>
              <a-input v-model:value="inputs.verification_code" :placeholder="t('auth.register.verification_code')">
                <template #prefix><LockOutlined /></template>
              </a-input>
            </a-form-item>
          </template>

          <div v-if="turnstileEnabled" class="flex justify-center mb-4">
            <vue-turnstile :site-key="turnstileSiteKey" v-model="turnstileToken" />
          </div>

          <a-button
            block
            size="large"
            type="primary"
            :loading="loading"
            class="mb-4"
            @click="handleSubmit"
          >
            {{ t('auth.register.button') }}
          </a-button>
        </a-form>

        <template v-if="status.github_oauth || status.wechat_login || (status.lark_oauth && status.lark_client_id)">
          <a-divider>{{ t('auth.login.other_methods') }}</a-divider>
          <div class="flex justify-center gap-4 mt-2 mb-2">
            <a-button v-if="status.github_oauth" shape="circle" @click="onGitHubOAuthClicked(status.github_client_id)">
              <template #icon><GithubOutlined /></template>
            </a-button>
            <a-button
              v-if="status.wechat_login"
              shape="circle"
              :title="t('auth.login.wechat.entry')"
              :aria-label="t('auth.login.wechat.entry')"
              @click="showWeChatLoginModal = true"
            >
              <template #icon><WechatOutlined /></template>
            </a-button>
            <img
              v-if="status.lark_oauth && status.lark_client_id"
              :src="larkIcon"
              class="w-9 h-9 cursor-pointer rounded-full"
              alt="lark"
              @click="onLarkOAuthClicked(status.lark_client_id)"
            />
          </div>
        </template>

        <a-divider />
        <div class="text-center text-sm text-gray-500">
          {{ t('auth.register.has_account') }}
          <router-link to="/login" class="app-register-login-link">
            {{ t('auth.register.login') }}
          </router-link>
        </div>
      </a-card>
    </div>

    <!-- WeChat modal -->
    <a-modal v-model:open="showWeChatLoginModal" :footer="null" width="360px">
      <div class="text-center">
        <h3 class="mb-1.5">{{ t('auth.login.wechat.title') }}</h3>
        <p class="text-gray-500 mt-0">{{ t('auth.login.wechat.subtitle') }}</p>
        <div class="border rounded-2xl mx-auto mb-4 p-4" style="max-width: 260px">
          <img
            v-if="status.wechat_qrcode"
            :src="status.wechat_qrcode"
            :alt="t('auth.login.wechat.qrcode_alt')"
            class="w-full"
            style="max-height: 220px; object-fit: contain"
          />
          <a-alert v-else type="warning" :message="t('auth.login.wechat.qrcode_missing')" />
        </div>
        <a-alert type="info" class="text-left mb-3" :message="t('auth.login.wechat.scan_tip')" />
        <a-form layout="vertical">
          <a-form-item :label="t('auth.login.wechat.code_label')">
            <a-input v-model:value="inputs.wechat_verification_code" :placeholder="t('auth.login.wechat.code_placeholder')" />
          </a-form-item>
          <a-button block type="primary" size="large" @click="onSubmitWeChatVerificationCode">
            {{ t('auth.login.wechat.submit') }}
          </a-button>
        </a-form>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import VueTurnstile from 'vue-turnstile';
import {
  TranslationOutlined, UserOutlined, LockOutlined, MailOutlined,
  GithubOutlined, WechatOutlined,
} from '@ant-design/icons-vue';
import NacosThemeToggle from './NacosThemeToggle.vue';
import larkIcon from '../assets/lark.svg';
import {
  API, getLogo, showError, showInfo, showSuccess,
  onGitHubOAuthClicked, onLarkOAuthClicked,
} from '../helpers';
import { useUserStore } from '../stores/user';
import { useStatusStore } from '../stores/status';
import { storeToRefs } from 'pinia';
import { setLocale, currentLocale } from '../i18n';

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const statusStore = useStatusStore();
const { status: globalStatus } = storeToRefs(statusStore);

const logo = getLogo();
const inputs = reactive({
  username: '',
  password: '',
  password2: '',
  email: '',
  verification_code: '',
  wechat_verification_code: '',
});

const showEmailVerification = ref(false);
const turnstileEnabled = ref(false);
const turnstileSiteKey = ref('');
const turnstileToken = ref('');
const loading = ref(false);
const disableButton = ref(false);
const countdown = ref(30);
const status = ref({});
const showWeChatLoginModal = ref(false);

let affCode = new URLSearchParams(window.location.search).get('aff');
if (affCode) {
  localStorage.setItem('aff', affCode);
}

let countdownTimer = null;

function mergeStatus(data) {
  if (!data) return;
  status.value = data;
  try {
    localStorage.setItem('status', JSON.stringify(data));
  } catch {
    /* ignore */
  }
}

function applyStatus() {
  const s = status.value;
  if (!s || typeof s !== 'object') return;
  showEmailVerification.value = !!s.email_verification;
  if (s.turnstile_check) {
    turnstileEnabled.value = true;
    turnstileSiteKey.value = s.turnstile_site_key || '';
  } else {
    turnstileEnabled.value = false;
  }
}

onMounted(() => {
  if (globalStatus.value && Object.keys(globalStatus.value).length) {
    mergeStatus(globalStatus.value);
  } else {
    const cached = localStorage.getItem('status');
    if (cached) {
      try {
        mergeStatus(JSON.parse(cached));
      } catch {
        /* ignore */
      }
    }
  }
  applyStatus();
});

watch(globalStatus, (v) => {
  if (v && Object.keys(v).length) mergeStatus(v);
});

watch(status, applyStatus);

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

function toggleLanguage() {
  setLocale(currentLocale() === 'en' ? 'zh' : 'en');
}

async function onSubmitWeChatVerificationCode() {
  const res = await API.get(`/api/oauth/wechat?code=${inputs.wechat_verification_code}`);
  const { success, message, data } = res.data;
  if (success) {
    userStore.login(data);
    router.push('/');
    showSuccess(t('messages.success.login'));
    showWeChatLoginModal.value = false;
  } else {
    showError(message);
  }
}

async function handleSubmit() {
  if (inputs.password.length < 8) {
    showInfo(t('messages.error.password_length'));
    return;
  }
  if (inputs.password !== inputs.password2) {
    showInfo(t('messages.error.password_mismatch'));
    return;
  }
  if (inputs.username && inputs.password) {
    if (turnstileEnabled.value && turnstileToken.value === '') {
      showInfo(t('messages.error.turnstile_wait'));
      return;
    }
    loading.value = true;
    if (!affCode) {
      affCode = localStorage.getItem('aff');
    }
    inputs.aff_code = affCode;
    const res = await API.post(`/api/user/register?turnstile=${turnstileToken.value}`, inputs);
    const { success, message } = res.data;
    if (success) {
      router.push('/login');
      showSuccess(t('messages.success.register'));
    } else {
      showError(message);
    }
    loading.value = false;
  }
}

async function sendVerificationCode() {
  if (inputs.email === '') return;
  if (turnstileEnabled.value && turnstileToken.value === '') {
    showInfo(t('messages.error.turnstile_wait'));
    return;
  }
  disableButton.value = true;
  loading.value = true;
  const res = await API.get(`/api/verification?email=${inputs.email}&turnstile=${turnstileToken.value}`);
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('messages.success.verification_code'));
  } else {
    showError(message);
    disableButton.value = false;
    countdown.value = 30;
  }
  loading.value = false;
}
</script>
