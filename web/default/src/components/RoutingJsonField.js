import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Form,
  Icon,
  Label,
  Message,
  Modal,
  Segment,
  TextArea,
} from 'semantic-ui-react';
import { copy, showError, showSuccess } from '../helpers';
import './RoutingJsonField.css';

function analyzeJson(text, allowEmpty) {
  const raw = text ?? '';
  const trimmed = raw.trim();
  if (!trimmed) {
    if (allowEmpty) {
      return { ok: true, kind: 'empty', message: null };
    }
    return {
      ok: false,
      kind: 'error',
      message: 'empty',
    };
  }
  try {
    JSON.parse(trimmed);
    return { ok: true, kind: 'json', message: null };
  } catch (e) {
    return {
      ok: false,
      kind: 'error',
      message: e instanceof Error ? e.message : String(e),
    };
  }
}

/**
 * JSON 专用编辑区：格式化 / 压缩 / 复制 / 全屏编辑 + 即时校验提示。
 */
function RoutingJsonField({
  id,
  value,
  onChange,
  disabled,
  readOnly,
  minRows = 12,
  allowEmpty = true,
  placeholder,
}) {
  const { t } = useTranslation();
  const [text, setText] = useState(value ?? '');
  const [expanded, setExpanded] = useState(false);
  const [draft, setDraft] = useState('');

  useEffect(() => {
    setText(value ?? '');
  }, [value]);

  const analysis = useMemo(
    () => analyzeJson(text, allowEmpty),
    [text, allowEmpty]
  );

  const commit = (next) => {
    setText(next);
    if (onChange) {
      onChange(next);
    }
  };

  const handleAreaChange = (e, { value: v }) => {
    commit(v);
  };

  const formatJson = () => {
    const a = analyzeJson(text, allowEmpty);
    if (a.kind === 'empty') {
      commit('');
      return;
    }
    if (!a.ok) {
      showError(t('routing.json_error') + ': ' + (a.message || ''));
      return;
    }
    try {
      const obj = JSON.parse(text.trim());
      commit(JSON.stringify(obj, null, 2));
    } catch (e) {
      showError(e.message || String(e));
    }
  };

  const minifyJson = () => {
    const a = analyzeJson(text, allowEmpty);
    if (a.kind === 'empty') {
      commit('');
      return;
    }
    if (!a.ok) {
      showError(t('routing.json_error') + ': ' + (a.message || ''));
      return;
    }
    try {
      const obj = JSON.parse(text.trim());
      commit(JSON.stringify(obj));
    } catch (e) {
      showError(e.message || String(e));
    }
  };

  const copyAll = async () => {
    const ok = await copy(text);
    if (ok) {
      showSuccess(t('routing.json_copied'));
    } else {
      showError(t('routing.json_copy_failed'));
    }
  };

  const openExpand = () => {
    setDraft(text);
    setExpanded(true);
  };

  const applyExpand = () => {
    const a = analyzeJson(draft, allowEmpty);
    if (!a.ok && a.message !== 'empty') {
      showError(t('routing.json_error') + ': ' + (a.message || ''));
      return;
    }
    commit(draft);
    setExpanded(false);
  };

  const locked = disabled || readOnly;
  const statusLabel =
    analysis.kind === 'empty' ? (
      <Label size="small" basic color="grey">
        {t('routing.json_empty_badge')}
      </Label>
    ) : analysis.ok ? (
      <Label size="small" color="green">
        <Icon name="checkmark" />
        {t('routing.json_ok')}
      </Label>
    ) : (
      <Label size="small" color="red">
        <Icon name="warning circle" />
        {t('routing.json_error')}
      </Label>
    );

  return (
    <div className="routing-json-field">
      <Segment className="routing-json-segment" basic>
        <div className="routing-json-toolbar">
        <Button.Group size="small" basic>
          <Button
            type="button"
            disabled={locked}
            onClick={formatJson}
            title={t('routing.json_format')}
          >
            <Icon name="align justify" />
            {t('routing.json_format')}
          </Button>
          <Button
            type="button"
            disabled={locked}
            onClick={minifyJson}
            title={t('routing.json_minify')}
          >
            <Icon name="compress" />
            {t('routing.json_minify')}
          </Button>
          <Button type="button" onClick={copyAll} title={t('routing.json_copy')}>
            <Icon name="copy outline" />
            {t('routing.json_copy')}
          </Button>
          <Button
            type="button"
            disabled={locked}
            onClick={openExpand}
            title={t('routing.json_expand')}
          >
            <Icon name="expand arrows alternate" />
            {t('routing.json_expand')}
          </Button>
        </Button.Group>
        <span className="routing-json-status">{statusLabel}</span>
      </div>

      {!analysis.ok && analysis.message && analysis.message !== 'empty' ? (
        <Message negative size="small" className="routing-json-parse-msg">
          <Message.Header>{t('routing.json_error')}</Message.Header>
          <pre className="routing-json-error-pre">{analysis.message}</pre>
        </Message>
      ) : null}

      <TextArea
        id={id}
        className={
          'routing-json-textarea' +
          (!analysis.ok && analysis.message !== 'empty'
            ? ' routing-json-textarea-invalid'
            : '')
        }
        placeholder={placeholder}
        disabled={locked}
        readOnly={readOnly}
        rows={minRows}
        value={text}
        onChange={handleAreaChange}
        spellCheck={false}
        autoCorrect="off"
        autoCapitalize="off"
      />
      </Segment>

      <Modal
        open={expanded}
        onClose={() => setExpanded(false)}
        size="fullscreen"
        closeIcon
        closeOnDimmerClick={false}
      >
        <Modal.Header>{t('routing.json_expand')}</Modal.Header>
        <Modal.Content scrolling style={{ flex: 1 }}>
          <Form>
            <TextArea
              className="routing-json-textarea routing-json-textarea-modal"
              rows={32}
              value={draft}
              onChange={(e, { value: v }) => setDraft(v)}
              spellCheck={false}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setExpanded(false)}>
            {t('routing.json_expand_cancel')}
          </Button>
          <Button primary onClick={applyExpand} disabled={locked}>
            {t('routing.json_expand_apply')}
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
}

export default RoutingJsonField;
