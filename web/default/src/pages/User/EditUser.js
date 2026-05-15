import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Form, Card } from 'semantic-ui-react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { API, showError, showSuccess } from '../../helpers';
import { renderQuotaWithPrompt } from '../../helpers/render';

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

const EditUser = () => {
  const { t } = useTranslation();
  const params = useParams();
  const [searchParams] = useSearchParams();
  const userId = params.id;
  const [loading, setLoading] = useState(true);
  const [inputs, setInputs] = useState({
    username: '',
    display_name: '',
    password: '',
    github_id: '',
    wechat_id: '',
    email: '',
    quota: 0,
    group: 'default',
  });
  const [groupOptions, setGroupOptions] = useState([]);
  const {
    username,
    display_name,
    password,
    github_id,
    wechat_id,
    email,
    quota,
  } = inputs;

  const handleInputChange = (e, { name, value }) => {
    if (name === 'quota') {
      const trimmed = String(value).trim();
      const n = trimmed === '' ? 0 : parseInt(trimmed, 10);
      setInputs((prev) => ({
        ...prev,
        quota: Number.isNaN(n) ? prev.quota : n,
      }));
      return;
    }
    setInputs((prev) => ({ ...prev, [name]: value }));
  };

  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      setGroupOptions(
        res.data.data.map((group) => ({
          key: group,
          text: group,
          value: group,
        }))
      );
    } catch (error) {
      showError(error.message);
    }
  };
  const navigate = useNavigate();
  const handleCancel = () => {
    navigate('/setting');
  };
  const loadUser = async () => {
    let res = undefined;
    if (userId) {
      res = await API.get(`/api/user/${userId}`);
    } else {
      res = await API.get(`/api/user/self`);
    }
    const { success, message, data } = res.data;
    if (success) {
      data.password = '';
      setInputs(data);
    } else {
      showError(message);
    }
    setLoading(false);
  };
  useEffect(() => {
    loadUser().then();
    if (userId) {
      fetchGroups().then();
    }
  }, []);

  const submit = async () => {
    let res = undefined;
    if (userId) {
      let data = { ...inputs, user_id: userId };
      if (typeof data.quota === 'string') {
        data.quota = parseInt(data.quota);
      }
      res = await API.put(`/api/user/`, data);
    } else {
      res = await API.put(`/api/user/self`, inputs);
    }
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('user.messages.update_success'));
      if (!userId) {
        const back = consumeSafeInternalRedirect(searchParams);
        if (back) {
          window.location.assign(back);
        }
      }
    } else {
      showError(message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>{t('user.edit.title')}</Card.Header>
          <Form loading={loading} autoComplete='new-password'>
            <Form.Field>
              <Form.Input
                label={t('user.edit.username')}
                name='username'
                placeholder={t('user.edit.username_placeholder')}
                onChange={handleInputChange}
                value={username}
                autoComplete='username'
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('user.edit.password')}
                name='password'
                type={'password'}
                placeholder={t('user.edit.password_placeholder')}
                onChange={handleInputChange}
                value={password}
                autoComplete='new-password'
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('user.edit.display_name')}
                name='display_name'
                placeholder={t('user.edit.display_name_placeholder')}
                onChange={handleInputChange}
                value={display_name}
                autoComplete='off'
              />
            </Form.Field>
            {userId && (
              <>
                <Form.Field>
                  <Form.Dropdown
                    label={t('user.edit.group')}
                    placeholder={t('user.edit.group_placeholder')}
                    name='group'
                    fluid
                    search
                    selection
                    allowAdditions
                    additionLabel={t('user.edit.group_addition')}
                    onChange={handleInputChange}
                    value={inputs.group}
                    autoComplete='new-password'
                    options={groupOptions}
                  />
                </Form.Field>
                <Form.Field>
                  <Form.Input
                    label={`${t('user.edit.quota')}${renderQuotaWithPrompt(
                      quota,
                      t
                    )}`}
                    name='quota'
                    type='text'
                    inputMode='numeric'
                    placeholder={t('user.edit.quota_placeholder')}
                    onChange={handleInputChange}
                    value={
                      inputs.quota === undefined || inputs.quota === null
                        ? ''
                        : String(inputs.quota)
                    }
                    autoComplete='off'
                  />
                </Form.Field>
              </>
            )}
            <Form.Field>
              <Form.Input
                label={t('user.edit.github_id')}
                name='github_id'
                placeholder={t('user.edit.github_id_placeholder')}
                value={github_id}
                readOnly
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('user.edit.wechat_id')}
                name='wechat_id'
                placeholder={t('user.edit.wechat_id_placeholder')}
                value={wechat_id}
                readOnly
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('user.edit.email')}
                name='email'
                placeholder={t('user.edit.email_placeholder')}
                value={email}
                readOnly
              />
            </Form.Field>
            <Button onClick={handleCancel}>
              {t('user.edit.buttons.cancel')}
            </Button>
            <Button positive onClick={submit}>
              {t('user.edit.buttons.submit')}
            </Button>
          </Form>
        </Card.Content>
      </Card>
    </div>
  );
};

export default EditUser;
