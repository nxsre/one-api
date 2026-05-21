import '@/styles/login-captcha.css';

export default function LoginCaptchaModal({
  open,
  onClose,
  thumbSrc,
  masterSrc,
  loading,
  loadError,
  dotNum,
  clicks,
  masterSize,
  onMasterLoad,
  onMasterClick,
  onClear,
  onRefresh,
}) {
  if (!open) return null;

  return (
    <div className="tob-modal-overlay" role="dialog" aria-modal="true">
      <div className="tob-modal captcha-modal">
        <div className="tob-modal-header">
          <span>安全验证</span>
          <button type="button" className="tob-modal-close" onClick={onClose} aria-label="关闭">
            ×
          </button>
        </div>
        <div className="tob-modal-body">
          {thumbSrc ? (
            <div className="captcha-thumb-wrap">
              <img alt="提示" src={thumbSrc} className="captcha-thumb" />
            </div>
          ) : null}

          {masterSrc ? (
            <div className="captcha-master-wrap">
              <div className="captcha-master-inner">
                <img
                  alt="验证码"
                  src={masterSrc}
                  className="captcha-master"
                  onLoad={onMasterLoad}
                  onClick={onMasterClick}
                />
                {masterSize.w > 0 &&
                  masterSize.h > 0 &&
                  clicks.map((p, i) => (
                    <span
                      key={i}
                      className="captcha-dot"
                      style={{
                        left: `${(p.x / masterSize.w) * 100}%`,
                        top: `${(p.y / masterSize.h) * 100}%`,
                      }}
                    >
                      {i + 1}
                    </span>
                  ))}
              </div>
            </div>
          ) : (
            <p className="captcha-hint">
              {loading ? '验证码加载中…' : loadError || '请点击刷新获取验证码'}
            </p>
          )}

          <p className="captcha-progress">
            已点击 {clicks.length} / {dotNum || '—'}
          </p>

          <div className="captcha-actions-row">
            <button type="button" className="tob-btn-secondary" onClick={onClear}>
              清除点击
            </button>
            <button type="button" className="tob-btn-secondary" onClick={onRefresh}>
              换一张
            </button>
          </div>
        </div>
        <div className="tob-modal-footer">
          <button type="button" className="tob-btn-primary" onClick={onClose}>
            完成
          </button>
        </div>
      </div>
    </div>
  );
}
