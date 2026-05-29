<template>
  <!-- Public auth pages render without the chrome, matching the React theme. -->
  <div v-if="isPublicLayout" class="app-public-wrap min-h-screen flex flex-col">
    <div class="app-public-scroll flex-1">
      <slot />
    </div>
    <Footer />
  </div>

  <a-layout v-else style="min-height: 100vh">
    <a-layout-sider
      v-model:collapsed="collapsed"
      collapsible
      :trigger="null"
      :theme="isDark ? 'dark' : 'light'"
      class="app-sidebar"
      :width="240"
    >
      <div class="app-sidebar-brand h-14 flex items-center px-4 overflow-hidden cursor-pointer" @click="go('/')">
        <img :src="logo" alt="logo" class="h-7 w-7 flex-shrink-0" />
        <span v-if="!collapsed" class="ml-2 font-semibold truncate">{{ systemName }}</span>
      </div>
      <a-menu
        mode="inline"
        :theme="isDark ? 'dark' : 'light'"
        :selected-keys="selectedKeys"
        v-model:open-keys="openKeys"
        :items="menuItems"
        @click="onMenuClick"
      />
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="app-topbar flex items-center justify-between" :style="{ background: isDark ? '#141414' : '#fff', padding: '0 16px' }">
        <a-button type="text" @click="collapsed = !collapsed">
          <template #icon>
            <MenuUnfoldOutlined v-if="collapsed" />
            <MenuFoldOutlined v-else />
          </template>
        </a-button>
        <div class="flex items-center gap-2">
          <a-button type="text" size="small" :title="t('header.language_switch_tooltip')" @click="toggleLanguage">
            <template #icon><TranslationOutlined /></template>
          </a-button>
          <NacosThemeToggle />
          <template v-if="user">
            <a-dropdown>
              <span class="flex items-center gap-2 cursor-pointer">
                <a-avatar size="small">{{ userInitial }}</a-avatar>
                <span class="hidden sm:inline">{{ userDisplayName }}</span>
              </span>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="setting" @click="go('/setting')">{{ t('header.setting') }}</a-menu-item>
                  <a-menu-item key="logout" @click="logout">{{ t('header.logout') }}</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </template>
          <template v-else>
            <a-button size="small" @click="go('/login')">{{ t('header.login') }}</a-button>
            <a-button size="small" type="primary" @click="go('/register')">{{ t('header.register') }}</a-button>
          </template>
        </div>
      </a-layout-header>

      <a-layout-content class="app-main-content">
        <div class="app-main-page-scroll">
          <TenantConsoleActingBar v-if="showActingBar" />
          <slot />
        </div>
        <Footer />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { h, ref, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import {
  HomeOutlined, CloudOutlined, BankOutlined, FileTextOutlined,
  DeploymentUnitOutlined, UnorderedListOutlined, ControlOutlined,
  ApartmentOutlined, KeyOutlined, DollarOutlined, ShoppingCartOutlined,
  UserOutlined, BarChartOutlined, BookOutlined, SettingOutlined,
  InfoCircleOutlined, MessageOutlined, ExportOutlined, FolderOpenOutlined,
  BranchesOutlined, ApiOutlined, TeamOutlined, FileOutlined,
  OrderedListOutlined, SafetyOutlined, MenuFoldOutlined, MenuUnfoldOutlined,
  TranslationOutlined, ClusterOutlined, CheckSquareOutlined,
} from '@ant-design/icons-vue';
import Footer from '../components/Footer.vue';
import NacosThemeToggle from '../components/NacosThemeToggle.vue';
import TenantConsoleActingBar from '../components/TenantConsoleActingBar.vue';
import { useUserStore } from '../stores/user';
import { useStatusStore } from '../stores/status';
import { storeToRefs } from 'pinia';
import {
  API, getLogo, getSystemName, hasTenantPermission, isAdmin, isRoot,
  isNacosEnabled, isTenantAdmin, isTenantConsoleDelegate, showSuccess,
} from '../helpers';
import { setLocale, currentLocale } from '../i18n';
import { useDark } from '../composables/useDark';

const isDark = useDark();

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const statusStore = useStatusStore();
const { user } = storeToRefs(userStore);
const { status } = storeToRefs(statusStore);

const logo = getLogo();
const systemName = getSystemName();

const collapsed = ref(localStorage.getItem('one-api-sidebar-collapsed') === '1');
watch(collapsed, (v) => {
  try { localStorage.setItem('one-api-sidebar-collapsed', v ? '1' : '0'); } catch { /* ignore */ }
});

const publicPaths = new Set(['/login', '/tenant-login', '/register', '/reset', '/user/reset']);
const isPublicLayout = computed(() => {
  const p = route.path;
  return publicPaths.has(p) || p.startsWith('/oauth/');
});

const nacosMenuEnabled = computed(() => {
  const v = status.value?.nacos_enabled;
  return v !== false && v !== 'false' && isNacosEnabled();
});

const showActingBar = computed(
  () => route.path.startsWith('/tenant-console') && isAdmin() && !isTenantAdmin()
);

const userDisplayName = computed(() => {
  const u = user.value;
  if (!u) return '';
  const d = u.display_name && String(u.display_name).trim();
  return d || u.username || '';
});
const userInitial = computed(() => {
  const n = userDisplayName.value;
  return n ? n.charAt(0).toUpperCase() : '?';
});

function icon(comp) {
  return () => h(comp);
}

// Build the Antd Menu items tree with role gating, mirroring the React nav.
const menuItems = computed(() => {
  const items = [];

  if (isTenantAdmin() || isTenantConsoleDelegate()) {
    const showTenantNacos = nacosMenuEnabled.value && (isTenantAdmin() || hasTenantPermission('manage_nacos'));
    if (showTenantNacos) items.push(nacosGroup());
    items.push({ key: '/tenant-console', icon: icon(BankOutlined), label: t('header.tenant_console') });
    if (hasTenantPermission('manage_users') || hasTenantPermission('manage_tokens')) {
      items.push({ key: '/tenant-console/users', icon: icon(TeamOutlined), label: t('header.tenant_subusers') });
    }
    if (hasTenantPermission('manage_channels')) {
      items.push({ key: '/tenant-console/channels', icon: icon(ApartmentOutlined), label: t('header.tenant_channels') });
    }
    if (hasTenantPermission('manage_billing')) {
      items.push({ key: '/tenant-console/reports', icon: icon(FileTextOutlined), label: t('header.tenant_billing_reports') });
    }
    if (isTenantAdmin()) {
      items.push(
        { key: '/token', icon: icon(KeyOutlined), label: t('header.token') },
        { key: '/log', icon: icon(BookOutlined), label: t('header.log') },
        { key: '/dashboard', icon: icon(BarChartOutlined), label: t('header.dashboard') },
        { key: '/topup', icon: icon(ShoppingCartOutlined), label: t('header.topup') }
      );
    }
    items.push(
      { key: '/setting', icon: icon(SettingOutlined), label: t('header.setting') },
      { key: '/about', icon: icon(InfoCircleOutlined), label: t('header.about') }
    );
    return items;
  }

  const admin = isAdmin();
  const adminOrTenant = admin || isTenantAdmin();

  items.push({ key: '/', icon: icon(HomeOutlined), label: t('header.home') });
  if (localStorage.getItem('chat_link')) {
    items.push({ key: '/chat', icon: icon(MessageOutlined), label: t('header.chat') });
  }
  if (nacosMenuEnabled.value && adminOrTenant) items.push(nacosGroup());
  if (adminOrTenant) items.push(tenantGroup());
  if (admin) items.push({ key: '/platform-reports', icon: icon(FileTextOutlined), label: t('header.tenant_reports') });
  if (admin) items.push({ key: '/routing', icon: icon(DeploymentUnitOutlined), label: t('header.routing') });
  if (admin) items.push({ key: '/model-catalog', icon: icon(UnorderedListOutlined), label: t('header.model_catalog') });
  if (admin && isRoot()) items.push({ key: '/operations', icon: icon(ControlOutlined), label: t('header.operations_management') });
  if (admin) items.push({ key: '/channel', icon: icon(ApartmentOutlined), label: t('header.channel') });
  items.push({ key: '/token', icon: icon(KeyOutlined), label: t('header.token') });
  if (admin) items.push({ key: '/redemption', icon: icon(DollarOutlined), label: t('header.redemption') });
  items.push({ key: '/topup', icon: icon(ShoppingCartOutlined), label: t('header.topup') });
  if (admin) items.push({ key: '/user', icon: icon(UserOutlined), label: t('header.user') });
  items.push(
    { key: '/dashboard', icon: icon(BarChartOutlined), label: t('header.dashboard') },
    { key: '/log', icon: icon(BookOutlined), label: t('header.log') },
    { key: '/setting', icon: icon(SettingOutlined), label: t('header.setting') },
    { key: '/about', icon: icon(InfoCircleOutlined), label: t('header.about') }
  );
  return items;
});

function nacosGroup() {
  return {
    key: 'nacos',
    icon: icon(CloudOutlined),
    label: t('header.nacos_menu'),
    children: [
      { key: 'nacos-console-ext', icon: icon(ExportOutlined), label: t('header.nacos_sub_console') },
      { key: '/nacos/namespaces', icon: icon(FolderOpenOutlined), label: t('header.nacos_sub_namespaces') },
      { key: '/nacos/cs', icon: icon(SettingOutlined), label: t('header.nacos_sub_cs') },
      { key: '/nacos/skills', icon: icon(BranchesOutlined), label: t('header.nacos_sub_skills') },
      { key: '/nacos/agentspecs', icon: icon(ApartmentOutlined), label: t('header.nacos_sub_agentspecs') },
      { key: '/nacos/mcp', icon: icon(ApiOutlined), label: t('header.nacos_sub_mcp') },
      { key: '/nacos/a2a', icon: icon(TeamOutlined), label: t('header.nacos_sub_a2a') },
      { key: '/nacos/prompts', icon: icon(FileOutlined), label: t('header.nacos_sub_prompts') },
      { key: '/nacos/pipelines', icon: icon(OrderedListOutlined), label: t('header.nacos_sub_pipelines') },
      { key: '/nacos/permissions', icon: icon(SafetyOutlined), label: t('header.nacos_sub_perm') },
    ],
  };
}

function tenantGroup() {
  return {
    key: 'tenant',
    icon: icon(BankOutlined),
    label: t('header.tenant_platform'),
    children: [
      { type: 'group', label: t('header.tenant_platform_mgmt'), children: [
        { key: '/tenant-upgrades', icon: icon(CheckSquareOutlined), label: t('header.tenant_upgrades') },
        { key: '/tenant-management', icon: icon(ClusterOutlined), label: t('header.tenant_management') },
      ] },
      { type: 'group', label: t('header.tenant_platform_console'), children: [
        { key: '/tenant-console/users', icon: icon(TeamOutlined), label: t('header.tenant_subusers') },
        { key: '/tenant-console/channels', icon: icon(ApartmentOutlined), label: t('header.tenant_channels') },
        { key: '/tenant-console/reports', icon: icon(FileTextOutlined), label: t('header.tenant_billing_reports') },
      ] },
    ],
  };
}

// Highlight the deepest matching nav entry for the current route.
const selectedKeys = computed(() => {
  const p = route.path;
  const keys = [];
  const walk = (list) => {
    for (const it of list) {
      if (it.children) walk(it.children);
      else if (it.key && it.key.startsWith('/')) {
        if (p === it.key || p.startsWith(it.key + '/')) keys.push(it.key);
      }
    }
  };
  walk(menuItems.value);
  // Prefer the longest match.
  keys.sort((a, b) => b.length - a.length);
  return keys.length ? [keys[0]] : [];
});

const openKeys = ref([]);
watch(
  () => route.path,
  (p) => {
    if (p.startsWith('/nacos') && !openKeys.value.includes('nacos')) openKeys.value = [...openKeys.value, 'nacos'];
    if ((p.startsWith('/tenant-upgrades') || p.startsWith('/tenant-management') || p.startsWith('/tenant-console')) && !openKeys.value.includes('tenant')) {
      openKeys.value = [...openKeys.value, 'tenant'];
    }
  },
  { immediate: true }
);

function onMenuClick({ key }) {
  if (key === 'nacos-console-ext') {
    window.open(`${window.location.origin}/nacos-ui/`, '_blank', 'noopener,noreferrer');
    return;
  }
  if (typeof key === 'string' && key.startsWith('/')) {
    router.push(key);
  }
}

function go(path) {
  router.push(path);
}

async function toggleLanguage() {
  const next = currentLocale() === 'en' ? 'zh' : 'en';
  setLocale(next);
}

async function logout() {
  try {
    await API.get('/api/user/logout');
  } catch { /* ignore */ }
  showSuccess('注销成功!');
  userStore.logout();
  router.push('/login');
}
</script>
