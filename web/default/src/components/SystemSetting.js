import React, { useContext, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Divider,
  Form,
  Grid,
  Header,
  Modal,
  Message,
} from 'semantic-ui-react';
import { API, removeTrailingSlash, showError, noAutofillSecretProps, noAutofillTextProps } from '../helpers';
import { StatusContext } from '../context/Status';

const SystemSetting = () => {
  const { t } = useTranslation();
  const [, statusDispatch] = useContext(StatusContext);
  let [inputs, setInputs] = useState({
    PasswordLoginEnabled: '',
    PasswordRegisterEnabled: '',
    EmailVerificationEnabled: '',
    Force2FAForAllUsers: '',
    GitHubOAuthEnabled: '',
    LarkOAuthEnabled: '',
    GitHubClientId: '',
    GitHubClientSecret: '',
    LarkClientId: '',
    LarkClientSecret: '',
    Notice: '',
    SMTPServer: '',
    SMTPPort: '',
    SMTPAccount: '',
    SMTPFrom: '',
    SMTPToken: '',
    ServerAddress: '',
    Footer: '',
    WeChatAuthEnabled: '',
    WeChatServerAddress: '',
    WeChatServerToken: '',
    WeChatAccountQRCodeImageURL: '',
    MessagePusherAddress: '',
    MessagePusherToken: '',
    TurnstileCheckEnabled: '',
    LoginCaptchaEnabled: '',
    TurnstileSiteKey: '',
    TurnstileSecretKey: '',
    RegisterEnabled: '',
    EmailDomainRestrictionEnabled: '',
    EmailDomainWhitelist: '',
    S3SiteEnabled: '',
    NacosEnabled: '',
    SecurePasswordLoginEnabled: '',
    OutboundSSRFSProtectionEnabled: 'true',
    OutboundURLWhitelistEnabled: 'false',
    OutboundURLWhitelistDomains: '',
    OutboundURLWhitelistIPs: '',
  });
  const [originInputs, setOriginInputs] = useState({});
  let [loading, setLoading] = useState(false);
  const [EmailDomainWhitelist, setEmailDomainWhitelist] = useState([]);
  const [restrictedDomainInput, setRestrictedDomainInput] = useState('');
  const [showPasswordWarningModal, setShowPasswordWarningModal] =
    useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        newInputs[item.key] = item.value;
      });
      const merged = {
        OutboundSSRFSProtectionEnabled: 'true',
        OutboundURLWhitelistEnabled: 'false',
        OutboundURLWhitelistDomains: '',
        OutboundURLWhitelistIPs: '',
        ...newInputs,
      };
      setOriginInputs(merged);
      setInputs({
        ...merged,
        EmailDomainWhitelist: (merged.EmailDomainWhitelist || '').split(','),
      });

      setEmailDomainWhitelist(
        (merged.EmailDomainWhitelist || '').split(',').map((item) => {
          return { key: item, text: item, value: item };
        })
      );
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions().then();
  }, []);

  const updateOption = async (key, value) => {
    setLoading(true);
    switch (key) {
      case 'PasswordLoginEnabled':
      case 'PasswordRegisterEnabled':
      case 'EmailVerificationEnabled':
      case 'Force2FAForAllUsers':
      case 'GitHubOAuthEnabled':
      case 'LarkOAuthEnabled':
      case 'WeChatAuthEnabled':
      case 'TurnstileCheckEnabled':
      case 'LoginCaptchaEnabled':
      case 'EmailDomainRestrictionEnabled':
      case 'RegisterEnabled':
      case 'S3SiteEnabled':
      case 'NacosEnabled':
      case 'SecurePasswordLoginEnabled':
      case 'OutboundSSRFSProtectionEnabled':
      case 'OutboundURLWhitelistEnabled':
        value = inputs[key] === 'true' ? 'false' : 'true';
        break;
      default:
        break;
    }
    const res = await API.put('/api/option/', {
      key,
      value,
    });
    const { success, message } = res.data;
    if (success) {
      if (key === 'EmailDomainWhitelist') {
        value = value.split(',');
      }
      if (key === 'NacosEnabled' || key === 'SecurePasswordLoginEnabled') {
        try {
          const st = JSON.parse(localStorage.getItem('status') || '{}');
          if (key === 'NacosEnabled') {
            st.nacos_enabled = value === 'true';
          }
          if (key === 'SecurePasswordLoginEnabled') {
            st.secure_password_login = value === 'true';
          }
          localStorage.setItem('status', JSON.stringify(st));
          statusDispatch({ type: 'set', payload: st });
        } catch {
          /* ignore */
        }
      }
      if (key === 'LoginCaptchaEnabled') {
        try {
          const st = JSON.parse(localStorage.getItem('status') || '{}');
          if (value === 'false') {
            st.login_math_captcha = false;
            localStorage.setItem('status', JSON.stringify(st));
            statusDispatch({ type: 'set', payload: st });
          } else {
            API.get('/api/status').then((res) => {
              const payload = res.data?.data;
              if (!payload) return;
              const st = JSON.parse(localStorage.getItem('status') || '{}');
              Object.assign(st, payload);
              localStorage.setItem('status', JSON.stringify(st));
              statusDispatch({ type: 'set', payload: st });
            });
          }
        } catch {
          /* ignore */
        }
      }
      setInputs((inputs) => ({
        ...inputs,
        [key]: value,
      }));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const handleInputChange = async (e, { name, value }) => {
    if (name === 'PasswordLoginEnabled' && inputs[name] === 'true') {
      // block disabling password login
      setShowPasswordWarningModal(true);
      return;
    }
    if (
      name === 'Notice' ||
      name.startsWith('SMTP') ||
      name === 'ServerAddress' ||
      name === 'GitHubClientId' ||
      name === 'GitHubClientSecret' ||
      name === 'LarkClientId' ||
      name === 'LarkClientSecret' ||
      name === 'WeChatServerAddress' ||
      name === 'WeChatServerToken' ||
      name === 'WeChatAccountQRCodeImageURL' ||
      name === 'MessagePusherAddress' ||
      name === 'MessagePusherToken' ||
      name === 'TurnstileSiteKey' ||
      name === 'TurnstileSecretKey' ||
      name === 'EmailDomainWhitelist' ||
      name === 'OutboundURLWhitelistDomains' ||
      name === 'OutboundURLWhitelistIPs'
    ) {
      setInputs((inputs) => ({ ...inputs, [name]: value }));
    } else {
      await updateOption(name, value);
    }
  };

  const submitServerAddress = async () => {
    let ServerAddress = removeTrailingSlash(inputs.ServerAddress);
    await updateOption('ServerAddress', ServerAddress);
  };

  const submitOutboundWhitelist = async () => {
    const domainsChanged =
      (originInputs.OutboundURLWhitelistDomains || '') !==
      (inputs.OutboundURLWhitelistDomains || '');
    const ipsChanged =
      (originInputs.OutboundURLWhitelistIPs || '') !==
      (inputs.OutboundURLWhitelistIPs || '');
    if (!domainsChanged && !ipsChanged) {
      return;
    }
    if (domainsChanged) {
      await updateOption(
        'OutboundURLWhitelistDomains',
        inputs.OutboundURLWhitelistDomains || ''
      );
    }
    if (ipsChanged) {
      await updateOption(
        'OutboundURLWhitelistIPs',
        inputs.OutboundURLWhitelistIPs || ''
      );
    }
    await getOptions();
  };

  const submitSMTP = async () => {
    if (originInputs['SMTPServer'] !== inputs.SMTPServer) {
      await updateOption('SMTPServer', inputs.SMTPServer);
    }
    if (originInputs['SMTPAccount'] !== inputs.SMTPAccount) {
      await updateOption('SMTPAccount', inputs.SMTPAccount);
    }
    if (originInputs['SMTPFrom'] !== inputs.SMTPFrom) {
      await updateOption('SMTPFrom', inputs.SMTPFrom);
    }
    if (
      originInputs['SMTPPort'] !== inputs.SMTPPort &&
      inputs.SMTPPort !== ''
    ) {
      await updateOption('SMTPPort', inputs.SMTPPort);
    }
    if (
      originInputs['SMTPToken'] !== inputs.SMTPToken &&
      inputs.SMTPToken !== ''
    ) {
      await updateOption('SMTPToken', inputs.SMTPToken);
    }
  };

  const submitEmailDomainWhitelist = async () => {
    if (
      originInputs['EmailDomainWhitelist'] !==
        inputs.EmailDomainWhitelist.join(',') &&
      inputs.SMTPToken !== ''
    ) {
      await updateOption(
        'EmailDomainWhitelist',
        inputs.EmailDomainWhitelist.join(',')
      );
    }
  };

  const submitWeChat = async () => {
    if (originInputs['WeChatServerAddress'] !== inputs.WeChatServerAddress) {
      await updateOption(
        'WeChatServerAddress',
        removeTrailingSlash(inputs.WeChatServerAddress)
      );
    }
    if (
      originInputs['WeChatAccountQRCodeImageURL'] !==
      inputs.WeChatAccountQRCodeImageURL
    ) {
      await updateOption(
        'WeChatAccountQRCodeImageURL',
        inputs.WeChatAccountQRCodeImageURL
      );
    }
    if (
      originInputs['WeChatServerToken'] !== inputs.WeChatServerToken &&
      inputs.WeChatServerToken !== ''
    ) {
      await updateOption('WeChatServerToken', inputs.WeChatServerToken);
    }
  };

  const submitMessagePusher = async () => {
    if (originInputs['MessagePusherAddress'] !== inputs.MessagePusherAddress) {
      await updateOption(
        'MessagePusherAddress',
        removeTrailingSlash(inputs.MessagePusherAddress)
      );
    }
    if (
      originInputs['MessagePusherToken'] !== inputs.MessagePusherToken &&
      inputs.MessagePusherToken !== ''
    ) {
      await updateOption('MessagePusherToken', inputs.MessagePusherToken);
    }
  };

  const submitGitHubOAuth = async () => {
    if (originInputs['GitHubClientId'] !== inputs.GitHubClientId) {
      await updateOption('GitHubClientId', inputs.GitHubClientId);
    }
    if (
      originInputs['GitHubClientSecret'] !== inputs.GitHubClientSecret &&
      inputs.GitHubClientSecret !== ''
    ) {
      await updateOption('GitHubClientSecret', inputs.GitHubClientSecret);
    }
  };

  const submitLarkOAuth = async () => {
    if (originInputs['LarkClientId'] !== inputs.LarkClientId) {
      await updateOption('LarkClientId', inputs.LarkClientId);
    }
    if (
      originInputs['LarkClientSecret'] !== inputs.LarkClientSecret &&
      inputs.LarkClientSecret !== ''
    ) {
      await updateOption('LarkClientSecret', inputs.LarkClientSecret);
    }
  };

  const submitTurnstile = async () => {
    if (originInputs['TurnstileSiteKey'] !== inputs.TurnstileSiteKey) {
      await updateOption('TurnstileSiteKey', inputs.TurnstileSiteKey);
    }
    if (
      originInputs['TurnstileSecretKey'] !== inputs.TurnstileSecretKey &&
      inputs.TurnstileSecretKey !== ''
    ) {
      await updateOption('TurnstileSecretKey', inputs.TurnstileSecretKey);
    }
  };

  const submitNewRestrictedDomain = () => {
    const localDomainList = inputs.EmailDomainWhitelist;
    if (
      restrictedDomainInput !== '' &&
      !localDomainList.includes(restrictedDomainInput)
    ) {
      setRestrictedDomainInput('');
      setInputs({
        ...inputs,
        EmailDomainWhitelist: [...localDomainList, restrictedDomainInput],
      });
      setEmailDomainWhitelist([
        ...EmailDomainWhitelist,
        {
          key: restrictedDomainInput,
          text: restrictedDomainInput,
          value: restrictedDomainInput,
        },
      ]);
    }
  };

  return (
    <Grid columns={1}>
      <Grid.Column>
        <Form loading={loading}>
          <Header as='h3'>{t('setting.system.general.title')}</Header>
          <Form.Input
            label={t('setting.system.general.server_address')}
            name='ServerAddress'
            placeholder={t('setting.system.general.server_address_placeholder')}
            value={inputs.ServerAddress}
            onChange={handleInputChange}
            {...noAutofillTextProps}
          />
          <Form.Button onClick={submitServerAddress}>
            {t('setting.system.general.buttons.update')}
          </Form.Button>
          <Divider />
          <Header as='h3'>{t('setting.system.security.title')}</Header>
          <Message info>{t('setting.system.security.subtitle')}</Message>
          <Header as='h4'>{t('setting.system.security.ssrf.title')}</Header>
          <Message warning>{t('setting.system.security.ssrf.subtitle')}</Message>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.OutboundSSRFSProtectionEnabled === 'true'}
              label={t('setting.system.security.ssrf.enable')}
              name='OutboundSSRFSProtectionEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Divider section />
          <Header as='h4'>
            {t('setting.system.security.outbound_whitelist.title')}
          </Header>
          <Message warning>
            {t('setting.system.security.outbound_whitelist.subtitle')}
          </Message>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.OutboundURLWhitelistEnabled === 'true'}
              label={t('setting.system.security.outbound_whitelist.enable')}
              name='OutboundURLWhitelistEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.TextArea
            label={t('setting.system.security.outbound_whitelist.domains')}
            name='OutboundURLWhitelistDomains'
            placeholder={t(
              'setting.system.security.outbound_whitelist.domains_placeholder'
            )}
            value={inputs.OutboundURLWhitelistDomains || ''}
            onChange={handleInputChange}
            rows={4}
          />
          <Form.TextArea
            label={t('setting.system.security.outbound_whitelist.ips')}
            name='OutboundURLWhitelistIPs'
            placeholder={t(
              'setting.system.security.outbound_whitelist.ips_placeholder'
            )}
            value={inputs.OutboundURLWhitelistIPs || ''}
            onChange={handleInputChange}
            rows={4}
          />
          <Form.Button onClick={() => submitOutboundWhitelist().then()}>
            {t('setting.system.security.outbound_whitelist.buttons.save')}
          </Form.Button>
          <Divider />
          <Header as='h3'>{t('setting.system.login.title')}</Header>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.PasswordLoginEnabled === 'true'}
              label={t('setting.system.login.password_login')}
              name='PasswordLoginEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.SecurePasswordLoginEnabled === 'true'}
              label={t('setting.system.login.secure_password_login')}
              name='SecurePasswordLoginEnabled'
              onChange={handleInputChange}
            />
            {showPasswordWarningModal && (
              <Modal
                open={showPasswordWarningModal}
                onClose={() => setShowPasswordWarningModal(false)}
                size={'tiny'}
                style={{ maxWidth: '450px' }}
              >
                <Modal.Header>
                  {t('setting.system.password_login.warning.title')}
                </Modal.Header>
                <Modal.Content>
                  <p>{t('setting.system.password_login.warning.content')}</p>
                </Modal.Content>
                <Modal.Actions>
                  <Button onClick={() => setShowPasswordWarningModal(false)}>
                    {t('setting.system.password_login.warning.buttons.cancel')}
                  </Button>
                  <Button
                    color='yellow'
                    onClick={async () => {
                      setShowPasswordWarningModal(false);
                      await updateOption('PasswordLoginEnabled', 'false');
                    }}
                  >
                    {t('setting.system.password_login.warning.buttons.confirm')}
                  </Button>
                </Modal.Actions>
              </Modal>
            )}
            <Form.Checkbox
              checked={inputs.PasswordRegisterEnabled === 'true'}
              label={t('setting.system.login.password_register')}
              name='PasswordRegisterEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.EmailVerificationEnabled === 'true'}
              label={t('setting.system.login.email_verification')}
              name='EmailVerificationEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.Force2FAForAllUsers === 'true'}
              label={t('setting.system.login.force_mfa')}
              name='Force2FAForAllUsers'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.GitHubOAuthEnabled === 'true'}
              label={t('setting.system.login.github_oauth')}
              name='GitHubOAuthEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.WeChatAuthEnabled === 'true'}
              label={t('setting.system.login.wechat_login')}
              name='WeChatAuthEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.LarkOAuthEnabled === 'true'}
              label={t('setting.system.login.lark_oauth')}
              name='LarkOAuthEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.RegisterEnabled === 'true'}
              label={t('setting.system.login.registration')}
              name='RegisterEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.TurnstileCheckEnabled === 'true'}
              label={t('setting.system.login.turnstile')}
              name='TurnstileCheckEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.LoginCaptchaEnabled === 'true'}
              label={t('setting.system.login.login_captcha')}
              name='LoginCaptchaEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Divider />
          <Header as='h3'>{t('setting.system.nacos.title')}</Header>
          <Message>{t('setting.system.nacos.subtitle')}</Message>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.NacosEnabled === 'true'}
              label={t('setting.system.nacos.enable')}
              name='NacosEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Divider />
          <Header as='h3'>{t('setting.system.s3.title')}</Header>
          <Message>{t('setting.system.s3.subtitle')}</Message>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.S3SiteEnabled === 'true'}
              label={t('setting.system.s3.enable_site')}
              name='S3SiteEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Divider />
          <Header as='h3'>{t('setting.system.email_restriction.title')}</Header>
          <Message>{t('setting.system.email_restriction.subtitle')}</Message>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.EmailDomainRestrictionEnabled === 'true'}
              label={t('setting.system.email_restriction.enable')}
              name='EmailDomainRestrictionEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.email_restriction.add_domain')}
              placeholder={t(
                'setting.system.email_restriction.add_domain_placeholder'
              )}
              value={restrictedDomainInput}
              onChange={(e, { value }) => {
                setRestrictedDomainInput(value);
              }}
              action={
                <Button
                  onClick={() => {
                    if (restrictedDomainInput === '') return;
                    setEmailDomainWhitelist([
                      ...EmailDomainWhitelist,
                      {
                        key: restrictedDomainInput,
                        text: restrictedDomainInput,
                        value: restrictedDomainInput,
                      },
                    ]);
                    setRestrictedDomainInput('');
                  }}
                >
                  {t('setting.system.email_restriction.buttons.fill')}
                </Button>
              }
            />
          </Form.Group>
          <Form.Dropdown
            label={t('setting.system.email_restriction.allowed_domains')}
            placeholder={t('setting.system.email_restriction.allowed_domains')}
            fluid
            multiple
            search
            selection
            allowAdditions
            value={EmailDomainWhitelist.map((item) => item.value)}
            options={EmailDomainWhitelist}
            onAddItem={(e, { value }) => {
              setEmailDomainWhitelist([
                ...EmailDomainWhitelist,
                {
                  key: value,
                  text: value,
                  value: value,
                },
              ]);
            }}
            onChange={(e, { value }) => {
              let newEmailDomainWhitelist = [];
              value.forEach((item) => {
                newEmailDomainWhitelist.push({
                  key: item,
                  text: item,
                  value: item,
                });
              });
              setEmailDomainWhitelist(newEmailDomainWhitelist);
            }}
          />
          <Form.Button onClick={submitEmailDomainWhitelist}>
            {t('setting.system.email_restriction.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>{t('setting.system.smtp.title')}</Header>
          <Message>{t('setting.system.smtp.subtitle')}</Message>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.smtp.server')}
              name='SMTPServer'
              placeholder={t('setting.system.smtp.server_placeholder')}
              value={inputs.SMTPServer}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.smtp.port')}
              name='SMTPPort'
              placeholder={t('setting.system.smtp.port_placeholder')}
              value={inputs.SMTPPort}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.smtp.account')}
              name='SMTPAccount'
              placeholder={t('setting.system.smtp.account_placeholder')}
              value={inputs.SMTPAccount}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
          </Form.Group>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.smtp.from')}
              name='SMTPFrom'
              placeholder={t('setting.system.smtp.from_placeholder')}
              value={inputs.SMTPFrom}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.smtp.token')}
              placeholder={t('setting.system.smtp.token_placeholder')}
              name='SMTPToken'
              onChange={handleInputChange}
              type='password'
              value={inputs.SMTPToken}
            />
          </Form.Group>
          <Form.Button onClick={submitSMTP}>
            {t('setting.system.smtp.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>{t('setting.system.github.title')}</Header>
          <Message>
            {t('setting.system.github.subtitle')}
            <a href='https://github.com/settings/developers' target='_blank'>
              {t('setting.system.github.manage_link')}
            </a>
            {t('setting.system.github.manage_text')}
          </Message>
          <Message>
            {t('setting.system.github.url_notice', {
              server_url: originInputs.ServerAddress,
              callback_url: `${originInputs.ServerAddress}/oauth/github`,
            })}
          </Message>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.github.client_id')}
              name='GitHubClientId'
              placeholder={t('setting.system.github.client_id_placeholder')}
              value={inputs.GitHubClientId}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.github.client_secret')}
              placeholder={t('setting.system.github.client_secret_placeholder')}
              name='GitHubClientSecret'
              onChange={handleInputChange}
              type='password'
              value={inputs.GitHubClientSecret}
            />
          </Form.Group>
          <Form.Button onClick={submitGitHubOAuth}>
            {t('setting.system.github.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>
            {t('setting.system.lark.title')}
            <Header.Subheader>
              {t('setting.system.lark.subtitle')}
              <a href='https://open.feishu.cn/app' target='_blank'>
                {t('setting.system.lark.manage_link')}
              </a>
              {t('setting.system.lark.manage_text')}
            </Header.Subheader>
          </Header>
          <Message>
            {t('setting.system.lark.url_notice', {
              server_url: inputs.ServerAddress,
              callback_url: `${inputs.ServerAddress}/oauth/lark`,
            })}
          </Message>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.lark.client_id')}
              name='LarkClientId'
              placeholder={t('setting.system.lark.client_id_placeholder')}
              value={inputs.LarkClientId}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.lark.client_secret')}
              name='LarkClientSecret'
              onChange={handleInputChange}
              type='password'
              {...noAutofillSecretProps}
              value={inputs.LarkClientSecret}
              placeholder={t('setting.system.lark.client_secret_placeholder')}
            />
          </Form.Group>
          <Form.Button onClick={submitLarkOAuth}>
            {t('setting.system.lark.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>
            {t('setting.system.wechat.title')}
            <Header.Subheader>
              {t('setting.system.wechat.subtitle')}
              <a
                href='https://github.com/songquanpeng/wechat-server'
                target='_blank'
              >
                {t('setting.system.wechat.learn_more')}
              </a>
            </Header.Subheader>
          </Header>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.wechat.server_address')}
              name='WeChatServerAddress'
              placeholder={t('setting.system.wechat.server_address_placeholder')}
              value={inputs.WeChatServerAddress}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.wechat.token')}
              name='WeChatServerToken'
              onChange={handleInputChange}
              type='password'
              {...noAutofillSecretProps}
              value={inputs.WeChatServerToken}
              placeholder={t('setting.system.wechat.token_placeholder')}
            />
            <Form.Input
              label={t('setting.system.wechat.qrcode')}
              name='WeChatAccountQRCodeImageURL'
              placeholder={t('setting.system.wechat.qrcode_placeholder')}
              value={inputs.WeChatAccountQRCodeImageURL}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
          </Form.Group>
          <Form.Button onClick={submitWeChat}>
            {t('setting.system.wechat.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>
            {t('setting.system.turnstile.title')}
            <Header.Subheader>
              {t('setting.system.turnstile.subtitle')}
              <a href='https://dash.cloudflare.com/' target='_blank'>
                {t('setting.system.turnstile.manage_link')}
              </a>
              {t('setting.system.turnstile.manage_text')}
            </Header.Subheader>
          </Header>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.system.turnstile.site_key')}
              name='TurnstileSiteKey'
              placeholder={t('setting.system.turnstile.site_key_placeholder')}
              value={inputs.TurnstileSiteKey}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.system.turnstile.secret_key')}
              name='TurnstileSecretKey'
              onChange={handleInputChange}
              type='password'
              {...noAutofillSecretProps}
              value={inputs.TurnstileSecretKey}
              placeholder={t('setting.system.turnstile.secret_key_placeholder')}
            />
          </Form.Group>
          <Form.Button onClick={submitTurnstile}>
            {t('setting.system.turnstile.buttons.save')}
          </Form.Button>
        </Form>
      </Grid.Column>
    </Grid>
  );
};

export default SystemSetting;
