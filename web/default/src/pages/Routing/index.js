import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import moment from 'moment';
import {
  Button,
  Card,
  Form,
  Header,
  Icon,
  Message,
  Popup,
  Tab,
  Table,
} from 'semantic-ui-react';
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { API, isAdmin, isRoot, showError, showSuccess } from '../../helpers';
import { ROUTING_POLICY_JSON_SAMPLES } from './policyExamples';
import RelayRetryPolicyForm from './forms/RelayRetryPolicyForm';
import RoutingPolicyForm from './forms/RoutingPolicyForm';
import ModelAliasPolicyForm from './forms/ModelAliasPolicyForm';
import ModelRateLimitPolicyForm from './forms/ModelRateLimitPolicyForm';
import './Routing.css';
import {
  fetchRoutingChannels,
  fetchRoutingGroups,
  fetchRoutingModelCatalog,
  parseChannelModelsField,
} from './routingLookupLoaders';

function PolicyJsonExample({ summary, jsonText }) {
  return (
    <details style={{ marginBottom: '0.85rem' }}>
      <summary style={{ cursor: 'pointer', userSelect: 'none', fontWeight: 500 }}>
        {summary}
      </summary>
      <pre
        style={{
          marginTop: '0.5rem',
          padding: '0.75rem',
          background: 'rgba(34, 36, 38, 0.06)',
          borderRadius: 4,
          overflow: 'auto',
          fontSize: 12,
          lineHeight: 1.45,
          whiteSpace: 'pre',
        }}
      >
        {jsonText}
      </pre>
    </details>
  );
}

const POLICY_KEYS_BASIC = ['RoutingPolicy', 'RelayRetryPolicy', 'ModelRateLimitPolicy'];

/** 表头 + 悬停说明（渠道预览表 Direction～Fuse 列） */
function PreviewColumnHeader({ label, hint }) {
  return (
    <Popup
      wide
      hoverable
      position='top center'
      trigger={
        <span className='routing-preview-th-popup'>
          {label}
          <Icon name='info circle' />
        </span>
      }
      content={
        <div className='routing-preview-th-popup-body'>{hint}</div>
      }
    />
  );
}

