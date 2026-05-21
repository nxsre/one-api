import axios from 'axios';

const baseURL = import.meta.env.VITE_API_BASE || '';

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
  (error) => Promise.reject(error)
);

export function getApiErrorMessage(error) {
  const data = error?.response?.data;
  if (typeof data?.message === 'string' && data.message.trim()) {
    return data.message.trim();
  }
  return error?.message || '请求失败';
}
