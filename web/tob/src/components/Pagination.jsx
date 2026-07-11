import './pagination.css';

/**
 * 构建页码序列（含省略号）
 * @returns {(number|'ellipsis')[]}
 */
export function buildPageItems(page, totalPages, sibling = 1) {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const pages = new Set([1, totalPages, page]);
  for (let i = page - sibling; i <= page + sibling; i += 1) {
    if (i >= 1 && i <= totalPages) pages.add(i);
  }
  const sorted = [...pages].sort((a, b) => a - b);
  const result = [];
  for (let i = 0; i < sorted.length; i += 1) {
    if (i > 0 && sorted[i] - sorted[i - 1] > 1) result.push('ellipsis');
    result.push(sorted[i]);
  }
  return result;
}

/** 通用分页（mockup .pagination） */
export default function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  className = '',
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize) || 1);
  const safePage = Math.min(Math.max(1, page), totalPages);
  const items = buildPageItems(safePage, totalPages);

  if (total <= 0) return null;

  return (
    <div className={`tob-pagination${className ? ` ${className}` : ''}`}>
      <span className="tob-pagination-info">
        共 {total.toLocaleString()} 条记录，第 {safePage} / {totalPages} 页
      </span>
      <div className="tob-pagination-btns">
        <button
          type="button"
          className="tob-page-btn"
          disabled={safePage <= 1}
          aria-label="上一页"
          onClick={() => onPageChange(safePage - 1)}
        >
          ‹
        </button>
        {items.map((item, idx) =>
          item === 'ellipsis' ? (
            <span key={`ellipsis-${idx}`} className="tob-page-ellipsis">
              …
            </span>
          ) : (
            <button
              key={item}
              type="button"
              className={`tob-page-btn${item === safePage ? ' active' : ''}`}
              onClick={() => onPageChange(item)}
            >
              {item}
            </button>
          )
        )}
        <button
          type="button"
          className="tob-page-btn"
          disabled={safePage >= totalPages}
          aria-label="下一页"
          onClick={() => onPageChange(safePage + 1)}
        >
          ›
        </button>
      </div>
    </div>
  );
}
