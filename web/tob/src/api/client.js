import axios from 'axios';
import { message } from 'antd';

const baseURL = import.meta.env.VITE_API_BASE || '';

let handlingUnauthorized = false;

function handleUnauthorized() {
  if (handlingUnauthorized) return;
  if (window.location.pathname === '/login') return;

  handlingUnauthorized = true;
  message.warning('登录已过期，请重新登录');
  localStorage.removeItem('user');
  window.location.replace('/login');
}

export const API = axios.create({
  baseURL,
  withCredentials: true,
});

API.interceptors.response.use(
  (response) => {
    if (response.data?.code === 'force_2fa_required') {
      if (window.location.pathname !== '/settings') {
        window.location.href = '/settings';
      }
    }
    return response;
  },
  (error) => {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    }
    return Promise.reject(error);
  }
);

export function getApiErrorMessage(error) {
  if (error?.response?.status === 429) {
    return '请求过于频繁，请稍后再试';
  }
  const data = error?.response?.data;
  if (typeof data?.message === 'string' && data.message.trim()) {
    return data.message.trim();
  }
  return error?.message || '请求失败';
}
