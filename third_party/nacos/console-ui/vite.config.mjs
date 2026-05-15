/**
 * 旧版 Nacos 控制台 — 与 console-ui-next 对齐使用 Vite 构建。
 * 嵌入 one-api 时 build 使用 base: './'，产物结构与原 webpack 一致（js/main.js、css/main.css）。
 */
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const nodeEmpty = path.resolve(rootDir, 'src/shims/node-empty.js');
const nodeEmptyCjs = path.resolve(rootDir, 'src/shims/node-empty.cjs');

const nodeBuiltinShims = [
  'fs',
  'path',
  'url',
  'http',
  'https',
  'stream',
  'util',
  'os',
  'crypto',
  'buffer',
  'node:fs',
  'node:path',
  'node:url',
  'node:http',
  'node:https',
  'node:stream',
  'node:util',
  'node:os',
  'node:crypto',
  'node:buffer',
].map((id) => ({ find: id, replacement: nodeEmpty }));

export default defineConfig(({ command }) => {
  const isBuild = command === 'build';
  // 勿使用 ../fonts、../icons：Vite 会在构建期解析 url()，本地不存在即产生大量告警。
  // Roboto 与 Fusion / Nacos 图标字体使用阿里 CDN（与 @alifd/next 包内默认一致），嵌入 base: './' 时也可用完整 https。
  const scssAdditionalData = `
$font-custom-path: "https://i.alicdn.com/artascope-font/20160419204543/font/";
`;

  return {
    base: isBuild ? './' : '/',
    publicDir: 'public',
    logLevel: isBuild ? 'warn' : 'info',
    plugins: [
      react({
        jsxRuntime: 'classic',
        include: /\.(jsx|js|tsx|ts)$/,
        // 生产 + classic 时若未配置 babel，插件会删掉 Babel transform，仅走 esbuild/oxc，.js 内 JSX 会失败
        babel: {
          babelrc: true,
          configFile: false,
          // lib.js / globalLib.js 无 JSX，跳过 Babel 可避免 sourcemap 定位报错
          ignore: [/[/\\]src[/\\]lib\.js$/, /[/\\]src[/\\]globalLib\.js$/],
        },
      }),
    ],
    define: {
      'process.env': {},
      'process.version': JSON.stringify(''),
    },
    resolve: {
      alias: [
        ...nodeBuiltinShims,
        {
          find: /^~(.*)$/,
          replacement: `${path.resolve(process.cwd(), 'node_modules')}/$1`,
        },
        { find: '@', replacement: path.resolve(rootDir, 'src') },
        { find: 'utils', replacement: path.resolve(rootDir, 'src/utils') },
        { find: 'components', replacement: path.resolve(rootDir, 'src/components') },
        { find: 'jquery', replacement: path.resolve(rootDir, 'src/shims/jquery.js') },
      ],
    },
    server: {
      port: Number(process.env.PORT) || 8000,
      proxy: {
        '/v1': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (p) => p.replace(/^\/v1/, '/nacos/v1'),
        },
        '/v2': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (p) => p.replace(/^\/v2/, '/nacos/v2'),
        },
        '/v3': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (p) => p.replace(/^\/v3/, '/nacos/v3'),
        },
      },
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      sourcemap: false,
      chunkSizeWarningLimit: 5000,
      commonjsOptions: {
        defaultIsModuleExports: true,
        requireReturnsDefault: 'auto',
      },
      rollupOptions: {
        output: {
          entryFileNames: 'js/main.js',
          chunkFileNames: 'js/[name].js',
          assetFileNames: (info) => {
            if (info.name?.endsWith('.css')) return 'css/main.css';
            return 'assets/[name]-[hash][extname]';
          },
        },
      },
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: scssAdditionalData,
          // @alifd/next 仍使用旧式 Sass 语法；依赖侧短期无法升级，构建时静默常见弃用类。
          quietDeps: true,
          silenceDeprecations: ['if-function', 'import', 'global-builtin'],
        },
      },
    },
  };
});
