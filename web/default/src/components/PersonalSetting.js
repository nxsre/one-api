import React, { useContext, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Divider,
  Form,
  Header,
  Image,
  Message,
  Modal,
} from 'semantic-ui-react';
import { Link, useNavigate } from 'react-router-dom';
import {
  API,
  clearNacosEmbeddedConsoleLocalSession,
  copy,
  showError,
  showInfo,
  showNotice,
  showSuccess,
} from '../helpers';
import Turnstile from 'react-turnstile';
import { UserContext } from '../context/User';
import { onGitHubOAuthClicked, onLarkOAuthClicked } from './utils';
import TwoFASetting from './TwoFASetting';

const PersonalSetting = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  let navigate = useNavigate();

  const [inputs, setInputs] = useState({
    wechat_verification_code: '',
    email_verification_code: '',
    email: '',
    self_account_deletion_confirmation: '',
  });
  const [status, setStatus] = useState({});
  const [showWeChatBindModal, setShowWeChatBindModal] = useState(false);
  const [showEmailBindModal, setShowEmailBindModal] = useState(false);
  const [showAccountDeleteModal, setShowAccountDeleteModal] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [disableButton, setDisableButton] = useState(false);
  const [countdown, setCountdown] = useState(30);
  const [affLink, setAffLink] = useState('');
  const [systemToken, setSystemToken] = useState('');
  const [s3Info, setS3Info] = useState(null);
  const [s3SecretModal, setS3SecretModal] = useState({
    open: false,
    accessKey: '',
    secretKey: '',
    subtitle: '',
  });

  useEffect(() => {
    let status = localStorage.getItem('status');
    if (status) {
      status = JSON.parse(status);
      setStatus(status);
      if (status.turnstile_check) {
        setTurnstileEnabled(true);
        setTurnstileSiteKey(status.turnstile_site_key);
      }
    }
  }, []);

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
    return () => clearInterval(countdownInterval); // Clean up on unmount
  }, [disableButton, countdown]);

  const loadS3Self = async () => {
    try {
      const res = await API.get('/api/user/self');
      const { success, message, data } = res.data;
      if (success) {
        setS3Info({
          site: data.s3_site_enabled,
          enabled: data.s3_enabled,
          region: data.s3_region,
          accessKey: data.s3_access_key || '',
        });
      } else {
        showError(message);
      }
    } catch {
      /* interceptor may have shown error */
    }
  };

  useEffect(() => {
    loadS3Self().then();
  }, []);

  const openS3SecretModal = (accessKey, secretKey, subtitle) => {
    setS3SecretModal({ open: true, accessKey, secretKey, subtitle });
  };

  const s3Enable = async () => {
    const res = await API.post('/api/user/s3/enable');
    const { success, message, data } = res.data;
    if (success) {
      openS3SecretModal(
        data.access_key,
        data.secret_key,
        t('setting.personal.s3.enable')
      );
      await loadS3Self();
      showSuccess('临时 S3 已启用');
    } else {
      showError(message);
    }
  };

  const s3Disable = async () => {
    if (!window.confirm('确定关闭并作废当前 S3 密钥？已存储的对象不会自动删除。')) {
      return;
    }
    const res = await API.post('/api/user/s3/disable');
    const { success, message } = res.data;
    if (success) {
      await loadS3Self();
      showSuccess('已关闭');
    } else {
      showError(message);
    }
  };

  const s3RegenerateSecret = async () => {
    const res = await API.post('/api/user/s3/regenerate_secret');
    const { success, message, data } = res.data;
    if (success) {
      openS3SecretModal(
        s3Info?.accessKey || '',
        data.secret_key,
        t('setting.personal.s3.regenerate_secret')
      );
      showSuccess('Secret 已更新');
    } else {
      showError(message);
    }
  };

  const s3RotateKeys = async () => {
    if (!window.confirm('将生成新的 Access Key 与 Secret，旧密钥立即失效。继续？')) {
      return;
    }
    const res = await API.post('/api/user/s3/rotate_keys');
    const { success, message, data } = res.data;
    if (success) {
      openS3SecretModal(
        data.access_key,
        data.secret_key,
        t('setting.personal.s3.rotate_keys')
      );
      await loadS3Self();
      showSuccess('密钥已轮换');
    } else {
      showError(message);
    }
  };

  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const generateAccessToken = async () => {
    const res = await API.get('/api/user/token');
    const { success, message, data } = res.data;
    if (success) {
      setSystemToken(data);
      setAffLink('');
      await copy(data);
      showSuccess(`令牌已重置并已复制到剪贴板`);
    } else {
      showError(message);
    }
  };

  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    const { success, message, data } = res.data;
    if (success) {
      let link = `${window.location.origin}/register?aff=${data}`;
      setAffLink(link);
      setSystemToken('');
      await copy(link);
      showSuccess(`邀请链接已复制到剪切板`);
    } else {
      showError(message);
    }
  };

  const handleAffLinkClick = async (e) => {
    e.target.select();
    await copy(e.target.value);
    showSuccess(`邀请链接已复制到剪切板`);
  };

  const handleSystemTokenClick = async (e) => {
    e.target.select();
    await copy(e.target.value);
    showSuccess(`系统令牌已复制到剪切板`);
  };

  const deleteAccount = async () => {
    if (inputs.self_account_deletion_confirmation !== userState.user.username) {
      showError('请输入你的账户名以确认删除！');
      return;
    }

    const res = await API.delete('/api/user/self');
    const { success, message } = res.data;

    if (success) {
      showSuccess('账户已删除！');
      await API.get('/api/user/logout');
      userDispatch({ type: 'logout' });
      localStorage.removeItem('user');
      clearNacosEmbeddedConsoleLocalSession();
      navigate('/login');
    } else {
      showError(message);
    }
  };

  const bindWeChat = async () => {
    if (inputs.wechat_verification_code === '') return;
    const res = await API.get(
      `/api/oauth/wechat/bind?code=${inputs.wechat_verification_code}`
    );
    const { success, message } = res.data;
    if (success) {
      showSuccess('微信账户绑定成功！');
      setShowWeChatBindModal(false);
    } else {
      showError(message);
    }
  };

  const sendVerificationCode = async () => {
    setDisableButton(true);
    if (inputs.email === '') return;
    if (turnstileEnabled && turnstileToken === '') {
      showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
      return;
    }
    setLoading(true);
    const res = await API.get(
      `/api/verification?email=${inputs.email}&turnstile=${turnstileToken}`
    );
    const { success, message } = res.data;
    if (success) {
      showSuccess('验证码发送成功，请检查邮箱！');
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const bindEmail = async () => {
    if (inputs.email_verification_code === '') return;
    setLoading(true);
    const res = await API.get(
      `/api/oauth/email/bind?email=${inputs.email}&code=${inputs.email_verification_code}`
    );
    const { success, message } = res.data;
    if (success) {
      showSuccess('邮箱账户绑定成功！');
      setShowEmailBindModal(false);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  return (
    <div className='settings-page-body'>
      <Header as='h3'>{t('setting.personal.general.title')}</Header>
      <Message>{t('setting.personal.general.system_token_notice')}</Message>
      <div className='settings-action-bar'>
      <Button as={Link} to={`/user/edit/`}>
        {t('setting.personal.general.buttons.update_profile')}
      </Button>
      <Button onClick={generateAccessToken}>
        {t('setting.personal.general.buttons.generate_token')}
      </Button>
      <Button onClick={getAffLink}>
        {t('setting.personal.general.buttons.copy_invite')}
      </Button>
      <Button
        onClick={() => {
          setShowAccountDeleteModal(true);
        }}
      >
        {t('setting.personal.general.buttons.delete_account')}
      </Button>
      </div>

      {systemToken && (
        <Form.Input
          fluid
          readOnly
          value={systemToken}
          onClick={handleSystemTokenClick}
          style={{ marginTop: '10px' }}
        />
      )}
      {affLink && (
        <Form.Input
          fluid
          readOnly
          value={affLink}
          onClick={handleAffLinkClick}
          style={{ marginTop: '10px' }}
        />
      )}
      <Divider />
      <Header as='h3'>安全 · 两步验证（2FA）</Header>
      <TwoFASetting />
      <Divider />
      <Header as='h3'>{t('setting.personal.s3.title')}</Header>
      <Message>{t('setting.personal.s3.notice')}</Message>
      {s3Info && !s3Info.site && (
        <Message warning>{t('setting.personal.s3.site_disabled')}</Message>
      )}
      {s3Info && s3Info.site && (
        <>
          <div>
            {t('setting.personal.s3.region')}:{' '}
            <code>{s3Info.region}</code>
          </div>
          <div style={{ marginTop: 8 }}>
            {s3Info.enabled
              ? t('setting.personal.s3.status_enabled')
              : t('setting.personal.s3.status_disabled')}
            {s3Info.accessKey
              ? ` — ${t('setting.personal.s3.access_key')}: ${s3Info.accessKey}`
              : ''}
          </div>
          <div className='settings-action-bar' style={{ marginTop: 12 }}>
            {!s3Info.enabled ? (
              <Button primary onClick={s3Enable}>
                {t('setting.personal.s3.enable')}
              </Button>
            ) : (
              <Button.Group>
                <Button onClick={s3RegenerateSecret}>
                  {t('setting.personal.s3.regenerate_secret')}
                </Button>
                <Button onClick={s3RotateKeys}>
                  {t('setting.personal.s3.rotate_keys')}
                </Button>
                <Button negative onClick={s3Disable}>
                  {t('setting.personal.s3.disable')}
                </Button>
              </Button.Group>
            )}
          </div>
        </>
      )}
      <Modal
        open={s3SecretModal.open}
        onClose={() =>
          setS3SecretModal((m) => ({
            ...m,
            open: false,
            secretKey: '',
          }))
        }
        size='small'
      >
        <Modal.Header>{t('setting.personal.s3.secret_once')}</Modal.Header>
        <Modal.Content>
          <p>{s3SecretModal.subtitle}</p>
          {s3SecretModal.accessKey ? (
            <Form.Input
              label={t('setting.personal.s3.access_key')}
              fluid
              readOnly
              value={s3SecretModal.accessKey}
              onClick={(e) => e.target.select()}
            />
          ) : null}
          <Form.Input
            label={t('setting.personal.s3.secret_once')}
            fluid
            readOnly
            value={s3SecretModal.secretKey}
            onClick={(e) => e.target.select()}
          />
        </Modal.Content>
        <Modal.Actions>
          <Button
            primary
            onClick={async () => {
              const text = [
                s3SecretModal.accessKey
                  ? `AK=${s3SecretModal.accessKey}`
                  : '',
                `SK=${s3SecretModal.secretKey}`,
              ]
                .filter(Boolean)
                .join('\n');
              await copy(text);
              showSuccess('已复制');
            }}
          >
            复制到剪贴板
          </Button>
          <Button
            onClick={() =>
              setS3SecretModal((m) => ({
                ...m,
                open: false,
                secretKey: '',
              }))
            }
          >
            关闭
          </Button>
        </Modal.Actions>
      </Modal>
      <Divider />
      <Header as='h3'>{t('setting.personal.binding.title')}</Header>
      {status.wechat_login && (
        <Button onClick={() => setShowWeChatBindModal(true)}>
          {t('setting.personal.binding.buttons.bind_wechat')}
        </Button>
      )}
      <Modal
        onClose={() => setShowWeChatBindModal(false)}
        onOpen={() => setShowWeChatBindModal(true)}
        open={showWeChatBindModal}
        size={'mini'}
      >
        <Modal.Content>
          <Modal.Description>
            <Image src={status.wechat_qrcode} fluid />
            <div style={{ textAlign: 'center' }}>
              <p>{t('setting.personal.binding.wechat.description')}</p>
            </div>
            <Form size='large'>
              <Form.Input
                fluid
                placeholder={t(
                  'setting.personal.binding.wechat.verification_code'
                )}
                name='wechat_verification_code'
                value={inputs.wechat_verification_code}
                onChange={handleInputChange}
              />
              <Button color='' fluid size='large' onClick={bindWeChat}>
                {t('setting.personal.binding.wechat.bind')}
              </Button>
            </Form>
          </Modal.Description>
        </Modal.Content>
      </Modal>
      {status.github_oauth && (
        <Button onClick={() => onGitHubOAuthClicked(status.github_client_id)}>
          {t('setting.personal.binding.buttons.bind_github')}
        </Button>
      )}
      {status.lark_client_id && (
        <Button onClick={() => onLarkOAuthClicked(status.lark_client_id)}>
          {t('setting.personal.binding.buttons.bind_lark')}
        </Button>
      )}
      <Button onClick={() => setShowEmailBindModal(true)}>
        {t('setting.personal.binding.buttons.bind_email')}
      </Button>
      <Modal
        onClose={() => setShowEmailBindModal(false)}
        onOpen={() => setShowEmailBindModal(true)}
        open={showEmailBindModal}
        size={'tiny'}
        style={{ maxWidth: '450px' }}
      >
        <Modal.Header>{t('setting.personal.binding.email.title')}</Modal.Header>
        <Modal.Content>
          <Modal.Description>
            <Form size='large'>
              <Form.Input
                fluid
                placeholder={t(
                  'setting.personal.binding.email.email_placeholder'
                )}
                onChange={handleInputChange}
                name='email'
                type='email'
                action={
                  <Button
                    onClick={sendVerificationCode}
                    disabled={disableButton || loading}
                  >
                    {disableButton
                      ? t('setting.personal.binding.email.get_code_retry', {
                          countdown,
                        })
                      : t('setting.personal.binding.email.get_code')}
                  </Button>
                }
              />
              <Form.Input
                fluid
                placeholder={t(
                  'setting.personal.binding.email.code_placeholder'
                )}
                name='email_verification_code'
                value={inputs.email_verification_code}
                onChange={handleInputChange}
              />
              {turnstileEnabled && (
                <Turnstile
                  sitekey={turnstileSiteKey}
                  onVerify={(token) => {
                    setTurnstileToken(token);
                  }}
                />
              )}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginTop: '1rem',
                }}
              >
                <Button
                  color=''
                  fluid
                  size='large'
                  onClick={bindEmail}
                  loading={loading}
                >
                  {t('setting.personal.binding.email.bind')}
                </Button>
                <div style={{ width: '1rem' }}></div>
                <Button
                  fluid
                  size='large'
                  onClick={() => setShowEmailBindModal(false)}
                >
                  {t('setting.personal.binding.email.cancel')}
                </Button>
              </div>
            </Form>
          </Modal.Description>
        </Modal.Content>
      </Modal>
      <Modal
        onClose={() => setShowAccountDeleteModal(false)}
        onOpen={() => setShowAccountDeleteModal(true)}
        open={showAccountDeleteModal}
        size={'tiny'}
        style={{ maxWidth: '450px' }}
      >
        <Modal.Header>
          {t('setting.personal.delete_account.title')}
        </Modal.Header>
        <Modal.Content>
          <Message>{t('setting.personal.delete_account.warning')}</Message>
          <Modal.Description>
            <Form size='large'>
              <Form.Input
                fluid
                placeholder={t(
                  'setting.personal.delete_account.confirm_placeholder',
                  {
                    username: userState?.user?.username,
                  }
                )}
                name='self_account_deletion_confirmation'
                value={inputs.self_account_deletion_confirmation}
                onChange={handleInputChange}
              />
              {turnstileEnabled && (
                <Turnstile
                  sitekey={turnstileSiteKey}
                  onVerify={(token) => {
                    setTurnstileToken(token);
                  }}
                />
              )}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginTop: '1rem',
                }}
              >
                <Button
                  color='red'
                  fluid
                  size='large'
                  onClick={deleteAccount}
                  loading={loading}
                >
                  {t('setting.personal.delete_account.buttons.confirm')}
                </Button>
                <div style={{ width: '1rem' }}></div>
                <Button
                  fluid
                  size='large'
                  onClick={() => setShowAccountDeleteModal(false)}
                >
                  {t('setting.personal.delete_account.buttons.cancel')}
                </Button>
              </div>
            </Form>
          </Modal.Description>
        </Modal.Content>
      </Modal>
    </div>
  );
};

export default PersonalSetting;
