import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import Chart from 'react-apexcharts';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  Paper,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Typography
} from '@mui/material';
import MainCard from 'ui-component/cards/MainCard';
import { gridSpacing } from 'store/constant';
import config from 'config';
import { API } from 'utils/api';
import { isAdmin, isRoot, showError, showSuccess } from 'utils/common';
import { ROUTING_POLICY_JSON_SAMPLES } from './policyExamples';
import {
  fetchRoutingChannels,
  fetchRoutingGroups,
  fetchRoutingModelCatalog,
  parseChannelModelsField,
} from './routingLookupLoaders';

const comboFieldSx = { minWidth: 260, flex: '1 1 260px' };

function PolicyJsonExample({ summary, jsonText }) {
  return (
    <details style={{ marginBottom: '0.85rem' }}>
      <summary style={{ cursor: 'pointer', userSelect: 'none', fontWeight: 600 }}>
        {summary}
      </summary>
      <Box
        component="pre"
        sx={{
          mt: 1,
          p: 1.5,
          borderRadius: 1,
          bgcolor: 'action.hover',
          overflow: 'auto',
          fontSize: 12,
          lineHeight: 1.45,
          whiteSpace: 'pre',
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace'
        }}
      >
        {jsonText}
      </Box>
    </details>
  );
}

dayjs.extend(utc);

const POLICY_KEYS = ['RoutingPolicy', 'RelayRetryPolicy', 'ModelAliasPolicy', 'ModelRateLimitPolicy'];

