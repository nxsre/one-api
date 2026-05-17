import * as monaco from 'monaco-editor';
import { loader, type Monaco } from '@monaco-editor/react';

import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';

/** 与静态资源目录一致：npm install / build 时拷入 public/monaco-editor/min/vs */
function monacoVsBase(): string {
  const base = import.meta.env.BASE_URL ?? '/';
  const trimmed = base.endsWith('/') ? base.slice(0, -1) : base;
  const joined = `${trimmed}/monaco-editor/min/vs`.replace(/([^:])\/{2,}/g, '$1/');
  return joined.replace(/\/$/, '');
}

// 本地打包 Monaco；Worker 走 Vite ?worker。paths.vs 指向同源拷贝，防止 loader 回退到 jsDelivr AMD。
self.MonacoEnvironment = {
  getWorker(_, label) {
    if (label === 'json') return new jsonWorker();
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker();
    return new editorWorker();
  },
};

loader.config({
  monaco: monaco as unknown as Monaco,
  paths: {
    vs: monacoVsBase(),
  },
});
