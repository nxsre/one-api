import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Checkbox, Divider, Form, Modal } from 'semantic-ui-react';

export const emptyCatalogForm = () => ({
  model_id: '',
  model_name: '',
  owned_by: '',
  enabled: true,
  source: 'manual',
  tags: '',
  provider_key: '',
  provider_display: '',
  family: '',
  modalities_in: '',
  modalities_out: '',
  context_limit: '',
  output_limit: '',
  cost_input: '',
  cost_output: '',
  cost_cache_read: '',
  cost_cache_write: '',
  reasoning: false,
  tool_call: false,
  temperature_ok: false,
  attachment_ok: false,
  open_weights: false,
  knowledge_cutoff: '',
  release_date: '',
  last_updated: '',
  npm_package: '',
  api_base: '',
  doc_url: '',
  notes: '',
});

export function rowToCatalogForm(row) {
  if (!row) return emptyCatalogForm();
  const numStr = (n) => {
    if (n === null || n === undefined || Number.isNaN(Number(n))) return '';
    if (Number(n) === 0) return '';
    return String(n);
  };
  return {
    model_id: row.model_id || '',
    model_name: row.model_name || '',
    owned_by: row.owned_by || '',
    enabled: !!row.enabled,
    source: row.source || 'manual',
    tags: row.tags || '',
    provider_key: row.provider_key || '',
    provider_display: row.provider_display || '',
    family: row.family || '',
    modalities_in: row.modalities_in || '',
    modalities_out: row.modalities_out || '',
    context_limit: row.context_limit ? String(row.context_limit) : '',
    output_limit: row.output_limit ? String(row.output_limit) : '',
    cost_input: numStr(row.cost_input),
    cost_output: numStr(row.cost_output),
    cost_cache_read: numStr(row.cost_cache_read),
    cost_cache_write: numStr(row.cost_cache_write),
    reasoning: !!row.reasoning,
    tool_call: !!row.tool_call,
    temperature_ok: !!row.temperature_ok,
    attachment_ok: !!row.attachment_ok,
    open_weights: !!row.open_weights,
    knowledge_cutoff: row.knowledge_cutoff || '',
    release_date: row.release_date || '',
    last_updated: row.last_updated || '',
    npm_package: row.npm_package || '',
    api_base: row.api_base || '',
    doc_url: row.doc_url || '',
    notes: row.notes || '',
  };
}

function parseIntField(v) {
  const s = String(v ?? '').trim();
  if (!s) return 0;
  const n = parseInt(s, 10);
  return Number.isFinite(n) ? n : 0;
}

function parseFloatField(v) {
  const s = String(v ?? '').trim();
  if (!s) return 0;
  const n = parseFloat(s);
  return Number.isFinite(n) ? n : 0;
}

export function catalogFormToPayload(form, editRow) {
  const body = {
    model_id: String(form.model_id || '').trim(),
    model_name: String(form.model_name || '').trim(),
    owned_by: String(form.owned_by || '').trim(),
    enabled: !!form.enabled,
    notes: String(form.notes || '').trim(),
    provider_key: String(form.provider_key || '').trim(),
    provider_display: String(form.provider_display || '').trim(),
    family: String(form.family || '').trim(),
    npm_package: String(form.npm_package || '').trim(),
    api_base: String(form.api_base || '').trim(),
    doc_url: String(form.doc_url || '').trim(),
    modalities_in: String(form.modalities_in || '').trim(),
    modalities_out: String(form.modalities_out || '').trim(),
    context_limit: parseIntField(form.context_limit),
    output_limit: parseIntField(form.output_limit),
    cost_input: parseFloatField(form.cost_input),
    cost_output: parseFloatField(form.cost_output),
    cost_cache_read: parseFloatField(form.cost_cache_read),
    cost_cache_write: parseFloatField(form.cost_cache_write),
    reasoning: !!form.reasoning,
    tool_call: !!form.tool_call,
    temperature_ok: !!form.temperature_ok,
    attachment_ok: !!form.attachment_ok,
    open_weights: !!form.open_weights,
    knowledge_cutoff: String(form.knowledge_cutoff || '').trim(),
    release_date: String(form.release_date || '').trim(),
    last_updated: String(form.last_updated || '').trim(),
    tags: String(form.tags || '').trim(),
  };
  if (editRow) {
    body.id = editRow.id;
    body.source = String(form.source || editRow.source || 'manual').trim();
  }
  return body;
}

