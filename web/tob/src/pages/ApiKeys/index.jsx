import { useCallback, useEffect, useMemo, useState } from 'react';
import { Modal } from 'antd';
import { useLocation } from 'react-router-dom';
import Pagination from '@/components/Pagination';
import { getApiErrorMessage } from '@/api/client';
import {
  TOKEN_PAGE_SIZE,
  TOKEN_TABS,
  buildTokenMetaParts,
  computeTokenStats,
  copyText,
  deleteToken,
  fetchTokenPage,
  filterTokensByTab,
  formatMaskedKey,
  formatQuotaDisplay,
  getCopyKeyValue,
  getTokenStatusMeta,
} from '@/lib/tokens';
import TokenFormModal from './TokenFormModal';
import './apiKeys.css';

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

function KeyIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  );
}

function EditIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <polyline points="3 6 5 6 21 6" />
      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  );
}

export default function ApiKeysPage() {
  const location = useLocation();
  const [tokens, setTokens] = useState([]);
  const [tab, setTab] = useState('all');
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toast, setToast] = useState('');

  const [formOpen, setFormOpen] = useState(false);
  const [editId, setEditId] = useState(null);
  const [initialModels, setInitialModels] = useState([]);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      let pageIdx = 0;
      let all = [];
      for (;;) {
        const batch = await fetchTokenPage(pageIdx, '');
        if (!batch.length) break;
        all = all.concat(batch);
        if (batch.length < TOKEN_PAGE_SIZE) break;
        pageIdx += 1;
        if (pageIdx > 50) break;
      }
      setTokens(all);
    } catch (e) {
      setError(getApiErrorMessage(e));
      setTokens([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  useEffect(() => {
    const modelId = location.state?.modelId;
    if (modelId) {
      setInitialModels([String(modelId)]);
      setEditId(null);
      setFormOpen(true);
      window.history.replaceState({}, document.title);
    }
  }, [location.state]);

  const filtered = useMemo(() => filterTokensByTab(tokens, tab), [tokens, tab]);
  const stats = useMemo(() => computeTokenStats(tokens), [tokens]);
  const pageRows = useMemo(() => {
    const start = (page - 1) * TOKEN_PAGE_SIZE;
    return filtered.slice(start, start + TOKEN_PAGE_SIZE);
  }, [filtered, page]);

  useEffect(() => {
    setPage(1);
  }, [tab]);

  const showToast = (msg) => {
    setToast(msg);
    window.setTimeout(() => setToast(''), 2000);
  };

  const openCreate = () => {
    setEditId(null);
    setInitialModels([]);
    setFormOpen(true);
  };

  const openEdit = (id) => {
    setEditId(id);
    setInitialModels([]);
    setFormOpen(true);
  };

  const handleCopy = async (token) => {
    const text = getCopyKeyValue(token.key);
    if (!text) {
      showToast('密钥不可复制');
      return;
    }
    const ok = await copyText(text);
    showToast(ok ? '已复制密钥' : '复制失败');
  };

  const handleDelete = (token) => {
    Modal.confirm({
      title: '删除令牌',
      content: `确定删除「${token.name || '未命名'}」？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await deleteToken(token.id);
        setTokens((list) => list.filter((t) => t.id !== token.id));
        showToast('已删除');
      },
    });
  };

  return (
    <div className="apikeys-page page-enter">
      <div className="apikeys-section-header">
        <div>
          <div className="apikeys-section-title">API KEY 管理</div>
          <div className="apikeys-section-sub">
            创建和管理用于调用 API 的密钥，密钥仅在创建时完整显示一次
          </div>
        </div>
        <div className="apikeys-header-actions">
          <button type="button" className="tob-btn tob-btn-primary" onClick={openCreate}>
            <PlusIcon />
            创建新 Key
          </button>
        </div>
      </div>

      <div className="apikeys-grid-4">
        <div className="apikeys-stat-card">
          <div className="apikeys-stat-title">活跃 Key</div>
          <div className="apikeys-stat-value">{stats.active}</div>
          <div className="apikeys-stat-sub">共 {stats.total} 个</div>
        </div>
        <div className="apikeys-stat-card">
          <div className="apikeys-stat-title">已用额度</div>
          <div className="apikeys-stat-value">¥{formatQuotaDisplay(stats.used)}</div>
          <div className="apikeys-stat-sub">全部令牌累计</div>
        </div>
        <div className="apikeys-stat-card">
          <div className="apikeys-stat-title">剩余额度</div>
          <div className="apikeys-stat-value">¥{formatQuotaDisplay(stats.remain)}</div>
          <div className="apikeys-stat-sub">不含无限额度令牌</div>
        </div>
        <div className="apikeys-stat-card">
          <div className="apikeys-stat-title">无限额度</div>
          <div className="apikeys-stat-value">{stats.unlimited}</div>
          <div className="apikeys-stat-sub">令牌数量</div>
        </div>
      </div>

      {error ? <div className="tob-error">{error}</div> : null}
      {toast ? <div className="apikeys-toast">{toast}</div> : null}

      <div className="apikeys-card">
        <div className="apikeys-tabs">
          {TOKEN_TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              className={`apikeys-tab${tab === t.key ? ' active' : ''}`}
              onClick={() => setTab(t.key)}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="apikeys-list">
          {loading ? (
            <div className="apikeys-empty">加载中…</div>
          ) : null}
          {!loading && pageRows.length === 0 ? (
            <div className="apikeys-empty">暂无令牌，点击「创建新 Key」开始</div>
          ) : null}
          {!loading &&
            pageRows.map((token) => {
              const status = getTokenStatusMeta(token.status);
              const muted = token.status !== 1;
              const meta = buildTokenMetaParts(token);
              return (
                <div
                  key={token.id}
                  className={`apikeys-key-row${muted ? ' is-muted' : ''}`}
                >
                  <div className={`apikeys-key-icon${muted ? ' is-muted' : ''}`}>
                    <KeyIcon />
                  </div>
                  <div className="apikeys-key-info">
                    <div className="apikeys-key-title">
                      <div className="apikeys-key-name">{token.name || '未命名'}</div>
                      <span className={`tag ${status.tagClass}`}>{status.label}</span>
                    </div>
                    <div className="apikeys-key-val">{formatMaskedKey(token.key)}</div>
                    <div className="apikeys-key-meta">{meta.join(' · ')}</div>
                  </div>
                  <div className="apikeys-key-actions">
                    <button
                      type="button"
                      className="apikeys-icon-btn"
                      title="复制密钥"
                      onClick={() => handleCopy(token)}
                    >
                      <CopyIcon />
                    </button>
                    <button
                      type="button"
                      className="apikeys-icon-btn"
                      title="编辑"
                      onClick={() => openEdit(token.id)}
                    >
                      <EditIcon />
                    </button>
                    <button
                      type="button"
                      className="apikeys-icon-btn danger"
                      title="删除"
                      onClick={() => handleDelete(token)}
                    >
                      <TrashIcon />
                    </button>
                  </div>
                </div>
              );
            })}
        </div>

        {!loading && filtered.length > 0 ? (
          <Pagination
            page={page}
            pageSize={TOKEN_PAGE_SIZE}
            total={filtered.length}
            onPageChange={setPage}
          />
        ) : null}
      </div>

      <TokenFormModal
        open={formOpen}
        tokenId={editId}
        initialModels={initialModels}
        onClose={() => {
          setFormOpen(false);
          setEditId(null);
          setInitialModels([]);
          loadAll();
        }}
        onSuccess={(result) => {
          loadAll();
          if (result?.type === 'update') {
            setFormOpen(false);
            setEditId(null);
            setInitialModels([]);
            showToast('已保存');
          }
          if (result?.type === 'create' && result?.key) {
            showToast('创建成功，请复制保存密钥');
          }
        }}
      />
    </div>
  );
}
