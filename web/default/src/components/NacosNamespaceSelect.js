import React, { useEffect, useRef, useState } from 'react';
import { Dropdown } from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../helpers';

const NacosNamespaceSelect = ({ value, onChange, fluid = true, disabled = false }) => {
  const { t } = useTranslation();
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const mounted = useRef(true);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  useEffect(() => {
    if (!value) return;
    setOptions((prev) => {
      if (prev.some((o) => o.value === value)) return prev;
      return [...prev, { key: value, value, text: value }].sort((a, b) =>
        a.text.localeCompare(b.text)
      );
    });
  }, [value]);

  useEffect(() => {
    const delay = searchQuery ? 280 : 0;
    const t = setTimeout(async () => {
      setLoading(true);
      try {
        const res = await API.get('/api/nacos/namespaces/options', {
          params: { q: searchQuery || '' },
        });
        if (!mounted.current) return;
        if (!res.data?.success) {
          showError(res.data?.message || 'namespace options');
          return;
        }
        const ns = res.data.data?.namespaces || [];
        setOptions(ns.map((id) => ({ key: id, value: id, text: id })));
      } catch (e) {
        if (mounted.current) showError(e.message);
      } finally {
        if (mounted.current) setLoading(false);
      }
    }, delay);
    return () => clearTimeout(t);
  }, [searchQuery]);

  const handleAddItem = (e, { value: newVal }) => {
    const v = String(newVal || '').trim();
    if (!v) return;
    onChange(v);
    setOptions((prev) => {
      if (prev.some((o) => o.value === v)) return prev;
      return [...prev, { key: v, value: v, text: v }].sort((a, b) =>
        a.text.localeCompare(b.text)
      );
    });
  };

  return (
    <Dropdown
      selection
      search
      fluid={fluid}
      disabled={disabled}
      loading={loading}
      placeholder={t('nacos.ns_select_placeholder')}
      options={options}
      value={value}
      onChange={(e, { value: v }) => onChange(v)}
      onSearchChange={(e, { searchQuery: sq }) => setSearchQuery(sq)}
      allowAdditions
      additionLabel={`${t('nacos.ns_custom')}: `}
      onAddItem={handleAddItem}
    />
  );
};

export default NacosNamespaceSelect;
