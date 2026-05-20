import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Form, Modal } from 'semantic-ui-react';
import ModelCatalogProviderSearch from '../../components/ModelCatalogProviderSearch';
import {
  fetchModelCatalogModelIds,
  fetchModelCatalogProviders,
  providerCatalogModelFilters,
  providerOptionsToDropdown,
} from '../../helpers/modelCatalog';
import { showError } from '../../helpers';

export default function FillCatalogModelsModal({
  open,
  onClose,
  onConfirm,
  tenantConsole = false,
}) {
  const { t } = useTranslation();
  const [provider, setProvider] = useState('');
  const [providerOptions, setProviderOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [previewCount, setPreviewCount] = useState(null);

  useEffect(() => {
    if (!open) return;
    setProvider('');
    setPreviewCount(null);
    fetchModelCatalogProviders('', 100, tenantConsole)
      .then((rows) => setProviderOptions(providerOptionsToDropdown(rows)))
      .catch(() => setProviderOptions([]));
  }, [open, tenantConsole]);

  useEffect(() => {
    if (!open) return;
    const filters = providerCatalogModelFilters(provider, providerOptions);
    if (!filters) {
      setPreviewCount(null);
      return;
    }
    const timer = setTimeout(() => {
      fetchModelCatalogModelIds(filters, tenantConsole)
        .then((ids) => setPreviewCount(ids.length))
        .catch(() => setPreviewCount(null));
    }, 250);
    return () => clearTimeout(timer);
  }, [open, provider, providerOptions, tenantConsole]);

  const submit = async () => {
    const filters = providerCatalogModelFilters(provider, providerOptions);
    if (!filters) {
      showError(t('channel.edit.fill_catalog_provider_required'));
      return;
    }
    setLoading(true);
    try {
      const ids = await fetchModelCatalogModelIds(filters, tenantConsole);
      if (!ids.length) {
        showError(t('channel.edit.fill_catalog_empty'));
        return;
      }
      onConfirm(ids);
      onClose();
    } catch (e) {
      showError(e.message || t('channel.edit.fill_catalog_fail'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal open={open} onClose={onClose} size='small'>
      <Modal.Header>{t('channel.edit.fill_catalog_title')}</Modal.Header>
      <Modal.Content>
        <p className='channel-edit-field-hint'>{t('channel.edit.fill_catalog_hint')}</p>
        <ModelCatalogProviderSearch
          label={t('model_catalog.filter_provider_label')}
          placeholder={t('channel.edit.fill_catalog_provider_ph')}
          value={provider}
          onChange={setProvider}
          tenantConsole={tenantConsole}
        />
        {previewCount != null && provider ? (
          <Form.Field>
            <label>{t('channel.edit.fill_catalog_preview', { count: previewCount })}</label>
          </Form.Field>
        ) : null}
      </Modal.Content>
      <Modal.Actions>
        <Button type='button' basic onClick={onClose}>
          {t('channel.edit.fill_catalog_cancel')}
        </Button>
        <Button type='button' primary loading={loading} disabled={loading} onClick={() => void submit()}>
          {t('channel.edit.fill_catalog_confirm')}
        </Button>
      </Modal.Actions>
    </Modal>
  );
}
