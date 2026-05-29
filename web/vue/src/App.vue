<template>
  <a-config-provider :locale="antdLocale" :theme="antdTheme">
    <div class="app-root-fill">
      <SidebarLayout>
        <router-view />
      </SidebarLayout>

      <a-modal
        :open="force2FAOpen"
        :closable="false"
        :mask-closable="false"
        :keyboard="false"
        :footer="null"
        width="520px"
        :title="t('auth.force_2fa.modal_title')"
      >
        <a-alert type="warning" :message="t('auth.force_2fa.modal_hint')" class="mb-3" />
        <TwoFASetting
          force-mode
          :cancel-login-busy="force2FACancelBusy"
          :cancel-login-label="t('auth.force_2fa.cancel_login')"
          @enabled="updateLocalForce2FAUser(false)"
          @cancel-login="cancelForce2FAAndReturnToLogin"
        />
        <div class="mt-3 text-right">
          <a-button :loading="force2FACancelBusy" :disabled="force2FACancelBusy" @click="cancelForce2FAAndReturnToLogin">
            {{ t('auth.force_2fa.cancel_login') }}
          </a-button>
        </div>
      </a-modal>
    </div>
  </a-config-provider>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
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
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const statusStore = useStatusStore();
const { user } = storeToRefs(userStore);
const { status } = storeToRefs(statusStore);

const antdLocale = computed(() => (currentLocale() === 'en' ? enUS : zhCN));

const isDark = useDark();
const antdTheme = computed(() => ({
  algorithm: isDark.value ? antdThemeApi.darkAlgorithm : antdThemeApi.defaultAlgorithm,
}));

const force2FAOpen = ref(false);
const force2FACancelBusy = ref(false);

function isPublicAuthPath() {
  const p = window.location.pathname;
  return (
    p === '/login' || p === '/tenant-login' || p === '/register' ||
    p === '/reset' || p === '/user/reset' || p.startsWith('/oauth/')
  );
}

function updateLocalForce2FAUser(required) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    force2FAOpen.value = false;
    return;
  }
  const data = JSON.parse(raw);
  const next = { ...data, require_force_2fa_setup: required };
  userStore.login(next);
  force2FAOpen.value = !!required;
}

async function cancelForce2FAAndReturnToLogin() {
  force2FACancelBusy.value = true;
  try {
    try {
      await API.get('/api/user/logout');
    } catch { /* clear local session regardless */ }
    userStore.logout();
    force2FAOpen.value = false;
    router.push('/login');
  } finally {
    force2FACancelBusy.value = false;
  }
}

onMounted(() => {
  userStore.loadFromStorage();
  statusStore.loadFromStorage();
  if (user.value) {
    force2FAOpen.value = !!user.value.require_force_2fa_setup && !isPublicAuthPath();
  }
  statusStore.loadStatus();
});

// Session change → drop the cached /api/status payload and refetch.
watch(
  () => user.value?.id,
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

watch(
  () => user.value?.require_force_2fa_setup,
  (v) => {
    force2FAOpen.value = !!v && !isPublicAuthPath();
  }
);
</script>
