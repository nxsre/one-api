import { API } from './api';

/** 并发合并：StrictMode 双次 effect、或多组件同时触发时只发一条 HTTP。成功后短路缓存（直至整页刷新或 clearSystemStatusFetchCache）。 */
let inflight = null;
let cachedBody = null;

/**
 * @returns {Promise<{ success?: boolean, message?: string, data?: object }>}
 */
export async function fetchSystemStatusOnce() {
  if (cachedBody) {
    return cachedBody;
  }
  if (!inflight) {
    inflight = API.get('/api/status')
      .then((res) => {
        const body = res?.data || {};
        const { success, data } = body;
        if (success && data) {
          cachedBody = body;
        }
        return body;
      })
      .finally(() => {
        inflight = null;
      });
  }
  return inflight;
}

/** 管理端等处若需在会话内强制重新拉取站点状态时调用（当前未使用）。 */
export function clearSystemStatusFetchCache() {
  cachedBody = null;
}
