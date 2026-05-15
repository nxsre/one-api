/**
 * Webpack externals 在 Vite 下用全局 jQuery（由 index.html 中 /js/jquery.js 注入）。
 */
const $ = typeof window !== 'undefined' ? window.jQuery : undefined;
export default $;
