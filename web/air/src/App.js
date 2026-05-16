import React, { lazy, Suspense, useContext, useEffect, useState } from 'react';
import { Route, Routes, useNavigate } from 'react-router-dom';
import { Button, Message, Modal } from 'semantic-ui-react';
import Loading from './components/Loading';
import User from './pages/User';
import { PrivateRoute } from './components/PrivateRoute';
import RegisterForm from './components/RegisterForm';
import LoginForm from './components/LoginForm';
import NotFound from './pages/NotFound';
import Setting from './pages/Setting';
import EditUser from './pages/User/EditUser';
import { API, getLogo, getSystemName } from './helpers';
import PasswordResetForm from './components/PasswordResetForm';
import GitHubOAuth from './components/GitHubOAuth';
import PasswordResetConfirm from './components/PasswordResetConfirm';
import { UserContext } from './context/User';
import Channel from './pages/Channel';
import Token from './pages/Token';
import EditChannel from './pages/Channel/EditChannel';
import Redemption from './pages/Redemption';
import TopUp from './pages/TopUp';
import Log from './pages/Log';
import Chat from './pages/Chat';
import Routing from './pages/Routing';
import { Layout } from '@douyinfe/semi-ui';
import Midjourney from './pages/Midjourney';
import Detail from './pages/Detail';
import TwoFASetting from './components/TwoFASetting';

const Home = lazy(() => import('./pages/Home'));
const About = lazy(() => import('./pages/About'));

function isPublicAuthPath() {
  const path = window.location.pathname;
  return (
    path === '/login' ||
    path === '/register' ||
    path === '/reset' ||
    path === '/user/reset' ||
    path.startsWith('/oauth/')
  );
}

function App() {
  const navigate = useNavigate();
  const [userState, userDispatch] = useContext(UserContext);
  // const [statusState, statusDispatch] = useContext(StatusContext);
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
        /* ignore */
      }
      localStorage.removeItem('user');
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

  useEffect(() => {
    loadUser();
    let systemName = getSystemName();
    if (systemName) {
      document.title = systemName;
    }
    let logo = getLogo();
    if (logo) {
      let linkElement = document.querySelector('link[rel~=\'icon\']');
      if (linkElement) {
        linkElement.href = logo;
      }
    }
  }, []);

  useEffect(() => {
    setForce2FASetupOpen(
      !!userState.user?.require_force_2fa_setup && !isPublicAuthPath()
    );
  }, [userState.user?.require_force_2fa_setup]);

  return (
    <Layout>
      <Layout.Content>
        <Routes>
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Suspense fallback={<Loading></Loading>}>
                  <Home />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path="/channel"
            element={
              <PrivateRoute>
                <Channel />
              </PrivateRoute>
            }
          />
          <Route
            path="/channel/edit/:id"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <EditChannel />
              </Suspense>
            }
          />
          <Route
            path="/channel/add"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <EditChannel />
              </Suspense>
            }
          />
          <Route
            path="/routing"
            element={
              <PrivateRoute>
                <Routing />
              </PrivateRoute>
            }
          />
          <Route
            path="/token"
            element={
              <PrivateRoute>
                <Token />
              </PrivateRoute>
            }
          />
          <Route
            path="/redemption"
            element={
              <PrivateRoute>
                <Redemption />
              </PrivateRoute>
            }
          />
          <Route
            path="/user"
            element={
              <PrivateRoute>
                <User />
              </PrivateRoute>
            }
          />
          <Route
            path="/user/edit/:id"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <EditUser />
              </Suspense>
            }
          />
          <Route
            path="/user/edit"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <EditUser />
              </Suspense>
            }
          />
          <Route
            path="/user/reset"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <PasswordResetConfirm />
              </Suspense>
            }
          />
          <Route
            path="/login"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <LoginForm />
              </Suspense>
            }
          />
          <Route
            path="/register"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <RegisterForm />
              </Suspense>
            }
          />
          <Route
            path="/reset"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <PasswordResetForm />
              </Suspense>
            }
          />
          <Route
            path="/oauth/github"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <GitHubOAuth />
              </Suspense>
            }
          />
          <Route
            path="/setting"
            element={
              <PrivateRoute>
                <Suspense fallback={<Loading></Loading>}>
                  <Setting />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path="/topup"
            element={
              <PrivateRoute>
                <Suspense fallback={<Loading></Loading>}>
                  <TopUp />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path="/log"
            element={
              <PrivateRoute>
                <Log />
              </PrivateRoute>
            }
          />
          <Route
            path="/detail"
            element={
              <PrivateRoute>
                <Detail />
              </PrivateRoute>
            }
          />
          <Route
            path="/midjourney"
            element={
              <PrivateRoute>
                <Midjourney />
              </PrivateRoute>
            }
          />
          <Route
            path="/about"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <About />
              </Suspense>
            }
          />
          <Route
            path="/chat"
            element={
              <Suspense fallback={<Loading></Loading>}>
                <Chat />
              </Suspense>
            }
          />
          <Route path="*" element={
            <NotFound />
          } />
        </Routes>
        <Modal
          open={force2FASetupOpen}
          closeOnDimmerClick={false}
          closeOnEscape={false}
          size="small"
        >
          <Modal.Header>需要配置两步验证</Modal.Header>
          <Modal.Content>
            <Message warning>
              管理员已开启全员 MFA。请先完成两步验证配置，完成前无法进行其他操作。
            </Message>
            <TwoFASetting
              forceMode
              onEnabled={() => updateLocalForce2FAUser(false)}
              onCancelLogin={cancelForce2FAAndReturnToLogin}
              cancelLoginBusy={force2FACancelBusy}
              cancelLoginLabel='取消并返回登录'
            />
          </Modal.Content>
          <Modal.Actions>
            <Button
              basic
              loading={force2FACancelBusy}
              disabled={force2FACancelBusy}
              onClick={cancelForce2FAAndReturnToLogin}
            >
              取消并返回登录
            </Button>
          </Modal.Actions>
        </Modal>
      </Layout.Content>
    </Layout>
  );
}

export default App;
