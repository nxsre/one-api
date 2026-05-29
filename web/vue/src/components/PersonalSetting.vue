<template>
  <div class="settings-page-body">
    <h3>{{ t('setting.personal.general.title') }}</h3>
    <a-alert
      type="info"
      :message="t('setting.personal.general.system_token_notice')"
      style="margin-bottom: 1rem"
    />
    <div class="settings-action-bar">
      <a-button @click="router.push('/user/edit/')">
        {{ t('setting.personal.general.buttons.update_profile') }}
      </a-button>
      <a-button @click="generateAccessToken">
        {{ t('setting.personal.general.buttons.generate_token') }}
      </a-button>
      <a-button @click="getAffLink">
        {{ t('setting.personal.general.buttons.copy_invite') }}
      </a-button>
      <a-button
        v-if="userStore.user?.role === 1"
        type="primary"
        @click="upgradeModalOpen = true"
      >
        升级为企业（租户）
      </a-button>
      <a-button @click="showAccountDeleteModal = true">
        {{ t('setting.personal.general.buttons.delete_account') }}
      </a-button>
    </div>

    <a-input
      v-if="systemToken"
      readonly
      :value="systemToken"
      style="margin-top: 10px"
      @click="handleSystemTokenClick"
    />
    <a-input
      v-if="affLink"
      readonly
      :value="affLink"
      style="margin-top: 10px"
      @click="handleAffLinkClick"
    />

    <a-divider />
    <h3>安全 · 两步验证（2FA）</h3>
    <TwoFASetting />

    <a-divider />
    <h3>{{ t('setting.personal.s3.title') }}</h3>
    <a-alert type="info" :message="t('setting.personal.s3.notice')" style="margin-bottom: 1rem" />
    <a-alert
      v-if="s3Info && !s3Info.site"
      type="warning"
      :message="t('setting.personal.s3.site_disabled')"
      style="margin-bottom: 1rem"
    />
    <template v-if="s3Info && s3Info.site">
      <div>
        {{ t('setting.personal.s3.region') }}: <code>{{ s3Info.region }}</code>
      </div>
      <div style="margin-top: 8px">
        {{ s3Info.enabled
          ? t('setting.personal.s3.status_enabled')
          : t('setting.personal.s3.status_disabled') }}
        {{ s3Info.accessKey
          ? ` — ${t('setting.personal.s3.access_key')}: ${s3Info.accessKey}`
          : '' }}
      </div>
      <div class="settings-action-bar" style="margin-top: 12px">
        <a-button v-if="!s3Info.enabled" type="primary" @click="s3Enable">
          {{ t('setting.personal.s3.enable') }}
        </a-button>
        <a-button-group v-else>
          <a-button @click="s3RegenerateSecret">
            {{ t('setting.personal.s3.regenerate_secret') }}
          </a-button>
          <a-button @click="s3RotateKeys">
            {{ t('setting.personal.s3.rotate_keys') }}
          </a-button>
          <a-button danger @click="s3Disable">
            {{ t('setting.personal.s3.disable') }}
          </a-button>
        </a-button-group>
      </div>
    </template>

    <a-modal
      v-model:open="s3SecretModal.open"
      :title="t('setting.personal.s3.secret_once')"
      @cancel="closeS3SecretModal"
    >
      <p>{{ s3SecretModal.subtitle }}</p>
      <a-form-item v-if="s3SecretModal.accessKey" :label="t('setting.personal.s3.access_key')">
        <a-input readonly :value="s3SecretModal.accessKey" @click="(e) => e.target.select()" />
      </a-form-item>
      <a-form-item :label="t('setting.personal.s3.secret_once')">
        <a-input readonly :value="s3SecretModal.secretKey" @click="(e) => e.target.select()" />
      </a-form-item>
      <template #footer>
        <a-button type="primary" @click="copyS3Secret">复制到剪贴板</a-button>
        <a-button @click="closeS3SecretModal">关闭</a-button>
      </template>
    </a-modal>

    <a-divider />
    <h3>{{ t('setting.personal.binding.title') }}</h3>
    <div class="settings-action-bar">
      <a-button v-if="status.wechat_login" @click="showWeChatBindModal = true">
        {{ t('setting.personal.binding.buttons.bind_wechat') }}
      </a-button>
      <a-button v-if="status.github_oauth" @click="onGitHubOAuthClicked(status.github_client_id)">
        {{ t('setting.personal.binding.buttons.bind_github') }}
      </a-button>
      <a-button
        v-if="status.lark_oauth && status.lark_client_id"
        @click="onLarkOAuthClicked(status.lark_client_id)"
      >
        {{ t('setting.personal.binding.buttons.bind_lark') }}
      </a-button>
      <a-button @click="showEmailBindModal = true">
        {{ t('setting.personal.binding.buttons.bind_email') }}
      </a-button>
    </div>

    <a-modal v-model:open="showWeChatBindModal" :footer="null" width="360px">
      <a-image :src="status.wechat_qrcode" :preview="false" />
      <div style="text-align: center">
        <p>{{ t('setting.personal.binding.wechat.description') }}</p>
      </div>
      <a-form>
        <a-form-item>
          <a-input
            v-model:value="inputs.wechat_verification_code"
            :placeholder="t('setting.personal.binding.wechat.verification_code')"
          />
        </a-form-item>
        <a-button block size="large" @click="bindWeChat">
          {{ t('setting.personal.binding.wechat.bind') }}
        </a-button>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="showEmailBindModal"
      :title="t('setting.personal.binding.email.title')"
      :footer="null"
      :width="450"
    >
      <a-form>
        <a-form-item>
          <a-input-search
            v-model:value="inputs.email"
            type="email"
            name="email"
            :placeholder="t('setting.personal.binding.email.email_placeholder')"
            :enter-button="disableButton
              ? t('setting.personal.binding.email.get_code_retry', { countdown })
              : t('setting.personal.binding.email.get_code')"
            :loading="false"
            :disabled="disableButton || loading"
            @search="sendVerificationCode"
          />
        </a-form-item>
        <a-form-item>
          <a-input
            v-model:value="inputs.email_verification_code"
            :placeholder="t('setting.personal.binding.email.code_placeholder')"
          />
        </a-form-item>
        <div v-if="turnstileEnabled" class="flex justify-center mb-4">
          <vue-turnstile :site-key="turnstileSiteKey" v-model="turnstileToken" />
        </div>
        <div style="display: flex; justify-content: space-between; margin-top: 1rem">
          <a-button block size="large" :loading="loading" @click="bindEmail">
            {{ t('setting.personal.binding.email.bind') }}
          </a-button>
          <div style="width: 1rem"></div>
          <a-button block size="large" @click="showEmailBindModal = false">
            {{ t('setting.personal.binding.email.cancel') }}
          </a-button>
        </div>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="showAccountDeleteModal"
      :title="t('setting.personal.delete_account.title')"
      :footer="null"
      :width="450"
    >
      <a-alert type="warning" :message="t('setting.personal.delete_account.warning')" style="margin-bottom: 1rem" />
      <a-form>
        <a-form-item>
          <a-input
            v-model:value="inputs.self_account_deletion_confirmation"
            :placeholder="t('setting.personal.delete_account.confirm_placeholder', {
              username: userStore.user?.username,
            })"
          />
        </a-form-item>
        <div v-if="turnstileEnabled" class="flex justify-center mb-4">
          <vue-turnstile :site-key="turnstileSiteKey" v-model="turnstileToken" />
        </div>
        <div style="display: flex; justify-content: space-between; margin-top: 1rem">
          <a-button block danger size="large" :loading="loading" @click="deleteAccount">
            {{ t('setting.personal.delete_account.buttons.confirm') }}
          </a-button>
          <div style="width: 1rem"></div>
          <a-button block size="large" @click="showAccountDeleteModal = false">
            {{ t('setting.personal.delete_account.buttons.cancel') }}
          </a-button>
        </div>
      </a-form>
    </a-modal>

    <a-modal v-model:open="upgradeModalOpen" title="申请升级为企业（租户）">
      <a-form layout="vertical">
        <a-form-item label="企业名称" required>
          <a-input v-model:value="upgradeInputs.name" placeholder="请输入您的企业或团队名称" />
        </a-form-item>
        <a-form-item label="租户标识 (Slug)" required>
          <a-input
            v-model:value="upgradeInputs.slug"
            placeholder="全英文或数字，将作为专属链接后缀，如：my-team"
          />
        </a-form-item>
        <a-form-item label="申请备注">
          <a-textarea
            v-model:value="upgradeInputs.remark"
            placeholder="请简要说明企业规模及主要用途（选填）"
          />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="upgradeModalOpen = false">取消</a-button>
        <a-button type="primary" :loading="upgradeLoading" @click="submitUpgrade">
          提交申请
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { Modal } from 'ant-design-vue';
import VueTurnstile from 'vue-turnstile';
import {
  API,
  clearNacosEmbeddedConsoleLocalSession,
  clearTenantConsoleActingTenantId,
  copy,
  showError,
  showInfo,
  showSuccess,
  onGitHubOAuthClicked,
  onLarkOAuthClicked,
} from '@/helpers';
import { useUserStore } from '@/stores/user';
import TwoFASetting from '@/components/TwoFASetting.vue';

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();

