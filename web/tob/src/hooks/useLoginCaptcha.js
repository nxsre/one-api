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

/**
 * 登录图形验证码（与 vue LoginForm + CaptchaWidget 逻辑一致）
 */
export function useLoginCaptcha({ mathCaptchaEnabled }) {
  const [challenge, setChallenge] = useState(null);
  const [answer, setAnswer] = useState(null);
  const [ready, setReady] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [modalOpen, setModalOpen] = useState(false);

  const loginProofRef = useRef(null);
  const captchaWidgetRef = useRef(null);
  const loadingRef = useRef(false);

  const resetCaptcha = useCallback(() => {
    setChallenge(null);
    setAnswer(null);
    setReady(false);
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

  const loadCaptcha = useCallback(async () => {
    if (!mathCaptchaEnabled || loadingRef.current) return;

    loadingRef.current = true;
    setLoadError('');
    setLoading(true);
    try {
      const res = await API.get('/api/user/login/captcha');
      const d = res.data?.data;
      if (res.data?.success && d?.master_image) {
        setChallenge({ ...d, mode: resolveCaptchaMode(d) });
        setAnswer(null);
        setReady(false);
        setLoadError('');
        applyProof(d);
      } else {
        resetCaptcha();
        setLoadError(
          (res.data?.message && String(res.data.message).trim()) || '验证码加载失败'
        );
      }
    } catch (err) {
      resetCaptcha();
      setLoadError(captchaErrorMessage(err));
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
  }, [mathCaptchaEnabled, resetCaptcha, applyProof]);

  const refreshCaptcha = useCallback(() => {
    if (loadingRef.current) return;
    resetCaptcha();
    void loadCaptcha();
  }, [loadCaptcha, resetCaptcha]);

  const handleModalOpenChange = useCallback(
    (open) => {
      setModalOpen(open);
      if (!open || !mathCaptchaEnabled) return;
      if (challenge || loadingRef.current) return;
      void loadCaptcha();
    },
    [challenge, loadCaptcha, mathCaptchaEnabled]
  );

  const clearWidget = useCallback(() => {
    captchaWidgetRef.current?.clear?.();
  }, []);

  const getLiveState = useCallback(() => {
    const widgetReady = captchaWidgetRef.current?.isReady?.() ?? ready;
    const widgetAnswer = captchaWidgetRef.current?.getAnswer?.() ?? answer;
    return { ready: widgetReady, answer: widgetAnswer };
  }, [ready, answer]);

  const getCaptchaPayload = useCallback(() => {
    if (!mathCaptchaEnabled || !challenge?.captcha_id) return undefined;
    const { ready: widgetReady, answer: widgetAnswer } = getLiveState();
    if (!widgetReady || widgetAnswer == null) return undefined;
    if (Array.isArray(widgetAnswer) && widgetAnswer.length === 0) return undefined;
    return {
      captcha_id: challenge.captcha_id,
      mode: challenge.mode,
      answer: widgetAnswer,
    };
  }, [mathCaptchaEnabled, challenge, getLiveState]);

  const getLoginProof = useCallback(() => {
    if (!mathCaptchaEnabled) return null;
    return loginProofRef.current;
  }, [mathCaptchaEnabled]);

  const confirmCaptcha = useCallback(() => {
    const { ready: widgetReady, answer: widgetAnswer } = getLiveState();
    if (!widgetReady || widgetAnswer == null) return;
    if (Array.isArray(widgetAnswer) && widgetAnswer.length === 0) return;
    setAnswer(widgetAnswer);
    setReady(true);
    setModalOpen(false);
  }, [getLiveState]);

  const validateBeforeLogin = useCallback(() => {
    if (!mathCaptchaEnabled) return { ok: true };
    if (!challenge) {
      if (loading) return { ok: false, message: '验证码加载中，请稍候', openModal: true };
      if (loadError) return { ok: false, message: loadError, openModal: true };
      return { ok: false, message: '请先完成安全验证', openModal: true };
    }
    const { ready: widgetReady } = getLiveState();
    if (!widgetReady) {
      return { ok: false, message: '请完成安全验证', openModal: true };
    }
    return { ok: true };
  }, [mathCaptchaEnabled, challenge, loading, loadError, getLiveState]);

  return {
    challenge,
    answer,
    ready,
    loading,
    loadError,
    modalOpen,
    setModalOpen: handleModalOpenChange,
    setAnswer,
    setReady,
    loadCaptcha: refreshCaptcha,
    refreshCaptcha,
    clearWidget,
    confirmCaptcha,
    getCaptchaPayload,
    getLoginProof,
    isComplete: ready,
    validateBeforeLogin,
    resetCaptcha,
    captchaWidgetRef,
  };
}
