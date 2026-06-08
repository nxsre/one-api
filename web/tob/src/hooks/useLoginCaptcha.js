import { useCallback, useEffect, useRef, useState } from 'react';
import { API, getApiErrorMessage } from '@/api/client';
import { isSecurePasswordLoginEnabled } from '@/lib/systemStatus';

export const CAPTCHA_MODE_CLICK = 'click';
export const CAPTCHA_MODE_ROTATE = 'rotate';
export const CAPTCHA_MODE_SLIDE = 'slide';

function captchaErrorMessage(err) {
  if (err?.response?.status === 429) {
    return '验证码请求过于频繁，请约 20 分钟后再试，或稍候手动点击「换一张」';
  }
  return getApiErrorMessage(err) || '验证码加载失败';
}

function resolveCaptchaMode(d) {
  const mode = String(d?.mode || '').toLowerCase();
  if (mode === CAPTCHA_MODE_SLIDE) return CAPTCHA_MODE_SLIDE;
  if (mode === CAPTCHA_MODE_ROTATE) return CAPTCHA_MODE_ROTATE;
  if (mode === CAPTCHA_MODE_CLICK || Number(d?.dot_num) > 0) return CAPTCHA_MODE_CLICK;
  if (Number(d?.tile_width) > 0 || Number(d?.tile_x) >= 0) return CAPTCHA_MODE_SLIDE;
  if (Number(d?.thumb_size) > 0) return CAPTCHA_MODE_ROTATE;
  return CAPTCHA_MODE_ROTATE;
}

function parseSlideMeta(d) {
  return {
    thumbX: Number(d?.tile_x) || 0,
    thumbY: Number(d?.tile_y) || 0,
    thumbWidth: Number(d?.tile_width) || 0,
    thumbHeight: Number(d?.tile_height) || 0,
  };
}

/**
 * 登录图形验证码（支持 slide / rotate / click，按后端 mode 字段切换）
 */
