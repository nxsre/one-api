import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Divider,
  Form,
  Grid,
  Header,
  Message,
  Modal,
} from 'semantic-ui-react';
import { Link } from 'react-router-dom';
import { API, showError, showSuccess, noAutofillTextProps } from '../helpers';
import { marked } from 'marked';

const OtherSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    Footer: '',
    Notice: '',
    About: '',
    SystemName: '',
    Logo: '',
    HomePageContent: '',
    Theme: '',
  });
  let [loading, setLoading] = useState(false);
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [updateData, setUpdateData] = useState({
    tag_name: '',
    content: '',
  });

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          newInputs[item.key] = item.value;
        }
      });
      setInputs(newInputs);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions().then();
  }, []);

  const updateOption = async (key, value) => {
    setLoading(true);
    const res = await API.put('/api/option/', {
      key,
      value,
    });
    const { success, message } = res.data;
    if (success) {
      setInputs((inputs) => ({ ...inputs, [key]: value }));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const submitNotice = async () => {
    await updateOption('Notice', inputs.Notice);
  };

  const submitSystemName = async () => {
    await updateOption('SystemName', inputs.SystemName);
  };

  const submitTheme = async () => {
    await updateOption('Theme', inputs.Theme);
  };

  const submitLogo = async () => {
    await updateOption('Logo', inputs.Logo);
  };

  const submitAbout = async () => {
    await updateOption('About', inputs.About);
  };

  const submitOption = async (key) => {
    await updateOption(key, inputs[key]);
  };

  const handleFieldChange = (e, { name, value }) => {
    setInputs((prev) => ({ ...prev, [name]: value }));
  };

  const openGitHubRelease = () => {
    window.location = 'https://github.com/songquanpeng/one-api/releases/latest';
  };

  const checkUpdate = async () => {
    const res = await API.get(
      'https://api.github.com/repos/songquanpeng/one-api/releases/latest'
    );
    const { tag_name, body } = res.data;
    if (tag_name === process.env.REACT_APP_VERSION) {
      showSuccess(`已是最新版本：${tag_name}`);
    } else {
      setUpdateData({
        tag_name: tag_name,
        content: marked.parse(body),
      });
      setShowUpdateModal(true);
    }
  };

  return (
    <Grid columns={1}>
      <Grid.Column>
        <Form loading={loading}>
          <Header as='h3'>{t('setting.other.notice.title')}</Header>
          <Form.Group widths='equal'>
            <Form.TextArea
              label={t('setting.other.notice.content')}
              name='Notice'
              placeholder={t('setting.other.notice.content_placeholder')}
              value={inputs.Notice}
              onChange={handleFieldChange}
              rows={10}
            />
          </Form.Group>
          <Form.Button onClick={submitNotice}>
            {t('setting.other.notice.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>{t('setting.other.system.title')}</Header>
          <Form.Group widths='equal'>
            <Form.Input
              label={t('setting.other.system.name')}
              name='SystemName'
              placeholder={t('setting.other.system.name_placeholder')}
              value={inputs.SystemName}
              onChange={handleFieldChange}
              {...noAutofillTextProps}
            />
          </Form.Group>
          <Form.Button onClick={submitSystemName}>
            {t('setting.other.system.buttons.save_name')}
          </Form.Button>
          <Form.Group widths='equal'>
            <Form.Field>
              <label>
                {t('setting.other.system.theme.title')}（
                <Link to='https://github.com/songquanpeng/one-api/blob/main/web/README.md'>
                  {t('setting.other.system.theme.link')}
                </Link>
                ）
              </label>
              <Form.Input
                name='Theme'
                placeholder={t('setting.other.system.theme.placeholder')}
                value={inputs.Theme}
                onChange={handleFieldChange}
                {...noAutofillTextProps}
              />
            </Form.Field>
          </Form.Group>
          <Form.Button onClick={submitTheme}>
            {t('setting.other.system.buttons.save_theme')}
          </Form.Button>
          <Form.Group widths='equal'>
            <Form.Input
              label={t('setting.other.system.logo')}
              name='Logo'
              placeholder={t('setting.other.system.logo_placeholder')}
              value={inputs.Logo}
              onChange={handleFieldChange}
              {...noAutofillTextProps}
            />
          </Form.Group>
          <Form.Button onClick={submitLogo}>
            {t('setting.other.system.buttons.save_logo')}
          </Form.Button>

          <Divider />
          <Header as='h3'>{t('setting.other.content.title')}</Header>
          <Form.Group widths='equal'>
            <Form.TextArea
              label={t('setting.other.content.homepage.title')}
              name='HomePageContent'
              placeholder={t('setting.other.content.homepage.placeholder')}
              value={inputs.HomePageContent}
              onChange={handleFieldChange}
              rows={14}
            />
          </Form.Group>
          <Form.Button onClick={() => submitOption('HomePageContent')}>
            {t('setting.other.content.buttons.save_homepage')}
          </Form.Button>
          <Form.Group widths='equal'>
            <Form.TextArea
              label={t('setting.other.content.about.title')}
              name='About'
              placeholder={t('setting.other.content.about.placeholder')}
              value={inputs.About}
              onChange={handleFieldChange}
              rows={14}
            />
          </Form.Group>
          <Form.Button onClick={submitAbout}>
            {t('setting.other.content.buttons.save_about')}
          </Form.Button>
          <Message>{t('setting.other.copyright.notice')}</Message>
          <Form.Group widths='equal'>
            <Form.TextArea
              label={t('setting.other.content.footer.title')}
              name='Footer'
              placeholder={t('setting.other.content.footer.placeholder')}
              value={inputs.Footer}
              onChange={handleFieldChange}
              rows={6}
            />
          </Form.Group>
          <Form.Button onClick={() => submitOption('Footer')}>
            {t('setting.other.content.buttons.save_footer')}
          </Form.Button>
        </Form>
      </Grid.Column>
      <Modal
        onClose={() => setShowUpdateModal(false)}
        onOpen={() => setShowUpdateModal(true)}
        open={showUpdateModal}
      >
        <Modal.Header>新版本：{updateData.tag_name}</Modal.Header>
        <Modal.Content>
          <Modal.Description>
            <div dangerouslySetInnerHTML={{ __html: updateData.content }}></div>
          </Modal.Description>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setShowUpdateModal(false)}>关闭</Button>
          <Button
            content='详情'
            onClick={() => {
              setShowUpdateModal(false);
              openGitHubRelease();
            }}
          />
        </Modal.Actions>
      </Modal>
    </Grid>
  );
};

export default OtherSetting;
