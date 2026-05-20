import React, { useContext, useEffect, useState } from 'react';
import {
  Button,
  Form,
  Grid,
  Header,
  Image,
  Message,
  Modal,
  Card,
  Divider,
  Icon,
} from 'semantic-ui-react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { API, getLogo, showError, showInfo, showSuccess } from '../helpers';
import { UserContext } from '../context/User';
import { StatusContext } from '../context/Status';
import Turnstile from 'react-turnstile';
import NacosThemeToggle from './NacosThemeToggle';
import { onGitHubOAuthClicked, onLarkOAuthClicked } from './utils';
import larkIcon from '../images/lark.svg';

const RegisterForm = () => {
  const { t, i18n } = useTranslation();
  const [, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const [inputs, setInputs] = useState({
    username: '',
    password: '',
    password2: '',
    email: '',
    verification_code: '',
    wechat_verification_code: '',
  });
  const { username, password, password2 } = inputs;
  const [showEmailVerification, setShowEmailVerification] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [disableButton, setDisableButton] = useState(false);
  const [countdown, setCountdown] = useState(30);
  const [status, setStatus] = useState({});

  const isEnglishUI =
    i18n.language && String(i18n.language).toLowerCase().startsWith('en');

  const toggleLanguage = async () => {
    await i18n.changeLanguage(isEnglishUI ? 'zh' : 'en');
    window.location.reload();
  };

  const mergeStatus = (data) => {
    if (!data) return;
    setStatus(data);
    try {
      localStorage.setItem('status', JSON.stringify(data));
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    if (statusState.status) {
      mergeStatus(statusState.status);
      return;
    }
    const cached = localStorage.getItem('status');
    if (!cached) return;
    try {
      mergeStatus(JSON.parse(cached));
    } catch {
      /* ignore */
    }
  }, [statusState.status]);
  const logo = getLogo();
  let affCode = new URLSearchParams(window.location.search).get('aff');
  if (affCode) {
    localStorage.setItem('aff', affCode);
  }

  useEffect(() => {
    if (!status || typeof status !== 'object') return;
    setShowEmailVerification(!!status.email_verification);
    if (status.turnstile_check) {
      setTurnstileEnabled(true);
      setTurnstileSiteKey(status.turnstile_site_key || '');
    } else {
      setTurnstileEnabled(false);
    }
  }, [status]);

  useEffect(() => {
    let countdownInterval = null;
    if (disableButton && countdown > 0) {
      countdownInterval = setInterval(() => {
        setCountdown(countdown - 1);
      }, 1000);
    } else if (countdown === 0) {
      setDisableButton(false);
      setCountdown(30);
    }
    return () => clearInterval(countdownInterval);
  }, [disableButton, countdown]);

  const navigate = useNavigate();
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
      navigate('/');
      showSuccess(t('messages.success.login'));
      setShowWeChatLoginModal(false);
    } else {
      showError(message);
    }
  };

  function handleChange(e) {
    const { name, value } = e.target;
    console.log(name, value);
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  }

  async function handleSubmit(e) {
    if (password.length < 8) {
      showInfo(t('messages.error.password_length'));
      return;
    }
    if (password !== password2) {
      showInfo(t('messages.error.password_mismatch'));
      return;
    }
    if (username && password) {
      if (turnstileEnabled && turnstileToken === '') {
        showInfo(t('messages.error.turnstile_wait'));
        return;
      }
      setLoading(true);
      if (!affCode) {
        affCode = localStorage.getItem('aff');
      }
      inputs.aff_code = affCode;
      const res = await API.post(
        `/api/user/register?turnstile=${turnstileToken}`,
        inputs
      );
      const { success, message } = res.data;
      if (success) {
        navigate('/login');
        showSuccess(t('messages.success.register'));
      } else {
        showError(message);
      }
      setLoading(false);
    }
  }

  const sendVerificationCode = async () => {
    if (inputs.email === '') return;
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('messages.error.turnstile_wait'));
      return;
    }
    setDisableButton(true);
    setLoading(true);
    const res = await API.get(
      `/api/verification?email=${inputs.email}&turnstile=${turnstileToken}`
    );
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('messages.success.verification_code'));
    } else {
      showError(message);
      setDisableButton(false);
      setCountdown(30);
    }
    setLoading(false);
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
                <Header.Content>{t('auth.register.title')}</Header.Content>
              </Header>
            </Card.Header>
            <Form size='large'>
              <Form.Input
                fluid
                icon='user'
                iconPosition='left'
                placeholder={t('auth.register.username')}
                onChange={handleChange}
                name='username'
                style={{ marginBottom: '1em' }}
              />
              <Form.Input
                fluid
                icon='lock'
                iconPosition='left'
                placeholder={t('auth.register.password')}
                onChange={handleChange}
                name='password'
                type='password'
                style={{ marginBottom: '1em' }}
              />
              <Form.Input
                fluid
                icon='lock'
                iconPosition='left'
                placeholder={t('auth.register.confirm_password')}
                onChange={handleChange}
                name='password2'
                type='password'
                style={{ marginBottom: '1em' }}
              />

              {showEmailVerification && (
                <>
                  <Form.Input
                    fluid
                    icon='mail'
                    iconPosition='left'
                    placeholder={t('auth.register.email')}
                    onChange={handleChange}
                    name='email'
                    type='email'
                    action={
                      <Button onClick={sendVerificationCode} disabled={loading}>
                        {disableButton
                          ? t('auth.register.get_code_retry', { countdown })
                          : t('auth.register.get_code')}
                      </Button>
                    }
                    style={{ marginBottom: '1em' }}
                  />
                  <Form.Input
                    fluid
                    icon='lock'
                    iconPosition='left'
                    placeholder={t('auth.register.verification_code')}
                    onChange={handleChange}
                    name='verification_code'
                    style={{ marginBottom: '1em' }}
                  />
                </>
              )}

              {turnstileEnabled && (
                <div
                  style={{
                    marginBottom: '1em',
                    display: 'flex',
                    justifyContent: 'center',
                  }}
                >
                  <Turnstile
                    sitekey={turnstileSiteKey}
                    onVerify={(token) => {
                      setTurnstileToken(token);
                    }}
                  />
                </div>
              )}

              <Button
                fluid
                size='large'
                onClick={handleSubmit}
                style={{
                  background: '#2F73FF', // 使用更现代的蓝色
                  color: 'white',
                  marginBottom: '1.5em',
                }}
                loading={loading}
              >
                {t('auth.register.button')}
              </Button>
            </Form>

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
                    marginBottom: '1em',
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

            <Divider />
            <Message style={{ background: 'transparent', boxShadow: 'none' }}>
              <div
                style={{
                  textAlign: 'center',
                  fontSize: '0.9em',
                  color: '#666',
                }}
              >
                {t('auth.register.has_account')}
                <Link to='/login' className='app-register-login-link'>
                  {t('auth.register.login')}
                </Link>
              </div>
            </Message>
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

export default RegisterForm;
