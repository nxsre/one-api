import axios from 'axios';
import { showError } from './utils';
import { getTenantConsoleActingTenantId } from './tenantConsoleImpersonation';
import { useUserStore } from '../stores/user';

export const API = axios.create({
  baseURL: import.meta.env.VITE_SERVER ? import.meta.env.VITE_SERVER : '',
});

const ROLE_TENANT_ADMIN = 20;

API.interceptors.request.use((config) => {
  const url = config.url || '';
  if (typeof url === 'string' && url.includes('/api/tenant_console')) {
    try {
      const raw = localStorage.getItem('user');
      if (!raw) return config;
      const u = JSON.parse(raw);
      if (Number(u.role) === ROLE_TENANT_ADMIN) return config;
      const tid = getTenantConsoleActingTenantId();
      if (tid) {
        const headers = config.headers ?? {};
        headers['X-Tenant-Console-Tenant-Id'] = tid;
        config.headers = headers;
      }
    } catch {
      /* ignore */
    }
  }
  return config;
});

function markForce2FARequired() {
  useUserStore().markForce2FASetupRequired();
  if (window.location.pathname !== '/setting') {
    window.location.href = '/setting';
  }
}

API.interceptors.response.use(
  (response) => {
    if (response.data?.code === 'force_2fa_required') {
      markForce2FARequired();
    }
    return response;
  },
  (error) => {
    showError(error);
    return Promise.reject(error);
  }
);