export default function Routing() {
  const navigate = useNavigate();
  const [tab, setTab] = useState(0);
  const [policies, setPolicies] = useState({});
  const [previewGroup, setPreviewGroup] = useState('default');
  const [previewModel, setPreviewModel] = useState('');
  const [previewMeta, setPreviewMeta] = useState(null);
  const [metricsDay, setMetricsDay] = useState(dayjs.utc().format('YYYYMMDD'));
  const [metrics, setMetrics] = useState([]);
  const [fuse, setFuse] = useState([]);
  const [tsCh, setTsCh] = useState('');
  const [tsModel, setTsModel] = useState('');
  const [tsHours, setTsHours] = useState(24);
  const [tsData, setTsData] = useState([]);
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
      navigate(config.defaultPath);
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

  const chartModelOptions = useMemo(() => {
    const idStr = String(tsCh).trim();
    if (!idStr || tsShowAllModels || tsResolvedModels.length === 0) {
      return fullCatalog;
    }
    return tsResolvedModels;
  }, [fullCatalog, tsCh, tsShowAllModels, tsResolvedModels]);

  const saveOption = async (key) => {
    if (!isRoot()) {
      showError('仅 Root 可通过运营选项保存策略 JSON');
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
        time: dayjs.unix(r.minute_unix).utc().format('MM-DD HH:mm')
      }));
      setTsData(rows);
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

  const tsChart = useMemo(() => {
    if (!tsData.length) return null;
    const categories = tsData.map((d) => d.time);
    return {
      options: {
        chart: { id: 'routing-ts', toolbar: { show: false }, zoom: { enabled: false } },
        stroke: { curve: 'smooth', width: 2 },
        dataLabels: { enabled: false },
        xaxis: { categories },
        yaxis: [
          { title: { text: '平均延迟 (ms)' } },
          { opposite: true, title: { text: '错误率' }, min: 0, max: 1 }
        ],
        tooltip: { shared: true },
        legend: { position: 'top' },
        colors: ['#5c6bc0', '#26a69a']
      },
      series: [
        {
          name: 'avg_latency_ms',
          data: tsData.map((d) => d.avg_latency_ms ?? 0),
          yAxisIndex: 0
        },
        {
          name: 'err_ratio',
          data: tsData.map((d) => d.err_ratio ?? 0),
          yAxisIndex: 1
        }
      ]
    };
  }, [tsData]);

  const policyPane = (
    <Box sx={{ p: 2 }}>
      <Alert severity="info" sx={{ mb: 2 }}>
        策略 JSON 持久化在运营选项中；保存单项需要 Root。可先校验别名策略再写入 ModelAliasPolicy。
      </Alert>
      <Box sx={{ mb: 3 }}>
        <Typography variant="h6">跨协议转发</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1, mt: 0.5 }}>
          开启后 Anthropic / Gemini 原生 API 可选用非对应协议渠道，经 OpenAI 语义转发（需模型映射）。关闭则仅允许协议匹配。
        </Typography>
        <FormControlLabel
          control={
            <Checkbox
              checked={policies.RelayProtocolBridgeEnabled === 'true'}
              onChange={(e) =>
                setPolicies((p) => ({
                  ...p,
                  RelayProtocolBridgeEnabled: e.target.checked ? 'true' : 'false',
                }))
              }
            />
          }
          label="允许跨协议转发（RelayProtocolBridgeEnabled）"
        />
        <Box sx={{ mt: 1 }}>
          <Button
            variant="outlined"
            size="small"
            disabled={!isRoot()}
            onClick={() => void saveOption('RelayProtocolBridgeEnabled')}
          >
            保存此项
          </Button>
        </Box>
      </Box>
      {POLICY_KEYS.map((k) => (
        <Box key={k} sx={{ mb: 3 }}>
          <Typography variant="h6">{k}</Typography>
          {k === 'RelayRetryPolicy' && (
            <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1, mt: 0.5 }}>
              省略 max_retries_override 时沿用系统「重试次数」。熔断阈值 circuit_fail_threshold 为 0 表示关闭 Redis 熔断。
            </Typography>
          )}
          <PolicyJsonExample summary="查看字段示例 JSON（仅供参考，可按业务删减）" jsonText={ROUTING_POLICY_JSON_SAMPLES[k]} />
          <TextField
            fullWidth
            multiline
            minRows={6}
            value={policies[k] ?? '{}'}
            onChange={(e) => setPolicies((p) => ({ ...p, [k]: e.target.value }))}
            sx={{ mt: 1, mb: 1, '& textarea': { fontFamily: 'monospace', fontSize: 13 } }}
          />
          <Button variant="outlined" size="small" disabled={!isRoot()} onClick={() => void saveOption(k)}>
            保存此项
          </Button>
        </Box>
      ))}
      <Typography variant="h6">别名策略校验（独立粘贴）</Typography>
      <PolicyJsonExample summary="查看字段示例 JSON（仅供参考，可按业务删减）" jsonText={ROUTING_POLICY_JSON_SAMPLES.ModelAliasPolicy} />
      <TextField
        fullWidth
        multiline
        minRows={6}
        value={aliasRaw}
        onChange={(e) => setAliasRaw(e.target.value)}
        sx={{ mt: 1, mb: 1, '& textarea': { fontFamily: 'monospace', fontSize: 13 } }}
      />
      <Button variant="contained" onClick={() => void validateAlias()}>
        校验 JSON
      </Button>
    </Box>
  );

  const observePane = (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6">路由预览</Typography>
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mt: 1, mb: 2, alignItems: 'flex-start' }}>
        <Autocomplete
          freeSolo
          options={groups}
          disabled={!lookupReady}
          sx={comboFieldSx}
          inputValue={previewGroup}
          onInputChange={(e, v) => setPreviewGroup(v ?? '')}
          onChange={(e, v) => {
            if (typeof v === 'string') setPreviewGroup(v);
          }}
          renderInput={(params) => (
            <TextField {...params} label="分组" placeholder="筛选或输入" size="small" />
          )}
        />
        <Autocomplete
          freeSolo
          options={fullCatalog}
          disabled={!lookupReady}
          sx={comboFieldSx}
          inputValue={previewModel}
          onInputChange={(e, v) => setPreviewModel(v ?? '')}
          onChange={(e, v) => {
            if (typeof v === 'string') setPreviewModel(v);
          }}
          renderInput={(params) => (
            <TextField {...params} label="模型" placeholder="筛选或输入" size="small" />
          )}
        />
        <Button variant="contained" onClick={() => void loadPreview()} sx={{ mt: 0.5 }}>
          刷新预览
        </Button>
      </Box>
      {previewMeta && (
        <>
          <Alert sx={{ mb: 2 }}>候选渠道数：{previewMeta.candidate_count}</Alert>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>ID</TableCell>
                <TableCell>名称</TableCell>
                <TableCell>Provider</TableCell>
                <TableCell>W</TableCell>
                <TableCell>×m</TableCell>
                <TableCell>×a</TableCell>
                <TableCell>自适应</TableCell>
                <TableCell>有效权重</TableCell>
                <TableCell>优先级</TableCell>
                <TableCell>熔断</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {(previewMeta.channels || []).map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.id}</TableCell>
                  <TableCell>{r.name}</TableCell>
                  <TableCell>{r.provider}</TableCell>
                  <TableCell>{r.base_weight}</TableCell>
                  <TableCell>{r.manual_multiplier}</TableCell>
                  <TableCell>{r.auto_multiplier}</TableCell>
                  <TableCell>{r.adaptive_enabled ? 'Y' : 'N'}</TableCell>
                  <TableCell>{r.effective_weight}</TableCell>
                  <TableCell>{r.priority}</TableCell>
                  <TableCell>{r.circuit_open ? 'OPEN' : '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </>
      )}
      <Typography variant="h6" sx={{ mt: 3 }}>
        手工倍率（Redis）
      </Typography>
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mt: 1, alignItems: 'flex-start' }}>
        <Autocomplete
          freeSolo
          options={channels}
          disabled={!lookupReady}
          sx={{ ...comboFieldSx, minWidth: 300 }}
          getOptionLabel={(opt) =>
            typeof opt === 'string' ? opt : `#${opt.id} ${opt.name || ''}`.trim()
          }
          value={
            mwCh === '' ? null : channels.find((c) => String(c.id) === mwCh) ?? mwCh
          }
          inputValue={mwCh}
          onInputChange={(e, v) => setMwCh(v ?? '')}
          onChange={(e, v) => {
            if (v == null || v === '') setMwCh('');
            else if (typeof v === 'string') setMwCh(v.trim());
            else setMwCh(String(v.id));
          }}
          renderInput={(params) => (
            <TextField {...params} label="渠道 ID" placeholder="筛选名称或输入 ID" size="small" />
          )}
        />
        <TextField label="倍率" size="small" value={mwMul} onChange={(e) => setMwMul(e.target.value)} sx={{ width: 140, mt: 0.5 }} />
        <Button variant="contained" onClick={() => void postManualWeight()} sx={{ mt: 0.5 }}>
          应用
        </Button>
      </Box>
      <Typography variant="h6" sx={{ mt: 3 }}>
        重置自适应倍率
      </Typography>
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mt: 1, alignItems: 'flex-start' }}>
        <Autocomplete
          freeSolo
          options={channels}
          disabled={!lookupReady}
          sx={{ ...comboFieldSx, minWidth: 300 }}
          getOptionLabel={(opt) =>
            typeof opt === 'string' ? opt : `#${opt.id} ${opt.name || ''}`.trim()
          }
          value={
            rwCh === '' ? null : channels.find((c) => String(c.id) === rwCh) ?? rwCh
          }
          inputValue={rwCh}
          onInputChange={(e, v) => setRwCh(v ?? '')}
          onChange={(e, v) => {
            if (v == null || v === '') setRwCh('');
            else if (typeof v === 'string') setRwCh(v.trim());
            else setRwCh(String(v.id));
          }}
          renderInput={(params) => (
            <TextField {...params} label="渠道 ID" placeholder="筛选名称或输入 ID" size="small" />
          )}
        />
        <Button variant="outlined" color="warning" onClick={() => void resetAuto()} sx={{ mt: 0.5 }}>
          重置自适应
        </Button>
      </Box>
    </Box>
  );

  const metricsPane = (
    <Box sx={{ p: 2 }}>
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mb: 2 }}>
        <TextField label="日期 YYYYMMDD (UTC)" value={metricsDay} onChange={(e) => setMetricsDay(e.target.value)} />
        <Button variant="contained" onClick={() => void loadMetrics()}>
          刷新
        </Button>
      </Box>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Redis Key</TableCell>
            <TableCell>ok</TableCell>
            <TableCell>fail</TableCell>
            <TableCell>lat_n</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {metrics.map((m, idx) => (
            <TableRow key={idx}>
              <TableCell>{m.redis_key}</TableCell>
              <TableCell>{m.ok}</TableCell>
              <TableCell>{m.fail}</TableCell>
              <TableCell>{m.lat_n}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );

  const fusePane = (
    <Box sx={{ p: 2 }}>
      <Button variant="contained" sx={{ mb: 2 }} onClick={() => void loadFuse()}>
        刷新熔断记录
      </Button>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>时间</TableCell>
            <TableCell>渠道</TableCell>
            <TableCell>状态</TableCell>
            <TableCell>原因</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {fuse.map((ev, idx) => (
            <TableRow key={idx}>
              <TableCell>{dayjs(ev.unix_ms).format('YYYY-MM-DD HH:mm:ss')}</TableCell>
              <TableCell>{ev.channel_id}</TableCell>
              <TableCell>{ev.state}</TableCell>
              <TableCell>{ev.reason}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );

  const chartPane = (
    <Box sx={{ p: 2 }}>
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mb: 2, alignItems: 'flex-start' }}>
        <Autocomplete
          freeSolo
          options={channels}
          disabled={!lookupReady}
          sx={{ ...comboFieldSx, minWidth: 300 }}
          getOptionLabel={(opt) =>
            typeof opt === 'string' ? opt : `#${opt.id} ${opt.name || ''}`.trim()
          }
          value={
            tsCh === '' ? null : channels.find((c) => String(c.id) === tsCh) ?? tsCh
          }
          inputValue={tsCh}
          onInputChange={(e, v) => setTsCh(v ?? '')}
          onChange={(e, v) => {
            if (v == null || v === '') setTsCh('');
            else if (typeof v === 'string') setTsCh(v.trim());
            else setTsCh(String(v.id));
          }}
          renderInput={(params) => (
            <TextField {...params} label="渠道 ID" placeholder="筛选名称或输入 ID" size="small" />
          )}
        />
        <Autocomplete
          freeSolo
          options={chartModelOptions}
          disabled={!lookupReady}
          sx={comboFieldSx}
          inputValue={tsModel}
          onInputChange={(e, v) => setTsModel(v ?? '')}
          onChange={(e, v) => {
            if (typeof v === 'string') setTsModel(v);
          }}
          renderInput={(params) => (
            <TextField {...params} label="模型" placeholder="筛选或输入" size="small" />
          )}
        />
        <FormControlLabel
          sx={{ mt: 0.5, alignSelf: 'flex-start' }}
          control={
            <Checkbox
              checked={tsShowAllModels}
              disabled={!String(tsCh).trim()}
              onChange={(e) => setTsShowAllModels(e.target.checked)}
            />
          }
          label="显示全部模型（合并目录）"
        />
        <TextField
          label="最近小时数"
          type="number"
          size="small"
          value={tsHours}
          sx={{ width: 140, mt: 0.5 }}
          onChange={(e) => setTsHours(parseInt(e.target.value, 10) || 24)}
        />
        <Button variant="contained" onClick={() => void loadTs()} sx={{ mt: 0.5 }}>
          加载曲线
        </Button>
      </Box>
      {tsChart ? (
        <Chart options={tsChart.options} series={tsChart.series} type="line" height={320} />
      ) : (
        <Typography color="textSecondary">加载数据后显示延迟与错误率曲线。</Typography>
      )}
    </Box>
  );

  const panels = [policyPane, observePane, metricsPane, fusePane, chartPane];

  return (
    <MainCard title="智能路由运维" sx={{ mb: gridSpacing }}>
      <Paper variant="outlined">
        <Tabs value={tab} onChange={(_, v) => setTab(v)} variant="scrollable" scrollButtons="auto">
          <Tab label="策略" />
          <Tab label="观测" />
          <Tab label="指标" />
          <Tab label="熔断" />
          <Tab label="曲线" />
        </Tabs>
        <Box>{panels[tab]}</Box>
      </Paper>
    </MainCard>
  );
}
