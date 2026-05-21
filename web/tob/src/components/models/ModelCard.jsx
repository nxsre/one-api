import {
  buildTagList,
  formatContextLimit,
  formatPriceLabel,
  getBadge,
  getCardTheme,
  getModelDescription,
  getModelIcon,
} from '@/lib/modelCatalog';

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

      <div className="model-actions">
        {row?.doc_url ? (
          <button
            type="button"
            className="tob-btn tob-btn-ghost"
            onClick={() => window.open(row.doc_url, '_blank', 'noopener,noreferrer')}
          >
            查看文档
          </button>
        ) : (
          <button type="button" className="tob-btn tob-btn-ghost" onClick={() => onCopyId(row)}>
            复制模型 ID
          </button>
        )}
        <button type="button" className="tob-btn tob-btn-primary" onClick={() => onUse(row)}>
          立即调用
        </button>
      </div>
    </article>
  );
}
