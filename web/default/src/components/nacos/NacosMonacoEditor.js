import React from 'react';
import Editor from '@monaco-editor/react';
import { Loader } from 'semantic-ui-react';
import './nacos-monaco.css';

const defaultOptions = (readOnly, minimap) => ({
  readOnly,
  minimap: { enabled: minimap },
  fontSize: 13,
  lineHeight: 22,
  padding: { top: 10, bottom: 10 },
  wordWrap: 'on',
  scrollBeyondLastLine: false,
  automaticLayout: true,
  tabSize: 2,
  formatOnPaste: !readOnly,
  formatOnType: !readOnly,
  folding: true,
  lineNumbers: 'on',
  renderWhitespace: 'selection',
  smoothScrolling: true,
  cursorBlinking: 'smooth',
  /* 当前行高亮含行号区，避免看起来被右侧「截断」 */
  renderLineHighlight: 'all',
  /* 去掉 overview 标尺右侧默认 1px 边框（常被看成一条黑线） */
  overviewRulerBorder: false,
  hideCursorInOverviewRuler: true,
});

/**
 * Nacos-style code editor (Monaco), for JSON / YAML / text configs.
 */
const NacosMonacoEditor = ({
  value,
  onChange,
  language = 'plaintext',
  readOnly = false,
  height = 380,
  minimap = true,
  theme = 'vs',
}) => (
  <div className='nacos-monaco-host' style={{ height }}>
    <Editor
      height='100%'
      language={language}
      theme={theme}
      value={value ?? ''}
      onChange={(v) => onChange && onChange(v ?? '')}
      loading={<Loader active inline='centered' />}
      options={defaultOptions(readOnly, minimap)}
    />
  </div>
);

export default NacosMonacoEditor;
