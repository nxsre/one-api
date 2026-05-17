import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Form,
  Card,
  Message,
} from 'semantic-ui-react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  API,
  copy,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { renderQuotaWithPrompt } from '../../helpers/render';
import SettingMonacoField from '../../components/SettingMonacoField';

const buildTokenBaseline = (d) => ({
  subnet: String(d?.subnet ?? ''),
});

const EditToken = () => {
  const { t } = useTranslation();
  const params = useParams();
  const tokenId = params.id;
  const isEdit = tokenId !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const [modelOptions, setModelOptions] = useState([]);
  const originInputs = {
    name: '',
    remain_quota: isEdit ? 0 : 500000,
    expired_time: -1,
    unlimited_quota: false,
    models: [],
    subnet: '',
  };
  const [inputs, setInputs] = useState(originInputs);
  const [inputBaseline, setInputBaseline] = useState(() =>
    buildTokenBaseline(originInputs)
  );
  const { name, remain_quota, expired_time, unlimited_quota } = inputs;
  const navigate = useNavigate();
  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };
  const handleCancel = () => {
    navigate('/token');
  };
  const setExpiredTime = (month, day, hour, minute) => {
    let now = new Date();
    let timestamp = now.getTime() / 1000;
    let seconds = month * 30 * 24 * 60 * 60;
    seconds += day * 24 * 60 * 60;
    seconds += hour * 60 * 60;
    seconds += minute * 60;
    if (seconds !== 0) {
      timestamp += seconds;
      setInputs({ ...inputs, expired_time: timestamp2string(timestamp) });
    } else {
      setInputs({ ...inputs, expired_time: -1 });
    }
  };

  const setUnlimitedQuota = () => {
    setInputs({ ...inputs, unlimited_quota: !unlimited_quota });
  };

  const loadToken = async () => {
    try {
      let res = await API.get(`/api/token/${tokenId}`);
      const { success, message, data } = res.data || {};
      if (success && data) {
        if (data.expired_time !== -1) {
          data.expired_time = timestamp2string(data.expired_time);
        }
        if (data.models === '') {
          data.models = [];
        } else {
          data.models = data.models.split(',');
        }
        setInputs(data);
        setInputBaseline(buildTokenBaseline(data));
      } else {
        showError(message || 'Failed to load token');
      }
    } catch (error) {
      showError(error.message || 'Network error');
    }
    setLoading(false);
  };

  const loadAvailableModels = async () => {
    try {
      let res = await API.get(`/api/user/available_models`);
      const { success, message, data } = res.data || {};
      if (success && data) {
        let options = data.map((model) => {
          return {
            key: model,
            text: model,
            value: model,
          };
        });
        setModelOptions(options);
      } else {
        showError(message || 'Failed to load models');
      }
    } catch (error) {
      showError(error.message || 'Network error');
    }
  };

  useEffect(() => {
    if (isEdit) {
      loadToken().catch((error) => {
        showError(error.message || 'Failed to load token');
        setLoading(false);
      });
    }
    loadAvailableModels().catch((error) => {
      showError(error.message || 'Failed to load models');
    });
  }, []);

  const submit = async () => {
    if (!isEdit && inputs.name === '') return;
    let localInputs = inputs;
    localInputs.remain_quota = parseInt(localInputs.remain_quota);
    if (localInputs.expired_time !== -1) {
      let time = Date.parse(localInputs.expired_time);
      if (isNaN(time)) {
        showError(t('token.edit.messages.expire_time_invalid'));
        return;
      }
      localInputs.expired_time = Math.ceil(time / 1000);
    }
    localInputs.models = localInputs.models.join(',');
    let res;
    if (isEdit) {
      res = await API.put(`/api/token/`, {
        ...localInputs,
        id: parseInt(tokenId),
      });
    } else {
      res = await API.post(`/api/token/`, localInputs);
    }
    const { success, message } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('token.edit.messages.update_success'));
      } else {
        showSuccess(t('token.edit.messages.create_success'));
        setInputs(originInputs);
        setInputBaseline(buildTokenBaseline(originInputs));
      }
    } else {
      showError(message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {isEdit ? t('token.edit.title_edit') : t('token.edit.title_create')}
          </Card.Header>
          <Form loading={loading} autoComplete='off'>
            <Form.Field>
              <Form.Input
                label={t('token.edit.name')}
                placeholder={t('token.edit.name_placeholder')}
                name='name'
                value={name}
                onChange={handleInputChange}
                autoComplete='off'
              />
            </Form.Field>
            <Form.Field>
              <Form.Dropdown
                label={t('token.edit.models')}
                placeholder={t('token.edit.models_placeholder')}
                name='models'
                fluid
                multiple
                search
                onLabelClick={(e, { value }) => {
                  copy(value).then();
                }}
                selection
                onChange={handleInputChange}
                value={inputs.models}
                autoComplete='new-password'
                options={modelOptions}
              />
            </Form.Field>
            <SettingMonacoField
              label={t('token.edit.ip_limit')}
              hint={t('token.edit.ip_limit_placeholder')}
              value={inputs.subnet}
              originValue={inputBaseline.subnet}
              onChange={(v) =>
                setInputs((prev) => ({ ...prev, subnet: v }))
              }
              height={120}
            />
            <Form.Field>
              <Form.Input
                label={t('token.edit.expire_time')}
                name='expired_time'
                placeholder={t('token.edit.expire_time_placeholder')}
                onChange={handleInputChange}
                value={expired_time}
                autoComplete='new-password'
                type='datetime-local'
              />
            </Form.Field>
            <div style={{ lineHeight: '40px' }}>
              <Button
                type={'button'}
                onClick={() => {
                  setExpiredTime(0, 0, 0, 0);
                }}
              >
                {t('token.edit.buttons.never_expire')}
              </Button>
              <Button
                type={'button'}
                onClick={() => {
                  setExpiredTime(1, 0, 0, 0);
                }}
              >
                {t('token.edit.buttons.expire_1_month')}
              </Button>
              <Button
                type={'button'}
                onClick={() => {
                  setExpiredTime(0, 1, 0, 0);
                }}
              >
                {t('token.edit.buttons.expire_1_day')}
              </Button>
              <Button
                type={'button'}
                onClick={() => {
                  setExpiredTime(0, 0, 1, 0);
                }}
              >
                {t('token.edit.buttons.expire_1_hour')}
              </Button>
              <Button
                type={'button'}
                onClick={() => {
                  setExpiredTime(0, 0, 0, 1);
                }}
              >
                {t('token.edit.buttons.expire_1_minute')}
              </Button>
            </div>
            <Message>{t('token.edit.quota_notice')}</Message>
            <Form.Field>
              <Form.Input
                label={`${t('token.edit.quota')}${renderQuotaWithPrompt(
                  remain_quota,
                  t
                )}`}
                placeholder={t('token.edit.quota_placeholder')}
                value={String(remain_quota ?? '')}
                readOnly={unlimited_quota}
                autoComplete='off'
                onChange={(e, { value: v }) => {
                  if (unlimited_quota) return;
                  const n = v.trim() === '' ? 0 : parseInt(v, 10);
                  setInputs((prev) => ({
                    ...prev,
                    remain_quota: Number.isNaN(n) ? prev.remain_quota : n,
                  }));
                }}
              />
            </Form.Field>
            <Button
              type={'button'}
              onClick={() => {
                setUnlimitedQuota();
              }}
            >
              {unlimited_quota
                ? t('token.edit.buttons.cancel_unlimited')
                : t('token.edit.buttons.unlimited_quota')}
            </Button>
            <Button floated='right' positive onClick={submit}>
              {t('token.edit.buttons.submit')}
            </Button>
            <Button floated='right' onClick={handleCancel}>
              {t('token.edit.buttons.cancel')}
            </Button>
          </Form>
        </Card.Content>
      </Card>
    </div>
  );
};

export default EditToken;
