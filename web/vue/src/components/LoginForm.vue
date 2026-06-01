<template>
  <div>
    <div class="app-public-theme-bar flex justify-end gap-2 p-3">
      <a-button type="text" size="small" :title="t('header.language_switch_tooltip')" @click="toggleLanguage">
        <template #icon><TranslationOutlined /></template>
      </a-button>
      <NacosThemeToggle />
    </div>

    <div class="flex justify-center mt-6 px-3">
      <a-card class="w-full" style="max-width: 450px; box-shadow: 0 1px 3px rgba(0,0,0,0.12)">
        <div class="text-center mb-6">
          <img :src="logo" class="mx-auto mb-2 h-12" alt="logo" />
          <h2 class="text-xl font-semibold">
            {{ tenantPortal ? t('auth.login.tenant_title') : t('auth.login.title') }}
          </h2>
        </div>

        <a-alert
          v-if="tenantPortal"
          type="info"
          show-icon
          class="mb-4 text-left"
          :message="t('auth.login.tenant_portal_hint')"
        />

        <a-form layout="vertical" @submit.prevent="handleSubmit">
          <a-form-item v-if="tenantPortal">
            <a-input v-model:value="inputs.tenant_id" placeholder="租户ID (Tenant ID)">
              <template #prefix><BankOutlined /></template>
            </a-input>
          </a-form-item>
          <a-form-item :validate-status="submitted && !inputs.username ? 'error' : ''">
            <a-input v-model:value="inputs.username" :placeholder="t('auth.login.username')" @pressEnter="handleSubmit">
              <template #prefix><UserOutlined /></template>
            </a-input>
          </a-form-item>
          <a-form-item :validate-status="submitted && !inputs.password ? 'error' : ''">
            <a-input-password v-model:value="inputs.password" :placeholder="t('auth.login.password')" @pressEnter="handleSubmit">
              <template #prefix><LockOutlined /></template>
            </a-input-password>
          </a-form-item>

          <div v-if="status.login_math_captcha && !turnstileEnabled" class="border rounded p-3 mb-3">
            <div class="flex justify-between text-sm font-semibold mb-2">
              <span>{{ t('auth.login.captcha_title') }}</span>
              <span class="font-medium">{{ captchaReady ? t('auth.login.captcha_done') : '' }}</span>
            </div>
            <a-button
              block
              :type="captchaReady ? 'primary' : 'default'"
              @click="showCaptchaModal = true"
            >
              {{ captchaReady ? t('auth.login.captcha_done') : t('auth.login.captcha_open') }}
            </a-button>
            <a-alert v-if="captchaLoadError" type="error" class="mt-2" :message="captchaLoadError" />
          </div>

          <a-button block size="large" type="primary" :loading="loginBusy" class="mb-4" @click="handleSubmit">
            {{ t('auth.login.button') }}
          </a-button>
        </a-form>

        <a-divider />

        <div v-if="tenantPortal" class="flex justify-between text-sm">
          <router-link to="/reset">{{ t('auth.login.tenant_footer_reset') }}</router-link>
          <router-link to="/login">{{ t('auth.login.platform_login_link') }}</router-link>
        </div>
        <div v-else class="text-sm space-y-1">
          <div>
            {{ t('auth.login.forgot_password') }}
            <router-link to="/reset">{{ t('auth.login.reset_password') }}</router-link>
          </div>
          <div>
            {{ t('auth.login.no_account') }}
            <router-link to="/register">{{ t('auth.login.register') }}</router-link>
            <span class="mx-1">·</span>
            <router-link to="/tenant-login">{{ t('auth.login.tenant_login_link') }}</router-link>
          </div>
        </div>

        <template v-if="status.github_oauth || status.wechat_login || (status.lark_oauth && status.lark_client_id)">
          <a-divider>{{ t('auth.login.other_methods') }}</a-divider>
          <div class="flex justify-center gap-4 mt-2">
            <a-button v-if="status.github_oauth" shape="circle" @click="onGitHubOAuthClicked(status.github_client_id)">
              <template #icon><GithubOutlined /></template>
            </a-button>
            <a-button
              v-if="status.wechat_login"
              shape="circle"
              :title="t('auth.login.wechat.entry')"
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
      </a-card>
    </div>

    <!-- 2FA modal -->
    <a-modal v-model:open="showTwoFA" title="两步验证" :confirm-loading="loginBusy" ok-text="确认登录" cancel-text="取消" @ok="submitTwoFA">
      <p>请输入认证器 6 位验证码或 8 位备用码</p>
      <a-input v-model:value="twoFACode" placeholder="验证码 / 备用码" @pressEnter="submitTwoFA" />
    </a-modal>

    <!-- Captcha modal (click / slide / rotate) -->
    <a-modal v-model:open="showCaptchaModal" :title="t('auth.login.captcha_modal_title')" :footer="null" width="fit-content">
      <CaptchaWidget
        v-if="captchaChallenge"
        ref="captchaWidget"
        :challenge="captchaChallenge"
        v-model:answer="captchaAnswer"
        v-model:ready="captchaReady"
      />
      <a-alert
        v-else
        type="info"
        :message="captchaLoading ? t('auth.login.captcha_loading') : (captchaLoadError || t('auth.login.captcha_refresh_hint'))"
      />
      <div class="grid grid-cols-2 gap-2 mt-3">
        <a-button block @click="captchaWidget?.clear()">{{ t('auth.login.captcha_clear') }}</a-button>
        <a-button block type="primary" @click="loadLoginCaptcha">{{ t('auth.login.captcha_refresh') }}</a-button>
      </div>
      <div class="text-right mt-3">
        <a-button @click="showCaptchaModal = false">{{ t('auth.login.captcha_close') }}</a-button>
      </div>
    </a-modal>

    <!-- WeChat modal -->
    <a-modal v-model:open="showWeChatLoginModal" :footer="null" width="360px">
      <div class="text-center">
        <h3 class="mb-1.5">{{ t('auth.login.wechat.title') }}</h3>
        <p class="text-gray-500 mt-0">{{ t('auth.login.wechat.subtitle') }}</p>
        <div class="border rounded-2xl mx-auto mb-4 p-4" style="max-width:260px">
          <img v-if="status.wechat_qrcode" :src="status.wechat_qrcode" :alt="t('auth.login.wechat.qrcode_alt')" class="w-full" style="max-height:220px;object-fit:contain" />
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
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import {
  TranslationOutlined, UserOutlined, LockOutlined, BankOutlined,
  GithubOutlined, WechatOutlined,
} from '@ant-design/icons-vue';
import NacosThemeToggle from './NacosThemeToggle.vue';
import CaptchaWidget from './CaptchaWidget.vue';
import larkIcon from '../assets/lark.svg';
import {
  API, buildLoginPayload, getLogo, isSecurePasswordLoginEnabled,
  postLoginDefaultPath, showError, showInfo, showSuccess, showWarning,
  onGitHubOAuthClicked, onLarkOAuthClicked,
} from '../helpers';
import { useUserStore } from '../stores/user';
import { useStatusStore } from '../stores/status';
import { storeToRefs } from 'pinia';
import { setLocale, currentLocale } from '../i18n';

