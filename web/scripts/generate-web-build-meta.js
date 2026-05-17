#!/usr/bin/env node
'use strict';

/**
 * 在 npm run build（prebuild）前写入 public/web-build.json，
 * 每次生产构建生成唯一 build_id，供「关于」页展示前端构建版本。
 *
 * 用法：node ../scripts/generate-web-build-meta.js [主题目录]
 * 默认目录：与本脚本相对位置的 ../default
 */
const fs = require('fs');
const path = require('path');

const targetRoot = path.resolve(
  process.argv[2] || path.join(__dirname, '..', 'default')
);
const publicDir = path.join(targetRoot, 'public');
const pkgPath = path.join(targetRoot, 'package.json');

if (!fs.existsSync(pkgPath)) {
  console.error('generate-web-build-meta: missing package.json at', targetRoot);
  process.exit(1);
}

const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
const pkgVersion = pkg.version || '0.0.0';
const pad = (n) => String(n).padStart(2, '0');
const d = new Date();
const stamp =
  '' +
  d.getUTCFullYear() +
  pad(d.getUTCMonth() + 1) +
  pad(d.getUTCDate()) +
  pad(d.getUTCHours()) +
  pad(d.getUTCMinutes()) +
  pad(d.getUTCSeconds());

const meta = {
  package_version: pkgVersion,
  build_id: `${pkgVersion}+${stamp}`,
  built_at: d.toISOString(),
};

fs.mkdirSync(publicDir, { recursive: true });
fs.writeFileSync(
  path.join(publicDir, 'web-build.json'),
  JSON.stringify(meta) + '\n',
  'utf8'
);
console.log(
  `generate-web-build-meta [${path.basename(targetRoot)}]: ${meta.build_id}`
);
