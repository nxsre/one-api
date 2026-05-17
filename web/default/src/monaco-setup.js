/**
 * 使用本地 monaco 包 + public/monaco-editor/min/vs，禁止 @monaco-editor/loader 默认 jsDelivr AMD。
 */
import * as monaco from 'monaco-editor';
import { loader } from '@monaco-editor/react';

function monacoVsPath() {
  const pub = typeof process !== 'undefined' && process.env.PUBLIC_URL ? process.env.PUBLIC_URL : '';
  const trimmed = String(pub).replace(/\/$/, '');
  const path = `${trimmed}/monaco-editor/min/vs`.replace(/\/+/g, '/');
  return path.startsWith('/') ? path : `/${path}`;
}

self.MonacoEnvironment = {
  getWorker(_, label) {
    const workerOpts = { type: 'module' };
    if (label === 'json') {
      return new Worker(new URL('monaco-editor/esm/vs/language/json/json.worker.js', import.meta.url), workerOpts);
    }
    if (label === 'html' || label === 'handlebars' || label === 'razor') {
      return new Worker(new URL('monaco-editor/esm/vs/language/html/html.worker.js', import.meta.url), workerOpts);
    }
    return new Worker(new URL('monaco-editor/esm/vs/editor/editor.worker.js', import.meta.url), workerOpts);
  },
};

loader.config({
  monaco,
  paths: {
    vs: monacoVsPath(),
  },
});
