import { useCallback, useEffect, useMemo, useState } from 'react';
import { message } from 'antd';
import { useNavigate } from 'react-router-dom';
import Pagination from '@/components/Pagination';
import ModelCard from '@/components/models/ModelCard';
import { getApiErrorMessage } from '@/api/client';
import { copyText } from '@/lib/tokens';
import {
  MODEL_FILTERS,
  MODEL_PAGE_SIZE,
  fetchModelSquare,
  filterToModelSquareCategory,
  paginateModelItems,
} from '@/lib/modelCatalog';
import './models.css';

function SearchIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  );
}

export default function ModelsPage() {
  const navigate = useNavigate();
  const [allItems, setAllItems] = useState([]);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState('all');
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    const timer = window.setTimeout(() => setSearchQuery(searchInput.trim()), 300);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    setPage(1);
  }, [searchQuery, filter]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError('');
      try {
        const { items } = await fetchModelSquare({
          category: filterToModelSquareCategory(filter),
          keyword: searchQuery,
        });
        if (!cancelled) setAllItems(items);
      } catch (e) {
        if (!cancelled) {
          setError(getApiErrorMessage(e));
          setAllItems([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [searchQuery, filter]);

  const { items: rows, total } = useMemo(
    () => paginateModelItems(allItems, page, MODEL_PAGE_SIZE),
    [allItems, page]
  );

  const handleCopyId = async (row) => {
    const id = row?.model_id || '';
    const ok = await copyText(id);
    if (ok) message.success('模型ID已复制成功');
    else message.error('复制失败');
  };

  const handleUse = (row) => {
    navigate('/api-keys', { state: { modelId: row?.model_id } });
  };

  return (
    <div className="models-page">
      <div className="models-section-header">
        <div>
          <div className="models-section-title">模型广场</div>
          <div className="models-section-sub">
            以下为当前账号可调用的模型；可按分类筛选或搜索。
          </div>
        </div>
      </div>

      {error ? <div className="tob-error">{error}</div> : null}

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
            placeholder="搜索模型 ID / 名称…"
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
            {total === 0 ? '暂无可用模型' : '本页没有匹配的模型，请调整筛选或搜索关键词。'}
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

      {!loading && total > 0 ? (
        <div className="models-count">共 {total} 个可用模型</div>
      ) : null}

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
