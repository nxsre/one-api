#!/usr/bin/env node
/**
 * semantic-ui-css 默认通过 @import 拉取 Google Fonts（Lato）。
 * 本项目已在 public/fonts 下自建字体，需在安装依赖后去掉该行，避免外链请求。
 */
'use strict';

const fs = require('fs');
const path = require('path');

const semanticRoot = path.join(__dirname, '..', 'node_modules', 'semantic-ui-css');
const files = ['semantic.min.css', 'semantic.css', 'components/site.min.css', 'components/site.css'];

function stripGoogleFontImport(content) {
  return content.replace(
    /@import\s+url\(\s*['"]?https?:\/\/fonts\.googleapis\.com\/[^'")\s]+['"]?\s*\)\s*;?/gi,
    ''
  );
}

if (!fs.existsSync(semanticRoot)) {
  process.stderr.write(
    '[patch-semantic-ui-no-google-fonts] skip: semantic-ui-css not installed\n'
  );
  process.exit(0);
}

for (const rel of files) {
  const p = path.join(semanticRoot, rel);
  if (!fs.existsSync(p)) continue;
  const orig = fs.readFileSync(p, 'utf8');
  const next = stripGoogleFontImport(orig);
  if (orig !== next) {
    fs.writeFileSync(p, next);
    console.log('[patch-semantic-ui-no-google-fonts] stripped:', rel);
  }
}
