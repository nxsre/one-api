import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Checkbox, Form, Header, Segment } from 'semantic-ui-react';
import {
  EMPTY_ROUTING_POLICY,
  safeParseObject,
  stringifyPolicy,
  toFloat,
  toInt,
} from './policySerialize';
import PolicyFormResetRow from './PolicyFormResetRow';

/** @param {{ value: string, onChange: (json: string) => void, disabled?: boolean }} props */
export default function RoutingPolicyForm({ value, onChange, disabled }) {
  const { t } = useTranslation();
  const [p, setP] = useState(() => safeParseObject(value, EMPTY_ROUTING_POLICY));

  useEffect(() => {
    setP(safeParseObject(value, EMPTY_ROUTING_POLICY));
  }, [value]);

  const commit = (next) => {
    setP(next);
    onChange(stringifyPolicy(next));
  };

  const nf = (name) => (e, { value: v }) => {
    commit({ ...p, [name]: v });
  };

  const nfi = (name) => (e, { value: v }) => {
    commit({ ...p, [name]: toInt(v, p[name] ?? 0) });
  };

  const nff = (name) => (e, { value: v }) => {
    commit({ ...p, [name]: toFloat(v, p[name] ?? 0) });
  };

  const nfcb = (name) => (e, { checked }) => {
    commit({ ...p, [name]: !!checked });
  };

  const selModeOpts = [
    { key: 'weighted_random', text: t('routing.form_sel_weighted_random'), value: 'weighted_random' },
    { key: 'consistent_hash', text: t('routing.form_sel_consistent_hash'), value: 'consistent_hash' },
  ];

  const stickyOpts = [
    { key: 'token_id', text: 'token_id', value: 'token_id' },
    { key: 'user_id', text: 'user_id', value: 'user_id' },
    { key: 'none', text: 'none', value: 'none' },
  ];

  const probeOpts = [
    { key: 'secondary_uniform', text: 'secondary_uniform', value: 'secondary_uniform' },
    { key: 'worst_first', text: 'worst_first', value: 'worst_first' },
  ];

  return (
    <Segment basic className="routing-policy-form-segment">
      <Header as="h5">{t('routing.form_routing_core')}</Header>
      <Form>
        <Form.Group widths="equal">
          <Form.Select
            label={t('routing.form_selection_mode')}
            options={selModeOpts}
            value={p.selection_mode || 'weighted_random'}
            onChange={(e, { value: v }) => commit({ ...p, selection_mode: v })}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_consistent_hash_seed')}
            value={p.consistent_hash_seed ?? ''}
            onChange={nf('consistent_hash_seed')}
            disabled={disabled}
            placeholder="consistent_hash"
          />
          <Form.Select
            label={t('routing.form_sticky_source')}
            options={stickyOpts}
            value={p.sticky_source || 'token_id'}
            onChange={(e, { value: v }) => commit({ ...p, sticky_source: v })}
            disabled={disabled}
          />
        </Form.Group>

        <Header as="h5">{t('routing.form_direction')}</Header>
        <Form.Group widths="equal">
          <Form.Field>
            <Checkbox
              label={t('routing.form_direction_enabled')}
              checked={!!p.direction_enabled}
              onChange={nfcb('direction_enabled')}
              disabled={disabled}
            />
          </Form.Field>
          <Form.Input
            label={t('routing.form_direction_probe_ratio')}
            type="number"
            step="0.01"
            value={p.direction_probe_ratio}
            onChange={nff('direction_probe_ratio')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_direction_min_probe_ratio')}
            type="number"
            step="0.01"
            value={p.direction_min_probe_ratio}
            onChange={nff('direction_min_probe_ratio')}
            disabled={disabled}
          />
        </Form.Group>
        <Form.Group widths="equal">
          <Form.Input
            label={t('routing.form_direction_latency_weight')}
            type="number"
            step="0.1"
            value={p.direction_latency_weight}
            onChange={nff('direction_latency_weight')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_direction_error_weight')}
            type="number"
            step="0.1"
            value={p.direction_error_weight}
            onChange={nff('direction_error_weight')}
            disabled={disabled}
          />
          <Form.Select
            label={t('routing.form_probe_pick_strategy')}
            options={probeOpts}
            value={p.probe_pick_strategy || 'secondary_uniform'}
            onChange={(e, { value: v }) => commit({ ...p, probe_pick_strategy: v })}
            disabled={disabled}
          />
        </Form.Group>

        <Header as="h5">{t('routing.form_adaptive_circuit')}</Header>
        <Form.Group widths="equal">
          <Form.Field>
            <Checkbox
              label={t('routing.form_auto_adaptive_enabled')}
              checked={!!p.auto_adaptive_enabled}
              onChange={nfcb('auto_adaptive_enabled')}
              disabled={disabled}
            />
          </Form.Field>
          <Form.Input
            label={t('routing.form_adaptive_interval_sec')}
            type="number"
            value={p.adaptive_interval_sec}
            onChange={nfi('adaptive_interval_sec')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_circuit_fail_threshold')}
            type="number"
            value={p.circuit_fail_threshold}
            onChange={nfi('circuit_fail_threshold')}
            disabled={disabled}
          />
        </Form.Group>
        <Form.Group widths="equal">
          <Form.Input
            label={t('routing.form_circuit_cooldown_sec')}
            type="number"
            value={p.circuit_cooldown_sec}
            onChange={nfi('circuit_cooldown_sec')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_half_open_probe_ms')}
            type="number"
            value={p.half_open_probe_ms}
            onChange={nfi('half_open_probe_ms')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_aggregation_interval_sec')}
            type="number"
            value={p.aggregation_interval_sec}
            onChange={nfi('aggregation_interval_sec')}
            disabled={disabled}
          />
        </Form.Group>
        <Form.Group widths="equal">
          <Form.Input
            label={t('routing.form_adaptive_aggressive_penalty')}
            type="number"
            step="0.01"
            value={p.adaptive_aggressive_penalty}
            onChange={nff('adaptive_aggressive_penalty')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_adaptive_gentle_boost')}
            type="number"
            step="0.01"
            value={p.adaptive_gentle_boost}
            onChange={nff('adaptive_gentle_boost')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_adaptive_err_ratio_threshold')}
            type="number"
            step="0.01"
            value={p.adaptive_err_ratio_threshold}
            onChange={nff('adaptive_err_ratio_threshold')}
            disabled={disabled}
          />
        </Form.Group>
        <Form.Group widths="equal">
          <Form.Input
            label={t('routing.form_auto_weight_min')}
            type="number"
            step="0.01"
            value={p.auto_weight_min}
            onChange={nff('auto_weight_min')}
            disabled={disabled}
          />
          <Form.Input
            label={t('routing.form_auto_weight_max')}
            type="number"
            step="0.01"
            value={p.auto_weight_max}
            onChange={nff('auto_weight_max')}
            disabled={disabled}
          />
        </Form.Group>
        <PolicyFormResetRow
          disabled={disabled}
          onReset={() => commit({ ...EMPTY_ROUTING_POLICY })}
        />
      </Form>
    </Segment>
  );
}
