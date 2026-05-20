const STORAGE_KEY = 'tenant_console_acting_tenant_id';

export function getTenantConsoleActingTenantId() {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v && /^\d+$/.test(String(v).trim())) return String(v).trim();
  } catch {
    /* ignore */
  }
  return '';
}

export function setTenantConsoleActingTenantId(id) {
  const s = id != null ? String(id).trim() : '';
  try {
    if (!s || !/^\d+$/.test(s)) {
      localStorage.removeItem(STORAGE_KEY);
      return;
    }
    localStorage.setItem(STORAGE_KEY, s);
  } catch {
    /* ignore */
  }
}

export function clearTenantConsoleActingTenantId() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    /* ignore */
  }
}
