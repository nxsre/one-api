import React, { useCallback, useContext, useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { UserContext } from '../context/User';
import {
  API,
  buildLoginPayload,
  showError,
  showInfo,
  showSuccess,
  showWarning,
} from '../helpers';
import { onGitHubOAuthClicked } from './utils';
import Turnstile from 'react-turnstile';
import {
  Button,
  Card,
  Divider,
  Form,
  Icon,
  Input,
  Layout,
  Modal,
  Typography,
} from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import TelegramLoginButton from 'react-telegram-login';

import { IconGithubLogo } from '@douyinfe/semi-icons';
import WeChatIcon from './WeChatIcon';

const LoginForm = () => {
  const [inputs, setInputs] = useState({
    username: '',
    password: '',
    wechat_verification_code: '',
  });
  const [searchParams] = useSearchParams();
  const [submitted, setSubmitted] = useState(false);
  const { username, password } = inputs;
  const [userState, userDispatch] = useContext(UserContext);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const navigate = useNavigate();
  const [status, setStatus] = useState({});
  const logo = getLogo();

  const [captchaMasterSrc, setCaptchaMasterSrc] = useState('');
  const [captchaThumbSrc, setCaptchaThumbSrc] = useState('');
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [captchaLoadError, setCaptchaLoadError] = useState('');
  const [captchaDotNum, setCaptchaDotNum] = useState(0);
  const [captchaChallengeId, setCaptchaChallengeId] = useState('');
  const [captchaClicks, setCaptchaClicks] = useState([]);
  const [captchaMasterNaturalSize, setCaptchaMasterNaturalSize] = useState({
    w: 0,
    h: 0,
  });
  const loginRequestProofRef = useRef(null);

  const [showTwoFA, setShowTwoFA] = useState(false);
  const [twoFACode, setTwoFACode] = useState('');
  const [loginBusy, setLoginBusy] = useState(false);

  const mergeStatus = useCallback((data) => {
    if (!data) return;
    setStatus(data);
    if (data.turnstile_check) {
      setTurnstileEnabled(true);
      setTurnstileSiteKey(data.turnstile_site_key);
    }
    try {
      localStorage.setItem('status', JSON.stringify(data));
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError('未登录或登录已过期，请重新登录！');
    }
    (async () => {
      try {
        const res = await API.get('/api/status');
        if (res.data?.success && res.data.data) {
          mergeStatus(res.data.data);
        }
      } catch {
        const s = localStorage.getItem('status');
        if (s) {
          try {
            mergeStatus(JSON.parse(s));
          } catch {
            /* ignore */
          }
        }
      }
    })();
  }, [mergeStatus, searchParams]);

  const loadLoginCaptcha = useCallback(async () => {
    if (!status.login_math_captcha || turnstileEnabled) return;
    setCaptchaLoadError('');
    setCaptchaLoading(true);
    try {
      const res = await API.get('/api/user/login/captcha');
      const d = res.data?.data;
      if (res.data?.success && d?.master_image && d?.thumb_image) {
        setCaptchaMasterNaturalSize({ w: 0, h: 0 });
        setCaptchaMasterSrc(d.master_image);
        setCaptchaThumbSrc(d.thumb_image);
        setCaptchaDotNum(Number(d.dot_num) || 0);
        setCaptchaChallengeId(d.captcha_id || '');
        setCaptchaClicks([]);
        setCaptchaLoadError('');
        const secureLogin = (() => {
          try {
            const st = JSON.parse(localStorage.getItem('status') || '{}');
            return (
              st.secure_password_login === true ||
              st.secure_password_login === 'true'
            );
          } catch {
            return false;
          }
        })();
        if (secureLogin) {
          if (
            d.login_request_id &&
            d.login_request_sig != null &&
            d.login_request_ts != null &&
            d.login_enc_key
          ) {
            loginRequestProofRef.current = {
              id: d.login_request_id,
              ts: Number(d.login_request_ts),
              sig: d.login_request_sig,
              encKey: d.login_enc_key,
            };
          } else {
            loginRequestProofRef.current = null;
          }
        } else {
          loginRequestProofRef.current = null;
        }
      } else {
        setCaptchaMasterSrc('');
        setCaptchaThumbSrc('');
        setCaptchaDotNum(0);
        setCaptchaChallengeId('');
        setCaptchaClicks([]);
        loginRequestProofRef.current = null;
        setCaptchaLoadError(
          (res.data?.message && String(res.data.message).trim()) ||
            '验证码加载失败，请稍后点击「刷新题目」重试'
        );
      }
    } catch {
      setCaptchaMasterSrc('');
      setCaptchaThumbSrc('');
      setCaptchaDotNum(0);
      setCaptchaChallengeId('');
      setCaptchaClicks([]);
      loginRequestProofRef.current = null;
      setCaptchaLoadError('验证码加载失败，请稍后点击「刷新题目」重试');
    } finally {
      setCaptchaLoading(false);
    }
  }, [status.login_math_captcha, turnstileEnabled]);

  useEffect(() => {
    if (status.login_math_captcha && !turnstileEnabled) {
      void loadLoginCaptcha();
    }
  }, [status.login_math_captcha, turnstileEnabled, loadLoginCaptcha]);

  const [showWeChatLoginModal, setShowWeChatLoginModal] = useState(false);

  const onWeChatLoginClicked = () => {
    setShowWeChatLoginModal(true);
  };

  const onSubmitWeChatVerificationCode = async () => {
    if (turnstileEnabled && turnstileToken === '') {
      showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
      return;
    }
    const res = await API.get(
      `/api/oauth/wechat?code=${inputs.wechat_verification_code}`
    );
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
      localStorage.setItem('user', JSON.stringify(data));
      navigate('/');
      showSuccess('登录成功！');
      setShowWeChatLoginModal(false);
    } else {
      showError(message);
    }
  };

  function handleChange(name, value) {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  }

  const onMasterCaptchaClick = (e) => {
    if (!status.login_math_captcha || turnstileEnabled) return;
    if (!captchaDotNum || captchaClicks.length >= captchaDotNum) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    setCaptchaClicks((prev) => [...prev, { x, y }]);
  };

  async function handleSubmit(e) {
    if (turnstileEnabled && turnstileToken === '') {
      showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
      return;
    }
    setSubmitted(true);
    if (!username || !password) {
      showError('请输入用户名和密码！');
      return;
    }

    if (status.login_math_captcha && !turnstileEnabled) {
      if (!captchaMasterSrc) {
        if (captchaLoading) showInfo('加载验证码中…');
        else if (captchaLoadError) showInfo(captchaLoadError);
        else showInfo('请先加载验证码，或点击「刷新题目」');
        return;
      }
      if (!captchaDotNum || captchaClicks.length !== captchaDotNum) {
        showInfo('请按顺序完成图形验证码');
        return;
      }
    }

    let proof = loginRequestProofRef.current;
    if (!(status.login_math_captcha && !turnstileEnabled)) {
      proof = null;
    }

    setLoginBusy(true);
    try {
      const captcha =
        status.login_math_captcha && !turnstileEnabled && captchaMasterSrc
          ? { captcha_id: captchaChallengeId, captcha_clicks: captchaClicks }
          : undefined;
      const body = await buildLoginPayload(username, password, captcha, proof);
      const res = await API.post(
        `/api/user/login?turnstile=${encodeURIComponent(turnstileToken || '')}`,
        body
      );
      const { success, message, data } = res.data;
      if (success) {
        if (data?.require_2fa) {
          setShowTwoFA(true);
          setTwoFACode('');
          return;
        }
        userDispatch({ type: 'login', payload: data });
        localStorage.setItem('user', JSON.stringify(data));
        if (data?.require_force_2fa_setup) {
          showWarning('根据安全策略，请前往个人设置完成两步验证（TOTP）配置');
        }
        showSuccess('登录成功！');
        if (username === 'root' && password === '123456') {
          Modal.error({
            title: '您正在使用默认密码！',
            content: '请立刻修改默认密码！',
            centered: true,
          });
        }
        navigate(data?.require_force_2fa_setup ? '/setting' : '/token');
      } else {
        showError(message);
        if (status.login_math_captcha && !turnstileEnabled) {
          void loadLoginCaptcha();
        }
      }
    } catch (err) {
      showError(err.message || '登录准备失败');
      if (status.login_math_captcha && !turnstileEnabled) {
        void loadLoginCaptcha();
      }
    } finally {
      setLoginBusy(false);
    }
  }

  const submitTwoFA = async () => {
    if (!twoFACode.trim()) {
      showWarning('请输入验证码或备用码');
      return;
    }
    setLoginBusy(true);
    try {
      const res = await API.post('/api/user/login/2fa', {
        code: twoFACode.trim(),
      });
      const { success, message, data } = res.data;
      if (success) {
        setShowTwoFA(false);
        userDispatch({ type: 'login', payload: data });
        localStorage.setItem('user', JSON.stringify(data));
        if (data?.require_force_2fa_setup) {
          showWarning('请前往个人设置完成两步验证配置');
          navigate('/setting');
        } else {
          navigate('/token');
        }
        showSuccess('登录成功！');
      } else {
        showError(message);
      }
    } catch {
      showError('验证失败');
    } finally {
      setLoginBusy(false);
    }
  };

  const onTelegramLoginClicked = async (response) => {
    const fields = [
      'id',
      'first_name',
      'last_name',
      'username',
      'photo_url',
      'auth_date',
      'hash',
      'lang',
    ];
    const params = {};
    fields.forEach((field) => {
      if (response[field]) {
        params[field] = response[field];
      }
    });
    const res = await API.get(`/api/oauth/telegram/login`, { params });
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
      localStorage.setItem('user', JSON.stringify(data));
      showSuccess('登录成功！');
      navigate('/');
    } else {
      showError(message);
    }
  };

  return (
    <div>
      <Layout>
        <Layout.Header></Layout.Header>
        <Layout.Content>
          <div
            style={{ justifyContent: 'center', display: 'flex', marginTop: 120 }}
          >
            <div style={{ width: 500 }}>
              <Card>
                <Title heading={2} style={{ textAlign: 'center' }}>
                  用户登录
                </Title>
                <Form>
                  <Form.Input
                    field={'username'}
                    label={'用户名'}
                    placeholder="用户名"
                    name="username"
                    onChange={(value) => handleChange('username', value)}
                  />
                  <Form.Input
                    field={'password'}
                    label={'密码'}
                    placeholder="密码"
                    name="password"
                    type="password"
                    onChange={(value) => handleChange('password', value)}
                  />

                  {status.login_math_captcha && !turnstileEnabled && (
                    <div style={{ marginTop: 12, marginBottom: 12 }}>
                      <Typography.Text type="secondary">
                        图形验证：请按缩略图顺序点击主图对应位置
                      </Typography.Text>
                      {captchaThumbSrc ? (
                        <div style={{ marginTop: 8 }}>
                          <img
                            alt="thumb"
                            src={captchaThumbSrc}
                            style={{ maxHeight: 56 }}
                          />
                        </div>
                      ) : null}
                      {captchaMasterSrc ? (
                        <div
                          style={{
                            position: 'relative',
                            display: 'inline-block',
                            marginTop: 8,
                          }}
                        >
                          <img
                            alt="captcha"
                            src={captchaMasterSrc}
                            style={{ maxWidth: '100%', cursor: 'crosshair' }}
                            onLoad={(ev) => {
                              setCaptchaMasterNaturalSize({
                                w: ev.target.naturalWidth,
                                h: ev.target.naturalHeight,
                              });
                            }}
                            onClick={onMasterCaptchaClick}
                          />
                          {captchaMasterNaturalSize.w > 0 &&
                            captchaMasterNaturalSize.h > 0 &&
                            captchaClicks.map((p, i) => (
                              <span
                                key={i}
                                style={{
                                  position: 'absolute',
                                  left: `${(p.x / captchaMasterNaturalSize.w) * 100}%`,
                                  top: `${(p.y / captchaMasterNaturalSize.h) * 100}%`,
                                  transform: 'translate(-50%, -50%)',
                                  width: 22,
                                  height: 22,
                                  borderRadius: '50%',
                                  border: '2px solid var(--semi-color-primary)',
                                  background: 'rgba(0,100,255,0.2)',
                                  color: '#fff',
                                  fontSize: 12,
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'center',
                                  fontWeight: 700,
                                }}
                              >
                                {i + 1}
                              </span>
                            ))}
                        </div>
                      ) : (
                        <div style={{ marginTop: 8, color: '#888' }}>
                          {captchaLoading
                            ? '加载验证码中…'
                            : captchaLoadError || '点击「刷新题目」加载验证码'}
                        </div>
                      )}
                      <div style={{ marginTop: 8, fontSize: 13 }}>
                        已点击 {captchaClicks.length}/{captchaDotNum || '—'}
                      </div>
                      <Button.Group style={{ marginTop: 8 }}>
                        <Button onClick={() => setCaptchaClicks([])}>
                          清除点击
                        </Button>
                        <Button
                          theme="solid"
                          onClick={() => void loadLoginCaptcha()}
                        >
                          刷新题目
                        </Button>
                      </Button.Group>
                    </div>
                  )}

                  <Button
                    theme="solid"
                    style={{ width: '100%', marginTop: 12 }}
                    type={'primary'}
                    size="large"
                    htmlType={'submit'}
                    loading={loginBusy}
                    disabled={loginBusy}
                    onClick={handleSubmit}
                  >
                    登录
                  </Button>
                </Form>
                <Modal
                  title="两步验证"
                  visible={showTwoFA}
                  onCancel={() => setShowTwoFA(false)}
                  onOk={submitTwoFA}
                  okText="确认登录"
                  confirmLoading={loginBusy}
                >
                  <p>请输入认证器 6 位验证码或 8 位备用码</p>
                  <Input
                    value={twoFACode}
                    onChange={(v) => setTwoFACode(v)}
                    placeholder="验证码 / 备用码"
                  />
                </Modal>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    marginTop: 20,
                  }}
                >
                  <Text>
                    没有账号请先 <Link to="/register">注册账号</Link>
                  </Text>
                  <Text>
                    忘记密码 <Link to="/reset">点击重置</Link>
                  </Text>
                </div>
                {status.github_oauth ||
                status.wechat_login ||
                status.telegram_oauth ? (
                  <>
                    <Divider margin="12px" align="center">
                      第三方登录
                    </Divider>
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'center',
                        marginTop: 20,
                      }}
                    >
                      {status.github_oauth ? (
                        <Button
                          type="primary"
                          icon={<IconGithubLogo />}
                          onClick={() =>
                            onGitHubOAuthClicked(status.github_client_id)
                          }
                        />
                      ) : (
                        <></>
                      )}
                      {status.wechat_login ? (
                        <Button
                          type="primary"
                          style={{ color: 'rgba(var(--semi-green-5), 1)' }}
                          icon={<Icon svg={<WeChatIcon />} />}
                          onClick={onWeChatLoginClicked}
                        />
                      ) : (
                        <></>
                      )}

                      {status.telegram_oauth ? (
                        <TelegramLoginButton
                          dataOnauth={onTelegramLoginClicked}
                          botName={status.telegram_bot_name}
                        />
                      ) : (
                        <></>
                      )}
                    </div>
                  </>
                ) : (
                  <></>
                )}
                <Modal
                  title="微信扫码登录"
                  visible={showWeChatLoginModal}
                  maskClosable={true}
                  onOk={onSubmitWeChatVerificationCode}
                  onCancel={() => setShowWeChatLoginModal(false)}
                  okText={'登录'}
                  size={'small'}
                  centered={true}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItem: 'center',
                      flexDirection: 'column',
                    }}
                  >
                    <img src={status.wechat_qrcode} alt="" />
                  </div>
                  <div style={{ textAlign: 'center' }}>
                    <p>
                      微信扫码关注公众号，输入「验证码」获取验证码（三分钟内有效）
                    </p>
                  </div>
                  <Form size="large">
                    <Form.Input
                      field={'wechat_verification_code'}
                      placeholder="验证码"
                      label={'验证码'}
                      value={inputs.wechat_verification_code}
                      onChange={(value) =>
                        handleChange('wechat_verification_code', value)
                      }
                    />
                  </Form>
                </Modal>
              </Card>
              {turnstileEnabled ? (
                <div
                  style={{ display: 'flex', justifyContent: 'center', marginTop: 20 }}
                >
                  <Turnstile
                    sitekey={turnstileSiteKey}
                    onVerify={(token) => {
                      setTurnstileToken(token);
                    }}
                  />
                </div>
              ) : (
                <></>
              )}
            </div>
          </div>
        </Layout.Content>
      </Layout>
    </div>
  );
};

export default LoginForm;
