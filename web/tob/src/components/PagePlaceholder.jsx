/**
 * 业务页占位：样式在 toB，逻辑后续从 default 主题迁移或复用组件
 */
export default function PagePlaceholder({ title, description, apiNote, defaultPath }) {
  return (
    <div className="tob-card">
      <div className="tob-card-title">{title}</div>
      <p className="tob-card-desc" style={{ marginBottom: 12 }}>
        {description}
      </p>
      {apiNote && (
        <p className="tob-card-desc">
          <span className="tob-tag">API</span>
          {apiNote}
        </p>
      )}
      {defaultPath && (
        <p className="tob-card-desc" style={{ marginTop: 12 }}>
          <span className="tob-tag">default 路由</span>
          <code>{defaultPath}</code> — 可逐步将 default 对应页面逻辑迁入此路由
        </p>
      )}
    </div>
  );
}
