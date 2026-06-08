import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import Components from 'unplugin-vue-components/vite';
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers';

// Backend (gin) serves the embedded SPA from web/build/<THEME>. To mirror the
// CRA layout the Dockerfile validates (static/js/*.js), emit hashed assets under
// static/js and static/css instead of Vite's default assets/.
export default defineConfig({
  base: '/',
  plugins: [
    vue(),
    Components({
      dts: false,
      resolvers: [
        AntDesignVueResolver({ importStyle: false, resolveIcons: true }),
      ],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  define: {
    // Mirror CRA's REACT_APP_VERSION so shared expectations keep working.
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(
      process.env.VITE_APP_VERSION || process.env.REACT_APP_VERSION || ''
    ),
  },
  build: {
    outDir: 'dist',
    assetsDir: 'static',
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      output: {
        entryFileNames: 'static/js/[name].[hash].js',
        // Strip leading underscores: Go's embed (even with all:) and some static
        // hosts treat `_`-prefixed files specially; keep chunk names plain.
        chunkFileNames: (chunkInfo) => {
          const name = (chunkInfo.name || 'chunk').replace(/^[_.]+/, '');
          return `static/js/${name || 'chunk'}.[hash].js`;
        },
        assetFileNames: (assetInfo) => {
          const name = assetInfo.name || '';
          if (name.endsWith('.css')) return 'static/css/[name].[hash][extname]';
          return 'static/media/[name].[hash][extname]';
        },
      },
    },
  },
  server: {
    port: 3001,
    proxy: {
      '/api': 'http://localhost:3000',
      '/v1': 'http://localhost:3000',
      '/dashboard': 'http://localhost:3000',
      '/pg': 'http://localhost:3000',
      '/nacos-ui': 'http://localhost:3000',
    },
  },
});