export default function RoutingOperations() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [policies, setPolicies] = useState({});
  const [previewGroup, setPreviewGroup] = useState('default');
  const [previewModel, setPreviewModel] = useState('');
  const [previewMeta, setPreviewMeta] = useState(null);
  const [metricsDay, setMetricsDay] = useState(moment().utc().format('YYYYMMDD'));
  const [metrics, setMetrics] = useState([]);
  const [fuse, setFuse] = useState([]);
  const [tsCh, setTsCh] = useState('');
  const [tsModel, setTsModel] = useState('');
  const [tsHours, setTsHours] = useState(24);
  const [tsData, setTsData] = useState([]);
  const [mwCh, setMwCh] = useState('');
  const [mwMul, setMwMul] = useState('1');
  const [rwCh, setRwCh] = useState('');
  const [groupOpts, setGroupOpts] = useState([]);
  const [fullModelCatalog, setFullModelCatalog] = useState([]);
  const [modelOpts, setModelOpts] = useState([]);
  const [routingChannels, setRoutingChannels] = useState([]);
  const [channelOpts, setChannelOpts] = useState([]);
  const [lookupReady, setLookupReady] = useState(false);
  const [tsShowAllModels, setTsShowAllModels] = useState(false);
  const [tsResolvedModels, setTsResolvedModels] = useState([]);

  useEffect(() => {
    if (!isAdmin()) {
      navigate('/');
    }
  }, [navigate]);

  const loadBundle = async () => {
    try {
      const res = await API.get('/api/routing/policy-bundle');
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      const data = res.data.data || {};
      setPolicies({
        ...data,
        RelayProtocolBridgeEnabled:
          data.RelayProtocolBridgeEnabled === 'true' ? 'true' : 'false',
      });
    } catch (e) {
      showError(e.message);
    }
  };

  useEffect(() => {
    void loadBundle();
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [groups, catalog, channelRows] = await Promise.all([
          fetchRoutingGroups(API),
          fetchRoutingModelCatalog(API),
          fetchRoutingChannels(API),
        ]);
        if (cancelled) return;
        setGroupOpts(groups.map((g) => ({ key: g, text: g, value: g })));
        setFullModelCatalog(catalog);
        setModelOpts(catalog.map((m) => ({ key: m, text: m, value: m })));
        setRoutingChannels(channelRows);
        setChannelOpts(
          channelRows.map((ch) => ({
            key: String(ch.id),
            value: String(ch.id),
            text: `#${ch.id} ${ch.name || ''}`.trim(),
          }))
        );
      } catch (e) {
        showError(e.message);
      } finally {
        if (!cancelled) setLookupReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const idStr = String(tsCh).trim();
    if (!idStr) {
      setTsShowAllModels(false);
      setTsResolvedModels([]);
      return undefined;
    }
    let cancelled = false;
    const fromList = routingChannels.find((c) => String(c.id) === idStr);
    if (fromList && fromList.models.length > 0) {
      setTsResolvedModels(fromList.models);
      return undefined;
    }
    const num = parseInt(idStr, 10);
    if (!num) {
      setTsResolvedModels([]);
      return undefined;
    }
    API.get(`/api/channel/${num}`)
      .then((res) => {
        if (cancelled) return;
        if (res.data?.success && res.data.data) {
          setTsResolvedModels(parseChannelModelsField(res.data.data.models));
        } else setTsResolvedModels([]);
      })
      .catch(() => {
        if (!cancelled) setTsResolvedModels([]);
      });
    return () => {
      cancelled = true;
    };
  }, [tsCh, routingChannels]);

  const chartModelOpts = useMemo(() => {
    const toOpts = (ids) =>
      ids.map((m) => ({ key: m, text: m, value: m }));
    const full = toOpts(fullModelCatalog);
    const idStr = String(tsCh).trim();
    if (!idStr || tsShowAllModels || tsResolvedModels.length === 0) {
      return full;
    }
    return toOpts(tsResolvedModels);
  }, [fullModelCatalog, tsCh, tsShowAllModels, tsResolvedModels]);

  const addGroupOpt = (raw) => {
    const v = String(raw ?? '').trim();
    if (!v) return;
    setGroupOpts((prev) =>
      prev.some((o) => o.value === v)
        ? prev
        : [...prev, { key: v, text: v, value: v }]
    );
  };

  const addModelOpt = (raw) => {
    const v = String(raw ?? '').trim();
    if (!v) return;
    setFullModelCatalog((prev) => {
      if (prev.some((x) => x === v)) return prev;
      return [...prev, v].sort((a, b) => a.localeCompare(b));
    });
    setModelOpts((prev) =>
      prev.some((o) => o.value === v)
        ? prev
        : [...prev, { key: v, text: v, value: v }]
    );
  };

  const addChannelOpt = (raw) => {
    const v = String(raw ?? '').trim();
    if (!v) return;
    setChannelOpts((prev) =>
      prev.some((o) => o.value === v)
        ? prev
        : [...prev, { key: v, value: v, text: v }]
    );
  };

  const saveOption = async (key) => {
    if (!isRoot()) {
      showError(t('routing.need_root_save'));
      return;
    }
    try {
      let raw;
      if (key === 'RelayProtocolBridgeEnabled') {
        raw = policies[key] === 'true' ? 'true' : 'false';
      } else {
        raw = String(policies[key] ?? '').trim() || '{}';
      }
      const res = await API.put('/api/option', {
        key,
        value: raw,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'save failed');
        return;
      }
      showSuccess(t('routing.saved'));
    } catch (e) {
      showError(e.message);
    }
  };

  const loadPreview = async () => {
    try {
      const res = await API.get(
        `/api/routing/channel-preview?group=${encodeURIComponent(
          previewGroup
        )}&model=${encodeURIComponent(previewModel)}`
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'preview failed');
        return;
      }
      setPreviewMeta(res.data.data);
    } catch (e) {
      showError(e.message);
    }
  };

  const loadMetrics = async () => {
    try {
      const res = await API.get(
        `/api/routing/metrics-day?day=${encodeURIComponent(metricsDay)}`
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'metrics failed');
        return;
      }
      setMetrics(res.data.data || []);
    } catch (e) {
      showError(e.message);
    }
  };

  const loadFuse = async () => {
    try {
      const res = await API.get('/api/routing/fuse-events?limit=80');
      if (!res.data?.success) {
        showError(res.data?.message || 'fuse failed');
        return;
      }
      setFuse(res.data.data || []);
    } catch (e) {
      showError(e.message);
    }
  };

  const loadTs = async () => {
    try {
      const res = await API.get(
        `/api/routing/timeseries?channel_id=${encodeURIComponent(
          tsCh
        )}&model=${encodeURIComponent(tsModel)}&hours=${encodeURIComponent(
          String(tsHours)
        )}`
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'timeseries failed');
        return;
      }
      const rows = (res.data.data || []).map((r) => ({
        ...r,
        time: moment.unix(r.minute_unix).utc().format('MM-DD HH:mm'),
      }));
      setTsData(rows);
    } catch (e) {
      showError(e.message);
    }
  };

  const validateAlias = async () => {
    try {
      const res = await API.post('/api/routing/validate-alias-policy', {
        model_alias_policy_json: policies.ModelAliasPolicy ?? '{}',
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'invalid');
        return;
      }
      showSuccess(t('routing.alias_ok'));
    } catch (e) {
      showError(e.message);
    }
  };

  const postManualWeight = async () => {
    try {
      const res = await API.post('/api/routing/manual-weight', {
        channel_id: parseInt(mwCh, 10),
        multiplier: parseFloat(mwMul),
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'failed');
        return;
      }
      showSuccess(t('routing.saved'));
    } catch (e) {
      showError(e.message);
    }
  };

  const resetAuto = async () => {
    try {
      const res = await API.post('/api/routing/reset-auto-weight', {
        channel_id: parseInt(rwCh, 10),
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'failed');
        return;
      }
      showSuccess(t('routing.saved'));
    } catch (e) {
      showError(e.message);
    }
  };

  const policyPane = (
    <Tab.Pane attached={false}>
      <Message info>{t('routing.policy_hint')}</Message>
      <div className='routing-policy-card' style={{ marginBottom: '1rem' }}>
        <Form>
          <Header as='h4' className='routing-policy-card__title'>
            {t('routing.protocol_bridge_title')}
          </Header>
          <p style={{ marginBottom: '0.75rem', opacity: 0.85 }}>
            {t('routing.protocol_bridge_hint')}
          </p>
          <Form.Checkbox
            toggle
            label={t('routing.protocol_bridge_label')}
            checked={policies.RelayProtocolBridgeEnabled === 'true'}
            onChange={(e, { checked }) =>
              setPolicies((p) => ({
                ...p,
                RelayProtocolBridgeEnabled: checked ? 'true' : 'false',
              }))
            }
          />
        </Form>
        <div className='routing-policy-save-footer' style={{ marginTop: '0.75rem' }}>
          {!isRoot() ? (
            <span className='routing-policy-save-footer__hint'>
              {t('routing.need_root_save')}
            </span>
          ) : null}
          <Button
            primary
            type='button'
            size='small'
            icon
            labelPosition='left'
            disabled={!isRoot()}
            onClick={() => void saveOption('RelayProtocolBridgeEnabled')}
          >
            <Icon name='save' />
            {t('routing.save')}
          </Button>
        </div>
      </div>
      {POLICY_KEYS_BASIC.map((k) => (
        <div key={k} className='routing-policy-card'>
          <Form>
            <Header as='h4' className='routing-policy-card__title'>
              {k}
            </Header>
          {k === 'RelayRetryPolicy' && (
            <Message size='small' info style={{ marginBottom: '0.75rem' }}>
              {t('routing.policy_example_note_relay')}
            </Message>
          )}
          <PolicyJsonExample
            summary={t('routing.policy_example_toggle')}
            jsonText={ROUTING_POLICY_JSON_SAMPLES[k]}
          />
          {k === 'RoutingPolicy' && (
            <RoutingPolicyForm
              value={policies[k] ?? '{}'}
              onChange={(v) => setPolicies((p) => ({ ...p, [k]: v }))}
              disabled={false}
            />
          )}
          {k === 'RelayRetryPolicy' && (
            <RelayRetryPolicyForm
              value={policies[k] ?? '{}'}
              onChange={(v) => setPolicies((p) => ({ ...p, [k]: v }))}
              disabled={false}
            />
          )}
          {k === 'ModelRateLimitPolicy' && (
            <ModelRateLimitPolicyForm
              value={policies[k] ?? '{}'}
              onChange={(v) => setPolicies((p) => ({ ...p, [k]: v }))}
              disabled={false}
              modelOpts={modelOpts}
              groupOpts={groupOpts}
              onAddModel={(v) => addModelOpt(v)}
              onAddGroup={(v) => addGroupOpt(v)}
              lookupReady={lookupReady}
            />
          )}
          </Form>
          <div className='routing-policy-save-footer'>
            {!isRoot() ? (
              <span className='routing-policy-save-footer__hint'>
                {t('routing.need_root_save')}
              </span>
            ) : null}
            <Button
              primary
              type='button'
              size='small'
              icon
              labelPosition='left'
              disabled={!isRoot()}
              onClick={() => void saveOption(k)}
            >
              <Icon name='save' />
              {t('routing.save')}
            </Button>
          </div>
        </div>
      ))}
      <div className='routing-policy-card'>
        <Form>
          <Header as='h4' className='routing-policy-card__title'>
            ModelAliasPolicy
          </Header>
        <Message size='small' info>
          {t('routing.alias_hint')}
        </Message>
        <PolicyJsonExample
          summary={t('routing.policy_example_toggle')}
          jsonText={ROUTING_POLICY_JSON_SAMPLES.ModelAliasPolicy}
        />
        <ModelAliasPolicyForm
          value={policies.ModelAliasPolicy ?? '{}'}
          onChange={(v) =>
            setPolicies((p) => ({ ...p, ModelAliasPolicy: v }))
          }
          disabled={false}
          modelOpts={modelOpts}
          groupOpts={groupOpts}
          onAddModel={(v) => addModelOpt(v)}
          onAddGroup={(v) => addGroupOpt(v)}
          lookupReady={lookupReady}
        />
        </Form>
        <div className='routing-policy-save-footer routing-policy-save-footer--split'>
          {!isRoot() ? (
            <span className='routing-policy-save-footer__hint'>
              {t('routing.need_root_save')}
            </span>
          ) : null}
          <div className='routing-policy-save-footer__buttons'>
            <Button
              primary
              type='button'
              size='small'
              icon
              labelPosition='left'
              disabled={!isRoot()}
              onClick={() => void saveOption('ModelAliasPolicy')}
            >
              <Icon name='save' />
              {t('routing.save')}
            </Button>
            <Button
              type='button'
              basic
              size='small'
              icon
              labelPosition='left'
              onClick={() => void validateAlias()}
            >
              <Icon name='check circle outline' />
              {t('routing.validate_alias')}
            </Button>
          </div>
        </div>
      </div>
    </Tab.Pane>
  );

  const observePane = (
    <Tab.Pane attached={false}>
      <Header as='h4'>{t('routing.preview_title')}</Header>
      <Form>
        <Form.Group widths='equal'>
          <Form.Dropdown
            label={t('routing.group')}
            placeholder={t('routing.combo_search_placeholder')}
            fluid
            search
            selection
            clearable
            allowAdditions
            additionLabel={t('routing.combo_add_custom')}
            disabled={!lookupReady}
            options={groupOpts}
            value={previewGroup}
            onAddItem={(e, { value }) => addGroupOpt(value)}
            onChange={(e, { value }) => setPreviewGroup(value ?? '')}
          />
          <Form.Dropdown
            label={t('routing.model')}
            placeholder={t('routing.combo_search_placeholder')}
            fluid
            search
            selection
            clearable
            allowAdditions
            additionLabel={t('routing.combo_add_custom')}
            disabled={!lookupReady}
            options={modelOpts}
            value={previewModel}
            onAddItem={(e, { value }) => addModelOpt(value)}
            onChange={(e, { value }) => setPreviewModel(value ?? '')}
          />
        </Form.Group>
        <Button type='button' onClick={() => void loadPreview()}>
          {t('routing.refresh')}
        </Button>
      </Form>
      {previewMeta && (
        <>
          <Message>
            {t('routing.candidate_count')}: {previewMeta.candidate_count}
          </Message>
          <Table compact celled size='small'>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>ID</Table.HeaderCell>
                <Table.HeaderCell>{t('routing.col_name')}</Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label={t('routing.col_provider')}
                    hint={t('routing.preview_hint_direction')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label='W'
                    hint={t('routing.preview_hint_w')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label='×m'
                    hint={t('routing.preview_hint_mul_m')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label='×a'
                    hint={t('routing.preview_hint_mul_a')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label={t('routing.adaptive')}
                    hint={t('routing.preview_hint_adaptive')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label='Eff'
                    hint={t('routing.preview_hint_eff')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label='Pri'
                    hint={t('routing.preview_hint_pri')}
                  />
                </Table.HeaderCell>
                <Table.HeaderCell>
                  <PreviewColumnHeader
                    label='Fuse'
                    hint={t('routing.preview_hint_fuse')}
                  />
                </Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {(previewMeta.channels || []).map((r) => (
                <Table.Row key={r.id}>
                  <Table.Cell>{r.id}</Table.Cell>
                  <Table.Cell>{r.name}</Table.Cell>
                  <Table.Cell>{r.provider}</Table.Cell>
                  <Table.Cell>{r.base_weight}</Table.Cell>
                  <Table.Cell>{r.manual_multiplier}</Table.Cell>
                  <Table.Cell>{r.auto_multiplier}</Table.Cell>
                  <Table.Cell>{r.adaptive_enabled ? 'Y' : 'N'}</Table.Cell>
                  <Table.Cell>{r.effective_weight}</Table.Cell>
                  <Table.Cell>{r.priority}</Table.Cell>
                  <Table.Cell>{r.circuit_open ? 'OPEN' : '-'}</Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        </>
      )}

      <Header as='h4' dividing>
        {t('routing.manual_weight')}
      </Header>
      <Form>
        <Form.Group widths='equal'>
          <Form.Dropdown
            label={t('routing.channel_id')}
            placeholder={t('routing.combo_channel_placeholder')}
            fluid
            search
            selection
            clearable
            allowAdditions
            additionLabel={t('routing.combo_add_custom')}
            disabled={!lookupReady}
            options={channelOpts}
            value={mwCh}
            onAddItem={(e, { value }) => addChannelOpt(value)}
            onChange={(e, { value }) => setMwCh(value ?? '')}
          />
          <Form.Input
            label={t('routing.multiplier')}
            value={mwMul}
            onChange={(e, { value }) => setMwMul(value)}
          />
        </Form.Group>
        <Button type='button' onClick={() => void postManualWeight()}>
          {t('routing.apply')}
        </Button>
      </Form>

      <Header as='h4' dividing>
        {t('routing.reset_auto')}
      </Header>
      <Form>
        <Form.Dropdown
          label={t('routing.channel_id')}
          placeholder={t('routing.combo_channel_placeholder')}
          fluid
          search
          selection
          clearable
          allowAdditions
          additionLabel={t('routing.combo_add_custom')}
          disabled={!lookupReady}
          options={channelOpts}
          value={rwCh}
          onAddItem={(e, { value }) => addChannelOpt(value)}
          onChange={(e, { value }) => setRwCh(value ?? '')}
        />
        <Button type='button' onClick={() => void resetAuto()}>
          {t('routing.reset_auto_btn')}
        </Button>
      </Form>
    </Tab.Pane>
  );

  const metricsPane = (
    <Tab.Pane attached={false}>
      <Form>
        <Form.Input
          label={t('routing.metrics_day')}
          value={metricsDay}
          onChange={(e, { value }) => setMetricsDay(value)}
        />
        <Button type='button' onClick={() => void loadMetrics()}>
          {t('routing.refresh')}
        </Button>
      </Form>
      <Table compact celled size='small'>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell>Key</Table.HeaderCell>
            <Table.HeaderCell>ok</Table.HeaderCell>
            <Table.HeaderCell>fail</Table.HeaderCell>
            <Table.HeaderCell>lat_n</Table.HeaderCell>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {metrics.map((m, idx) => (
            <Table.Row key={idx}>
              <Table.Cell>{m.redis_key}</Table.Cell>
              <Table.Cell>{m.ok}</Table.Cell>
              <Table.Cell>{m.fail}</Table.Cell>
              <Table.Cell>{m.lat_n}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table>
    </Tab.Pane>
  );

  const fusePane = (
    <Tab.Pane attached={false}>
      <Button type='button' onClick={() => void loadFuse()}>
        {t('routing.refresh')}
      </Button>
      <Table compact celled size='small'>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell>time</Table.HeaderCell>
            <Table.HeaderCell>ch</Table.HeaderCell>
            <Table.HeaderCell>state</Table.HeaderCell>
            <Table.HeaderCell>reason</Table.HeaderCell>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {fuse.map((ev, idx) => (
            <Table.Row key={idx}>
              <Table.Cell>
                {moment(ev.unix_ms).format('YYYY-MM-DD HH:mm:ss')}
              </Table.Cell>
              <Table.Cell>{ev.channel_id}</Table.Cell>
              <Table.Cell>{ev.state}</Table.Cell>
              <Table.Cell>{ev.reason}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table>
    </Tab.Pane>
  );

  const chartPane = (
    <Tab.Pane attached={false}>
      <Form>
        <Form.Group widths='equal'>
          <Form.Dropdown
            label={t('routing.channel_id')}
            placeholder={t('routing.combo_channel_placeholder')}
            fluid
            search
            selection
            clearable
            allowAdditions
            additionLabel={t('routing.combo_add_custom')}
            disabled={!lookupReady}
            options={channelOpts}
            value={tsCh}
            onAddItem={(e, { value }) => addChannelOpt(value)}
            onChange={(e, { value }) => setTsCh(value ?? '')}
          />
          <Form.Dropdown
            label={t('routing.model')}
            placeholder={t('routing.combo_search_placeholder')}
            fluid
            search
            selection
            clearable
            allowAdditions
            additionLabel={t('routing.combo_add_custom')}
            disabled={!lookupReady}
            options={chartModelOpts}
            value={tsModel}
            onAddItem={(e, { value }) => addModelOpt(value)}
            onChange={(e, { value }) => setTsModel(value ?? '')}
          />
          <Form.Input
            label={t('routing.hours')}
            type='number'
            value={tsHours}
            onChange={(e, { value }) => setTsHours(parseInt(value, 10) || 24)}
          />
        </Form.Group>
        <Button type='button' onClick={() => void loadTs()}>
          {t('routing.refresh')}
        </Button>
      </Form>
      <Form.Checkbox
        style={{ marginBottom: '0.75rem' }}
        label={t('routing.chart_show_all_models')}
        checked={tsShowAllModels}
        disabled={!String(tsCh).trim()}
        onChange={(e, { checked }) => setTsShowAllModels(!!checked)}
      />
      <Message size='small' info style={{ marginBottom: '1rem' }}>
        {t('routing.chart_models_hint')}
      </Message>
      <ResponsiveContainer width='100%' height={320}>
        <LineChart data={tsData}>
          <CartesianGrid strokeDasharray='3 3' />
          <XAxis dataKey='time' />
          <YAxis yAxisId='left' />
          <YAxis yAxisId='right' orientation='right' />
          <Tooltip />
          <Line
            yAxisId='left'
            type='monotone'
            dataKey='avg_latency_ms'
            stroke='#8884d8'
            dot={false}
            name='avg_latency_ms'
          />
          <Line
            yAxisId='right'
            type='monotone'
            dataKey='err_ratio'
            stroke='#82ca9d'
            dot={false}
            name='err_ratio'
          />
        </LineChart>
      </ResponsiveContainer>
    </Tab.Pane>
  );

  const panes = [
    { menuItem: t('routing.tab_policies'), render: () => policyPane },
    { menuItem: t('routing.tab_observe'), render: () => observePane },
    { menuItem: t('routing.tab_metrics'), render: () => metricsPane },
    { menuItem: t('routing.tab_fuse'), render: () => fusePane },
    { menuItem: t('routing.tab_chart'), render: () => chartPane },
  ];

  return (
    <div className='dashboard-container routing-page'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>{t('routing.title')}</Card.Header>
          <Tab
            menu={{ secondary: true, pointing: true }}
            panes={panes}
          />
        </Card.Content>
      </Card>
    </div>
  );
}
