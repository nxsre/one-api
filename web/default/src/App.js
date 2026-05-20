import React, { lazy, Suspense, useContext, useEffect, useState } from 'react';
import { Route, Routes, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Message, Modal } from 'semantic-ui-react';
import Loading from './components/Loading';
import User from './pages/User';
import { PrivateRoute } from './components/PrivateRoute';
import { TenantConsoleRoute, PrivatePlatformRoute } from './components/TenantScopeRoute';
import NacosPrivateRoute from './components/NacosPrivateRoute';
import RegisterForm from './components/RegisterForm';
import LoginForm from './components/LoginForm';
import NotFound from './pages/NotFound';
import Setting from './pages/Setting';
import EditUser from './pages/User/EditUser';
import AddUser from './pages/User/AddUser';
import {
  API,
  fetchSystemStatusOnce,
  clearSystemStatusFetchCache,
  getLogo,
  getSystemName,
  showError,
  showNotice,
  clearNacosEmbeddedConsoleLocalSession,
  clearTenantConsoleActingTenantId,
} from './helpers';
import PasswordResetForm from './components/PasswordResetForm';
import GitHubOAuth from './components/GitHubOAuth';
import PasswordResetConfirm from './components/PasswordResetConfirm';
import { UserContext } from './context/User';
import { StatusContext } from './context/Status';
import Channel from './pages/Channel';
import ModelCatalogPage from './pages/ModelCatalog';
import OperationPage from './pages/Operation';
import RoutingOperations from './pages/Routing';
import Token from './pages/Token';
import EditToken from './pages/Token/EditToken';
import EditChannel from './pages/Channel/EditChannel';
import Redemption from './pages/Redemption';
import EditRedemption from './pages/Redemption/EditRedemption';
import TenantUpgrades from './pages/TenantUpgrades';
import TenantManagement from './pages/TenantManagement';
import PlatformReports from './pages/PlatformReports';
import TopUp from './pages/TopUp';
import Log from './pages/Log';
import Chat from './pages/Chat';
import LarkOAuth from './components/LarkOAuth';
import Dashboard from './pages/Dashboard';
import TwoFASetting from './components/TwoFASetting';
import NacosSkillsRegistry from './pages/Nacos/SkillsRegistry';
import NacosAgentSpecsRegistry from './pages/Nacos/AgentSpecsRegistry';
import NacosMcpRegistry from './pages/Nacos/McpRegistry';
import NacosA2aRegistry from './pages/Nacos/A2aRegistry';
import NacosPromptsRegistry from './pages/Nacos/PromptsRegistry';
import NacosPipelinesRegistry from './pages/Nacos/PipelinesRegistry';
import NacosPermissions from './pages/Nacos/NacosPermissions';
import NacosNamespaces from './pages/Nacos/Namespaces';
import NacosCsConfigs from './pages/Nacos/CsConfigs';
import NacosConsoleExternalOpen from './pages/Nacos/NacosConsoleExternalOpen';
import TenantConsoleHome from './pages/TenantConsole';
import TenantReports from './pages/TenantConsole/TenantReports';
import TenantUsers from './pages/TenantConsole/TenantUsers';
import TenantEditUser from './pages/TenantConsole/TenantEditUser';
import TenantChannels from './pages/TenantConsole/TenantChannels';
import TenantEditChannel from './pages/TenantConsole/TenantEditChannel';
import TenantUserTokens from './pages/TenantConsole/TenantUserTokens';
import TenantEditToken from './pages/TenantConsole/TenantEditToken';

const Home = lazy(() => import('./pages/Home'));
const About = lazy(() => import('./pages/About'));

function TenantLoginPage() {
  return <LoginForm tenantPortal />;
}

function isPublicAuthPath() {
  const path = window.location.pathname;
  return (
    path === '/login' ||
    path === '/tenant-login' ||
    path === '/register' ||
    path === '/reset' ||
    path === '/user/reset' ||
    path.startsWith('/oauth/')
  );
}

