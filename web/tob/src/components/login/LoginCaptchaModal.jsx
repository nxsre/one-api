import GoCaptcha from 'go-captcha-react';
import 'go-captcha-react/dist/go-captcha-react.cjs.development.css';
import {
  CAPTCHA_MODE_CLICK,
  CAPTCHA_MODE_ROTATE,
  CAPTCHA_MODE_SLIDE,
} from '@/hooks/useLoginCaptcha';
import '@/styles/login-captcha.css';

export default function LoginCaptchaModal({
  open,
  onClose,
  mode,
  thumbSrc,
  masterSrc,
  loading,
  loadError,
  thumbSize,
  slideMeta,
  masterSize,
  onMasterLoad,
  onRefresh,
  onRotateConfirm,
  onClickConfirm,
  onSlideConfirm,
  captchaRef,
}) {
  if (!open) return null;

  const hasImages = Boolean(masterSrc && thumbSrc);
  const widgetConfig = {
    title: '安全验证',
    buttonText: '完成',
    showTheme: true,
  };

  const rotateData = {
    image: masterSrc || '',
    thumb: thumbSrc || '',
    thumbSize: thumbSize || 160,
    angle: 0,
  };

  const clickData = {
    image: masterSrc || '',
    thumb: thumbSrc || '',
  };

  const slideData = {
    image: masterSrc || '',
    thumb: thumbSrc || '',
    thumbX: slideMeta?.thumbX ?? 0,
    thumbY: slideMeta?.thumbY ?? 0,
    thumbWidth: slideMeta?.thumbWidth ?? 0,
    thumbHeight: slideMeta?.thumbHeight ?? 0,
  };

  const renderWidget = () => {
    if (mode === CAPTCHA_MODE_SLIDE) {
      return (
        <GoCaptcha.Slide
          ref={captchaRef}
          config={widgetConfig}
          data={slideData}
          events={{
            refresh: onRefresh,
            close: onClose,
            confirm: (point, reset) => {
              onSlideConfirm(point);
              reset?.();
              return true;
            },
          }}
        />
      );
    }
    if (mode === CAPTCHA_MODE_CLICK) {
      return (
        <GoCaptcha.Click
          ref={captchaRef}
          config={widgetConfig}
          data={clickData}
          events={{
            refresh: onRefresh,
            close: onClose,
            confirm: (dots, reset) => {
              onClickConfirm(dots);
              reset?.();
              return true;
            },
          }}
        />
      );
    }
    return (
      <GoCaptcha.Rotate
        ref={captchaRef}
        config={widgetConfig}
        data={rotateData}
        events={{
          refresh: onRefresh,
          close: onClose,
          confirm: (nextAngle, reset) => {
            onRotateConfirm(nextAngle);
            reset?.();
            return true;
          },
        }}
      />
    );
  };

  return (
    <div className="tob-modal-overlay" role="dialog" aria-modal="true">
      <div className="tob-captcha-widget-wrap">
        {!hasImages ? (
          <div className="tob-modal captcha-modal captcha-modal-fallback">
            <div className="tob-modal-header">
              <span>安全验证</span>
              <button type="button" className="tob-modal-close" onClick={onClose} aria-label="关闭">
                ×
              </button>
            </div>
            <div className="tob-modal-body">
              <p className="captcha-hint">
                {loading ? '验证码加载中…' : loadError || '请点击刷新获取验证码'}
              </p>
              {!loading && (
                <button type="button" className="tob-btn-primary captcha-reload-btn" onClick={onRefresh}>
                  换一张
                </button>
              )}
            </div>
          </div>
        ) : (
          renderWidget()
        )}

        {mode === CAPTCHA_MODE_CLICK && hasImages && masterSize.w === 0 && (
          <img alt="" src={masterSrc} className="captcha-size-probe" onLoad={onMasterLoad} />
        )}
      </div>
    </div>
  );
}

export { CAPTCHA_MODE_CLICK, CAPTCHA_MODE_ROTATE, CAPTCHA_MODE_SLIDE };