const props = defineProps({ tenantPortal: Boolean });
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const statusStore = useStatusStore();
const { status: globalStatus } = storeToRefs(statusStore);

const logo = getLogo();
const inputs = reactive({ username: '', password: '', tenant_id: '', wechat_verification_code: '' });
const submitted = ref(false);
const status = ref({});

const captchaChallenge = ref(null); // { mode, master_image, thumb_image, dot_num, tile_*, thumb_size, captcha_id }
const captchaAnswer = ref(null);
const captchaReady = ref(false);
const captchaLoading = ref(false);
const captchaLoadError = ref('');
const captchaWidget = ref(null);
let loginRequestProof = null;

const showTwoFA = ref(false);
const showCaptchaModal = ref(false);
const showWeChatLoginModal = ref(false);
const twoFACode = ref('');
const loginBusy = ref(false);

const turnstileEnabled = computed(() => !!status.value.turnstile_check);

function consumeSafeInternalRedirect() {
  const raw = route.query.redirect;
  if (!raw) return null;
  let decoded;
  try {
    decoded = decodeURIComponent(String(raw).trim());
  } catch {
    return null;
  }
  if (!decoded.startsWith('/') || decoded.startsWith('//')) return null;
  return decoded;
}

function afterLoginNavigate(defaultPath) {
  const target = consumeSafeInternalRedirect();
  if (target) {
    window.location.assign(target);
    return;
  }
  router.push(defaultPath);
}

function mergeStatus(data) {
  if (!data) return;
  status.value = data;
  try {
    localStorage.setItem('status', JSON.stringify(data));
  } catch {
    /* ignore */
  }
}

onMounted(() => {
  if (route.query.expired) showError(t('messages.error.login_expired'));
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
});

watch(globalStatus, (v) => {
  if (v && Object.keys(v).length) mergeStatus(v);
});

watch(showCaptchaModal, (open) => {
  if (!open) return;
  if (!status.value.login_math_captcha || turnstileEnabled.value) return;
  if (captchaChallenge.value || captchaLoading.value) return;
  loadLoginCaptcha();
});