const inputs = reactive({
  wechat_verification_code: '',
  email_verification_code: '',
  email: '',
  self_account_deletion_confirmation: '',
});
const status = ref({});
const showWeChatBindModal = ref(false);
const showEmailBindModal = ref(false);
const showAccountDeleteModal = ref(false);
const turnstileEnabled = ref(false);
const turnstileSiteKey = ref('');
const turnstileToken = ref('');
const loading = ref(false);
const upgradeModalOpen = ref(false);
const upgradeLoading = ref(false);
const upgradeInputs = reactive({ name: '', slug: '', remark: '' });
const disableButton = ref(false);
const countdown = ref(30);
const affLink = ref('');
const systemToken = ref('');
const s3Info = ref(null);
const s3SecretModal = reactive({
  open: false,
  accessKey: '',
  secretKey: '',
  subtitle: '',
});

let countdownInterval = null;

onMounted(() => {
  const raw = localStorage.getItem('status');
  if (raw) {
    const st = JSON.parse(raw);
    status.value = st;
    if (st.turnstile_check) {
      turnstileEnabled.value = true;
      turnstileSiteKey.value = st.turnstile_site_key;
    }
  }
  loadS3Self();
});

watch([disableButton, countdown], () => {
  clearInterval(countdownInterval);
  if (disableButton.value && countdown.value > 0) {
    countdownInterval = setInterval(() => {
      countdown.value -= 1;
    }, 1000);
  } else if (countdown.value === 0) {
    disableButton.value = false;
    countdown.value = 30;
  }
});

