import { useEffect } from 'react';
import { Toaster } from 'sonner';
import { AppRouter } from './router';
import { useAppStore } from './stores/app-store';
import { useAuthStore } from './stores/auth-store';
import { isEmbeddedUnderOneApi, redirectToOneApiLoginPage } from '@/lib/one-api-embed-auth';

function App() {
  const { initFromStorage } = useAppStore();

  useEffect(() => {
    initFromStorage();
  }, [initFromStorage]);

  useEffect(() => {
    if (!isEmbeddedUnderOneApi()) return;
    const onStorage = (e: StorageEvent) => {
      if (e.key !== 'token') return;
      if (e.newValue != null) return;
      useAuthStore.getState().loadFromStorage();
      if (!useAuthStore.getState().isAuthenticated) {
        redirectToOneApiLoginPage();
      }
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  return (
    <>
      <AppRouter />
      <Toaster position="top-right" richColors closeButton />
    </>
  );
}

export default App;