async function loadLoginCaptcha() {
  if (!status.value.login_math_captcha || turnstileEnabled.value) return;
  captchaLoadError.value = '';
  captchaLoading.value = true;
  try {
    const res = await API.get('/api/user/login/captcha');
    const d = res.data?.data;
    if (res.data?.success && d?.master_image) {
      captchaChallenge.value = d;
      captchaAnswer.value = null;
      captchaReady.value = false;
      if (isSecurePasswordLoginEnabled()) {
        if (d.login_request_id && d.login_request_sig != null && d.login_request_ts != null && d.login_enc_key) {
          loginRequestProof = {
            id: d.login_request_id,
            ts: Number(d.login_request_ts),
            sig: d.login_request_sig,
            encKey: d.login_enc_key,
          };
        } else {
          loginRequestProof = null;
        }
      } else {
        loginRequestProof = null;
      }
    } else {
      captchaChallenge.value = null;
      loginRequestProof = null;
      captchaLoadError.value =
        (res.data?.message && String(res.data.message).trim()) || t('auth.login.captcha_load_failed');
    }
  } catch {
    captchaChallenge.value = null;
    loginRequestProof = null;
    captchaLoadError.value = t('auth.login.captcha_load_failed');
  } finally {
    captchaLoading.value = false;
  }
}

async function onSubmitWeChatVerificationCode() {
  const res = await API.get(`/api/oauth/wechat?code=${inputs.wechat_verification_code}`);
  const { success, message, data } = res.data;
  if (success) {
    userStore.login(data);
    afterLoginNavigate(postLoginDefaultPath(data));
    showSuccess(t('messages.success.login'));
    showWeChatLoginModal.value = false;
  } else {
    showError(message);
  }
}

async function handleSubmit() {
  submitted.value = true;
  if (!inputs.username || !inputs.password) return;

  // Read live state from the widget (fall back to the emitted refs) so submit
  // never trips on stale/late v-model updates.
  const widgetReady = captchaWidget.value ? captchaWidget.value.isReady() : captchaReady.value;
  const widgetAnswer = captchaWidget.value ? captchaWidget.value.getAnswer() : captchaAnswer.value;

  if (status.value.login_math_captcha && !turnstileEnabled.value) {
    if (!captchaChallenge.value) {
      if (captchaLoading.value) showInfo(t('auth.login.captcha_loading'));
      else if (captchaLoadError.value) showInfo(captchaLoadError.value);
      else showInfo(t('auth.login.captcha_need_load'));
      showCaptchaModal.value = true;
      return;
    }
    if (!widgetReady) {
      showInfo(t('auth.login.captcha_incomplete'));
      showCaptchaModal.value = true;
      return;
    }
  }

  let proof = loginRequestProof;
  if (!(status.value.login_math_captcha && !turnstileEnabled.value)) proof = null;

  loginBusy.value = true;
  try {
    const captcha =
      status.value.login_math_captcha && !turnstileEnabled.value && captchaChallenge.value
        ? {
            captcha_id: captchaChallenge.value.captcha_id,
            mode: captchaChallenge.value.mode,
            answer: widgetAnswer,
          }
        : undefined;
    const body = await buildLoginPayload(inputs.username, inputs.password, captcha, proof);
    if (props.tenantPortal) {
      const tid = String(inputs.tenant_id ?? '').trim();
      if (tid !== '') body.tenant_id = tid;
    }
    const res = await API.post('/api/user/login', body);
    const { success, message, data } = res.data;
    if (success) {
      if (data?.require_2fa) {
        showTwoFA.value = true;
        twoFACode.value = '';
        return;
      }
      userStore.login(data);
      if (data?.require_force_2fa_setup) showWarning(t('auth.login.force_2fa_hint'));
      if (inputs.username === 'root' && inputs.password === '123456') {
        afterLoginNavigate('/user/edit');
        showSuccess(t('messages.success.login'));
        showWarning(t('messages.error.root_password'));
      } else {
        afterLoginNavigate(data?.require_force_2fa_setup ? '/setting' : postLoginDefaultPath(data));
        showSuccess(t('messages.success.login'));
      }
    } else {
      showError(message);
      if (status.value.login_math_captcha && !turnstileEnabled.value) loadLoginCaptcha();
    }
  } catch (err) {
    showError(err.message || '登录准备失败');
    if (status.value.login_math_captcha && !turnstileEnabled.value) loadLoginCaptcha();
  } finally {
    loginBusy.value = false;
  }
}

async function submitTwoFA() {
  if (!twoFACode.value.trim()) {
    showWarning('请输入验证码或备用码');
    return;
  }
  loginBusy.value = true;
  try {
    const res = await API.post('/api/user/login/2fa', { code: twoFACode.value.trim() });
    const { success, message, data } = res.data;
    if (success) {
      showTwoFA.value = false;
      userStore.login(data);
      if (data?.require_force_2fa_setup) {
        showWarning('请前往个人设置完成两步验证配置');
        afterLoginNavigate('/setting');
      } else {
        afterLoginNavigate(postLoginDefaultPath(data));
      }
      showSuccess(t('messages.success.login'));
    } else {
      showError(message);
    }
  } catch {
    showError('验证失败');
  } finally {
    loginBusy.value = false;
  }
}

function toggleLanguage() {
  setLocale(currentLocale() === 'en' ? 'zh' : 'en');
}
</script>
