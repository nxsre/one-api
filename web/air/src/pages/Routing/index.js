import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  AutoComplete,
  Banner,
  Button,
  Checkbox,
  Input,
  Layout,
  Space,
  TabPane,
  Table,
  Tabs,
  TextArea,
  Typography
} from '@douyinfe/semi-ui';
import { API, isAdmin, isRoot, showError, showSuccess } from '../../helpers';
import { ROUTING_POLICY_JSON_SAMPLES } from './policyExamples';
import {
  fetchRoutingChannels,
  fetchRoutingGroups,
  fetchRoutingModelCatalog,
  parseChannelModelsField,
} from './routingLookupLoaders';

function PolicyJsonExample({ summary, jsonText }) {
  return (
    <details style={{ marginBottom: '0.85rem' }}>
      <summary style={{ cursor: 'pointer', userSelect: 'none', fontWeight: 600 }}>
        {summary}
      </summary>
      <pre
        style={{
          marginTop: 8,
          padding: 12,
          background: 'var(--semi-color-fill-0)',
          borderRadius: 6,
          overflow: 'auto',
          fontSize: 12,
          lineHeight: 1.45,
          whiteSpace: 'pre',
          fontFamily: 'JetBrains Mono, Consolas, ui-monospace, monospace'
        }}
      >
        {jsonText}
      </pre>
    </details>
  );
}

const POLICY_KEYS = ['RoutingPolicy', 'RelayRetryPolicy', 'ModelAliasPolicy', 'ModelRateLimitPolicy'];

