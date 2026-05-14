import { showError } from './common';
import axios from 'axios';
import { store } from 'store/index';
import { LOGIN } from 'store/actions';
import config from 'config';

export const API = axios.create({
  baseURL: process.env.REACT_APP_SERVER ? process.env.REACT_APP_SERVER : '/'
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
  if (window.location.pathname !== `${config.basename}setting`) {
    window.location.href = `${config.basename}setting`;
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
    if (error.response?.status === 401) {
      localStorage.removeItem('user');
      store.dispatch({ type: LOGIN, payload: null });
      window.location.href = config.basename + 'login';
    }

    if (error.response?.data?.message) {
      error.message = error.response.data.message;
    }

    showError(error);
  }
);
