import { defineStore } from 'pinia';
import { ref } from 'vue';
import {
  fetchSystemStatusOnce,
  clearSystemStatusFetchCache,
} from '../helpers/fetchSystemStatusOnce';
import { showError, showNotice, getSystemName } from '../helpers/utils';

export const useStatusStore = defineStore('status', () => {
  const status = ref({});

  function loadFromStorage() {
    try {
      const raw = localStorage.getItem('status');
      status.value = raw ? JSON.parse(raw) : {};
    } catch {
      status.value = {};
    }
    return status.value;
  }

  async function loadStatus() {
    try {
      const body = await fetchSystemStatusOnce();
      const { success, message, data } = body || {};
      if (success && data) {
        localStorage.setItem('status', JSON.stringify(data));
        status.value = data;
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
        const ver = import.meta.env.VITE_APP_VERSION;
        if (data.version !== ver && data.version !== 'v0.0.0' && ver !== '') {
          showNotice(`新版本可用：${data.version}，请使用快捷键 Shift + F5 刷新页面`);
        }
      } else {
        showError(message || '无法正常连接至服务器！');
      }
    } catch (error) {
      showError(error.message || '无法正常连接至服务器！');
    }
  }

  function reload() {
    clearSystemStatusFetchCache();
    return loadStatus();
  }

  return { status, loadFromStorage, loadStatus, reload };
});