onBeforeUnmount(() => {
  clearInterval(countdownInterval);
});

const loadS3Self = async () => {
  try {
    const res = await API.get('/api/user/self');
    const { success, message, data } = res.data;
    if (success) {
      s3Info.value = {
        site: data.s3_site_enabled,
        enabled: data.s3_enabled,
        region: data.s3_region,
        accessKey: data.s3_access_key || '',
      };
    } else {
      showError(message);
    }
  } catch {
    /* interceptor may have shown error */
  }
};

const openS3SecretModal = (accessKey, secretKey, subtitle) => {
  s3SecretModal.open = true;
  s3SecretModal.accessKey = accessKey;
  s3SecretModal.secretKey = secretKey;
  s3SecretModal.subtitle = subtitle;
};

const closeS3SecretModal = () => {
  s3SecretModal.open = false;
  s3SecretModal.secretKey = '';
};

const copyS3Secret = async () => {
  const text = [
    s3SecretModal.accessKey ? `AK=${s3SecretModal.accessKey}` : '',
    `SK=${s3SecretModal.secretKey}`,
  ]
    .filter(Boolean)
    .join('\n');
  await copy(text);
  showSuccess('已复制');
};

const s3Enable = async () => {
  const res = await API.post('/api/user/s3/enable');
  const { success, message, data } = res.data;
  if (success) {
    openS3SecretModal(data.access_key, data.secret_key, t('setting.personal.s3.enable'));
    await loadS3Self();
    showSuccess('临时 S3 已启用');
  } else {
    showError(message);
  }
};

const s3Disable = () => {
  Modal.confirm({
    title: '确定关闭并作废当前 S3 密钥？已存储的对象不会自动删除。',
    onOk: async () => {
      const res = await API.post('/api/user/s3/disable');
      const { success, message } = res.data;
      if (success) {
        await loadS3Self();
        showSuccess('已关闭');
      } else {
        showError(message);
      }
    },
  });
};

