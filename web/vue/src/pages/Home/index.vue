<template>
  <template v-if="homePageContentLoaded && homePageContent === ''">
    <div class="dashboard-container">
      <a-card class="chart-card">
        <div class="header" style="font-size: 1.2em; font-weight: 600; margin-bottom: 15px;">
          {{ t('home.welcome.title') }}
        </div>
        <div style="line-height: 1.6;">
          <p>{{ t('home.welcome.description') }}</p>
          <p v-if="!userStore.user">{{ t('home.welcome.login_notice') }}</p>
        </div>
      </a-card>

      <a-card class="chart-card">
        <h3>{{ t('home.system_status.title') }}</h3>
        <a-row :gutter="16">
          <a-col :xs="24" :md="12">
            <a-card class="chart-card" :style="{ boxShadow: '0 1px 3px rgba(0,0,0,0.12)' }">
              <h3 style="color: #444;">{{ t('home.system_status.info.title') }}</h3>
              <div style="line-height: 2; margin-top: 1em;">
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <InfoCircleOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.info.name') }}</span>
                  <span>{{ status?.system_name }}</span>
                </p>
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <BranchesOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.info.version') }}</span>
                  <span>{{ status?.version || 'unknown' }}</span>
                </p>
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <GithubOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.info.source') }}</span>
                  <a href="https://github.com/songquanpeng/one-api" target="_blank" style="color: #2185d0;">
                    {{ t('home.system_status.info.source_link') }}
                  </a>
                </p>
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <ClockCircleOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.info.start_time') }}</span>
                  <span>{{ getStartTimeString() }}</span>
                </p>
              </div>
            </a-card>
          </a-col>

          <a-col :xs="24" :md="12">
            <a-card class="chart-card" :style="{ boxShadow: '0 1px 3px rgba(0,0,0,0.12)' }">
              <h3 style="color: #444;">{{ t('home.system_status.config.title') }}</h3>
              <div style="line-height: 2; margin-top: 1em;">
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <MailOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.config.email_verify') }}</span>
                  <span :style="{ color: status?.email_verification ? '#21ba45' : '#db2828', fontWeight: '500' }">
                    {{ status?.email_verification ? t('home.system_status.config.enabled') : t('home.system_status.config.disabled') }}
                  </span>
                </p>
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <GithubOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.config.github_oauth') }}</span>
                  <span :style="{ color: status?.github_oauth ? '#21ba45' : '#db2828', fontWeight: '500' }">
                    {{ status?.github_oauth ? t('home.system_status.config.enabled') : t('home.system_status.config.disabled') }}
                  </span>
                </p>
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <WechatOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.config.wechat_login') }}</span>
                  <span :style="{ color: status?.wechat_login ? '#21ba45' : '#db2828', fontWeight: '500' }">
                    {{ status?.wechat_login ? t('home.system_status.config.enabled') : t('home.system_status.config.disabled') }}
                  </span>
                </p>
                <p style="display: flex; align-items: center; gap: 0.5em;">
                  <SafetyOutlined />
                  <span style="font-weight: bold;">{{ t('home.system_status.config.turnstile') }}</span>
                  <span :style="{ color: status?.turnstile_check ? '#21ba45' : '#db2828', fontWeight: '500' }">
                    {{ status?.turnstile_check ? t('home.system_status.config.enabled') : t('home.system_status.config.disabled') }}
                  </span>
                </p>
              </div>
            </a-card>
          </a-col>
        </a-row>
      </a-card>
    </div>
  </template>

  <template v-else>
    <iframe
      v-if="homePageContent.startsWith('https://')"
      :src="homePageContent"
      style="width: 100%; height: 100vh; border: none;"
    />
    <div v-else style="font-size: larger;" v-html="homePageContent"></div>
  </template>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { marked } from 'marked';
import {
  InfoCircleOutlined,
  BranchesOutlined,
  GithubOutlined,
  ClockCircleOutlined,
  MailOutlined,
  WechatOutlined,
  SafetyOutlined,
} from '@ant-design/icons-vue';
import {
  API,
  showError,
  showNotice,
  timestamp2string,
  isTenantAdmin,
  isTenantConsoleDelegate,
} from '@/helpers';
import { useStatusStore } from '@/stores/status';
import { useUserStore } from '@/stores/user';

const { t } = useI18n();
const router = useRouter();
const statusStore = useStatusStore();
const userStore = useUserStore();

const homePageContentLoaded = ref(false);
const homePageContent = ref('');

const status = computed(() => statusStore.status);

const displayNotice = async () => {
  const res = await API.get('/api/notice');
  const { success, message, data } = res.data;
  if (success) {
    let oldNotice = localStorage.getItem('notice');
    if (data !== oldNotice && data !== '') {
      const htmlNotice = marked(data);
      showNotice(htmlNotice, true);
      localStorage.setItem('notice', data);
    }
  } else {
    showError(message);
  }
};

const displayHomePageContent = async () => {
  homePageContent.value = localStorage.getItem('home_page_content') || '';
  const res = await API.get('/api/home_page_content');
  const { success, message, data } = res.data;
  if (success) {
    let content = data;
    if (!data.startsWith('https://')) {
      content = marked.parse(data);
    }
    homePageContent.value = content;
    localStorage.setItem('home_page_content', content);
  } else {
    showError(message);
    homePageContent.value = t('home.loading_failed');
  }
  homePageContentLoaded.value = true;
};

const getStartTimeString = () => {
  const timestamp = status.value?.start_time;
  return timestamp2string(timestamp);
};

onMounted(() => {
  if (isTenantAdmin() || isTenantConsoleDelegate()) {
    router.replace('/tenant-console');
    return;
  }
  displayNotice();
  displayHomePageContent();
});
</script>
