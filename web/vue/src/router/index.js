import { createRouter, createWebHistory } from 'vue-router';
import {
  ROLE_TENANT_ADMIN,
  isAdmin,
  isTenantAdmin,
  isTenantConsoleDelegate,
  isNacosEnabled,
  hasTenantPermission,
} from '../helpers/utils';

// Lazy page imports keep the initial bundle small, matching the React
// theme's React.lazy/Suspense split.
const Home = () => import('../pages/Home/index.vue');
const About = () => import('../pages/About/index.vue');
const NotFound = () => import('../pages/NotFound/index.vue');
const LoginForm = () => import('../components/LoginForm.vue');
const RegisterForm = () => import('../components/RegisterForm.vue');
const PasswordResetForm = () => import('../components/PasswordResetForm.vue');
const PasswordResetConfirm = () => import('../components/PasswordResetConfirm.vue');
const GitHubOAuth = () => import('../components/GitHubOAuth.vue');
const LarkOAuth = () => import('../components/LarkOAuth.vue');

const Channel = () => import('../pages/Channel/index.vue');
const EditChannel = () => import('../pages/Channel/EditChannel.vue');
const ModelCatalog = () => import('../pages/ModelCatalog/index.vue');
const ModelSquare = () => import('../pages/ModelSquare/index.vue');
const Operation = () => import('../pages/Operation/index.vue');
const Routing = () => import('../pages/Routing/index.vue');
const Token = () => import('../pages/Token/index.vue');
const EditToken = () => import('../pages/Token/EditToken.vue');
const Redemption = () => import('../pages/Redemption/index.vue');
const EditRedemption = () => import('../pages/Redemption/EditRedemption.vue');
const User = () => import('../pages/User/index.vue');
const EditUser = () => import('../pages/User/EditUser.vue');
const AddUser = () => import('../pages/User/AddUser.vue');
const TenantUpgrades = () => import('../pages/TenantUpgrades/index.vue');
const TenantManagement = () => import('../pages/TenantManagement/index.vue');
const PlatformReports = () => import('../pages/PlatformReports/index.vue');
const TopUp = () => import('../pages/TopUp/index.vue');
const Log = () => import('../pages/Log/index.vue');
const Chat = () => import('../pages/Chat/index.vue');
const Dashboard = () => import('../pages/Dashboard/index.vue');
const Setting = () => import('../pages/Setting/index.vue');

const TenantConsoleHome = () => import('../pages/TenantConsole/index.vue');
const TenantReports = () => import('../pages/TenantConsole/TenantReports.vue');
const TenantUsers = () => import('../pages/TenantConsole/TenantUsers.vue');
const TenantEditUser = () => import('../pages/TenantConsole/TenantEditUser.vue');
const TenantChannels = () => import('../pages/TenantConsole/TenantChannels.vue');
const TenantEditChannel = () => import('../pages/TenantConsole/TenantEditChannel.vue');
const TenantUserTokens = () => import('../pages/TenantConsole/TenantUserTokens.vue');
const TenantEditToken = () => import('../pages/TenantConsole/TenantEditToken.vue');

const NacosNamespaces = () => import('../pages/Nacos/Namespaces.vue');
const NacosConsoleExternalOpen = () => import('../pages/Nacos/NacosConsoleExternalOpen.vue');
const NacosCsConfigs = () => import('../pages/Nacos/CsConfigs.vue');
const NacosSkillsRegistry = () => import('../pages/Nacos/SkillsRegistry.vue');
const NacosAgentSpecsRegistry = () => import('../pages/Nacos/AgentSpecsRegistry.vue');
const NacosMcpRegistry = () => import('../pages/Nacos/McpRegistry.vue');
const NacosA2aRegistry = () => import('../pages/Nacos/A2aRegistry.vue');
const NacosPromptsRegistry = () => import('../pages/Nacos/PromptsRegistry.vue');
const NacosPipelinesRegistry = () => import('../pages/Nacos/PipelinesRegistry.vue');
const NacosPermissions = () => import('../pages/Nacos/NacosPermissions.vue');

