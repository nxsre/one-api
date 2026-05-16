const STORAGE_KEY = 'nacos_registry_selected_namespace';

export function getStoredNacosNamespace() {
  const v = localStorage.getItem(STORAGE_KEY);
  if (v != null && String(v).trim() !== '') {
    return String(v).trim();
  }
  return 'public';
}

export function setStoredNacosNamespace(ns) {
  if (ns != null && String(ns).trim() !== '') {
    localStorage.setItem(STORAGE_KEY, String(ns).trim());
  }
}
