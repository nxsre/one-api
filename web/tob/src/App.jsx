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

export default function App() {
  return (
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
        <Route path="/usage" element={<UsagePage />} />
        <Route path="/logs" element={<LogsPage />} />
        <Route path="/api-keys" element={<ApiKeysPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
      <Route path="/" element={<Navigate to="/overview" replace />} />
      <Route path="*" element={<Navigate to="/overview" replace />} />
    </Routes>
  );
}
