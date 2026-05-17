/**
 * 将 monaco-editor/min/vs 拷到 public，供 loader.paths.vs 使用，避免从 jsDelivr 拉 loader/nls。
 */
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..');
const src = path.join(root, 'node_modules/monaco-editor/min/vs');
const dest = path.join(root, 'public/monaco-editor/min/vs');

if (!fs.existsSync(src)) {
  console.warn('[copy-monaco-vs] skip: monaco-editor/min/vs not found (run npm install)');
  process.exit(0);
}

fs.rmSync(dest, { recursive: true, force: true });
fs.mkdirSync(path.dirname(dest), { recursive: true });
fs.cpSync(src, dest, { recursive: true });
console.log('[copy-monaco-vs] copied to public/monaco-editor/min/vs');
