import React from 'react';
import { Modal, Button } from 'semantic-ui-react';
import { DiffEditor } from '@monaco-editor/react';
import { useTranslation } from 'react-i18next';
import './nacos-monaco.css';

/**
 * Side-by-side diff (Monaco), same UX family as Nacos console DiffEditorDialog.
 */
const NacosMonacoDiffModal = ({
  open,
  onClose,
  title,
  original,
  modified,
  language = 'plaintext',
  theme = 'vs',
}) => {
  const { t } = useTranslation();
  return (
    <Modal open={open} onClose={onClose} size='large' closeIcon>
      <Modal.Header>{title || t('nacos.cs_diff_title')}</Modal.Header>
      <Modal.Content style={{ paddingTop: 8 }}>
        <div
          className='nacos-monaco-diff-wrap'
          style={{ height: 'min(72vh, 680px)' }}
        >
          <DiffEditor
            height='100%'
            language={language}
            original={original ?? ''}
            modified={modified ?? ''}
            theme={theme}
            options={{
              readOnly: true,
              renderSideBySide: true,
              scrollBeyondLastLine: false,
              minimap: { enabled: false },
              fontSize: 13,
              automaticLayout: true,
            }}
          />
        </div>
      </Modal.Content>
      <Modal.Actions>
        <Button type='button' onClick={onClose}>
          {t('nacos.skills_close')}
        </Button>
      </Modal.Actions>
    </Modal>
  );
};

export default NacosMonacoDiffModal;
