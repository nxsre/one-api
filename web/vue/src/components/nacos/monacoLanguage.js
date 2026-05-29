/**
 * Map Nacos config `type` / filename hints to Monaco language id.
 * Aligns with Nacos console supported formats (yaml/json/xml/html/text/…).
 */
export function monacoLanguageFromNacosType(type) {
  if (type == null) return 'plaintext';
  const t = String(type).trim().toLowerCase();
  if (!t) return 'plaintext';
  if (t === 'yaml' || t === 'yml') return 'yaml';
  if (t === 'json') return 'json';
  if (t === 'xml') return 'xml';
  if (t === 'html' || t === 'htm') return 'html';
  if (t === 'sh' || t === 'shell' || t === 'bash') return 'shell';
  if (t === 'md' || t === 'markdown') return 'markdown';
  if (t === 'sql') return 'sql';
  if (t === 'properties' || t === 'props') return 'plaintext';
  return 'plaintext';
}

/** Match Nacos console diff behavior for line endings. */
export function normalizeConfigEOL(s) {
  return String(s ?? '').replace(/\r\n/g, '\n');
}