// guard: 'public' | 'private' | 'platform' | 'tenantConsole' | 'nacos'
const routes = [
  { path: '/', component: Home, meta: { guard: 'platform' } },
  { path: '/routing', component: Routing, meta: { guard: 'platform' } },
  { path: '/channel', component: Channel, meta: { guard: 'platform' } },
  { path: '/model-catalog', component: ModelCatalog, meta: { guard: 'platform' } },
  { path: '/operations', component: Operation, meta: { guard: 'platform' } },
  { path: '/channel/edit/:id', component: EditChannel, meta: { guard: 'platform' } },
  { path: '/channel/add', component: EditChannel, meta: { guard: 'platform' } },
  { path: '/model-square', component: ModelSquare, meta: { guard: 'private' } },
  { path: '/token', component: Token, meta: { guard: 'private' } },
  { path: '/token/edit/:id', component: EditToken, meta: { guard: 'private' } },
  { path: '/token/add', component: EditToken, meta: { guard: 'private' } },
  { path: '/redemption', component: Redemption, meta: { guard: 'platform' } },
  { path: '/redemption/edit/:id', component: EditRedemption, meta: { guard: 'platform' } },
  { path: '/redemption/add', component: EditRedemption, meta: { guard: 'platform' } },
  { path: '/user', component: User, meta: { guard: 'platform' } },
  { path: '/tenant-upgrades', component: TenantUpgrades, meta: { guard: 'platform' } },
  { path: '/tenant-management', component: TenantManagement, meta: { guard: 'platform' } },
  { path: '/platform-reports', component: PlatformReports, meta: { guard: 'platform' } },
  { path: '/user/edit/:id', component: EditUser, meta: { guard: 'platform' } },
  { path: '/user/edit', component: EditUser, meta: { guard: 'platform' } },
  { path: '/user/add', component: AddUser, meta: { guard: 'platform' } },
  { path: '/user/reset', component: PasswordResetConfirm, meta: { guard: 'public' } },
  { path: '/login', component: LoginForm, meta: { guard: 'public' } },
  { path: '/tenant-login', component: LoginForm, props: { tenantPortal: true }, meta: { guard: 'public' } },
  { path: '/register', component: RegisterForm, meta: { guard: 'public' } },
  { path: '/reset', component: PasswordResetForm, meta: { guard: 'public' } },
  { path: '/oauth/github', component: GitHubOAuth, meta: { guard: 'public' } },
  { path: '/oauth/lark', component: LarkOAuth, meta: { guard: 'public' } },
  { path: '/setting', component: Setting, meta: { guard: 'private' } },
  { path: '/topup', component: TopUp, meta: { guard: 'private' } },
  { path: '/log', component: Log, meta: { guard: 'private' } },
  { path: '/about', component: About, meta: { guard: 'public' } },
  { path: '/chat', component: Chat, meta: { guard: 'public' } },
  { path: '/dashboard', component: Dashboard, meta: { guard: 'private' } },
  { path: '/nacos/namespaces', component: NacosNamespaces, meta: { guard: 'nacos' } },
  { path: '/nacos/console', component: NacosConsoleExternalOpen, meta: { guard: 'nacos' } },
  { path: '/nacos/cs', component: NacosCsConfigs, meta: { guard: 'nacos' } },
  { path: '/nacos/skills', component: NacosSkillsRegistry, meta: { guard: 'nacos' } },
  { path: '/nacos/agentspecs', component: NacosAgentSpecsRegistry, meta: { guard: 'nacos' } },
  { path: '/nacos/mcp', component: NacosMcpRegistry, meta: { guard: 'nacos' } },
  { path: '/nacos/a2a', component: NacosA2aRegistry, meta: { guard: 'nacos' } },
  { path: '/nacos/prompts', component: NacosPromptsRegistry, meta: { guard: 'nacos' } },
  { path: '/nacos/pipelines', component: NacosPipelinesRegistry, meta: { guard: 'nacos' } },
  { path: '/nacos/permissions', component: NacosPermissions, meta: { guard: 'nacos' } },
  { path: '/tenant-console', component: TenantConsoleHome, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/reports', component: TenantReports, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/users', component: TenantUsers, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/users/add', component: TenantEditUser, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/users/edit/:userId', component: TenantEditUser, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/channels', component: TenantChannels, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/channels/add', component: TenantEditChannel, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/channels/edit/:id', component: TenantEditChannel, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/users/:ownerId/tokens', component: TenantUserTokens, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/users/:ownerId/tokens/add', component: TenantEditToken, meta: { guard: 'tenantConsole' } },
  { path: '/tenant-console/users/:ownerId/tokens/edit/:tid', component: TenantEditToken, meta: { guard: 'tenantConsole' } },
  { path: '/:pathMatch(.*)*', component: NotFound, meta: { guard: 'public' } },
];

const router = createRouter({
  history: createWebHistory('/'),
  routes,
  scrollBehavior() {
    return { top: 0 };
  },
});

function loggedIn() {
  return !!localStorage.getItem('user');
}

function storedUser() {
  try {
    const raw = localStorage.getItem('user');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

router.beforeEach((to) => {
  const guard = to.meta?.guard || 'public';
  if (guard === 'public') return true;

  if (!loggedIn()) {
    return { path: '/login', query: { redirect: to.fullPath } };
  }

  const u = storedUser();

  if (guard === 'platform') {
    // Tenant admins and tenant-console delegates are bounced to their console.
    if (u && Number(u.role) === ROLE_TENANT_ADMIN) return { path: '/tenant-console' };
    if (isTenantConsoleDelegate()) return { path: '/tenant-console' };
    return true;
  }

  if (guard === 'tenantConsole') {
    if (u && Number(u.role) === ROLE_TENANT_ADMIN) return true;
    if (isAdmin()) return true;
    if (isTenantConsoleDelegate()) return true;
    return { path: '/' };
  }

  if (guard === 'nacos') {
    if (!isNacosEnabled()) return { path: '/' };
    if (isAdmin() || isTenantAdmin()) return true;
    if (isTenantConsoleDelegate() && hasTenantPermission('manage_nacos')) return true;
    return { path: '/' };
  }

  // 'private'
  return true;
});

export default router;