function SectionHeader({ title }) {
  return (
    <Divider horizontal className='model-catalog-form-section'>
      <span>{title}</span>
    </Divider>
  );
}

function BoolField({ label, checked, onChange }) {
  return (
    <Form.Field className='model-catalog-form-bool'>
      <Checkbox label={label} checked={checked} onChange={(e, { checked: c }) => onChange(!!c)} />
    </Form.Field>
  );
}

export default function ModelCatalogEditModal({
  open,
  editRow,
  saving,
  onClose,
  onSave,
}) {
  const { t } = useTranslation();
  const [form, setForm] = useState(emptyCatalogForm);

  useEffect(() => {
    if (open) {
      setForm(rowToCatalogForm(editRow));
    }
  }, [open, editRow]);

  const set = (key) => (e, { value, checked }) => {
    setForm((prev) => ({
      ...prev,
      [key]: value !== undefined ? value : checked,
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onSave(form);
  };

  return (
    <Modal open={open} onClose={onClose} size='large' closeIcon className='model-catalog-edit-modal'>
      <Modal.Header>
        {editRow
          ? t('model_catalog.modal_edit_title')
          : t('model_catalog.modal_add_title')}
      </Modal.Header>
      <Modal.Content scrolling className='model-catalog-edit-modal__content'>
        <Form id='model-catalog-edit-form' onSubmit={handleSubmit}>
          <SectionHeader title={t('model_catalog.form_section_basic')} />
          <Form.Group widths='equal'>
            <Form.Input
              required
              label={t('model_catalog.col_model_id')}
              value={form.model_id}
              onChange={set('model_id')}
              placeholder='gpt-4o'
            />
            <Form.Input
              label={t('model_catalog.col_model_name')}
              value={form.model_name}
              onChange={set('model_name')}
              placeholder={t('model_catalog.form_model_name_ph')}
            />
          </Form.Group>
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_owned_by')}
              value={form.owned_by}
              onChange={set('owned_by')}
            />
            <Form.Input
              label={t('model_catalog.col_tags', '标签')}
              value={form.tags}
              onChange={set('tags')}
              placeholder={t('model_catalog.form_tags_ph')}
            />
          </Form.Group>
          <Form.Group widths='equal'>
            {editRow ? (
              <Form.Input
                label={t('model_catalog.col_source')}
                value={form.source}
                onChange={set('source')}
              />
            ) : (
              <Form.Input label={t('model_catalog.col_source')} value='manual' readOnly />
            )}
            <Form.Field className='model-catalog-form-enabled'>
              <label>{t('model_catalog.col_enabled')}</label>
              <Checkbox
                toggle
                checked={form.enabled}
                onChange={(e, { checked }) =>
                  setForm((prev) => ({ ...prev, enabled: !!checked }))
                }
              />
            </Form.Field>
          </Form.Group>

          <SectionHeader title={t('model_catalog.form_section_provider')} />
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_provider_key')}
              value={form.provider_key}
              onChange={set('provider_key')}
            />
            <Form.Input
              label={t('model_catalog.col_provider_name')}
              value={form.provider_display}
              onChange={set('provider_display')}
            />
            <Form.Input
              label={t('model_catalog.col_family')}
              value={form.family}
              onChange={set('family')}
            />
          </Form.Group>

          <SectionHeader title={t('model_catalog.form_section_capability')} />
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_modalities_in')}
              value={form.modalities_in}
              onChange={set('modalities_in')}
              placeholder='text,image'
            />
            <Form.Input
              label={t('model_catalog.col_modalities_out')}
              value={form.modalities_out}
              onChange={set('modalities_out')}
              placeholder='text'
            />
          </Form.Group>
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_context_limit')}
              value={form.context_limit}
              onChange={set('context_limit')}
              type='number'
              min={0}
            />
            <Form.Input
              label={t('model_catalog.col_output_limit')}
              value={form.output_limit}
              onChange={set('output_limit')}
              type='number'
              min={0}
            />
          </Form.Group>
          <div className='model-catalog-form-bool-row'>
            <BoolField
              label={t('model_catalog.col_reasoning')}
              checked={form.reasoning}
              onChange={(v) => setForm((p) => ({ ...p, reasoning: v }))}
            />
            <BoolField
              label={t('model_catalog.col_tool_call')}
              checked={form.tool_call}
              onChange={(v) => setForm((p) => ({ ...p, tool_call: v }))}
            />
            <BoolField
              label={t('model_catalog.col_temperature')}
              checked={form.temperature_ok}
              onChange={(v) => setForm((p) => ({ ...p, temperature_ok: v }))}
            />
            <BoolField
              label={t('model_catalog.col_attachment')}
              checked={form.attachment_ok}
              onChange={(v) => setForm((p) => ({ ...p, attachment_ok: v }))}
            />
            <BoolField
              label={t('model_catalog.col_open_weights')}
              checked={form.open_weights}
              onChange={(v) => setForm((p) => ({ ...p, open_weights: v }))}
            />
          </div>

          <SectionHeader title={t('model_catalog.form_section_pricing')} />
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_cost_input')}
              value={form.cost_input}
              onChange={set('cost_input')}
              type='number'
              step='any'
              min={0}
            />
            <Form.Input
              label={t('model_catalog.col_cost_output')}
              value={form.cost_output}
              onChange={set('cost_output')}
              type='number'
              step='any'
              min={0}
            />
            <Form.Input
              label={t('model_catalog.col_cost_cache_read')}
              value={form.cost_cache_read}
              onChange={set('cost_cache_read')}
              type='number'
              step='any'
              min={0}
            />
            <Form.Input
              label={t('model_catalog.col_cost_cache_write')}
              value={form.cost_cache_write}
              onChange={set('cost_cache_write')}
              type='number'
              step='any'
              min={0}
            />
          </Form.Group>

          <SectionHeader title={t('model_catalog.form_section_meta')} />
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_knowledge')}
              value={form.knowledge_cutoff}
              onChange={set('knowledge_cutoff')}
            />
            <Form.Input
              label={t('model_catalog.col_release_date')}
              value={form.release_date}
              onChange={set('release_date')}
            />
            <Form.Input
              label={t('model_catalog.col_last_updated')}
              value={form.last_updated}
              onChange={set('last_updated')}
            />
          </Form.Group>
          <Form.Group widths='equal'>
            <Form.Input
              label={t('model_catalog.col_npm')}
              value={form.npm_package}
              onChange={set('npm_package')}
            />
            <Form.Input
              label={t('model_catalog.col_api_base')}
              value={form.api_base}
              onChange={set('api_base')}
            />
          </Form.Group>
          <Form.Input
            label={t('model_catalog.col_doc')}
            value={form.doc_url}
            onChange={set('doc_url')}
            placeholder='https://'
          />
          <Form.TextArea
            label={t('model_catalog.col_notes')}
            value={form.notes}
            onChange={set('notes')}
            rows={3}
          />
        </Form>
      </Modal.Content>
      <Modal.Actions>
        <Button type='button' onClick={onClose} disabled={saving}>
          {t('model_catalog.btn_cancel')}
        </Button>
        <Button
          primary
          type='submit'
          form='model-catalog-edit-form'
          loading={saving}
          disabled={saving}
        >
          {t('model_catalog.btn_save')}
        </Button>
      </Modal.Actions>
    </Modal>
  );
}
