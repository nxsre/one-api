import React, { useCallback, useContext, useEffect, useRef, useState } from 'react';
import {
  Button,
  Divider,
  Form,
  Grid,
  Header,
  Image,
  Message,
  Modal,
  Segment,
  Card,
  Icon,
} from 'semantic-ui-react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../context/User';
import { StatusContext } from '../context/Status';
import {
  API,
  buildLoginPayload,
  getLogo,
  isSecurePasswordLoginEnabled,
  postLoginDefaultPath,
  showError,
  showInfo,
  showSuccess,
  showWarning,
} from '../helpers';
import { onGitHubOAuthClicked, onLarkOAuthClicked } from './utils';
import larkIcon from '../images/lark.svg';
import NacosThemeToggle from './NacosThemeToggle';

/** 仅允许站内相对路径，防止开放重定向 */
function consumeSafeInternalRedirect(searchParams) {
  const raw = searchParams.get('redirect');
  if (!raw) return null;
  let decoded;
  try {
    decoded = decodeURIComponent(String(raw).trim());
  } catch {
    return null;
  }
  if (!decoded.startsWith('/') || decoded.startsWith('//')) return null;
  return decoded;
}

const LoginForm = ({ tenantPortal }) => {
  const { t, i18n } = useTranslation();
  const [inputs, setInputs] = useState({
    username: '',
    password: '',
    tenant_id: '',
    wechat_verification_code: '',
  });
  const [searchParams] = useSearchParams();
  const [submitted, setSubmitted] = useState(false);
  const { username, password, tenant_id } = inputs;
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const navigate = useNavigate();
  const afterLoginNavigate = (defaultPath) => {
    const target = consumeSafeInternalRedirect(searchParams);
    if (target) {
      window.location.assign(target);
      return;
    }
    navigate(defaultPath);
  };
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
  const [showCaptchaModal, setShowCaptchaModal] = useState(false);
  const [twoFACode, setTwoFACode] = useState('');
  const [loginBusy, setLoginBusy] = useState(false);

  const turnstileEnabled = !!status.turnstile_check;

  const isEnglishUI =
    i18n.language && String(i18n.language).toLowerCase().startsWith('en');

  const toggleLanguage = async () => {
    await i18n.changeLanguage(isEnglishUI ? 'zh' : 'en');
    window.location.reload();
  };

  const mergeStatus = useCallback((data) => {
    if (!data) return;
    setStatus(data);
    try {
      localStorage.setItem('status', JSON.stringify(data));
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('messages.error.login_expired'));
    }
  }, [searchParams, t]);

  useEffect(() => {
    if (statusState.status) {
      mergeStatus(statusState.status);
      return;
    }
    const cachedStatus = localStorage.getItem('status');
    if (!cachedStatus) return;
    try {
      mergeStatus(JSON.parse(cachedStatus));
    } catch {
      /* ignore */
    }
  }, [mergeStatus, statusState.status]);

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
        if (isSecurePasswordLoginEnabled()) {
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
            t('auth.login.captcha_load_failed')
        );
      }
    } catch {
      setCaptchaMasterSrc('');
      setCaptchaThumbSrc('');
      setCaptchaDotNum(0);
      setCaptchaChallengeId('');
      setCaptchaClicks([]);
      loginRequestProofRef.current = null;
      setCaptchaLoadError(t('auth.login.captcha_load_failed'));
    } finally {
      setCaptchaLoading(false);
    }
  }, [status.login_math_captcha, turnstileEnabled, t]);

  // 不在进入登录页时请求验证码；仅在打开验证码弹窗且尚未加载时拉取（含从「登录」入口打开弹窗）
  useEffect(() => {
    if (!showCaptchaModal) return;
    if (!status.login_math_captcha || turnstileEnabled) return;
    if (captchaMasterSrc || captchaLoading) return;
    void loadLoginCaptcha();
  }, [
    showCaptchaModal,
    status.login_math_captcha,
    turnstileEnabled,
    captchaMasterSrc,
    captchaLoading,
    loadLoginCaptcha,
  ]);

  const [showWeChatLoginModal, setShowWeChatLoginModal] = useState(false);

  const onWeChatLoginClicked = () => {
    setShowWeChatLoginModal(true);
  };

  const onSubmitWeChatVerificationCode = async () => {
    const res = await API.get(
      `/api/oauth/wechat?code=${inputs.wechat_verification_code}`
    );
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
      localStorage.setItem('user', JSON.stringify(data));
      afterLoginNavigate(postLoginDefaultPath(data));
      showSuccess(t('messages.success.login'));
      setShowWeChatLoginModal(false);
    } else {
      showError(message);
    }
  };

  function handleChange(e) {
    const { name, value } = e.target;
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  }

  const onMasterCaptchaClick = (e) => {
    if (!status.login_math_captcha || turnstileEnabled) return;
    if (!captchaDotNum || captchaClicks.length >= captchaDotNum) return;
    const rect = e.currentTarget.getBoundingClientRect();
    // Backend expects integer coordinates; decimals can trigger "invalid_parameter".
    const x = Math.round(e.clientX - rect.left);
    const y = Math.round(e.clientY - rect.top);
    setCaptchaClicks((prev) => [...prev, { x, y }]);
  };

  async function handleSubmit() {
    setSubmitted(true);
    if (!username || !password) return;

    if (status.login_math_captcha && !turnstileEnabled) {
      if (!captchaMasterSrc) {
        if (captchaLoading) {
          showInfo(t('auth.login.captcha_loading'));
        } else if (captchaLoadError) {
          showInfo(captchaLoadError);
        } else {
          showInfo(t('auth.login.captcha_need_load'));
        }
        setShowCaptchaModal(true);
        return;
      }
      if (!captchaDotNum || captchaClicks.length !== captchaDotNum) {
        showInfo(t('auth.login.captcha_incomplete'));
        setShowCaptchaModal(true);
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
          ? {
              captcha_id: captchaChallengeId,
              captcha_clicks: captchaClicks,
            }
          : undefined;
      const body = await buildLoginPayload(username, password, captcha, proof);
      if (tenantPortal) {
        const tid = String(tenant_id ?? '').trim();
        if (tid !== '') {
          body.tenant_id = tid;
        }
      }
      const res = await API.post(`/api/user/login`, body);
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
          showWarning(
            t('auth.login.force_2fa_hint')
          );
        }
        if (username === 'root' && password === '123456') {
          afterLoginNavigate('/user/edit');
          showSuccess(t('messages.success.login'));
          showWarning(t('messages.error.root_password'));
        } else {
          afterLoginNavigate(data?.require_force_2fa_setup ? '/setting' : postLoginDefaultPath(data));
          showSuccess(t('messages.success.login'));
        }
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
          afterLoginNavigate('/setting');
        } else {
          afterLoginNavigate(postLoginDefaultPath(data));
        }
        showSuccess(t('messages.success.login'));
      } else {
        showError(message);
      }
    } catch {
      showError('验证失败');
    } finally {
      setLoginBusy(false);
    }
  };

  return (
    <>
      <div className='app-public-theme-bar'>
        <Button
          icon
          basic
          size='small'
          className='app-theme-toggle'
          onClick={() => void toggleLanguage()}
          title={t('header.language_switch_tooltip')}
          aria-label={t('header.language_switch_tooltip')}
        >
          <Icon name='language' />
        </Button>
        <NacosThemeToggle />
      </div>
      <Grid textAlign='center' style={{ marginTop: '24px' }}>
        <Grid.Column style={{ maxWidth: 450 }}>
        <Card
          fluid
          className='chart-card'
          style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.12)' }}
        >
          <Card.Content>
            <Card.Header>
              <Header
                as='h2'
                textAlign='center'
                style={{ marginBottom: '1.5em' }}
              >
                <Image src={logo} style={{ marginBottom: '10px' }} />
                <Header.Content>
                  {tenantPortal
                    ? t('auth.login.tenant_title')
                    : t('auth.login.title')}
                </Header.Content>
              </Header>
            </Card.Header>
            {tenantPortal ? (
              <Message
                info
                size='small'
                style={{ marginBottom: '1em', textAlign: 'left' }}
              >
                {t('auth.login.tenant_portal_hint')}
              </Message>
            ) : null}
            <Form size='large'>
              {tenantPortal ? (
                <Form.Input
                  fluid
                  icon='building'
                  iconPosition='left'
                  placeholder='租户ID (Tenant ID)'
                  name='tenant_id'
                  type='text'
                  value={tenant_id}
                  onChange={handleChange}
                  style={{ marginBottom: '1em' }}
                />
              ) : null}
              <Form.Input
                fluid
                icon='user'
                iconPosition='left'
                placeholder={t('auth.login.username')}
                name='username'
                value={username}
                onChange={handleChange}
                style={{ marginBottom: '1em' }}
              />
              <Form.Input
                fluid
                icon='lock'
                iconPosition='left'
                placeholder={t('auth.login.password')}
                name='password'
                type='password'
                value={password}
                onChange={handleChange}
                style={{ marginBottom: '1em' }}
              />

              {status.login_math_captcha && !turnstileEnabled && (
              <Segment className='auth-captcha-segment'>
                  <div
                    className='auth-captcha-title'
                    style={{
                      display: 'flex',
                      fontSize: 13,
                      fontWeight: 600,
                      justifyContent: 'space-between',
                      marginBottom: 8,
                    }}
                  >
                    <span>{t('auth.login.captcha_title')}</span>
                    <span className='auth-captcha-sub' style={{ fontWeight: 500 }}>
                      {t('auth.login.captcha_progress', {
                        n: captchaClicks.length,
                        m: captchaDotNum || '—',
                      })}
                    </span>
                  </div>
                  <Button
                    fluid
                    type='button'
                    primary={
                      captchaDotNum > 0 && captchaClicks.length === captchaDotNum
                    }
                    onClick={() => setShowCaptchaModal(true)}
                  >
                    {captchaDotNum > 0 && captchaClicks.length === captchaDotNum
                      ? t('auth.login.captcha_done')
                      : t('auth.login.captcha_open')}
                  </Button>
                  {captchaLoadError ? (
                    <Message
                      negative
                      size='small'
                      style={{ marginBottom: 0, marginTop: 8 }}
                    >
                      {captchaLoadError}
                    </Message>
                  ) : null}
                </Segment>
              )}

              <Button
                fluid
                size='large'
                loading={loginBusy}
                disabled={loginBusy}
                style={{
                  background: '#2F73FF',
                  color: 'white',
                  marginBottom: '1.5em',
                }}
                onClick={handleSubmit}
              >
                {t('auth.login.button')}
              </Button>
            </Form>

            <Modal open={showTwoFA} size='small' onClose={() => setShowTwoFA(false)}>
              <Modal.Header>两步验证</Modal.Header>
              <Modal.Content>
                <p>请输入认证器 6 位验证码或 8 位备用码</p>
                <Form.Input
                  fluid
                  placeholder='验证码 / 备用码'
                  value={twoFACode}
                  onChange={(e) => setTwoFACode(e.target.value)}
                />
              </Modal.Content>
              <Modal.Actions>
                <Button onClick={() => setShowTwoFA(false)}>取消</Button>
                <Button primary loading={loginBusy} onClick={submitTwoFA}>
                  确认登录
                </Button>
              </Modal.Actions>
            </Modal>

            <Modal
              open={showCaptchaModal}
              onClose={() => setShowCaptchaModal(false)}
              style={{
                maxWidth: '92vw',
                minWidth: 'unset',
                width: 'fit-content',
              }}
            >
              <Modal.Header>{t('auth.login.captcha_modal_title')}</Modal.Header>
              <Modal.Content
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  width: 'fit-content',
                }}
              >
                {captchaThumbSrc ? (
                  <div
                    style={{
                      alignItems: 'center',
                      background: '#f9fafb',
                      border: '1px solid #d1d5db',
                      borderRadius: 8,
                      display: 'inline-flex',
                      marginBottom: 10,
                      padding: '3px 8px',
                    }}
                  >
                    <img
                      alt='thumb'
                      src={captchaThumbSrc}
                      style={{ display: 'block', maxHeight: 52 }}
                    />
                  </div>
                ) : null}
                {captchaMasterSrc ? (
                  <div
                    style={{
                      alignItems: 'center',
                      background: '#f9fafb',
                      border: '1px solid #d1d5db',
                      borderRadius: 10,
                      display: 'flex',
                      justifyContent: 'center',
                      padding: 6,
                    }}
                  >
                    <div
                      style={{
                        display: 'inline-block',
                        lineHeight: 0,
                        position: 'relative',
                      }}
                    >
                      <img
                        alt='captcha'
                        src={captchaMasterSrc}
                        style={{
                          borderRadius: 6,
                          cursor: 'crosshair',
                          display: 'block',
                          maxHeight: 380,
                          maxWidth: '100%',
                          objectFit: 'contain',
                        }}
                        onLoad={(ev) => {
                          setCaptchaMasterNaturalSize({
                            h: ev.target.naturalHeight,
                            w: ev.target.naturalWidth,
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
                              alignItems: 'center',
                              background: 'rgba(37, 99, 235, 0.22)',
                              border: '2px solid #2563eb',
                              borderRadius: '50%',
                              color: '#fff',
                              display: 'flex',
                              fontSize: 12,
                              fontWeight: 700,
                              height: 22,
                              justifyContent: 'center',
                              left: `${(p.x / captchaMasterNaturalSize.w) * 100}%`,
                              position: 'absolute',
                              top: `${(p.y / captchaMasterNaturalSize.h) * 100}%`,
                              transform: 'translate(-50%, -50%)',
                              width: 22,
                            }}
                          >
                            {i + 1}
                          </span>
                        ))}
                    </div>
                  </div>
                ) : (
                  <Message size='small' style={{ marginBottom: 0, marginTop: 0 }}>
                    {captchaLoading
                      ? t('auth.login.captcha_loading')
                      : captchaLoadError || t('auth.login.captcha_refresh_hint')}
                  </Message>
                )}

                <div
                  style={{
                    color: '#6b7280',
                    fontSize: 13,
                    fontWeight: 500,
                    marginTop: 10,
                    textAlign: 'right',
                  }}
                >
                  {t('auth.login.captcha_progress', {
                    n: captchaClicks.length,
                    m: captchaDotNum || '—',
                  })}
                </div>

                <Grid columns={2} stackable style={{ marginBottom: 8, marginTop: 8 }}>
                  <Grid.Column style={{ paddingBottom: 0, paddingTop: 0 }}>
                    <Button fluid type='button' onClick={() => setCaptchaClicks([])}>
                      {t('auth.login.captcha_clear')}
                    </Button>
                  </Grid.Column>
                  <Grid.Column style={{ paddingBottom: 0, paddingTop: 0 }}>
                    <Button type='button' fluid primary onClick={() => void loadLoginCaptcha()}>
                      {t('auth.login.captcha_refresh')}
                    </Button>
                  </Grid.Column>
                </Grid>
              </Modal.Content>
              <Modal.Actions>
                <Button onClick={() => setShowCaptchaModal(false)}>
                  {t('auth.login.captcha_close')}
                </Button>
              </Modal.Actions>
            </Modal>

            <Divider />
            <Message style={{ background: 'transparent', boxShadow: 'none' }}>
              {tenantPortal ? (
                <div className='login-form-footer login-form-footer--tenant'>
                  <Link to='/reset' className='login-form-footer-link'>
                    {t('auth.login.tenant_footer_reset')}
                  </Link>
                  <Link to='/login' className='login-form-footer-link'>
                    {t('auth.login.platform_login_link')}
                  </Link>
                </div>
              ) : (
                <div className='login-form-footer'>
                  <span className='login-form-footer-line'>
                    {t('auth.login.forgot_password')}
                    <Link to='/reset' className='login-form-footer-link'>
                      {t('auth.login.reset_password')}
                    </Link>
                  </span>
                  <span className='login-form-footer-line'>
                    {t('auth.login.no_account')}
                    <Link to='/register' className='login-form-footer-link'>
                      {t('auth.login.register')}
                    </Link>
                    <span className='login-form-footer-sep'>·</span>
                    <Link to='/tenant-login' className='login-form-footer-link'>
                      {t('auth.login.tenant_login_link')}
                    </Link>
                  </span>
                </div>
              )}
            </Message>

            {(status.github_oauth ||
              status.wechat_login ||
              (status.lark_oauth && status.lark_client_id)) && (
              <>
                <Divider
                  horizontal
                  style={{ color: '#666', fontSize: '0.9em' }}
                >
                  {t('auth.login.other_methods')}
                </Divider>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'center',
                    gap: '1em',
                    marginTop: '1em',
                  }}
                >
                  {status.github_oauth && (
                    <Button
                      circular
                      color='black'
                      icon='github'
                      onClick={() =>
                        onGitHubOAuthClicked(status.github_client_id)
                      }
                    />
                  )}
                  {status.wechat_login && (
                    <Button
                      circular
                      color='green'
                      icon='wechat'
                      aria-label={t('auth.login.wechat.entry')}
                      title={t('auth.login.wechat.entry')}
                      onClick={onWeChatLoginClicked}
                    />
                  )}
                  {status.lark_oauth && status.lark_client_id && (
                    <div
                      style={{
                        background:
                          'radial-gradient(circle, #FFFFFF, #FFFFFF, #FFFFFF, #FFFFFF, #FFFFFF)',
                        width: '36px',
                        height: '36px',
                        borderRadius: '10em',
                        display: 'flex',
                        cursor: 'pointer',
                      }}
                      onClick={() => onLarkOAuthClicked(status.lark_client_id)}
                    >
                      <Image
                        src={larkIcon}
                        avatar
                        style={{
                          width: '36px',
                          height: '36px',
                          cursor: 'pointer',
                          margin: 'auto',
                        }}
                      />
                    </div>
                  )}
                </div>
              </>
            )}
          </Card.Content>
        </Card>
        <Modal
          onClose={() => setShowWeChatLoginModal(false)}
          onOpen={() => setShowWeChatLoginModal(true)}
          open={showWeChatLoginModal}
          size='mini'
        >
          <Modal.Content>
            <Modal.Description style={{ textAlign: 'center' }}>
              <Header as='h3' style={{ marginBottom: 6 }}>
                {t('auth.login.wechat.title')}
              </Header>
              <p style={{ color: '#6b7280', marginTop: 0 }}>
                {t('auth.login.wechat.subtitle')}
              </p>
              <div
                style={{
                  background:
                    'linear-gradient(180deg, #f8fafc 0%, #ffffff 100%)',
                  border: '1px solid #e5e7eb',
                  borderRadius: 16,
                  boxShadow: '0 12px 30px rgba(15, 23, 42, 0.08)',
                  margin: '0 auto 1em',
                  maxWidth: 260,
                  padding: 16,
                }}
              >
                {status.wechat_qrcode ? (
                  <Image
                    src={status.wechat_qrcode}
                    alt={t('auth.login.wechat.qrcode_alt')}
                    centered
                    style={{
                      background: '#fff',
                      borderRadius: 12,
                      boxShadow: 'inset 0 0 0 1px rgba(0, 0, 0, 0.04)',
                      maxHeight: 220,
                      objectFit: 'contain',
                      padding: 8,
                      width: '100%',
                    }}
                  />
                ) : (
                  <Message warning style={{ margin: 0 }}>
                    {t('auth.login.wechat.qrcode_missing')}
                  </Message>
                )}
              </div>
              <Message
                info
                size='small'
                style={{ textAlign: 'left', lineHeight: 1.6 }}
              >
                {t('auth.login.wechat.scan_tip')}
              </Message>
              <Form size='large'>
                <Form.Input
                  fluid
                  label={t('auth.login.wechat.code_label')}
                  placeholder={t('auth.login.wechat.code_placeholder')}
                  name='wechat_verification_code'
                  value={inputs.wechat_verification_code}
                  onChange={handleChange}
                />
                <Button
                  fluid
                  size='large'
                  style={{
                    background: '#2F73FF',
                    color: 'white',
                    marginBottom: '1.5em',
                  }}
                  onClick={onSubmitWeChatVerificationCode}
                >
                  {t('auth.login.wechat.submit')}
                </Button>
              </Form>
            </Modal.Description>
          </Modal.Content>
        </Modal>
      </Grid.Column>
    </Grid>
    </>
  );
};

export default LoginForm;
