import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Pagination from '@/components/Pagination';
import ModelCard from '@/components/models/ModelCard';
import { getApiErrorMessage } from '@/api/client';
import { getStoredStatus } from '@/lib/systemStatus';
import {
  MODEL_FILTERS,
  MODEL_PAGE_SIZE,
  fetchModelsPage,
  fetchUserAvailableModelIds,
  isAdminUser,
} from '@/lib/modelCatalog';
import { useUser } from '@/context/UserContext';
import './models.css';

function SearchIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  );
}

function DocIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  );
}

export default function ModelsPage() {
  const { user } = useUser();
  const navigate = useNavigate();
  const [rows, setRows] = useState([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [availableSet, setAvailableSet] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState('all');
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [toast, setToast] = useState('');

  const admin = isAdminUser(user);

  useEffect(() => {
    if (admin) {
      setAvailableSet(null);
      return;
    }
    let cancelled = false;
    fetchUserAvailableModelIds()
      .then((ids) => {
        if (!cancelled) setAvailableSet(new Set(ids));
      })
      .catch(() => {
        if (!cancelled) setAvailableSet(new Set());
      });
    return () => {
      cancelled = true;
    };
  }, [admin, user]);

  useEffect(() => {
    const timer = window.setTimeout(() => setSearchQuery(searchInput.trim()), 300);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    setPage(1);
  }, [searchQuery, filter]);

  const load = useCallback(async () => {
    if (!admin && availableSet === null) return;

    setLoading(true);
    setError('');
    try {
      const result = await fetchModelsPage({
        user,
        page,
        pageSize: MODEL_PAGE_SIZE,
        search: searchQuery,
        filterKey: filter,
        availableSet,
      });
      setRows(result.items);
      setTotal(result.total);
    } catch (e) {
      setError(getApiErrorMessage(e));
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [user, page, searchQuery, filter, availableSet, admin]);

  useEffect(() => {
    load();
  }, [load]);

  const showToast = (msg) => {
    setToast(msg);
    window.setTimeout(() => setToast(''), 2200);
  };

  const handleCopyId = async (row) => {
    const id = row?.model_id || '';
    try {
      await navigator.clipboard.writeText(id);
      showToast(`已复制：${id}`);
    } catch {
      showToast(id);
    }
  };

  const handleUse = (row) => {
    navigate('/api-keys', { state: { modelId: row?.model_id } });
  };

  const docsUrl = useMemo(() => {
    const status = getStoredStatus();
    const base = String(status?.server_address || status?.api_base || '').replace(/\/$/, '');
    if (status?.docs_url) return status.docs_url;
    if (base) return `${base}/docs`;
    return null;
  }, []);

  return (
    <div className="models-page">
      <div className="models-section-header">
        <div>
          <div className="models-section-title">模型广场</div>
          <div className="models-section-sub">
            100+ 全球顶级模型，OpenAI 兼容接口，一键接入
          </div>
        </div>
        {docsUrl ? (
          <a className="tob-btn tob-btn-primary" href={docsUrl} target="_blank" rel="noreferrer">
            <DocIcon />
            API 接入文档
          </a>
        ) : null}
      </div>

      {error ? <div className="tob-error">{error}</div> : null}
      {toast ? <div className="models-toast">{toast}</div> : null}

      <div className="models-filter-bar">
        {MODEL_FILTERS.map((f) => (
          <button
            key={f.key}
            type="button"
            className={`models-filter-btn${filter === f.key ? ' active' : ''}`}
            onClick={() => setFilter(f.key)}
          >
            {f.label}
          </button>
        ))}
        <div className="models-search">
          <SearchIcon />
          <input
            type="search"
            placeholder="搜索模型..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </div>
      </div>

      <div className="models-grid">
        {loading ? (
          <div className="models-loading">加载模型目录…</div>
        ) : rows.length === 0 ? (
          <div className="models-empty">
            {total === 0
              ? '暂无可用模型，请联系管理员配置渠道与模型目录。'
              : '本页没有匹配的模型，请调整筛选或搜索关键词。'}
          </div>
        ) : (
          rows.map((row) => (
            <ModelCard
              key={row.id != null ? `catalog-${row.id}-${page}` : `${row.model_id}-${page}`}
              row={row}
              onUse={handleUse}
              onCopyId={handleCopyId}
            />
          ))
        )}
      </div>

      {!loading ? (
        <Pagination
          page={page}
          pageSize={MODEL_PAGE_SIZE}
          total={total}
          onPageChange={setPage}
        />
      ) : null}
    </div>
  );
}
