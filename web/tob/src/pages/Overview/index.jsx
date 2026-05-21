import { useEffect, useState } from 'react';
import { API } from '@/api/client';
import { getApiErrorMessage } from '@/api/client';
import './overview.css';

export default function OverviewPage() {
  const [summary, setSummary] = useState({
    todayRequests: 0,
    todayQuota: 0,
    todayTokens: 0,
  });
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
        const rows = res.data.data || [];
        const today = new Date().toISOString().split('T')[0];
        const todayRows = rows.filter((item) => item.Day === today);
        if (!cancelled) {
          setSummary({
            todayRequests: todayRows.reduce((s, i) => s + (i.RequestCount || 0), 0),
            todayQuota: todayRows.reduce((s, i) => s + (i.Quota || 0), 0) / 1_000_000,
            todayTokens: todayRows.reduce(
              (s, i) => s + (i.PromptTokens || 0) + (i.CompletionTokens || 0),
              0
            ),
          });
        }
      } catch (e) {
        if (!cancelled) setError(getApiErrorMessage(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="overview-page">
      <div className="overview-banner">
        <span>欢迎使用 TokenHub 企业控制台 — 数据来自 one-api</span>
      </div>
      {error && <div className="tob-error" style={{ marginBottom: 16 }}>{error}</div>}
      <div className="overview-stats">
        <div className="stat-card glow">
          <div className="stat-label">今日请求</div>
          <div className="stat-value">{loading ? '—' : summary.todayRequests.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">今日额度 (M)</div>
          <div className="stat-value">{loading ? '—' : summary.todayQuota.toFixed(2)}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">今日 Token</div>
          <div className="stat-value">{loading ? '—' : summary.todayTokens.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">状态</div>
          <div className="stat-value" style={{ fontSize: 16 }}>
            {loading ? '加载中' : '正常'}
          </div>
        </div>
      </div>
      <div className="tob-card" style={{ marginTop: 20 }}>
        <div className="tob-card-title">下一步</div>
        <p className="tob-card-desc">
          概览已对接 <code>GET /api/user/dashboard</code>。图表与模型卡片 UI 可参考
          mockup，业务表格可迁移 default 的 Dashboard / Channel 等组件。
        </p>
      </div>
    </div>
  );
}
