import { defineStore } from 'pinia';
import { ref } from 'vue';
import {
  clearTenantConsoleActingTenantId,
} from '../helpers/tenantConsoleImpersonation';
import { clearNacosEmbeddedConsoleLocalSession } from '../helpers/nacosEmbeddedConsole';

// Mirrors the React UserContext reducer: the source of truth stays in
// localStorage (key `user`) so API helpers and route guards agree.
export const useUserStore = defineStore('user', () => {
  const user = ref(undefined);

  function loadFromStorage() {
    try {
      const raw = localStorage.getItem('user');
      user.value = raw ? JSON.parse(raw) : undefined;
    } catch {
      user.value = undefined;
    }
    return user.value;
  }

  function login(payload) {
    user.value = payload;
    try {
      localStorage.setItem('user', JSON.stringify(payload));
    } catch {
      /* ignore */
    }
  }

  function logout() {
    user.value = undefined;
    localStorage.removeItem('user');
    clearTenantConsoleActingTenantId();
    clearNacosEmbeddedConsoleLocalSession();
  }

  return { user, loadFromStorage, login, logout };
});
