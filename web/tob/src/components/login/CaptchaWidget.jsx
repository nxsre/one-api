import { forwardRef, useEffect, useImperativeHandle, useMemo, useState } from 'react';

function getAnswer(mode, { clicks, slideX, tileY, rotateAngle }) {
  if (mode === 'slide') {
    return { x: Math.round(slideX), y: Math.round(tileY) };
  }
  if (mode === 'rotate') {
    return { angle: Math.round(rotateAngle) };
  }
  return clicks;
}

function isReady(mode, { touched, clicks, dotNum }) {
  if (mode === 'slide' || mode === 'rotate') return touched;
  return dotNum > 0 && clicks.length === dotNum;
}

const CaptchaWidget = forwardRef(function CaptchaWidget(
  { challenge, onAnswerChange, onReadyChange },
  ref
) {
  const mode = challenge?.mode || 'click';
  const masterImage = challenge?.master_image || '';
  const thumbImage = challenge?.thumb_image || '';
  const dotNum = Number(challenge?.dot_num) || 0;
  const tileW = Number(challenge?.tile_width) || 0;
  const tileY = Number(challenge?.tile_y) || 0;
  const thumbSize = Number(challenge?.thumb_size) || 0;

  const [natural, setNatural] = useState({ w: 0, h: 0 });
  const [clicks, setClicks] = useState([]);
  const [slideX, setSlideX] = useState(0);
  const [rotateAngle, setRotateAngle] = useState(0);
  const [touched, setTouched] = useState(false);

  useEffect(() => {
    setNatural({ w: 0, h: 0 });
    setClicks([]);
    setSlideX(Number(challenge?.tile_x) || 0);
    setRotateAngle(0);
    setTouched(false);
  }, [challenge]);

  const slideMax = Math.max(0, (natural.w || 0) - tileW);
  const answer = useMemo(
    () => getAnswer(mode, { clicks, slideX, tileY, rotateAngle }),
    [mode, clicks, slideX, tileY, rotateAngle]
  );
  const ready = useMemo(
    () => isReady(mode, { touched, clicks, dotNum }),
    [mode, touched, clicks, dotNum]
  );

  useEffect(() => {
    onAnswerChange?.(answer);
  }, [answer, onAnswerChange]);

  useEffect(() => {
    onReadyChange?.(ready);
  }, [ready, onReadyChange]);

  useImperativeHandle(ref, () => ({
    clear() {
      setClicks([]);
      setSlideX(Number(challenge?.tile_x) || 0);
      setRotateAngle(0);
      setTouched(false);
    },
    isReady: () => ready,
    getAnswer: () => answer,
  }));

  const onMasterLoad = (ev) => {
    setNatural({
      w: ev.target.naturalWidth,
      h: ev.target.naturalHeight,
    });
  };

  const onClickMaster = (e) => {
    if (mode !== 'click' || !dotNum || clicks.length >= dotNum) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const scale = rect.width > 0 ? natural.w / rect.width : 1;
    const x = Math.round((e.clientX - rect.left) * scale);
    const y = Math.round((e.clientY - rect.top) * scale);
    setClicks((prev) => [...prev, { x, y }]);
    setTouched(true);
  };

  if (mode === 'click') {
    return (
      <div className="captcha-widget">
        {thumbImage ? (
          <div className="captcha-thumb-wrap">
            <img src={thumbImage} alt="提示" className="captcha-thumb" />
          </div>
        ) : null}
        {masterImage ? (
          <div className="captcha-master-wrap">
            <div className="captcha-master-inner">
              <img
                src={masterImage}
                alt="验证码"
                className="captcha-master"
                onLoad={onMasterLoad}
                onClick={onClickMaster}
              />
              {natural.w > 0
                ? clicks.map((p, i) => (
                    <span
                      key={`${p.x}-${p.y}-${i}`}
                      className="captcha-dot"
                      style={{
                        left: `${(p.x / natural.w) * 100}%`,
                        top: `${(p.y / natural.h) * 100}%`,
                      }}
                    >
                      {i + 1}
                    </span>
                  ))
                : null}
            </div>
          </div>
        ) : null}
        {dotNum > 0 ? (
          <p className="captcha-progress">
            {clicks.length} / {dotNum}
          </p>
        ) : null}
      </div>
    );
  }

  if (mode === 'slide') {
    return (
      <div className="captcha-widget">
        {masterImage ? (
          <div className="captcha-master-wrap">
            <div className="captcha-master-inner">
              <img
                src={masterImage}
                alt="验证码"
                className="captcha-master captcha-master-slide"
                onLoad={onMasterLoad}
              />
              {thumbImage ? (
                <img
                  src={thumbImage}
                  alt="拼图块"
                  className="captcha-slide-tile"
                  style={{
                    left: natural.w > 0 ? `${(slideX / natural.w) * 100}%` : '0',
                    top: natural.h > 0 ? `${(tileY / natural.h) * 100}%` : '0',
                    width: natural.w > 0 ? `${(tileW / natural.w) * 100}%` : 'auto',
                  }}
                />
              ) : null}
            </div>
          </div>
        ) : null}
        <div className="captcha-slider-wrap">
          <input
            type="range"
            className="captcha-slider"
            min={0}
            max={slideMax}
            value={slideX}
            onChange={(e) => {
              setSlideX(Number(e.target.value));
              setTouched(true);
            }}
          />
          <p className="captcha-hint">拖动滑块，将拼图移至正确位置</p>
        </div>
      </div>
    );
  }

  return (
    <div className="captcha-widget">
      {masterImage ? (
        <div className="captcha-master-wrap captcha-rotate-wrap">
          <div className="captcha-master-inner captcha-rotate-inner">
            <img
              src={masterImage}
              alt="验证码"
              className="captcha-master captcha-master-rotate"
              onLoad={onMasterLoad}
            />
            {thumbImage ? (
              <img
                src={thumbImage}
                alt="旋转块"
                className="captcha-rotate-thumb"
                style={{
                  width: natural.w > 0 && thumbSize > 0 ? `${(thumbSize / natural.w) * 100}%` : '50%',
                  transform: `translate(-50%, -50%) rotate(${rotateAngle}deg)`,
                }}
              />
            ) : null}
          </div>
        </div>
      ) : null}
      <div className="captcha-slider-wrap">
        <input
          type="range"
          className="captcha-slider"
          min={0}
          max={360}
          value={rotateAngle}
          onChange={(e) => {
            setRotateAngle(Number(e.target.value));
            setTouched(true);
          }}
        />
        <p className="captcha-hint">拖动滑块旋转图片至正确角度（{rotateAngle}°）</p>
      </div>
    </div>
  );
});

export default CaptchaWidget;