function App() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [force2FASetupOpen, setForce2FASetupOpen] = useState(false);
  const [force2FACancelBusy, setForce2FACancelBusy] = useState(false);

  const updateLocalForce2FAUser = (required) => {
    const user = localStorage.getItem('user');
    if (!user) {
      setForce2FASetupOpen(false);
      return;
    }
    const data = JSON.parse(user);
    const next = {
      ...data,
      require_force_2fa_setup: required,
    };
    localStorage.setItem('user', JSON.stringify(next));
    userDispatch({ type: 'login', payload: next });
    setForce2FASetupOpen(!!required);
  };

  const cancelForce2FAAndReturnToLogin = async () => {
    setForce2FACancelBusy(true);
    try {
      try {
        await API.get('/api/user/logout');
      } catch {
        /* 仍清理本地会话并返回登录页 */
      }
      localStorage.removeItem('user');
      clearTenantConsoleActingTenantId();
      clearNacosEmbeddedConsoleLocalSession();
      userDispatch({ type: 'logout' });
      setForce2FASetupOpen(false);
      navigate('/login');
    } finally {
      setForce2FACancelBusy(false);
    }
  };

  const loadUser = () => {
    let user = localStorage.getItem('user');
    if (user) {
      let data = JSON.parse(user);
      userDispatch({ type: 'login', payload: data });
      setForce2FASetupOpen(
        !!data.require_force_2fa_setup && !isPublicAuthPath()
      );
    }
  };
  const loadStatus = async () => {
    try {
      const body = await fetchSystemStatusOnce();
      const { success, message, data } = body || {};
      if (success && data) {
        // Check data exists
        localStorage.setItem('status', JSON.stringify(data));
        statusDispatch({ type: 'set', payload: data });
        localStorage.setItem('system_name', data.system_name);
        localStorage.setItem('logo', data.logo);
        localStorage.setItem('footer_html', data.footer_html);
        const qpu = data.quota_per_unit;
        const qpuStr =
          qpu !== undefined &&
          qpu !== null &&
          Number.isFinite(Number(qpu)) &&
          Number(qpu) > 0
            ? String(qpu)
            : String(500 * 1000);
        localStorage.setItem('quota_per_unit', qpuStr);
        localStorage.setItem(
          'display_in_currency',
          data.display_in_currency === true || data.display_in_currency === 'true'
            ? 'true'
            : 'false'
        );
        if (data.chat_link) {
          localStorage.setItem('chat_link', data.chat_link);
        } else {
          localStorage.removeItem('chat_link');
        }
        if (
          data.version !== process.env.REACT_APP_VERSION &&
          data.version !== 'v0.0.0' &&
          process.env.REACT_APP_VERSION !== ''
        ) {
          showNotice(
            `新版本可用：${data.version}，请使用快捷键 Shift + F5 刷新页面`
          );
        }
      } else {
        showError(message || '无法正常连接至服务器！');
      }
    } catch (error) {
      showError(error.message || '无法正常连接至服务器！');
    }
  };

  useEffect(() => {
    loadUser();
    loadStatus().then();
  }, []);

  // 登录/注销后会话变化：必须丢弃 fetchSystemStatusOnce 的内存缓存，否则仍返回未登录时的 public
  // payload（无 nacos_enabled），侧栏 Nacos 菜单会一直隐藏，直到整页刷新。
  useEffect(() => {
    clearSystemStatusFetchCache();
    loadStatus().then();
  }, [userState.user?.id]);

  useEffect(() => {
    const name =
      statusState.status?.system_name || getSystemName();
    if (name) {
      document.title = name;
    }
    const logo = statusState.status?.logo || getLogo();
    if (logo) {
      const linkElement = document.querySelector("link[rel~='icon']");
      if (linkElement) {
        linkElement.href = logo;
      }
    }
  }, [statusState.status]);

  useEffect(() => {
    setForce2FASetupOpen(
      !!userState.user?.require_force_2fa_setup && !isPublicAuthPath()
    );
  }, [userState.user?.require_force_2fa_setup]);

  return (
    <>
      <Routes>
      <Route
        path='/'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <Home />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/routing'
        element={
          <PrivatePlatformRoute>
            <RoutingOperations />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/channel'
        element={
          <PrivatePlatformRoute>
            <Channel />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/model-catalog'
        element={
          <PrivatePlatformRoute>
            <ModelCatalogPage />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/operations'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <OperationPage />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/channel/edit/:id'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditChannel />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/channel/add'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditChannel />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/token'
        element={
          <PrivateRoute>
            <Token />
          </PrivateRoute>
        }
      />
      <Route
        path='/token/edit/:id'
        element={
          <PrivateRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditToken />
            </Suspense>
          </PrivateRoute>
        }
      />
      <Route
        path='/token/add'
        element={
          <PrivateRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditToken />
            </Suspense>
          </PrivateRoute>
        }
      />
      <Route
        path='/redemption'
        element={
          <PrivatePlatformRoute>
            <Redemption />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/redemption/edit/:id'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditRedemption />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/redemption/add'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditRedemption />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/user'
        element={
          <PrivatePlatformRoute>
            <User />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/tenant-upgrades'
        element={
          <PrivatePlatformRoute>
            <TenantUpgrades />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/tenant-management'
        element={
          <PrivatePlatformRoute>
            <TenantManagement />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/platform-reports'
        element={
          <PrivatePlatformRoute>
            <PlatformReports />
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/user/edit/:id'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditUser />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/user/edit'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <EditUser />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/user/add'
        element={
          <PrivatePlatformRoute>
            <Suspense fallback={<Loading></Loading>}>
              <AddUser />
            </Suspense>
          </PrivatePlatformRoute>
        }
      />
      <Route
        path='/user/reset'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <PasswordResetConfirm />
          </Suspense>
        }
      />
      <Route
        path='/login'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <LoginForm />
          </Suspense>
        }
      />
      <Route path='/tenant-login' element={<TenantLoginPage />} />
      <Route
        path='/register'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <RegisterForm />
          </Suspense>
        }
      />
      <Route
        path='/reset'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <PasswordResetForm />
          </Suspense>
        }
      />
      <Route
        path='/oauth/github'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <GitHubOAuth />
          </Suspense>
        }
      />
      <Route
        path='/oauth/lark'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <LarkOAuth />
          </Suspense>
        }
      />
      <Route
        path='/setting'
        element={
          <PrivateRoute>
            <Suspense fallback={<Loading></Loading>}>
              <Setting />
            </Suspense>
          </PrivateRoute>
        }
      />
      <Route
        path='/topup'
        element={
          <PrivateRoute>
            <Suspense fallback={<Loading></Loading>}>
              <TopUp />
            </Suspense>
          </PrivateRoute>
        }
      />
      <Route
        path='/log'
        element={
          <PrivateRoute>
            <Log />
          </PrivateRoute>
        }
      />
      <Route
        path='/about'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <About />
          </Suspense>
        }
      />
      <Route
        path='/chat'
        element={
          <Suspense fallback={<Loading></Loading>}>
            <Chat />
          </Suspense>
        }
      />
      <Route
        path='/dashboard'
        element={
          <PrivateRoute>
            <Dashboard />
          </PrivateRoute>
        }
      />
      <Route
        path='/nacos/namespaces'
        element={
          <NacosPrivateRoute>
            <NacosNamespaces />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/console'
        element={
          <NacosPrivateRoute>
            <NacosConsoleExternalOpen />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/cs'
        element={
          <NacosPrivateRoute>
            <NacosCsConfigs />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/skills'
        element={
          <NacosPrivateRoute>
            <NacosSkillsRegistry />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/agentspecs'
        element={
          <NacosPrivateRoute>
            <NacosAgentSpecsRegistry />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/mcp'
        element={
          <NacosPrivateRoute>
            <NacosMcpRegistry />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/a2a'
        element={
          <NacosPrivateRoute>
            <NacosA2aRegistry />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/prompts'
        element={
          <NacosPrivateRoute>
            <NacosPromptsRegistry />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/pipelines'
        element={
          <NacosPrivateRoute>
            <NacosPipelinesRegistry />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/nacos/permissions'
        element={
          <NacosPrivateRoute>
            <NacosPermissions />
          </NacosPrivateRoute>
        }
      />
      <Route
        path='/tenant-console'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantConsoleHome />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/reports'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantReports />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/users'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantUsers />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/users/add'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantEditUser />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/users/edit/:userId'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantEditUser />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/channels'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantChannels />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/channels/add'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantEditChannel />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/channels/edit/:id'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantEditChannel />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/users/:ownerId/tokens'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantUserTokens />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/users/:ownerId/tokens/add'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantEditToken />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route
        path='/tenant-console/users/:ownerId/tokens/edit/:tid'
        element={
          <PrivateRoute>
            <TenantConsoleRoute>
              <TenantEditToken />
            </TenantConsoleRoute>
          </PrivateRoute>
        }
      />
      <Route path='*' element={<NotFound />} />
      </Routes>
      <Modal
        open={force2FASetupOpen}
        closeOnDimmerClick={false}
        closeOnEscape={false}
        size='small'
      >
        <Modal.Header>{t('auth.force_2fa.modal_title')}</Modal.Header>
        <Modal.Content>
          <Message warning>{t('auth.force_2fa.modal_hint')}</Message>
          <TwoFASetting
            forceMode
            onEnabled={() => updateLocalForce2FAUser(false)}
            onCancelLogin={cancelForce2FAAndReturnToLogin}
            cancelLoginBusy={force2FACancelBusy}
            cancelLoginLabel={t('auth.force_2fa.cancel_login')}
          />
        </Modal.Content>
        <Modal.Actions>
          <Button
            basic
            loading={force2FACancelBusy}
            disabled={force2FACancelBusy}
            onClick={cancelForce2FAAndReturnToLogin}
          >
            {t('auth.force_2fa.cancel_login')}
          </Button>
        </Modal.Actions>
      </Modal>
    </>
  );
}

export default App;