function utcYmd() {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getUTCFullYear()}${pad(d.getUTCMonth() + 1)}${pad(d.getUTCDate())}`;
}

function formatUtcMinute(ts) {
  const d = new Date(ts * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  return `${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

function formatMs(ms) {
  try {
    return new Date(ms).toISOString().replace('T', ' ').slice(0, 19);
  } catch {
    return String(ms);
  }
}

export default function Routing() {
  const navigate = useNavigate();
  const [policies, setPolicies] = useState({});
  const [previewGroup, setPreviewGroup] = useState('default');
  const [previewModel, setPreviewModel] = useState('');
  const [previewMeta, setPreviewMeta] = useState(null);
  const [metricsDay, setMetricsDay] = useState(utcYmd());
  const [metrics, setMetrics] = useState([]);
  const [fuse, setFuse] = useState([]);
  const [tsCh, setTsCh] = useState('');
  const [tsModel, setTsModel] = useState('');
  const [tsHours, setTsHours] = useState(24);
  const [tsRows, setTsRows] = useState([]);
  const [aliasRaw, setAliasRaw] = useState('{}');
  const [mwCh, setMwCh] = useState('');
  const [mwMul, setMwMul] = useState('1');
  const [rwCh, setRwCh] = useState('');
  const [groups, setGroups] = useState([]);
  const [fullCatalog, setFullCatalog] = useState([]);
  const [channels, setChannels] = useState([]);
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
        showError(res.data?.message || '加载失败');
        return;
      }
      const data = res.data.data || {};
      setPolicies({
        ...data,
        RelayProtocolBridgeEnabled:
          data.RelayProtocolBridgeEnabled === 'true' ? 'true' : 'false',
      });
      const alias = data.ModelAliasPolicy;
      if (alias) setAliasRaw(alias);
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
        const [g, catalog, chRows] = await Promise.all([
          fetchRoutingGroups(API),
          fetchRoutingModelCatalog(API),
          fetchRoutingChannels(API),
        ]);
        if (cancelled) return;
        setGroups(g);
        setFullCatalog(catalog);
        setChannels(chRows);
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
    const fromList = channels.find((c) => String(c.id) === idStr);
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
  }, [tsCh, channels]);

  const groupAutoData = useMemo(
    () => groups.map((g) => ({ value: g, label: g })),
    [groups]
  );
  const modelAutoData = useMemo(
    () => fullCatalog.map((m) => ({ value: m, label: m })),
    [fullCatalog]
  );
  const chartModelAutoData = useMemo(() => {
    const idStr = String(tsCh).trim();
    const src =
      !idStr || tsShowAllModels || tsResolvedModels.length === 0
        ? fullCatalog
        : tsResolvedModels;
    return src.map((m) => ({ value: m, label: m }));
  }, [fullCatalog, tsCh, tsShowAllModels, tsResolvedModels]);
  const channelAutoData = useMemo(
    () =>
      channels.map((c) => ({
        value: String(c.id),
        label: `#${c.id} ${c.name || ''}`.trim(),
      })),
    [channels]
  );

  const saveOption = async (key) => {
    if (!isRoot()) {
      showError('仅 Root 可保存运营选项');
      return;
    }
    try {
      let value;
      if (key === 'RelayProtocolBridgeEnabled') {
        value = policies[key] === 'true' ? 'true' : 'false';
      } else {
        value = policies[key] ?? '{}';
      }
      const res = await API.put('/api/option', {
        key,
        value
      });
      if (!res.data?.success) {
        showError(res.data?.message || '保存失败');
        return;
      }
      showSuccess('已保存');
    } catch (e) {
      showError(e.message);
    }
  };

  const loadPreview = async () => {
    try {
      const res = await API.get(
        `/api/routing/channel-preview?group=${encodeURIComponent(previewGroup)}&model=${encodeURIComponent(previewModel)}`
      );
      if (!res.data?.success) {
        showError(res.data?.message || '预览失败');
        return;
      }
      setPreviewMeta(res.data.data);
    } catch (e) {
      showError(e.message);
    }
  };

  const loadMetrics = async () => {
    try {
      const res = await API.get(`/api/routing/metrics-day?day=${encodeURIComponent(metricsDay)}`);
      if (!res.data?.success) {
        showError(res.data?.message || '加载指标失败');
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
        showError(res.data?.message || '加载熔断记录失败');
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
        `/api/routing/timeseries?channel_id=${encodeURIComponent(tsCh)}&model=${encodeURIComponent(
          tsModel
        )}&hours=${encodeURIComponent(String(tsHours))}`
      );
      if (!res.data?.success) {
        showError(res.data?.message || '加载时序失败');
        return;
      }
      const rows = (res.data.data || []).map((r) => ({
        ...r,
        time: formatUtcMinute(r.minute_unix)
      }));
      setTsRows(rows);
    } catch (e) {
      showError(e.message);
    }
  };

  const validateAlias = async () => {
    try {
      const res = await API.post('/api/routing/validate-alias-policy', {
        model_alias_policy_json: aliasRaw
      });
      if (!res.data?.success) {
        showError(res.data?.message || '校验失败');
        return;
      }
      showSuccess('别名策略格式合法');
    } catch (e) {
      showError(e.message);
    }
  };

  const postManualWeight = async () => {
    try {
      const res = await API.post('/api/routing/manual-weight', {
        channel_id: parseInt(mwCh, 10),
        multiplier: parseFloat(mwMul)
      });
      if (!res.data?.success) {
        showError(res.data?.message || '操作失败');
        return;
      }
      showSuccess('已更新手工倍率');
    } catch (e) {
      showError(e.message);
    }
  };

  const resetAuto = async () => {
    try {
      const res = await API.post('/api/routing/reset-auto-weight', {
        channel_id: parseInt(rwCh, 10)
      });
      if (!res.data?.success) {
        showError(res.data?.message || '操作失败');
        return;
      }
      showSuccess('已重置该渠道自适应倍率');
    } catch (e) {
      showError(e.message);
    }
  };

  const previewColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: 'Provider', dataIndex: 'provider', key: 'provider' },
    { title: 'W', dataIndex: 'base_weight', key: 'bw' },
    { title: '×m', dataIndex: 'manual_multiplier', key: 'mm' },
    { title: '×a', dataIndex: 'auto_multiplier', key: 'am' },
    {
      title: '自适应',
      dataIndex: 'adaptive_enabled',
      key: 'ad',
      render: (v) => (v ? 'Y' : 'N')
    },
    { title: '有效权重', dataIndex: 'effective_weight', key: 'ew' },
    { title: '优先级', dataIndex: 'priority', key: 'pri' },
    {
      title: '熔断',
      dataIndex: 'circuit_open',
      key: 'fuse',
      render: (v) => (v ? 'OPEN' : '-')
    }
  ];

  const metricsColumns = [
    { title: 'Redis Key', dataIndex: 'redis_key', key: 'k' },
    { title: 'ok', dataIndex: 'ok', key: 'ok' },
    { title: 'fail', dataIndex: 'fail', key: 'fail' },
    { title: 'lat_n', dataIndex: 'lat_n', key: 'lat' }
  ];

  const fuseColumns = [
    {
      title: '时间',
      dataIndex: 'unix_ms',
      key: 't',
      render: (ms) => formatMs(ms)
    },
    { title: '渠道', dataIndex: 'channel_id', key: 'ch' },
    { title: '状态', dataIndex: 'state', key: 'st' },
    { title: '原因', dataIndex: 'reason', key: 'rs' }
  ];

  const tsColumns = [
    { title: '时间(UTC分桶)', dataIndex: 'time', key: 'tm' },
    { title: '平均延迟 ms', dataIndex: 'avg_latency_ms', key: 'lat' },
    { title: '错误率', dataIndex: 'err_ratio', key: 'err' }
  ];

  const policyTab = (
    <div style={{ paddingTop: 12 }}>
      <Banner type="info" description="策略 JSON 保存在运营选项；单项保存需 Root。可先校验别名再写入 ModelAliasPolicy。" />
      <Space vertical align="start" style={{ width: '100%', marginTop: 16 }} spacing="loose">
        <div style={{ width: '100%' }}>
          <Typography.Title heading={6}>跨协议转发</Typography.Title>
          <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            开启后 Anthropic / Gemini 原生 API 可选用非对应协议渠道，经 OpenAI 语义转发（需模型映射）。关闭则仅允许协议匹配。
          </Typography.Text>
          <Checkbox
            checked={policies.RelayProtocolBridgeEnabled === 'true'}
            onChange={(e) =>
              setPolicies((p) => ({
                ...p,
                RelayProtocolBridgeEnabled: e.target.checked ? 'true' : 'false',
              }))
            }
          >
            允许跨协议转发（RelayProtocolBridgeEnabled）
          </Checkbox>
          <Button
            theme="light"
            disabled={!isRoot()}
            onClick={() => void saveOption('RelayProtocolBridgeEnabled')}
            style={{ marginTop: 8 }}
          >
            保存此项
          </Button>
        </div>
        {POLICY_KEYS.map((k) => (
          <div key={k} style={{ width: '100%' }}>
            <Typography.Title heading={6}>{k}</Typography.Title>
            {k === 'RelayRetryPolicy' && (
              <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
                省略 max_retries_override 时沿用系统「重试次数」。熔断阈值 circuit_fail_threshold（在 RoutingPolicy 中）为 0 表示关闭 Redis 熔断。
              </Typography.Text>
            )}
            <PolicyJsonExample summary="查看字段示例 JSON（仅供参考，可按业务删减）" jsonText={ROUTING_POLICY_JSON_SAMPLES[k]} />
            <TextArea
              rows={8}
              style={{ fontFamily: 'JetBrains Mono, Consolas, monospace', fontSize: 13 }}
              value={policies[k] ?? '{}'}
              onChange={(v) => setPolicies((p) => ({ ...p, [k]: v }))}
            />
            <Button theme="light" disabled={!isRoot()} onClick={() => void saveOption(k)} style={{ marginTop: 8 }}>
              保存此项
            </Button>
          </div>
        ))}
        <Typography.Title heading={6}>别名策略校验（独立粘贴）</Typography.Title>
        <PolicyJsonExample summary="查看字段示例 JSON（仅供参考，可按业务删减）" jsonText={ROUTING_POLICY_JSON_SAMPLES.ModelAliasPolicy} />
        <TextArea
          rows={8}
          style={{ fontFamily: 'JetBrains Mono, Consolas, monospace', fontSize: 13 }}
          value={aliasRaw}
          onChange={setAliasRaw}
        />
        <Button onClick={() => void validateAlias()}>校验 JSON</Button>
      </Space>
    </div>
  );

  const observeTab = (
    <div style={{ paddingTop: 12 }}>
      <Typography.Title heading={6}>路由预览</Typography.Title>
      <Space style={{ marginTop: 8, marginBottom: 16 }} wrap>
        <AutoComplete
          style={{ width: 280 }}
          disabled={!lookupReady}
          data={groupAutoData}
          value={previewGroup}
          onChange={setPreviewGroup}
          placeholder="分组：筛选或输入"
          showClear
        />
        <AutoComplete
          style={{ width: 280 }}
          disabled={!lookupReady}
          data={modelAutoData}
          value={previewModel}
          onChange={setPreviewModel}
          placeholder="模型：筛选或输入（可空）"
          showClear
        />
        <Button theme="solid" onClick={() => void loadPreview()}>
          刷新预览
        </Button>
      </Space>
      {previewMeta && (
        <>
          <Banner
            type="info"
            description={`候选渠道数：${previewMeta.candidate_count}`}
            style={{ marginBottom: 12 }}
          />
          <Table columns={previewColumns} dataSource={previewMeta.channels || []} pagination={false} />
        </>
      )}
      <Typography.Title heading={6} style={{ marginTop: 24 }}>
        手工倍率
      </Typography.Title>
      <Space style={{ marginTop: 8 }} wrap>
        <AutoComplete
          style={{ width: 320 }}
          disabled={!lookupReady}
          data={channelAutoData}
          value={mwCh}
          onChange={setMwCh}
          placeholder="渠道：筛选名称/ID 或手动输入"
          showClear
        />
        <Input value={mwMul} onChange={setMwMul} placeholder="倍率" style={{ width: 120 }} />
        <Button theme="solid" onClick={() => void postManualWeight()}>
          应用
        </Button>
      </Space>
      <Typography.Title heading={6} style={{ marginTop: 24 }}>
        重置自适应倍率
      </Typography.Title>
      <Space style={{ marginTop: 8 }} wrap>
        <AutoComplete
          style={{ width: 320 }}
          disabled={!lookupReady}
          data={channelAutoData}
          value={rwCh}
          onChange={setRwCh}
          placeholder="渠道：筛选名称/ID 或手动输入"
          showClear
        />
        <Button theme="solid" type="warning" onClick={() => void resetAuto()}>
          重置自适应
        </Button>
      </Space>
    </div>
  );

  const metricsTab = (
    <div style={{ paddingTop: 12 }}>
      <Space style={{ marginBottom: 16 }}>
        <Input value={metricsDay} onChange={setMetricsDay} placeholder="YYYYMMDD UTC" />
        <Button theme="solid" onClick={() => void loadMetrics()}>
          刷新
        </Button>
      </Space>
      <Table columns={metricsColumns} dataSource={metrics} pagination={false} />
    </div>
  );

  const fuseTab = (
    <div style={{ paddingTop: 12 }}>
      <Button theme="solid" onClick={() => void loadFuse()} style={{ marginBottom: 16 }}>
        刷新熔断记录
      </Button>
      <Table columns={fuseColumns} dataSource={fuse} pagination={false} />
    </div>
  );

  const chartTab = (
    <div style={{ paddingTop: 12 }}>
      <Space style={{ marginBottom: 16 }} wrap>
        <AutoComplete
          style={{ width: 320 }}
          disabled={!lookupReady}
          data={channelAutoData}
          value={tsCh}
          onChange={setTsCh}
          placeholder="渠道：筛选名称/ID 或手动输入"
          showClear
        />
        <AutoComplete
          style={{ width: 280 }}
          disabled={!lookupReady}
          data={chartModelAutoData}
          value={tsModel}
          onChange={setTsModel}
          placeholder="模型：筛选或输入"
          showClear
        />
        <Space align='center'>
          <Checkbox
            checked={tsShowAllModels}
            disabled={!String(tsCh).trim()}
            onChange={(checked) => setTsShowAllModels(!!checked)}
          />
          <Typography.Text type='secondary'>显示全部模型（合并目录）</Typography.Text>
        </Space>
        <Input
          value={String(tsHours)}
          onChange={(v) => setTsHours(parseInt(v, 10) || 24)}
          placeholder="小时数"
          style={{ width: 120 }}
        />
        <Button theme="solid" onClick={() => void loadTs()}>
          加载时序
        </Button>
      </Space>
      <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
        Air 主题以表格展示分钟桶；亦可通过 default / berry 主题查看曲线图。
      </Typography.Text>
      <Table columns={tsColumns} dataSource={tsRows} pagination={false} />
    </div>
  );

  return (
    <Layout>
      <Layout.Content style={{ padding: 24 }}>
        <Typography.Title heading={4}>智能路由运维</Typography.Title>
        <Tabs type="line" defaultActiveKey="p">
          <TabPane tab="策略" itemKey="p">
            {policyTab}
          </TabPane>
          <TabPane tab="观测" itemKey="o">
            {observeTab}
          </TabPane>
          <TabPane tab="指标" itemKey="m">
            {metricsTab}
          </TabPane>
          <TabPane tab="熔断" itemKey="f">
            {fuseTab}
          </TabPane>
          <TabPane tab="时序" itemKey="c">
            {chartTab}
          </TabPane>
        </Tabs>
      </Layout.Content>
    </Layout>
  );
}
