import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiTarget = env.VITE_API_PROXY || 'http://127.0.0.1:3000';
  const isHttps = apiTarget.startsWith('https://');

  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      port: 5173,
      proxy: {
        // 必须用 /api/，不能用 /api，否则会误代理前端路由 /api-keys
        '/api/': {
          target: apiTarget,
          changeOrigin: true,
          secure: false,
          ...(isHttps && { protocolRewrite: 'https' }),
        },
      },
    },
    build: {
      outDir: 'dist',
    },
  };
});
