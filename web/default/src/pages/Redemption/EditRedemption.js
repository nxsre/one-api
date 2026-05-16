import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Form, Card } from 'semantic-ui-react';
import { useParams, useNavigate } from 'react-router-dom';
import { API, downloadTextAsFile, showError, showSuccess } from '../../helpers';
import { renderQuotaWithPrompt } from '../../helpers/render';
import SettingMonacoField from '../../components/SettingMonacoField';

const buildRedemptionBaseline = (d) => ({
  name: String(d?.name ?? ''),
  quota:
    d?.quota === undefined || d?.quota === null ? '' : String(d.quota),
  count:
    d?.count === undefined || d?.count === null ? '' : String(d.count),
});

const EditRedemption = () => {
  const { t } = useTranslation();
  const params = useParams();
  const navigate = useNavigate();
  const redemptionId = params.id;
  const isEdit = redemptionId !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const originInputs = {
    name: '',
    quota: 100000,
    count: 1,
  };
  const [inputs, setInputs] = useState(originInputs);
  const [inputBaseline, setInputBaseline] = useState(() =>
    buildRedemptionBaseline(originInputs)
  );
  const { name, quota, count } = inputs;

  const handleCancel = () => {
    navigate('/redemption');
  };

  const loadRedemption = async () => {
    let res = await API.get(`/api/redemption/${redemptionId}`);
    const { success, message, data } = res.data;
    if (success) {
      setInputs(data);
      setInputBaseline(buildRedemptionBaseline(data));
    } else {
      showError(message);
    }
    setLoading(false);
  };
  useEffect(() => {
    if (isEdit) {
      loadRedemption().then();
    }
  }, []);

  const submit = async () => {
    if (!isEdit && inputs.name === '') return;
    let localInputs = inputs;
    localInputs.count = parseInt(localInputs.count);
    localInputs.quota = parseInt(localInputs.quota);
    let res;
    if (isEdit) {
      res = await API.put(`/api/redemption/`, {
        ...localInputs,
        id: parseInt(redemptionId),
      });
    } else {
      res = await API.post(`/api/redemption/`, {
        ...localInputs,
      });
    }
    const { success, message, data } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('redemption.messages.update_success'));
      } else {
        showSuccess(t('redemption.messages.create_success'));
        setInputs(originInputs);
        setInputBaseline(buildRedemptionBaseline(originInputs));
      }
    } else {
      showError(message);
    }
    if (!isEdit && data) {
      let text = '';
      for (let i = 0; i < data.length; i++) {
        text += data[i] + '\n';
      }
      downloadTextAsFile(text, `${inputs.name}.txt`);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {isEdit ? t('redemption.edit.title_edit') : t('redemption.edit.title_create')}
          </Card.Header>
          <Form loading={loading} autoComplete='new-password'>
            <SettingMonacoField
              label={t('redemption.edit.name')}
              hint={t('redemption.edit.name_placeholder')}
              value={name}
              originValue={inputBaseline.name}
              onChange={(v) =>
                setInputs((prev) => ({ ...prev, name: v }))
              }
              height={96}
            />
            <SettingMonacoField
              label={`${t('redemption.edit.quota')}${renderQuotaWithPrompt(quota, t)}`}
              hint={t('redemption.edit.quota_placeholder')}
              value={String(quota ?? '')}
              originValue={inputBaseline.quota}
              onChange={(v) => {
                const n = v.trim() === '' ? 0 : parseInt(v, 10);
                setInputs((prev) => ({
                  ...prev,
                  quota: Number.isNaN(n) ? prev.quota : n,
                }));
              }}
              height={88}
            />
            {!isEdit && (
              <>
                <SettingMonacoField
                  label={t('redemption.edit.count')}
                  hint={t('redemption.edit.count_placeholder')}
                  value={String(count ?? '')}
                  originValue={inputBaseline.count}
                  onChange={(v) => {
                    const n = v.trim() === '' ? 0 : parseInt(v, 10);
                    setInputs((prev) => ({
                      ...prev,
                      count: Number.isNaN(n) ? prev.count : n,
                    }));
                  }}
                  height={88}
                />
              </>
            )}
            <Button positive onClick={submit}>
              {t('redemption.edit.buttons.submit')}
            </Button>
            <Button onClick={handleCancel}>
              {t('redemption.edit.buttons.cancel')}
            </Button>
          </Form>
        </Card.Content>
      </Card>
    </div>
  );
};

export default EditRedemption;
