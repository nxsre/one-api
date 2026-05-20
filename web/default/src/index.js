import './monaco-setup';
import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import SidebarLayout from './components/SidebarLayout';
import 'semantic-ui-css/semantic.min.css';
import './index.css';
import './nacos-theme.css';
import './pages/Dashboard/Dashboard.css';
import { UserProvider } from './context/User';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import { StatusProvider } from './context/Status';
import './i18n';
import { applyNacosThemeClass, getNacosTheme } from './helpers/nacosTheme';

applyNacosThemeClass(getNacosTheme());

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <StatusProvider>
      <UserProvider>
        <BrowserRouter>
          <div className='app-root-fill'>
            <SidebarLayout>
              <div className='main-content app-page-root'>
                <App />
              </div>
            </SidebarLayout>
            <ToastContainer position='top-right' autoClose={3000} />
          </div>
        </BrowserRouter>
      </UserProvider>
    </StatusProvider>
  </React.StrictMode>
);
