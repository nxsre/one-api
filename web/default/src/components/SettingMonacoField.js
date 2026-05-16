import React, { useMemo, useState, useSyncExternalStore } from 'react';
import { Button, Form } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import NacosMonacoEditor from './nacos/NacosMonacoEditor';
import NacosMonacoDiffModal from './nacos/NacosMonacoDiffModal';
import { showError } from '../helpers';
import './setting-monaco-field.css';

function subscribeHtmlClass(cb) {
  const el = document.documentElement;
  const obs = new MutationObserver(cb);
  obs.observe(el, { attributes: true, attributeFilter: ['class'] });
  return () => obs.disconnect();
}

function htmlHasDarkClass() {
  return document.documentElement.classList.contains('dark');
}

function serverDarkSnapshot() {
  return false;
}

/**
 * 与 Nacos 控制台一致的 Monaco 编辑区：可选 JSON 格式化、与进入页面时的内容对比（Diff）。
 * 密码等敏感字段请仍使用 Form.Input type="password"。
 */
export default function SettingMonacoField({
  label,
  hint,
  value,
  originValue,
  onChange,
  language = 'plaintext',
  height = 220,
  minimap = false,
  enableJsonFormat = false,
  enableDiff = true,
  readOnly = false,
}) {
  const { t } = useTranslation();
  const [diffOpen, setDiffOpen] = useState(false);
  const isDark = useSyncExternalStore(
    subscribeHtmlClass,
    htmlHasDarkClass,
    serverDarkSnapshot
  );
  const monacoTheme = isDark ? 'vs-dark' : 'vs';

  const diffLanguage = useMemo(() => {
    if (language === 'json') return 'json';
    if (language === 'markdown' || language === 'html') return language;
    return 'plaintext';
  }, [language]);

  const formatJson = () => {
    try {
      const trimmed = (value || '').trim();
      if (!trimmed) {
        onChange('');
        return;
      }
      const parsed = JSON.parse(trimmed);
      onChange(JSON.stringify(parsed, null, 2));
    } catch (e) {
      showError(
        (e && e.message) || t('setting.monaco.json_invalid')
      );
    }
  };

  const strVal = value == null ? '' : String(value);
  const strOrigin = originValue == null ? '' : String(originValue);

  return (
    <Form.Field className='setting-monaco-field'>
      <div className='setting-monaco-field__head'>
        <div className='setting-monaco-field__label-block'>
          {typeof label === 'string' || typeof label === 'number' ? (
            <label>{label}</label>
          ) : (
            label
          )}
          {hint ? <div className='setting-monaco-field__hint'>{hint}</div> : null}
        </div>
        <div className='setting-monaco-field__actions'>
          {!readOnly && enableJsonFormat ? (
            <Button size='small' type='button' onClick={formatJson}>
              {t('setting.monaco.format_json')}
            </Button>
          ) : null}
          {enableDiff ? (
            <Button size='small' type='button' onClick={() => setDiffOpen(true)}>
              {t('setting.monaco.compare_saved')}
            </Button>
          ) : null}
        </div>
      </div>
      <div className='setting-monaco-field__surface'>
        <NacosMonacoEditor
          value={strVal}
          onChange={readOnly ? () => {} : onChange}
          language={language}
          height={height}
          minimap={minimap}
          theme={monacoTheme}
          readOnly={readOnly}
        />
      </div>
      {enableDiff ? (
        <NacosMonacoDiffModal
          open={diffOpen}
          onClose={() => setDiffOpen(false)}
          title={t('setting.monaco.diff_title')}
          original={strOrigin}
          modified={strVal}
          language={diffLanguage}
          theme={monacoTheme}
        />
      ) : null}
    </Form.Field>
  );
}
