import { useEffect, useMemo, useState } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { API, getApiErrorMessage } from '@/api/client';
import {
  CHART_COLORS,
  calculateSummary,
  formatChartDate,
  getBarColor,
  processModelData,
  processTimeSeriesData,
} from '@/lib/dashboard';
import './overview.css';

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
  interval: 0,
  minTickGap: 5,
  padding: { left: 24, right: 24 },
};

const LINE_PROPS = {
  type: 'monotone',
  strokeWidth: 2,
  dot: false,
  activeDot: { r: 4 },
};

function MiniLineChart({ data, dataKey, color, valueFormatter, tooltipLabel }) {
  return (
    <ResponsiveContainer width="100%" height={120}>
      <LineChart data={data} margin={{ left: 4, right: 4, top: 4, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" opacity={0.6} />
        <XAxis {...X_AXIS} />
        <YAxis hide />
        <Tooltip
          contentStyle={TOOLTIP_STYLE}
          formatter={(value) => [valueFormatter(value), tooltipLabel]}
          labelFormatter={(label) => `日期：${formatChartDate(label)}`}
        />
        <Line {...LINE_PROPS} dataKey={dataKey} stroke={color} />
      </LineChart>
    </ResponsiveContainer>
  );
}

export default function OverviewPage() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/user/dashboard');
        if (!res.data?.success) {
          throw new Error(res.data?.message || '加载失败');
        }
        if (!cancelled) setData(res.data.data || []);
      } catch (e) {
        if (!cancelled) {
          setError(getApiErrorMessage(e));
          setData([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const summary = useMemo(() => calculateSummary(data), [data]);
  const timeSeriesData = useMemo(() => processTimeSeriesData(data), [data]);
  const { series: modelData, models } = useMemo(() => processModelData(data), [data]);

  return (
    <div className="overview-page">
      {error && (
        <div className="tob-error overview-error">{error}</div>
      )}

      <div className="overview-stats">
        <div className="stat-card glow">
          <div className="stat-label">今日请求</div>
          <div className="stat-value">
            {loading ? '—' : summary.todayRequests.toLocaleString()}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">今日额度 (M)</div>
          <div className="stat-value">
            {loading ? '—' : summary.todayQuota.toFixed(3)}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">今日 Token</div>
          <div className="stat-value">
            {loading ? '—' : summary.todayTokens.toLocaleString()}
          </div>
        </div>
      </div>

      <div className={`overview-charts-grid${loading ? ' is-loading' : ''}`}>
        <div className="overview-chart-card">
          <div className="overview-chart-title">请求趋势</div>
          <MiniLineChart
            data={timeSeriesData}
            dataKey="requests"
            color={CHART_COLORS.requests}
            valueFormatter={(v) => Number(v).toLocaleString()}
            tooltipLabel="请求数"
          />
        </div>
        <div className="overview-chart-card">
          <div className="overview-chart-title">额度趋势 (M)</div>
          <MiniLineChart
            data={timeSeriesData}
            dataKey="quota"
            color={CHART_COLORS.quota}
            valueFormatter={(v) => Number(v).toFixed(6)}
            tooltipLabel="额度 (M)"
          />
        </div>
        <div className="overview-chart-card">
          <div className="overview-chart-title">Token 趋势</div>
          <MiniLineChart
            data={timeSeriesData}
            dataKey="tokens"
            color={CHART_COLORS.tokens}
            valueFormatter={(v) => Number(v).toLocaleString()}
            tooltipLabel="Token"
          />
        </div>
      </div>

      <div className={`overview-chart-card overview-chart-card--wide${loading ? ' is-loading' : ''}`}>
        <div className="overview-chart-title">模型使用统计（Token）</div>
        <div className="overview-bar-wrap">
          {models.length === 0 && !loading ? (
            <p className="overview-empty">暂无模型用量数据</p>
          ) : (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={modelData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" opacity={0.6} />
                <XAxis {...X_AXIS} />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  tick={{ fontSize: 11, fill: 'var(--text3)' }}
                  tickFormatter={(v) =>
                    v >= 1_000_000 ? `${(v / 1_000_000).toFixed(1)}M` : v >= 1000 ? `${(v / 1000).toFixed(0)}k` : v
                  }
                />
                <Tooltip
                  contentStyle={TOOLTIP_STYLE}
                  labelFormatter={(label) => `日期：${formatChartDate(label)}`}
                  formatter={(value, name) => [Number(value).toLocaleString(), name]}
                />
                <Legend wrapperStyle={{ fontSize: 12, paddingTop: 12 }} />
                {models.map((model, index) => (
                  <Bar
                    key={model}
                    dataKey={model}
                    stackId="a"
                    fill={getBarColor(index)}
                    name={model}
                    radius={index === models.length - 1 ? [4, 4, 0, 0] : [0, 0, 0, 0]}
                  />
                ))}
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </div>
  );
}
