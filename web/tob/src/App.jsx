import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { Navigate, Route, Routes } from 'react-router-dom';
import PrivateRoute from '@/components/PrivateRoute';
import AppLayout from '@/components/layout/AppLayout';
import LoginPage from '@/pages/Login';
import OverviewPage from '@/pages/Overview';
import ModelsPage from '@/pages/Models';
import UsagePage from '@/pages/Usage';
import LogsPage from '@/pages/Logs';
import ApiKeysPage from '@/pages/ApiKeys';
import SettingsPage from '@/pages/Settings';
import PlaygroundPage from '@/pages/Playground';

const antTheme = {
  token: {
    colorPrimary: '#6366f1',
    borderRadius: 8,
    controlHeight: 36,
    fontSize: 13,
  },
};

export default function App() {
  return (
    <ConfigProvider locale={zhCN} theme={antTheme}>
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <PrivateRoute>
            <AppLayout />
          </PrivateRoute>
        }
      >
        <Route path="/overview" element={<OverviewPage />} />
        <Route path="/models" element={<ModelsPage />} />
        <Route path="/playground" element={<PlaygroundPage />} />
        <Route path="/usage" element={<UsagePage />} />
        <Route path="/logs" element={<LogsPage />} />
        <Route path="/api-keys" element={<ApiKeysPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
      <Route path="/" element={<Navigate to="/overview" replace />} />
      <Route path="*" element={<Navigate to="/overview" replace />} />
    </Routes>
    </ConfigProvider>
  );
}
