import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Dropdown, Form } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import {
  fetchModelCatalogProviders,
  providerOptionsToDropdown,
} from '../helpers/modelCatalog';

export default function ModelCatalogProviderSearch({
  value,
  onChange,
  label,
  placeholder,
  tenantConsole = false,
  fluid = true,
}) {
  const { t } = useTranslation();
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const timerRef = useRef(null);

  const loadProviders = useCallback(
    async (query) => {
      setLoading(true);
      try {
        const rows = await fetchModelCatalogProviders(query, 40, tenantConsole);
        setOptions(providerOptionsToDropdown(rows));
      } catch {
        setOptions([]);
      } finally {
        setLoading(false);
      }
    },
    [tenantConsole]
  );

  useEffect(() => {
    loadProviders(value).then();
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [loadProviders]);

  const handleSearchChange = (_, { searchQuery }) => {
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      loadProviders(searchQuery).then();
    }, 220);
  };

  const mergedOptions = useMemo(() => {
    const current = String(value || '').trim();
    if (!current) return options;
    if (options.some((o) => o.value === current)) return options;
    return [{ key: `custom-${current}`, value: current, text: current }, ...options];
  }, [options, value]);

  return (
    <Form.Field>
      {label ? <label>{label}</label> : null}
      <Dropdown
        fluid={fluid}
        search
        selection
        clearable
        allowAdditions
        selectOnBlur={false}
        loading={loading}
        options={mergedOptions}
        value={value || ''}
        placeholder={placeholder || t('model_catalog.filter_ph_provider')}
        noResultsMessage={t('model_catalog.provider_no_results')}
        onSearchChange={handleSearchChange}
        onAddItem={(_, { value: v }) => onChange(v)}
        onChange={(_, { value: v }) => onChange(v || '')}
        onBlur={() => {
          if (!String(value || '').trim() && mergedOptions.length === 1) {
            onChange(mergedOptions[0].value);
          }
        }}
      />
    </Form.Field>
  );
}
