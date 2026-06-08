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

// vue-i18n compiles every message at runtime and treats `{...}` as an
// interpolation placeholder. Many translations carry literal JSON examples in
// plain text (e.g. the channel-edit hints: {"503":502}, {}, {"temperature":0.7}).
// The default compiler tries to parse those as placeholders, throws a
// SyntaxError (INVALID_TOKEN_IN_PLACEHOLDER), and aborts rendering of the
// surrounding node — which is why those fields showed only their label with no
// input. This app uses ONLY named interpolation (no plural `|`, linked `@:`, or
// list `{0}` syntax), so we swap in a brace-tolerant compiler that never throws:
// `{name}` is substituted only when it is a clean identifier AND a matching
// param was supplied; anything else (JSON braces, doc placeholders) is left
// verbatim.
function braceTolerantMessageCompiler(message) {
  if (typeof message === 'function') return message;
  const text = String(message);
  const fn = (ctx) => {
    const named = ctx && typeof ctx.named === 'function' ? ctx.named : null;
    if (!named) return text;
    return text.replace(/\{\s*(\w+)\s*\}/g, (whole, name) => {
      const value = named(name);
      return value === undefined || value === null ? whole : String(value);
    });
  };
  fn.source = text;
  return fn;
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
  messageCompiler: braceTolerantMessageCompiler,
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
