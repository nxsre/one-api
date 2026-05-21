import { useCallback, useEffect, useRef, useState } from 'react';
import { API } from '@/api/client';
import { isSecurePasswordLoginEnabled } from '@/lib/systemStatus';

/**
 * 点击验证码逻辑，对齐 default LoginForm
 */
export function useLoginCaptcha({ mathCaptchaEnabled }) {
  const [masterSrc, setMasterSrc] = useState('');
  const [thumbSrc, setThumbSrc] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [dotNum, setDotNum] = useState(0);
  const [challengeId, setChallengeId] = useState('');
  const [clicks, setClicks] = useState([]);
  const [masterSize, setMasterSize] = useState({ w: 0, h: 0 });
  const [modalOpen, setModalOpen] = useState(false);
  const loginProofRef = useRef(null);

  const resetCaptcha = useCallback(() => {
    setMasterSrc('');
    setThumbSrc('');
    setDotNum(0);
    setChallengeId('');
    setClicks([]);
    setMasterSize({ w: 0, h: 0 });
    loginProofRef.current = null;
  }, []);

  const loadCaptcha = useCallback(async () => {
    if (!mathCaptchaEnabled) return;
    setLoadError('');
    setLoading(true);
    try {
      const res = await API.get('/api/user/login/captcha');
      const d = res.data?.data;
      if (res.data?.success && d?.master_image && d?.thumb_image) {
        setMasterSize({ w: 0, h: 0 });
        setMasterSrc(d.master_image);
        setThumbSrc(d.thumb_image);
        setDotNum(Number(d.dot_num) || 0);
        setChallengeId(d.captcha_id || '');
        setClicks([]);
        setLoadError('');
        if (isSecurePasswordLoginEnabled()) {
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
        } else {
          loginProofRef.current = null;
        }
      } else {
        resetCaptcha();
        setLoadError(
          (res.data?.message && String(res.data.message).trim()) || '验证码加载失败'
        );
      }
    } catch {
      resetCaptcha();
      setLoadError('验证码加载失败');
    } finally {
      setLoading(false);
    }
  }, [mathCaptchaEnabled, resetCaptcha]);

  useEffect(() => {
    if (!modalOpen || !mathCaptchaEnabled) return;
    if (masterSrc || loading) return;
    void loadCaptcha();
  }, [modalOpen, mathCaptchaEnabled, masterSrc, loading, loadCaptcha]);

  const onMasterClick = (e) => {
    if (!mathCaptchaEnabled || !dotNum || clicks.length >= dotNum) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const x = Math.round(e.clientX - rect.left);
    const y = Math.round(e.clientY - rect.top);
    setClicks((prev) => [...prev, { x, y }]);
  };

  const getCaptchaPayload = () => {
    if (!mathCaptchaEnabled || !masterSrc) return undefined;
    return {
      captcha_id: challengeId,
      captcha_clicks: clicks,
    };
  };

  const getLoginProof = () => {
    if (!mathCaptchaEnabled) return null;
    return loginProofRef.current;
  };

  const isComplete = dotNum > 0 && clicks.length === dotNum;

  const validateBeforeLogin = () => {
    if (!mathCaptchaEnabled) return { ok: true };
    if (!masterSrc) {
      if (loading) return { ok: false, message: '验证码加载中，请稍候' };
      if (loadError) return { ok: false, message: loadError, openModal: true };
      return { ok: false, message: '请先完成点击验证', openModal: true };
    }
    if (!dotNum || clicks.length !== dotNum) {
      return { ok: false, message: '请按提示完成全部点击', openModal: true };
    }
    return { ok: true };
  };

  return {
    masterSrc,
    thumbSrc,
    loading,
    loadError,
    dotNum,
    clicks,
    masterSize,
    setMasterSize,
    modalOpen,
    setModalOpen,
    loadCaptcha,
    onMasterClick,
    setClicks,
    getCaptchaPayload,
    getLoginProof,
    isComplete,
    validateBeforeLogin,
    resetCaptcha,
  };
}