const s3RegenerateSecret = async () => {
  const res = await API.post('/api/user/s3/regenerate_secret');
  const { success, message, data } = res.data;
  if (success) {
    openS3SecretModal(
      s3Info.value?.accessKey || '',
      data.secret_key,
      t('setting.personal.s3.regenerate_secret')
    );
    showSuccess('Secret 已更新');
  } else {
    showError(message);
  }
};

const s3RotateKeys = () => {
  Modal.confirm({
    title: '将生成新的 Access Key 与 Secret，旧密钥立即失效。继续？',
    onOk: async () => {
      const res = await API.post('/api/user/s3/rotate_keys');
      const { success, message, data } = res.data;
      if (success) {
        openS3SecretModal(data.access_key, data.secret_key, t('setting.personal.s3.rotate_keys'));
        await loadS3Self();
        showSuccess('密钥已轮换');
      } else {
        showError(message);
      }
    },
  });
};

const generateAccessToken = async () => {
  const res = await API.get('/api/user/token');
  const { success, message, data } = res.data;
  if (success) {
    systemToken.value = data;
    affLink.value = '';
    await copy(data);
    showSuccess('令牌已重置并已复制到剪贴板');
  } else {
    showError(message);
  }
};

const getAffLink = async () => {
  const res = await API.get('/api/user/aff');
  const { success, message, data } = res.data;
  if (success) {
    const link = `${window.location.origin}/register?aff=${data}`;
    affLink.value = link;
    systemToken.value = '';
    await copy(link);
    showSuccess('邀请链接已复制到剪切板');
  } else {
    showError(message);
  }
};

const handleAffLinkClick = async (e) => {
  e.target.select();
  await copy(e.target.value);
  showSuccess('邀请链接已复制到剪切板');
};

const handleSystemTokenClick = async (e) => {
  e.target.select();
  await copy(e.target.value);
  showSuccess('系统令牌已复制到剪切板');
};

const deleteAccount = async () => {
  if (inputs.self_account_deletion_confirmation !== userStore.user?.username) {
    showError('请输入你的账户名以确认删除！');
    return;
  }
  const res = await API.delete('/api/user/self');
  const { success, message } = res.data;
  if (success) {
    showSuccess('账户已删除！');
    await API.get('/api/user/logout');
    userStore.logout();
    clearTenantConsoleActingTenantId();
    clearNacosEmbeddedConsoleLocalSession();
    router.push('/login');
  } else {
    showError(message);
  }
};

const bindWeChat = async () => {
  if (inputs.wechat_verification_code === '') return;
  const res = await API.get(
    `/api/oauth/wechat/bind?code=${inputs.wechat_verification_code}`
  );
  const { success, message } = res.data;
  if (success) {
    showSuccess('微信账户绑定成功！');
    showWeChatBindModal.value = false;
  } else {
    showError(message);
  }
};

const sendVerificationCode = async () => {
  disableButton.value = true;
  if (inputs.email === '') return;
  if (turnstileEnabled.value && turnstileToken.value === '') {
    showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
    return;
  }
  loading.value = true;
  const res = await API.get(
    `/api/verification?email=${inputs.email}&turnstile=${turnstileToken.value}`
  );
  const { success, message } = res.data;
  if (success) {
    showSuccess('验证码发送成功，请检查邮箱！');
  } else {
    showError(message);
  }
  loading.value = false;
};

const bindEmail = async () => {
  if (inputs.email_verification_code === '') return;
  loading.value = true;
  const res = await API.get(
    `/api/oauth/email/bind?email=${inputs.email}&code=${inputs.email_verification_code}`
  );
  const { success, message } = res.data;
  if (success) {
    showSuccess('邮箱账户绑定成功！');
    showEmailBindModal.value = false;
  } else {
    showError(message);
  }
  loading.value = false;
};

const submitUpgrade = async () => {
  if (!upgradeInputs.name || !upgradeInputs.slug) {
    showError('企业名称和租户标识为必填项');
    return;
  }
  upgradeLoading.value = true;
  const res = await API.post('/api/user/tenant_upgrade', { ...upgradeInputs });
  const { success, message } = res.data;
  if (success) {
    showSuccess('租户升级申请提交成功，请等待管理员审核。');
    upgradeModalOpen.value = false;
  } else {
    showError(message || '提交失败');
  }
  upgradeLoading.value = false;
};
</script>

<style scoped>
.settings-action-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

h3 {
  margin-top: 0.5rem;
  margin-bottom: 0.75rem;
}
</style>
