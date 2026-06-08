import CaptchaWidget from '@/components/login/CaptchaWidget';
import '@/styles/login-captcha.css';

export default function LoginCaptchaModal({
  open,
  onClose,
  challenge,
  ready,
  loading,
  loadError,
  onRefresh,
  onClear,
  onConfirm,
  onAnswerChange,
  onReadyChange,
  captchaWidgetRef,
}) {
  if (!open) return null;

  return (
    <div className="tob-modal-overlay" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="tob-modal captcha-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="tob-modal-header">
          <span>安全验证</span>
          <button type="button" className="tob-modal-close" onClick={onClose} aria-label="关闭">
            ×
          </button>
        </div>

        <div className="tob-modal-body">
          {challenge ? (
            <CaptchaWidget
              ref={captchaWidgetRef}
              challenge={challenge}
              onAnswerChange={onAnswerChange}
              onReadyChange={onReadyChange}
            />
          ) : (
            <p className="captcha-hint">
              {loading ? '验证码加载中…' : loadError || '请点击「换一张」获取验证码'}
            </p>
          )}

          <div className="captcha-actions-row">
            <button type="button" className="tob-btn-secondary" onClick={onClear} disabled={!challenge}>
              清除
            </button>
            <button type="button" className="tob-btn-primary" onClick={onRefresh}>
              换一张
            </button>
          </div>
        </div>

        <div className="tob-modal-footer">
          <button
            type="button"
            className="tob-btn-primary"
            disabled={!challenge || !ready}
            onClick={onConfirm}
          >
            完成
          </button>
        </div>
      </div>
    </div>
  );
}
