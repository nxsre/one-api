import {
  buildTagList,
  formatContextLimit,
  formatPriceLabel,
  getBadge,
  getCardTheme,
  getModelDescription,
  getModelIcon,
} from '@/lib/modelCatalog';

function CopyIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  );
}

export default function ModelCard({ row, onUse, onCopyId }) {
  const theme = getCardTheme(row);
  const badge = getBadge(row);
  const tags = buildTagList(row);
  const ctx = formatContextLimit(row?.context_limit);
  const displayName = row?.model_name || row?.model_id;
  const provider =
    row?.provider_display ||
    row?.provider_key ||
    row?.owned_by ||
    '平台模型';

  return (
    <article className={`model-card ${theme}`}>
      <div className="model-header">
        <div className={`model-icon model-icon--${theme}`}>{getModelIcon(row)}</div>
        <div className="model-header-text">
          <div className="model-name" title={row?.model_id}>
            {displayName}
          </div>
          <div className="model-provider">{provider}</div>
        </div>
        {badge ? (
          <span className={`tag ${badge.tag}`} style={{ marginLeft: 'auto' }}>
            {badge.text}
          </span>
        ) : null}
      </div>

      {row?.model_id ? (
        <div className="model-id-row">
          <span className="model-id-label">模型ID</span>
          <code className="model-id" title={row.model_id}>
            {row.model_id}
          </code>
          <button
            type="button"
            className="model-copy-btn"
            title="复制模型 ID"
            aria-label="复制模型 ID"
            onClick={() => onCopyId(row)}
          >
            <CopyIcon />
          </button>
        </div>
      ) : null}

      <p className="model-desc">{getModelDescription(row)}</p>

      <div className="model-meta">
        <span className="model-price">{formatPriceLabel(row)}</span>
        {ctx ? <span className="model-ctx">· {ctx}</span> : null}
      </div>

      {tags.length > 0 ? (
        <div className="model-tags">
          {tags.map((t) => (
            <span key={`${row.model_id}-${t.label}`} className={`tag ${t.cls}`}>
              {t.label}
            </span>
          ))}
        </div>
      ) : null}

      <div className={`model-actions${row?.doc_url ? '' : ' model-actions--single'}`}>
        {row?.doc_url ? (
          <button
            type="button"
            className="tob-btn tob-btn-ghost"
            onClick={() => window.open(row.doc_url, '_blank', 'noopener,noreferrer')}
          >
            查看文档
          </button>
        ) : null}
        <button type="button" className="tob-btn tob-btn-primary" onClick={() => onUse(row)}>
          立即调用
        </button>
      </div>
    </article>
  );
}
