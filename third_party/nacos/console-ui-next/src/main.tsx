import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './monaco-setup';
import './globals.css';
import './locales';
import App from './App';
import { isEmbeddedUnderOneApi, redirectToOneApiLoginPage } from '@/lib/one-api-embed-auth';

/** 同源嵌入 one-api 时，用已登录 Web 会话换取 Nacos 控制台 localStorage.token，避免二次登录 */
async function syncOneApiNacosEmbedSession(): Promise<void> {
  const path = window.location.pathname;
  if (path !== '/nacos-ui' && !path.startsWith('/nacos-ui/')) {
    return;
  }
  try {
    const res = await fetch(`${window.location.origin}/api/user/nacos-console-token`, {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    });
    if (!res.ok) {
      if (isEmbeddedUnderOneApi()) {
        try {
          localStorage.removeItem('token');
        } catch {
          /* ignore */
        }
      }
      return;
    }
    let j: {
      success?: boolean;
      data?: { accessToken?: string; username?: string; globalAdmin?: boolean };
    };
    try {
      j = (await res.json()) as typeof j;
    } catch {
      if (isEmbeddedUnderOneApi()) {
        try {
          localStorage.removeItem('token');
        } catch {
          /* ignore */
        }
      }
      return;
    }
    if (j.success && j.data?.accessToken) {
      localStorage.setItem(
        'token',
        JSON.stringify({
          accessToken: j.data.accessToken,
          username: j.data.username ?? '',
          globalAdmin: !!j.data.globalAdmin,
        })
      );
    } else if (isEmbeddedUnderOneApi()) {
      try {
        localStorage.removeItem('token');
      } catch {
        /* ignore */
      }
    }
  } catch {
    /* 未登录 one-api 时由路由跳转站点 /login */
    if (isEmbeddedUnderOneApi()) {
      try {
        localStorage.removeItem('token');
      } catch {
        /* ignore */
      }
    }
  }
}

// Handle OIDC Cookie-based authentication (cluster-friendly, no server-side storage)
(function oidcCookieSync() {
  const hash = window.location.hash;

  // Handle error response from OIDC callback
  if (hash && hash.includes('error=')) {
    try {
      const queryString = hash.split('?')[1];
      if (queryString) {
        const params = new URLSearchParams(queryString);
        const error = params.get('error');
        if (error) {
          console.error('[OIDC] Authentication failed:', decodeURIComponent(error));
          sessionStorage.setItem('oidcError', decodeURIComponent(error));
          if (isEmbeddedUnderOneApi()) {
            redirectToOneApiLoginPage();
          } else {
            const newUrl = window.location.href.split('#')[0] + '#/login';
            window.history.replaceState(null, '', newUrl);
          }
        }
      }
    } catch (e) {
      console.error('[OIDC] Failed to parse error from URL', e);
    }
    return;
  }

  function getCookie(name: string): string | null {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()!.split(';').shift() || null;
    return null;
  }

  function deleteCookie(name: string) {
    document.cookie = name + '=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
  }

  const accessToken = getCookie('accessToken');
  const username = getCookie('username');

  if (accessToken && username) {
    localStorage.setItem('token', JSON.stringify({
      accessToken,
      username: decodeURIComponent(username),
      globalAdmin: false,
      oidc: true,
    }));
    deleteCookie('accessToken');
    deleteCookie('username');
  }
})();

void syncOneApiNacosEmbedSession().finally(() => {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>
  );
});