export function useLoginCaptcha({ mathCaptchaEnabled }) {
  const [mode, setMode] = useState(CAPTCHA_MODE_ROTATE);
  const [masterSrc, setMasterSrc] = useState('');
  const [thumbSrc, setThumbSrc] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [dotNum, setDotNum] = useState(0);
  const [thumbSize, setThumbSize] = useState(0);
  const [slideMeta, setSlideMeta] = useState({
    thumbX: 0,
    thumbY: 0,
    thumbWidth: 0,
    thumbHeight: 0,
  });
  const [challengeId, setChallengeId] = useState('');
  const [clicks, setClicks] = useState([]);
  const [angle, setAngle] = useState(null);
  const [slidePoint, setSlidePoint] = useState(null);
  const [masterSize, setMasterSize] = useState({ w: 0, h: 0 });
  const [modalOpen, setModalOpen] = useState(false);
  const loginProofRef = useRef(null);
  const captchaRef = useRef(null);
  const loadingRef = useRef(false);
  const autoLoadBlockedRef = useRef(false);
  const modalOpenedRef = useRef(false);

  const resetCaptcha = useCallback(() => {
    setMasterSrc('');
    setThumbSrc('');
    setDotNum(0);
    setThumbSize(0);
    setSlideMeta({ thumbX: 0, thumbY: 0, thumbWidth: 0, thumbHeight: 0 });
    setChallengeId('');
    setClicks([]);
    setAngle(null);
    setSlidePoint(null);
    setMasterSize({ w: 0, h: 0 });
    loginProofRef.current = null;
  }, []);

  const applyProof = useCallback((d) => {
    if (!isSecurePasswordLoginEnabled()) {
      loginProofRef.current = null;
      return;
    }
    if (
      d.login_request_id &&
      d.login_request_sig != null &&
      d.login_request_ts != null &&
      d.login_enc_key
    ) {
      loginProofRef.current = {
        id: d.login_request_id,
        ts: Number(d.login_request_ts),
        sig: d.login_request_sig,
        encKey: d.login_enc_key,
      };
    } else {
      loginProofRef.current = null;
    }
  }, []);

  const loadCaptcha = useCallback(
    async ({ manual = false } = {}) => {
      if (!mathCaptchaEnabled || loadingRef.current) return;
      if (!manual && autoLoadBlockedRef.current) return;

      loadingRef.current = true;
      setLoadError('');
      setLoading(true);
      try {
        const res = await API.get('/api/user/login/captcha');
        const d = res.data?.data;
        if (res.data?.success && d?.master_image && d?.thumb_image) {
          const nextMode = resolveCaptchaMode(d);
          setMode(nextMode);
          setMasterSize({ w: 0, h: 0 });
          setMasterSrc(d.master_image);
          setThumbSrc(d.thumb_image);
          setDotNum(Number(d.dot_num) || 0);
          setThumbSize(Number(d.thumb_size) || 0);
          setSlideMeta(parseSlideMeta(d));
          setChallengeId(d.captcha_id || '');
          setClicks([]);
          setAngle(null);
          setSlidePoint(null);
          setLoadError('');
          autoLoadBlockedRef.current = false;
          applyProof(d);
          captchaRef.current?.clear?.();
        } else {
          resetCaptcha();
          autoLoadBlockedRef.current = true;
          setLoadError(
            (res.data?.message && String(res.data.message).trim()) || '验证码加载失败'
          );
        }
      } catch (err) {
        resetCaptcha();
        autoLoadBlockedRef.current = err?.response?.status === 429;
        setLoadError(captchaErrorMessage(err));
      } finally {
        loadingRef.current = false;
        setLoading(false);
      }
    },
    [mathCaptchaEnabled, resetCaptcha, applyProof]
  );

  const refreshCaptcha = useCallback(() => {
    if (loadingRef.current) return;
    autoLoadBlockedRef.current = false;
    resetCaptcha();
    void loadCaptcha({ manual: true });
  }, [loadCaptcha, resetCaptcha]);

  const handleModalOpenChange = useCallback(
    (open) => {
      setModalOpen(open);
      if (!open) {
        modalOpenedRef.current = false;
        return;
      }
      if (modalOpenedRef.current) return;
      modalOpenedRef.current = true;
      if (!masterSrc && !loadingRef.current && !autoLoadBlockedRef.current) {
        void loadCaptcha();
      }
    },
    [loadCaptcha, masterSrc]
  );

  useEffect(() => {
    if (!modalOpen) {
      modalOpenedRef.current = false;
    }
  }, [modalOpen]);

  const onRotateConfirm = (nextAngle) => {
    setAngle(Math.round(nextAngle));
    setModalOpen(false);
  };

  const onClickConfirm = (dots) => {
    if (Array.isArray(dots) && dots.length > 0) {
      setClicks(dots.map((p) => ({ x: Math.round(p.x), y: Math.round(p.y) })));
    }
    setModalOpen(false);
  };

  const onSlideConfirm = (point) => {
    if (point && typeof point.x === 'number') {
      setSlidePoint({
        x: Math.round(point.x),
        y: Math.round(point.y ?? slideMeta.thumbY ?? 0),
      });
    }
    setModalOpen(false);
  };

  const getCaptchaPayload = () => {
    if (!mathCaptchaEnabled || !masterSrc || !challengeId) return undefined;
    if (mode === CAPTCHA_MODE_SLIDE) {
      if (!slidePoint) return undefined;
      return {
        captcha_id: challengeId,
        mode: CAPTCHA_MODE_SLIDE,
        captcha_point: slidePoint,
      };
    }
    if (mode === CAPTCHA_MODE_ROTATE) {
      if (angle == null) return undefined;
      return {
        captcha_id: challengeId,
        mode: CAPTCHA_MODE_ROTATE,
        captcha_angle: angle,
      };
    }
    if (!dotNum || clicks.length !== dotNum) return undefined;
    return {
      captcha_id: challengeId,
      mode: CAPTCHA_MODE_CLICK,
      captcha_clicks: clicks,
    };
  };

  const getLoginProof = () => {
    if (!mathCaptchaEnabled) return null;
    return loginProofRef.current;
  };

  const isComplete =
    mode === CAPTCHA_MODE_SLIDE
      ? slidePoint != null
      : mode === CAPTCHA_MODE_ROTATE
        ? angle != null
        : dotNum > 0 && clicks.length === dotNum;

  const validateBeforeLogin = () => {
    if (!mathCaptchaEnabled) return { ok: true };
    if (!masterSrc) {
      if (loading) return { ok: false, message: '验证码加载中，请稍候' };
      if (loadError) return { ok: false, message: loadError, openModal: true };
      return { ok: false, message: '请先完成安全验证', openModal: true };
    }
    if (mode === CAPTCHA_MODE_SLIDE) {
      if (!slidePoint) {
        return { ok: false, message: '请完成滑动拼图验证', openModal: true };
      }
      return { ok: true };
    }
    if (mode === CAPTCHA_MODE_ROTATE) {
      if (angle == null) {
        return { ok: false, message: '请完成旋转验证', openModal: true };
      }
      return { ok: true };
    }
    if (!dotNum || clicks.length !== dotNum) {
      return { ok: false, message: '请按提示完成全部点击', openModal: true };
    }
    return { ok: true };
  };

  return {
    mode,
    masterSrc,
    thumbSrc,
    loading,
    loadError,
    dotNum,
    thumbSize,
    slideMeta,
    angle,
    slidePoint,
    clicks,
    masterSize,
    setMasterSize,
    modalOpen,
    setModalOpen: handleModalOpenChange,
    loadCaptcha: refreshCaptcha,
    refreshCaptcha,
    onRotateConfirm,
    onClickConfirm,
    onSlideConfirm,
    setClicks,
    getCaptchaPayload,
    getLoginProof,
    isComplete,
    validateBeforeLogin,
    resetCaptcha,
    captchaRef,
  };
};
