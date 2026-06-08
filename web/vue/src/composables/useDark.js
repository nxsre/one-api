import { ref } from 'vue';
import { applyNacosThemeClass, getNacosTheme } from '@/helpers/nacosTheme';

// Shared dark-mode flag driven by the `nacos_theme` localStorage key, which is
// toggled both by our top-bar bulb (NacosThemeToggle) and by the embedded Nacos
// native console (same origin, separate tab). We mirror it so Ant Design Vue's
// dark algorithm and the layout chrome react — AND we re-apply the `.dark` class
// on <html> on every sync, otherwise a cross-tab change leaves antd dark while
// the `.dark`-keyed CSS (e.g. sidebar recolor) is stale → mismatched colors.
const isDark = ref(getNacosTheme() === 'dark');

function sync() {
  const dark = getNacosTheme() === 'dark';
  isDark.value = dark;
  applyNacosThemeClass(dark ? 'dark' : 'light');
}

let wired = false;
export function useDark() {
  if (!wired && typeof window !== 'undefined') {
    wired = true;
    // Cross-tab change from the Nacos native console fires `storage`.
    window.addEventListener('storage', sync);
    // Same-tab change from our bulb fires this custom event.
    window.addEventListener('one-api:nacos-theme-changed', sync);
    // Returning to this tab after toggling elsewhere: reconcile in case the
    // storage event was missed (e.g. tab was backgrounded).
    window.addEventListener('focus', sync);
    document.addEventListener('visibilitychange', () => {
      if (!document.hidden) sync();
    });
  }
  return isDark;
}
