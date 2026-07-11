import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { API, getApiErrorMessage } from '@/api/client';
import { formatChartDate } from '@/lib/dashboard';
import { getModelIcon } from '@/lib/modelCatalog';
import UsageRangePicker from '@/components/UsageRangePicker';
import {
  aggregateUsageData,
  fetchBillingRows,
  formatQuotaM,
  formatTokenCount,
  getDefaultRange,
  resolveRangeFromPreset,
} from '@/lib/usageReports';
import { useUser } from '@/context/UserContext';
import './usage.css';

const TOOLTIP_STYLE = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius-sm)',
  boxShadow: 'var(--shadow)',
  fontSize: 12,
};

const X_AXIS = {
  dataKey: 'date',
  axisLine: false,
  tickLine: false,
  tick: { fontSize: 11, fill: 'var(--text3)' },
  tickFormatter: formatChartDate,
  minTickGap: 8,
};

const PROGRESS_CLASSES = ['indigo', 'cyan', 'green', 'amber'];

async function fetchDashboardDaily() {
  const res = await API.get('/api/user/dashboard');
  if (!res.data?.success) return [];
  const agg = aggregateUsageData(res.data.data || [], 'dashboard');
  return agg.daily;
}

export default function UsagePage() {
  const { user } = useUser();
  const defaultRange = useMemo(() => getDefaultRange(), []);
  const [preset, setPreset] = useState('month');
  const [startDate, setStartDate] = useState(defaultRange.start);
  const [endDate, setEndDate] = useState(defaultRange.end);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [agg, setAgg] = useState(null);
  const [dataSource, setDataSource] = useState('dashboard');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const { rows, source } = await fetchBillingRows(user, startDate, endDate);
      let result = aggregateUsageData(rows, source);

      try {
        const daily = await fetchDashboardDaily();
        if (daily.length) result = { ...result, daily };
      } catch {
        /* 趋势图可选 */
      }

      setAgg(result);
      setDataSource(source);
    } catch (e) {
      setError(getApiErrorMessage(e));
      setAgg(null);
    } finally {
      setLoading(false);
    }
  }, [user, startDate, endDate]);

  useEffect(() => {
    load();
  }, [load]);

  const handlePresetChange = (key) => {
    setPreset(key);
    if (key === 'custom') return;
    const range = resolveRangeFromPreset(key);
    if (range) {
      setStartDate(range.start);
      setEndDate(range.end);
    }
  };

  const handleCustomRangeChange = (start, end) => {
    setPreset('custom');
    setStartDate(start);
    setEndDate(end);
  };

  const dailyHint =
    dataSource === 'dashboard'
      ? '每日趋势来自个人仪表盘（最近 7 天）'
      : '每日趋势来自个人仪表盘（最近 7 天）；汇总来自账单报表';

  const summary = agg?.summary;
  const avgQuotaPerDay =
    summary && summary.dayCount > 0 ? summary.quotaM / summary.dayCount : summary?.quotaM || 0;

  return (
    <div className="usage-page page-enter">
      <div className="usage-page-header">
        <div>
          <div className="usage-section-title">用量统计</div>
          <div className="usage-section-sub">全面了解 Token 消耗与费用分布</div>
        </div>
        <div className="usage-header-actions">
          <UsageRangePicker
            preset={preset}
            startDate={startDate}
            endDate={endDate}
            loading={loading}
            onPresetChange={handlePresetChange}
            onCustomRangeChange={handleCustomRangeChange}
          />
        </div>
      </div>

      {error ? <div className="tob-error">{error}</div> : null}

      <div className="usage-grid-4">
        <div className="usage-card usage-accent-card">
          <div className="usage-card-title">总 Token 用量</div>
          <div className="usage-card-value">
            {loading ? '—' : formatTokenCount(summary?.totalTokens)}
          </div>
          <div className="usage-card-sub">
            {loading
              ? '—'
              : `输入 ${formatTokenCount(summary?.promptTokens)} · 输出 ${formatTokenCount(summary?.completionTokens)}`}
          </div>
        </div>
        <div className="usage-card">
          <div className="usage-card-title">总消费额度 (M)</div>
          <div className="usage-card-value">
            {loading ? '—' : formatQuotaM(summary?.quotaM)}
          </div>
          <div className="usage-card-sub">
            {loading ? '—' : `日均 ${formatQuotaM(avgQuotaPerDay)}`}
          </div>
        </div>
        <div className="usage-card">
          <div className="usage-card-title">请求次数</div>
          <div className="usage-card-value">
            {loading ? '—' : (summary?.requestCount || 0).toLocaleString()}
          </div>
          <div className="usage-card-sub">
            {loading ? '—' : `覆盖 ${summary?.modelCount || 0} 个模型`}
          </div>
        </div>
        <div className="usage-card">
          <div className="usage-card-title">统计周期</div>
          <div className="usage-card-value" style={{ fontSize: 18 }}>
            {loading ? '—' : `${summary?.dayCount || '—'} 天`}
          </div>
          <div className="usage-card-sub">
            {startDate} ~ {endDate}
          </div>
        </div>
      </div>

      {agg?.daily?.length > 0 ? (
        <>
          <p className="usage-hint">{dailyHint}</p>
          <div className="usage-grid-2">
            <div className="usage-card">
              <div className="usage-card-header">
                <div className="usage-section-title">每日 Token 用量</div>
              </div>
              <div className="usage-chart-wrap">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={agg.daily} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                    <XAxis {...X_AXIS} />
                    <YAxis hide />
                    <Tooltip
                      contentStyle={TOOLTIP_STYLE}
                      formatter={(v) => [formatTokenCount(v), 'Token']}
                      labelFormatter={(l) => `日期：${formatChartDate(l)}`}
                    />
                    <Bar dataKey="tokens" fill="#6366f1" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
            <div className="usage-card">
              <div className="usage-card-header">
                <div className="usage-section-title">每日消费额度 (M)</div>
              </div>
              <div className="usage-chart-wrap">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={agg.daily} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                    <XAxis {...X_AXIS} />
                    <YAxis hide />
                    <Tooltip
                      contentStyle={TOOLTIP_STYLE}
                      formatter={(v) => [formatQuotaM(v), '额度 (M)']}
                      labelFormatter={(l) => `日期：${formatChartDate(l)}`}
                    />
                    <Bar dataKey="quotaM" fill="#06b6d4" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>
        </>
      ) : null}

      <div className="usage-card">
        <div className="usage-card-header">
          <div className="usage-section-title">
            {agg?.tableMode === 'tenant' ? '按租户统计' : '按模型统计'}
          </div>
          <span className="tag tag-gray">本周期</span>
        </div>
        <div className="usage-table-wrap">
          {loading ? (
            <div className="usage-empty">加载中…</div>
          ) : !agg?.byModel?.length ? (
            <div className="usage-empty">所选时间范围内暂无用量数据</div>
          ) : agg.tableMode === 'tenant' ? (
            <table className="usage-table">
              <thead>
                <tr>
                  <th>租户</th>
                  <th>请求次数</th>
                  <th>输入 Tokens</th>
                  <th>输出 Tokens</th>
                  <th>消费额度 (M)</th>
                  <th>占比</th>
                </tr>
              </thead>
              <tbody>
                {agg.byModel.map((row, i) => (
                  <tr key={row.tenant_id ?? i}>
                    <td>
                      <div className="usage-model-name">{row.tenant_name}</div>
                      {row.tenant_id != null && Number(row.tenant_id) > 0 ? (
                        <div className="usage-model-sub">ID {row.tenant_id}</div>
                      ) : null}
                    </td>
                    <td>{(row.request_count || 0).toLocaleString()}</td>
                    <td>{formatTokenCount(row.prompt_tokens)}</td>
                    <td>{formatTokenCount(row.completion_tokens)}</td>
                    <td className="usage-quota">{formatQuotaM(row.quotaM)}</td>
                    <td>
                      <ShareBar share={row.share} index={i} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <table className="usage-table">
              <thead>
                <tr>
                  <th>模型</th>
                  <th>请求次数</th>
                  <th>输入 Tokens</th>
                  <th>输出 Tokens</th>
                  <th>消费额度 (M)</th>
                  <th>占比</th>
                </tr>
              </thead>
              <tbody>
                {agg.byModel.map((row, i) => (
                  <tr key={row.model_name}>
                    <td>
                      <div className="usage-model-cell">
                        <span style={{ fontSize: 16 }}>{getModelIcon({ model_id: row.model_name })}</span>
                        <div>
                          <div className="usage-model-name">{row.model_name}</div>
                        </div>
                      </div>
                    </td>
                    <td>{(row.request_count || 0).toLocaleString()}</td>
                    <td>{formatTokenCount(row.prompt_tokens)}</td>
                    <td>{formatTokenCount(row.completion_tokens)}</td>
                    <td className="usage-quota">{formatQuotaM(row.quotaM)}</td>
                    <td>
                      <ShareBar share={row.share} index={i} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}

function ShareBar({ share, index }) {
  const cls = PROGRESS_CLASSES[index % PROGRESS_CLASSES.length];
  return (
    <div className="usage-share">
      <div className="usage-progress-bar">
        <div
          className={`usage-progress-fill ${cls}`}
          style={{ width: `${Math.min(100, Math.max(0, share || 0))}%` }}
        />
      </div>
      <span style={{ fontSize: 12, color: 'var(--text2)' }}>{(share || 0).toFixed(0)}%</span>
    </div>
  );
}
