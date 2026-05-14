import { showError } from './utils';
import axios from 'axios';

export const API = axios.create({
  baseURL: process.env.REACT_APP_SERVER ? process.env.REACT_APP_SERVER : '',
});

function markForce2FARequired() {
  const raw = localStorage.getItem('user');
  if (raw) {
    try {
      const user = JSON.parse(raw);
      localStorage.setItem(
        'user',
        JSON.stringify({ ...user, require_force_2fa_setup: true })
      );
    } catch {
      /* ignore invalid local data */
    }
  }
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
  }
);
