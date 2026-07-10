<template>
  <a-config-provider :locale="antdLocale" :theme="antdTheme">
    <div v-if="showForce2FASetup" class="force-2fa-page">
      <div class="force-2fa-card">
        <h2 class="force-2fa-card__title">{{ t('auth.force_2fa.modal_title') }}</h2>
        <a-alert type="warning" :message="t('auth.force_2fa.modal_hint')" class="mb-3" />
        <TwoFASetting
          force-mode
          :cancel-login-busy="force2FACancelBusy"
          :cancel-login-label="t('auth.force_2fa.cancel_login')"
          @enabled="updateLocalForce2FAUser(false)"
          @cancel-login="cancelForce2FAAndReturnToLogin"
        />
      </div>
    </div>

    <div v-else class="app-root-fill">
      <SidebarLayout>
        <router-view />
      </SidebarLayout>
    </div>
  </a-config-provider>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { theme as antdThemeApi } from 'ant-design-vue';
import zhCN from 'ant-design-vue/es/locale/zh_CN';
import enUS from 'ant-design-vue/es/locale/en_US';
import { useDark } from './composables/useDark';
import SidebarLayout from './layouts/SidebarLayout.vue';
import TwoFASetting from './components/TwoFASetting.vue';
import { useUserStore } from './stores/user';
import { useStatusStore } from './stores/status';
import { storeToRefs } from 'pinia';
import { API, getLogo, getSystemName } from './helpers';
import { currentLocale } from './i18n';

const { t } = useI18n();
const userStore = useUserStore();
const statusStore = useStatusStore();
const { user } = storeToRefs(userStore);
const { status } = storeToRefs(statusStore);

const antdLocale = computed(() => (currentLocale() === 'en' ? enUS : zhCN));

const isDark = useDark();
const antdTheme = computed(() => ({
  algorithm: isDark.value ? antdThemeApi.darkAlgorithm : antdThemeApi.defaultAlgorithm,
}));

const force2FACancelBusy = ref(false);

function isPublicAuthPath() {
  const p = window.location.pathname;
  return (
    p === '/login' || p === '/tenant-login' || p === '/register' ||
    p === '/reset' || p === '/user/reset' || p.startsWith('/oauth/')
  );
}

const showForce2FASetup = computed(
  () => !!user.value?.require_force_2fa_setup && !isPublicAuthPath()
);

function updateLocalForce2FAUser(required) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    return;
  }
  const data = JSON.parse(raw);
  const next = { ...data, require_force_2fa_setup: required };
  userStore.login(next);
}

async function cancelForce2FAAndReturnToLogin() {
  if (force2FACancelBusy.value) return;
  force2FACancelBusy.value = true;
  try {
    try {
      await API.get('/api/user/logout');
    } catch { /* clear local session regardless */ }
    userStore.logout();
    window.location.assign('/login');
  } finally {
    force2FACancelBusy.value = false;
  }
}

onMounted(() => {
  userStore.loadFromStorage();
  statusStore.loadFromStorage();
  statusStore.loadStatus();
});

// Session change → drop the cached /api/status payload and refetch.
watch(
  () => user.value?.id ?? user.value?.user_id,
  () => {
    statusStore.reload();
  }
);

watch(
  () => status.value,
  () => {
    const name = status.value?.system_name || getSystemName();
    if (name) document.title = name;
    const logo = status.value?.logo || getLogo();
    if (logo) {
      const link = document.querySelector("link[rel~='icon']");
      if (link) link.href = logo;
    }
  },
  { deep: true }
);
</script>

<style scoped>
.force-2fa-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #f0f2f5;
}

.force-2fa-card {
  width: 100%;
  max-width: 560px;
  padding: 24px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.force-2fa-card__title {
  margin: 0 0 16px;
  font-size: 20px;
  font-weight: 600;
  text-align: center;
}
</style>
