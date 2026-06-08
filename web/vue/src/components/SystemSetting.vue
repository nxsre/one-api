<template>
  <a-spin :spinning="loading">
    <a-form layout="vertical" @submit.prevent>
      <h3>{{ t('setting.system.general.title') }}</h3>
      <a-form-item :label="t('setting.system.general.server_address')">
        <a-input
          v-model:value="inputs.ServerAddress"
          :placeholder="t('setting.system.general.server_address_placeholder')"
          v-bind="noAutofillTextProps"
        />
      </a-form-item>
      <a-button @click="submitServerAddress">
        {{ t('setting.system.general.buttons.update') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.security.title') }}</h3>
      <a-alert type="info" :message="t('setting.system.security.subtitle')" style="margin-bottom: 1rem" />
      <h4>{{ t('setting.system.security.ssrf.title') }}</h4>
      <a-alert type="warning" :message="t('setting.system.security.ssrf.subtitle')" style="margin-bottom: 1rem" />
      <a-form-item>
        <a-checkbox
          :checked="inputs.OutboundSSRFSProtectionEnabled === 'true'"
          @change="updateOption('OutboundSSRFSProtectionEnabled')"
        >
          {{ t('setting.system.security.ssrf.enable') }}
        </a-checkbox>
      </a-form-item>
      <a-divider />
      <h4>{{ t('setting.system.security.outbound_whitelist.title') }}</h4>
      <a-alert
        type="warning"
        :message="t('setting.system.security.outbound_whitelist.subtitle')"
        style="margin-bottom: 1rem"
      />
      <a-form-item>
        <a-checkbox
          :checked="inputs.OutboundURLWhitelistEnabled === 'true'"
          @change="updateOption('OutboundURLWhitelistEnabled')"
        >
          {{ t('setting.system.security.outbound_whitelist.enable') }}
        </a-checkbox>
      </a-form-item>
      <a-form-item :label="t('setting.system.security.outbound_whitelist.domains')">
        <a-textarea
          v-model:value="inputs.OutboundURLWhitelistDomains"
          :placeholder="t('setting.system.security.outbound_whitelist.domains_placeholder')"
          :rows="4"
        />
      </a-form-item>
      <a-form-item :label="t('setting.system.security.outbound_whitelist.ips')">
        <a-textarea
          v-model:value="inputs.OutboundURLWhitelistIPs"
          :placeholder="t('setting.system.security.outbound_whitelist.ips_placeholder')"
          :rows="4"
        />
      </a-form-item>
      <a-button @click="submitOutboundWhitelist">
        {{ t('setting.system.security.outbound_whitelist.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.login.title') }}</h3>
      <a-form-item>
        <div class="settings-checkbox-group">
          <a-checkbox
            :checked="inputs.PasswordLoginEnabled === 'true'"
            @change="onPasswordLoginChange"
          >
            {{ t('setting.system.login.password_login') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.SecurePasswordLoginEnabled === 'true'"
            @change="updateOption('SecurePasswordLoginEnabled')"
          >
            {{ t('setting.system.login.secure_password_login') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.PasswordRegisterEnabled === 'true'"
            @change="updateOption('PasswordRegisterEnabled')"
          >
            {{ t('setting.system.login.password_register') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.EmailVerificationEnabled === 'true'"
            @change="updateOption('EmailVerificationEnabled')"
          >
            {{ t('setting.system.login.email_verification') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.Force2FAForAllUsers === 'true'"
            @change="updateOption('Force2FAForAllUsers')"
          >
            {{ t('setting.system.login.force_mfa') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.GitHubOAuthEnabled === 'true'"
            @change="updateOption('GitHubOAuthEnabled')"
          >
            {{ t('setting.system.login.github_oauth') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.WeChatAuthEnabled === 'true'"
            @change="updateOption('WeChatAuthEnabled')"
          >
            {{ t('setting.system.login.wechat_login') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.LarkOAuthEnabled === 'true'"
            @change="updateOption('LarkOAuthEnabled')"
          >
            {{ t('setting.system.login.lark_oauth') }}
          </a-checkbox>
        </div>
      </a-form-item>
      <a-form-item>
        <div class="settings-checkbox-group">
          <a-checkbox
            :checked="inputs.RegisterEnabled === 'true'"
            @change="updateOption('RegisterEnabled')"
          >
            {{ t('setting.system.login.registration') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.TurnstileCheckEnabled === 'true'"
            @change="updateOption('TurnstileCheckEnabled')"
          >
            {{ t('setting.system.login.turnstile') }}
          </a-checkbox>
          <a-checkbox
            :checked="inputs.LoginCaptchaEnabled === 'true'"
            @change="updateOption('LoginCaptchaEnabled')"
          >
            {{ t('setting.system.login.login_captcha') }}
          </a-checkbox>
        </div>
      </a-form-item>

      <a-modal
        v-model:open="showPasswordWarningModal"
        :title="t('setting.system.password_login.warning.title')"
        :width="450"
      >
        <p>{{ t('setting.system.password_login.warning.content') }}</p>
        <template #footer>
          <a-button @click="showPasswordWarningModal = false">
            {{ t('setting.system.password_login.warning.buttons.cancel') }}
          </a-button>
          <a-button type="primary" danger @click="confirmDisablePasswordLogin">
            {{ t('setting.system.password_login.warning.buttons.confirm') }}
          </a-button>
        </template>
      </a-modal>

      <a-divider />
      <h3>{{ t('setting.system.nacos.title') }}</h3>
      <a-alert type="info" :message="t('setting.system.nacos.subtitle')" style="margin-bottom: 1rem" />
      <a-form-item>
        <a-checkbox
          :checked="inputs.NacosEnabled === 'true'"
          @change="updateOption('NacosEnabled')"
        >
          {{ t('setting.system.nacos.enable') }}
        </a-checkbox>
      </a-form-item>

      <a-divider />
      <h3>{{ t('setting.system.s3.title') }}</h3>
      <a-alert type="info" :message="t('setting.system.s3.subtitle')" style="margin-bottom: 1rem" />
      <a-form-item>
        <a-checkbox
          :checked="inputs.S3SiteEnabled === 'true'"
          @change="updateOption('S3SiteEnabled')"
        >
          {{ t('setting.system.s3.enable_site') }}
        </a-checkbox>
      </a-form-item>

      <a-divider />
      <h3>{{ t('setting.system.email_restriction.title') }}</h3>
      <a-alert type="info" :message="t('setting.system.email_restriction.subtitle')" style="margin-bottom: 1rem" />
      <a-form-item>
        <a-checkbox
          :checked="inputs.EmailDomainRestrictionEnabled === 'true'"
          @change="updateOption('EmailDomainRestrictionEnabled')"
        >
          {{ t('setting.system.email_restriction.enable') }}
        </a-checkbox>
      </a-form-item>
      <a-form-item :label="t('setting.system.email_restriction.add_domain')">
        <a-input-search
          v-model:value="restrictedDomainInput"
          :placeholder="t('setting.system.email_restriction.add_domain_placeholder')"
          :enter-button="t('setting.system.email_restriction.buttons.fill')"
          style="max-width: 420px"
          @search="fillRestrictedDomain"
        />
      </a-form-item>
      <a-form-item :label="t('setting.system.email_restriction.allowed_domains')">
        <a-select
          v-model:value="emailDomainWhitelist"
          mode="tags"
          style="width: 100%"
          :placeholder="t('setting.system.email_restriction.allowed_domains')"
          :token-separators="[',']"
        />
      </a-form-item>
      <a-button @click="submitEmailDomainWhitelist">
        {{ t('setting.system.email_restriction.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.smtp.title') }}</h3>
      <a-alert type="info" :message="t('setting.system.smtp.subtitle')" style="margin-bottom: 1rem" />
      <div class="settings-grid">
        <a-form-item :label="t('setting.system.smtp.server')">
          <a-input
            v-model:value="inputs.SMTPServer"
            :placeholder="t('setting.system.smtp.server_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.smtp.port')">
          <a-input
            v-model:value="inputs.SMTPPort"
            :placeholder="t('setting.system.smtp.port_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.smtp.account')">
          <a-input
            v-model:value="inputs.SMTPAccount"
            :placeholder="t('setting.system.smtp.account_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.smtp.from')">
          <a-input
            v-model:value="inputs.SMTPFrom"
            :placeholder="t('setting.system.smtp.from_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.smtp.token')">
          <a-input-password
            v-model:value="inputs.SMTPToken"
            :placeholder="t('setting.system.smtp.token_placeholder')"
          />
        </a-form-item>
      </div>
      <a-button @click="submitSMTP">
        {{ t('setting.system.smtp.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.github.title') }}</h3>
      <a-alert type="info" style="margin-bottom: 0.5rem">
        <template #message>
          {{ t('setting.system.github.subtitle') }}
          <a href="https://github.com/settings/developers" target="_blank">
            {{ t('setting.system.github.manage_link') }}
          </a>
          {{ t('setting.system.github.manage_text') }}
        </template>
      </a-alert>
      <a-alert
        type="info"
        :message="t('setting.system.github.url_notice', {
          server_url: originInputs.ServerAddress,
          callback_url: `${originInputs.ServerAddress}/oauth/github`,
        })"
        style="margin-bottom: 1rem"
      />
      <div class="settings-grid">
        <a-form-item :label="t('setting.system.github.client_id')">
          <a-input
            v-model:value="inputs.GitHubClientId"
            :placeholder="t('setting.system.github.client_id_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.github.client_secret')">
          <a-input-password
            v-model:value="inputs.GitHubClientSecret"
            :placeholder="t('setting.system.github.client_secret_placeholder')"
          />
        </a-form-item>
      </div>
      <a-button @click="submitGitHubOAuth">
        {{ t('setting.system.github.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.lark.title') }}</h3>
      <p class="settings-subheader">
        {{ t('setting.system.lark.subtitle') }}
        <a href="https://open.feishu.cn/app" target="_blank">
          {{ t('setting.system.lark.manage_link') }}
        </a>
        {{ t('setting.system.lark.manage_text') }}
      </p>
      <a-alert
        type="info"
        :message="t('setting.system.lark.url_notice', {
          server_url: inputs.ServerAddress,
          callback_url: `${inputs.ServerAddress}/oauth/lark`,
        })"
        style="margin-bottom: 1rem"
      />
      <div class="settings-grid">
        <a-form-item :label="t('setting.system.lark.client_id')">
          <a-input
            v-model:value="inputs.LarkClientId"
            :placeholder="t('setting.system.lark.client_id_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.lark.client_secret')">
          <a-input-password
            v-model:value="inputs.LarkClientSecret"
            :placeholder="t('setting.system.lark.client_secret_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
      </div>
      <a-button @click="submitLarkOAuth">
        {{ t('setting.system.lark.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.wechat.title') }}</h3>
      <p class="settings-subheader">
        {{ t('setting.system.wechat.subtitle') }}
        <a href="https://github.com/songquanpeng/wechat-server" target="_blank">
          {{ t('setting.system.wechat.learn_more') }}
        </a>
      </p>
      <div class="settings-grid">
        <a-form-item :label="t('setting.system.wechat.server_address')">
          <a-input
            v-model:value="inputs.WeChatServerAddress"
            :placeholder="t('setting.system.wechat.server_address_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.wechat.token')">
          <a-input-password
            v-model:value="inputs.WeChatServerToken"
            :placeholder="t('setting.system.wechat.token_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.wechat.qrcode')">
          <a-input
            v-model:value="inputs.WeChatAccountQRCodeImageURL"
            :placeholder="t('setting.system.wechat.qrcode_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
      </div>
      <a-button @click="submitWeChat">
        {{ t('setting.system.wechat.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.system.turnstile.title') }}</h3>
      <p class="settings-subheader">
        {{ t('setting.system.turnstile.subtitle') }}
        <a href="https://dash.cloudflare.com/" target="_blank">
          {{ t('setting.system.turnstile.manage_link') }}
        </a>
        {{ t('setting.system.turnstile.manage_text') }}
      </p>
      <div class="settings-grid">
        <a-form-item :label="t('setting.system.turnstile.site_key')">
          <a-input
            v-model:value="inputs.TurnstileSiteKey"
            :placeholder="t('setting.system.turnstile.site_key_placeholder')"
            v-bind="noAutofillTextProps"
          />
        </a-form-item>
        <a-form-item :label="t('setting.system.turnstile.secret_key')">
          <a-input-password
            v-model:value="inputs.TurnstileSecretKey"
            :placeholder="t('setting.system.turnstile.secret_key_placeholder')"
            v-bind="noAutofillSecretProps"
          />
        </a-form-item>
      </div>
      <a-button @click="submitTurnstile">
        {{ t('setting.system.turnstile.buttons.save') }}
      </a-button>
    </a-form>
  </a-spin>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  API,
  removeTrailingSlash,
  showError,
  noAutofillSecretProps,
  noAutofillTextProps,
} from '@/helpers';
import { useStatusStore } from '@/stores/status';

const { t } = useI18n();
const statusStore = useStatusStore();

const inputs = reactive({
  PasswordLoginEnabled: '',
  PasswordRegisterEnabled: '',
  EmailVerificationEnabled: '',
  Force2FAForAllUsers: '',
  GitHubOAuthEnabled: '',
  LarkOAuthEnabled: '',
  GitHubClientId: '',
  GitHubClientSecret: '',
  LarkClientId: '',
  LarkClientSecret: '',
  Notice: '',
  SMTPServer: '',
  SMTPPort: '',
  SMTPAccount: '',
  SMTPFrom: '',
  SMTPToken: '',
  ServerAddress: '',
  Footer: '',
  WeChatAuthEnabled: '',
  WeChatServerAddress: '',
  WeChatServerToken: '',
  WeChatAccountQRCodeImageURL: '',
  MessagePusherAddress: '',
  MessagePusherToken: '',
  TurnstileCheckEnabled: '',
  LoginCaptchaEnabled: '',
  TurnstileSiteKey: '',
  TurnstileSecretKey: '',
  RegisterEnabled: '',
  EmailDomainRestrictionEnabled: '',
  EmailDomainWhitelist: '',
  S3SiteEnabled: '',
  NacosEnabled: '',
  SecurePasswordLoginEnabled: '',
  OutboundSSRFSProtectionEnabled: 'true',
  OutboundURLWhitelistEnabled: 'false',
  OutboundURLWhitelistDomains: '',
  OutboundURLWhitelistIPs: '',
});
const originInputs = reactive({});
const loading = ref(false);
const emailDomainWhitelist = ref([]);
const restrictedDomainInput = ref('');
const showPasswordWarningModal = ref(false);

const TOGGLE_KEYS = new Set([
  'PasswordLoginEnabled',
  'PasswordRegisterEnabled',
  'EmailVerificationEnabled',
  'Force2FAForAllUsers',
  'GitHubOAuthEnabled',
  'LarkOAuthEnabled',
  'WeChatAuthEnabled',
  'TurnstileCheckEnabled',
  'LoginCaptchaEnabled',
  'EmailDomainRestrictionEnabled',
  'RegisterEnabled',
  'S3SiteEnabled',
  'NacosEnabled',
  'SecurePasswordLoginEnabled',
  'OutboundSSRFSProtectionEnabled',
  'OutboundURLWhitelistEnabled',
]);

const getOptions = async () => {
  const res = await API.get('/api/option/');
  const { success, message, data } = res.data;
  if (success) {
    const newInputs = {};
    data.forEach((item) => {
      newInputs[item.key] = item.value;
    });
    const merged = {
      OutboundSSRFSProtectionEnabled: 'true',
      OutboundURLWhitelistEnabled: 'false',
      OutboundURLWhitelistDomains: '',
      OutboundURLWhitelistIPs: '',
      ...newInputs,
    };
    Object.keys(originInputs).forEach((k) => delete originInputs[k]);
    Object.assign(originInputs, merged);
    Object.assign(inputs, merged);
    emailDomainWhitelist.value = (merged.EmailDomainWhitelist || '')
      .split(',')
      .filter((x) => x !== '');
  } else {
    showError(message);
  }
};

onMounted(() => {
  getOptions();
});

const applyStatusPatch = (patch) => {
  try {
    const st = JSON.parse(localStorage.getItem('status') || '{}');
    Object.assign(st, patch);
    localStorage.setItem('status', JSON.stringify(st));
    statusStore.loadFromStorage();
  } catch {
    /* ignore */
  }
};

const updateOption = async (key, value) => {
  loading.value = true;
  if (TOGGLE_KEYS.has(key)) {
    value = inputs[key] === 'true' ? 'false' : 'true';
  }
  const res = await API.put('/api/option/', { key, value });
  const { success, message } = res.data;
  if (success) {
    if (key === 'NacosEnabled') {
      applyStatusPatch({ nacos_enabled: value === 'true' });
    }
    if (key === 'SecurePasswordLoginEnabled') {
      applyStatusPatch({ secure_password_login: value === 'true' });
    }
    if (key === 'LoginCaptchaEnabled') {
      if (value === 'false') {
        applyStatusPatch({ login_math_captcha: false });
      } else {
        API.get('/api/status').then((r) => {
          const payload = r.data?.data;
          if (!payload) return;
          applyStatusPatch(payload);
        });
      }
    }
    inputs[key] = value;
  } else {
    showError(message);
  }
  loading.value = false;
};

const onPasswordLoginChange = () => {
  if (inputs.PasswordLoginEnabled === 'true') {
    // block disabling password login
    showPasswordWarningModal.value = true;
    return;
  }
  updateOption('PasswordLoginEnabled');
};

const confirmDisablePasswordLogin = async () => {
  showPasswordWarningModal.value = false;
  await updateOption('PasswordLoginEnabled', 'false');
};

const submitServerAddress = async () => {
  await updateOption('ServerAddress', removeTrailingSlash(inputs.ServerAddress));
};

const submitOutboundWhitelist = async () => {
  const domainsChanged =
    (originInputs.OutboundURLWhitelistDomains || '') !==
    (inputs.OutboundURLWhitelistDomains || '');
  const ipsChanged =
    (originInputs.OutboundURLWhitelistIPs || '') !==
    (inputs.OutboundURLWhitelistIPs || '');
  if (!domainsChanged && !ipsChanged) return;
  if (domainsChanged) {
    await updateOption('OutboundURLWhitelistDomains', inputs.OutboundURLWhitelistDomains || '');
  }
  if (ipsChanged) {
    await updateOption('OutboundURLWhitelistIPs', inputs.OutboundURLWhitelistIPs || '');
  }
  await getOptions();
};

const submitSMTP = async () => {
  if (originInputs.SMTPServer !== inputs.SMTPServer) {
    await updateOption('SMTPServer', inputs.SMTPServer);
  }
  if (originInputs.SMTPAccount !== inputs.SMTPAccount) {
    await updateOption('SMTPAccount', inputs.SMTPAccount);
  }
  if (originInputs.SMTPFrom !== inputs.SMTPFrom) {
    await updateOption('SMTPFrom', inputs.SMTPFrom);
  }
  if (originInputs.SMTPPort !== inputs.SMTPPort && inputs.SMTPPort !== '') {
    await updateOption('SMTPPort', inputs.SMTPPort);
  }
  if (originInputs.SMTPToken !== inputs.SMTPToken && inputs.SMTPToken !== '') {
    await updateOption('SMTPToken', inputs.SMTPToken);
  }
};

const fillRestrictedDomain = () => {
  const value = restrictedDomainInput.value;
  if (value === '') return;
  if (!emailDomainWhitelist.value.includes(value)) {
    emailDomainWhitelist.value = [...emailDomainWhitelist.value, value];
  }
  restrictedDomainInput.value = '';
};

const submitEmailDomainWhitelist = async () => {
  const joined = emailDomainWhitelist.value.join(',');
  if (originInputs.EmailDomainWhitelist !== joined && inputs.SMTPToken !== '') {
    await updateOption('EmailDomainWhitelist', joined);
  }
};

const submitWeChat = async () => {
  if (originInputs.WeChatServerAddress !== inputs.WeChatServerAddress) {
    await updateOption('WeChatServerAddress', removeTrailingSlash(inputs.WeChatServerAddress));
  }
  if (originInputs.WeChatAccountQRCodeImageURL !== inputs.WeChatAccountQRCodeImageURL) {
    await updateOption('WeChatAccountQRCodeImageURL', inputs.WeChatAccountQRCodeImageURL);
  }
  if (originInputs.WeChatServerToken !== inputs.WeChatServerToken && inputs.WeChatServerToken !== '') {
    await updateOption('WeChatServerToken', inputs.WeChatServerToken);
  }
};

const submitGitHubOAuth = async () => {
  if (originInputs.GitHubClientId !== inputs.GitHubClientId) {
    await updateOption('GitHubClientId', inputs.GitHubClientId);
  }
  if (originInputs.GitHubClientSecret !== inputs.GitHubClientSecret && inputs.GitHubClientSecret !== '') {
    await updateOption('GitHubClientSecret', inputs.GitHubClientSecret);
  }
};

const submitLarkOAuth = async () => {
  if (originInputs.LarkClientId !== inputs.LarkClientId) {
    await updateOption('LarkClientId', inputs.LarkClientId);
  }
  if (originInputs.LarkClientSecret !== inputs.LarkClientSecret && inputs.LarkClientSecret !== '') {
    await updateOption('LarkClientSecret', inputs.LarkClientSecret);
  }
};

const submitTurnstile = async () => {
  if (originInputs.TurnstileSiteKey !== inputs.TurnstileSiteKey) {
    await updateOption('TurnstileSiteKey', inputs.TurnstileSiteKey);
  }
  if (originInputs.TurnstileSecretKey !== inputs.TurnstileSecretKey && inputs.TurnstileSecretKey !== '') {
    await updateOption('TurnstileSecretKey', inputs.TurnstileSecretKey);
  }
};
</script>

<style scoped>
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0 1rem;
}

.settings-checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1.5rem;
}

.settings-subheader {
  color: rgba(0, 0, 0, 0.55);
  margin-bottom: 0.75rem;
}

h3 {
  margin-top: 0.5rem;
  margin-bottom: 0.75rem;
}
</style>
