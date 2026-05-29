import { createI18n } from 'vue-i18n';
import zh from '../locales/zh/translation.json';
import en from '../locales/en/translation.json';

// i18next stored the active language under i18nextLng; reuse it so a user's
// prior choice carries over when both themes are installed side by side.
function detectLocale() {
  try {
    const stored = localStorage.getItem('i18nextLng') || localStorage.getItem('lang');
    if (stored && stored.toLowerCase().startsWith('en')) return 'en';
    if (stored && stored.toLowerCase().startsWith('zh')) return 'zh';
  } catch {
    /* ignore */
  }
  const nav = (navigator.language || 'zh').toLowerCase();
  return nav.startsWith('en') ? 'en' : 'zh';
}

const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale: 'zh',
  messages: { zh, en },
  // Strings carried over from i18next occasionally contain unescaped braces in
  // plain text; do not warn loudly about them in production.
  missingWarn: false,
  fallbackWarn: false,
  warnHtmlMessage: false,
});

export function setLocale(lng) {
  const l = lng === 'en' ? 'en' : 'zh';
  i18n.global.locale.value = l;
  try {
    localStorage.setItem('i18nextLng', l);
  } catch {
    /* ignore */
  }
}

export function currentLocale() {
  return i18n.global.locale.value;
}

export default i18n;
