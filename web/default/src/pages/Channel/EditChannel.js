import React, {useEffect, useState} from 'react';
import {useTranslation} from 'react-i18next';
import {Button, Card, Form, Message, Modal} from 'semantic-ui-react';
import {useNavigate, useParams} from 'react-router-dom';
import {
  API,
  copy,
  getChannelModels,
  splitModelNameList,
  showError,
  showInfo,
  showSuccess,
  verifyJSON,
  verifyStepUp2FA,
  fetchChannelKeyAfterVerify,
} from '../../helpers';
import {CHANNEL_OPTIONS} from '../../constants';
import {renderChannelTip} from '../../helpers/render';
import SettingMonacoField from '../../components/SettingMonacoField';

const buildChannelInputBaseline = (inp, customModelStr) => ({
  name: String(inp?.name ?? ''),
  key: String(inp?.key ?? ''),
  base_url: String(inp?.base_url ?? ''),
  other: String(inp?.other ?? ''),
  model_mapping: String(inp?.model_mapping ?? ''),
  system_prompt: String(inp?.system_prompt ?? ''),
  customModel: String(customModelStr ?? ''),
});

const MODEL_MAPPING_EXAMPLE = {
  'gpt-3.5-turbo-0301': 'gpt-3.5-turbo',
  'gpt-4-0314': 'gpt-4',
  'gpt-4-32k-0314': 'gpt-4-32k',
};

const DEFAULT_CHANNEL_CONFIG = {
  region: '',
  sk: '',
  ak: '',
  user_id: '',
  vertex_ai_project_id: '',
  vertex_ai_adc: '',
};

function type2secretPrompt(type, t) {
  switch (type) {
    case 15:
      return t('channel.edit.key_prompts.zhipu');
    case 18:
      return t('channel.edit.key_prompts.spark');
    case 22:
      return t('channel.edit.key_prompts.fastgpt');
    case 23:
      return t('channel.edit.key_prompts.tencent');
    case 52:
      return t('channel.edit.key_prompts.aippt');
    case 53:
      return t('channel.edit.key_prompts.amap_poi');
    default:
      return t('channel.edit.key_prompts.default');
  }
}

const EditChannel = () => {
  const { t } = useTranslation();
  const params = useParams();
  const navigate = useNavigate();
  const channelId = params.id;
  const isEdit = channelId !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const handleCancel = () => {
    navigate('/channel');
  };

  const openLoadKeyModal = async () => {
    try {
      const res = await API.get('/api/user/2fa/status');
      if (!res.data?.success || !res.data?.data?.enabled) {
        showInfo(t('channel.edit.load_key_need_2fa'));
        return;
      }
      setLoadKeyCode('');
      setLoadKeyOpen(true);
    } catch (e) {
      showError(e.message || '无法检查两步验证状态');
    }
  };

  const submitLoadKey = async () => {
    if (!loadKeyCode.trim()) {
      showInfo(t('channel.edit.load_key_code_required'));
      return;
    }
    setLoadKeyBusy(true);
    try {
      await verifyStepUp2FA(loadKeyCode);
      const key = await fetchChannelKeyAfterVerify(channelId);
      setInputs((prev) => ({ ...prev, key }));
      setLoadKeyOpen(false);
      setLoadKeyCode('');
      showSuccess(t('channel.edit.load_key_success'));
    } catch (e) {
      showError(e.message || '加载失败');
    } finally {
      setLoadKeyBusy(false);
    }
  };

  const defaultChannelInputs = {
    name: '',
    type: 1,
    key: '',
    base_url: '',
    other: '',
    model_mapping: '',
    system_prompt: '',
    models: [],
    groups: ['default'],
  };
  const [batch, setBatch] = useState(false);
  const [loadKeyOpen, setLoadKeyOpen] = useState(false);
  const [loadKeyCode, setLoadKeyCode] = useState('');
  const [loadKeyBusy, setLoadKeyBusy] = useState(false);
  const [inputs, setInputs] = useState(defaultChannelInputs);
  const [inputBaseline, setInputBaseline] = useState(() =>
    buildChannelInputBaseline(defaultChannelInputs, '')
  );
  const [originModelOptions, setOriginModelOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [basicModels, setBasicModels] = useState([]);
  const [fullModels, setFullModels] = useState([]);
  const [customModel, setCustomModel] = useState('');
  const [config, setConfig] = useState({ ...DEFAULT_CHANNEL_CONFIG });
  const [configBaseline, setConfigBaseline] = useState({
    ...DEFAULT_CHANNEL_CONFIG,
  });
  const handleInputChange = (e, { name, value }) => {
    if (name === 'type') {
      const localModels = getChannelModels(value);
      setBasicModels(localModels);
      setInputs((prev) => {
        const typeChanged = prev.type !== value;
        const models =
          typeChanged && prev.models.length === 0 ? localModels : prev.models;
        return { ...prev, type: value, models };
      });
      return;
    }
    setInputs((prev) => ({ ...prev, [name]: value }));
  };

  const handleConfigChange = (e, { name, value }) => {
    setConfig((inputs) => ({ ...inputs, [name]: value }));
  };

  const loadChannel = async () => {
    let res = await API.get(`/api/channel/${channelId}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.models === '') {
        data.models = [];
      } else {
        data.models = [
          ...new Set(
            data.models
              .split(',')
              .map((id) => id.trim())
              .filter(Boolean)
          ),
        ];
      }
      if (data.group === '') {
        data.groups = [];
      } else {
        data.groups = data.group.split(',');
      }
      if (data.model_mapping !== '') {
        data.model_mapping = JSON.stringify(
          JSON.parse(data.model_mapping),
          null,
          2
        );
      }
      setInputs(data);
      setInputBaseline(buildChannelInputBaseline(data, ''));
      if (data.config !== '') {
        const cfg = JSON.parse(data.config);
        setConfig({ ...DEFAULT_CHANNEL_CONFIG, ...cfg });
        setConfigBaseline({ ...DEFAULT_CHANNEL_CONFIG, ...cfg });
      } else {
        setConfig({ ...DEFAULT_CHANNEL_CONFIG });
        setConfigBaseline({ ...DEFAULT_CHANNEL_CONFIG });
      }
      setBasicModels(getChannelModels(data.type));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const fetchModels = async () => {
    try {
      let res = await API.get(`/api/channel/models`);
      let localModelOptions = res.data.data.map((model) => ({
        key: model.id,
        text: model.id,
        value: model.id,
      }));
      setOriginModelOptions(localModelOptions);
      setFullModels(res.data.data.map((model) => model.id));
    } catch (error) {
      showError(error.message);
    }
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

  useEffect(() => {
    let localModelOptions = [...originModelOptions];
    inputs.models.forEach((model) => {
      if (!localModelOptions.find((option) => option.key === model)) {
        localModelOptions.push({
          key: model,
          text: model,
          value: model,
        });
      }
    });
    setModelOptions(localModelOptions);
  }, [originModelOptions, inputs.models]);

  useEffect(() => {
    if (isEdit) {
      loadChannel().then();
    } else {
      let localModels = getChannelModels(inputs.type);
      setBasicModels(localModels);
    }
    fetchModels().then();
    fetchGroups().then();
  }, []);

  const submit = async () => {
    if (inputs.key === '') {
      if (config.ak !== '' && config.sk !== '' && config.region !== '') {
        inputs.key = `${config.ak}|${config.sk}|${config.region}`;
      } else if (
        config.region !== '' &&
        config.vertex_ai_project_id !== '' &&
        config.vertex_ai_adc !== ''
      ) {
        inputs.key = `${config.region}|${config.vertex_ai_project_id}|${config.vertex_ai_adc}`;
      }
    }
    if (!isEdit && (inputs.name === '' || inputs.key === '')) {
      showInfo(t('channel.edit.messages.name_required'));
      return;
    }
    if (inputs.type !== 43 && inputs.models.length === 0) {
      showInfo(t('channel.edit.messages.models_required'));
      return;
    }
    if (inputs.model_mapping !== '' && !verifyJSON(inputs.model_mapping)) {
      showInfo(t('channel.edit.messages.model_mapping_invalid'));
      return;
    }
    let localInputs = { ...inputs };
    if (localInputs.key === 'undefined|undefined|undefined') {
      localInputs.key = ''; // prevent potential bug
    }
    if (localInputs.base_url && localInputs.base_url.endsWith('/')) {
      localInputs.base_url = localInputs.base_url.slice(
        0,
        localInputs.base_url.length - 1
      );
    }
    if (localInputs.type === 3 && localInputs.other === '') {
      localInputs.other = '2024-03-01-preview';
    }
    let res;
    localInputs.models = localInputs.models.join(',');
    localInputs.group = localInputs.groups.join(',');
    localInputs.config = JSON.stringify(config);
    if (isEdit) {
      res = await API.put(`/api/channel/`, {
        ...localInputs,
        id: parseInt(channelId),
      });
    } else {
      res = await API.post(`/api/channel/`, localInputs);
    }
    const { success, message } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('channel.edit.messages.update_success'));
        await loadChannel();
      } else {
        showSuccess(t('channel.edit.messages.create_success'));
        setInputs(defaultChannelInputs);
        setInputBaseline(
          buildChannelInputBaseline(defaultChannelInputs, '')
        );
        setCustomModel('');
        setConfig({ ...DEFAULT_CHANNEL_CONFIG });
        setConfigBaseline({ ...DEFAULT_CHANNEL_CONFIG });
      }
    } else {
      showError(message);
    }
  };

  const addCustomModel = () => {
    const names = splitModelNameList(customModel);
    if (names.length === 0) return;
    const localModels = [...inputs.models];
    const newOptions = [];
    for (const name of names) {
      if (localModels.includes(name)) continue;
      localModels.push(name);
      newOptions.push({ key: name, text: name, value: name });
    }
    if (newOptions.length === 0) {
      showInfo(t('channel.edit.messages.custom_models_all_exist'));
      return;
    }
    setModelOptions((modelOptions) => [...modelOptions, ...newOptions]);
    setCustomModel('');
    handleInputChange(null, { name: 'models', value: localModels });
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {isEdit
              ? t('channel.edit.title_edit')
              : t('channel.edit.title_create')}
          </Card.Header>
          <Form loading={loading} autoComplete='new-password'>
            <Form.Field>
              <Form.Select
                label={t('channel.edit.type')}
                name='type'
                required
                search
                options={CHANNEL_OPTIONS}
                value={inputs.type}
                onChange={handleInputChange}
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('channel.edit.name')}
                placeholder={t('channel.edit.name_placeholder')}
                name='name'
                required
                value={inputs.name}
                onChange={handleInputChange}
                autoComplete='off'
              />
            </Form.Field>
            <Form.Field>
              <Form.Dropdown
                label={t('channel.edit.group')}
                placeholder={t('channel.edit.group_placeholder')}
                name='groups'
                required
                fluid
                multiple
                selection
                allowAdditions
                additionLabel={t('channel.edit.group_addition')}
                onChange={handleInputChange}
                value={inputs.groups}
                autoComplete='new-password'
                options={groupOptions}
              />
            </Form.Field>
            {renderChannelTip(inputs.type)}

            {/* Azure OpenAI specific fields */}
            {inputs.type === 3 && (
              <>
                <Message>
                  注意，<strong>模型部署名称必须和模型名称保持一致</strong>
                  ，因为 One API 会把请求体中的 model
                  参数替换为你的部署名称（模型名称中的点会被剔除），
                  <a
                    target='_blank'
                    href='https://github.com/songquanpeng/one-api/issues/133?notification_referrer_id=NT_kwDOAmJSYrM2NjIwMzI3NDgyOjM5OTk4MDUw#issuecomment-1571602271'
                  >
                    图片演示
                  </a>
                  。
                </Message>
                <SettingMonacoField
                  label='AZURE_OPENAI_ENDPOINT'
                  hint='请输入 AZURE_OPENAI_ENDPOINT，例如：https://docs-test-001.openai.azure.com'
                  value={inputs.base_url}
                  originValue={inputBaseline.base_url}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'base_url', value: v })
                  }
                  height={96}
                />
                <SettingMonacoField
                  label='默认 API 版本'
                  hint='请输入默认 API 版本，例如：2024-03-01-preview，该配置可以被实际的请求查询参数所覆盖'
                  value={inputs.other}
                  originValue={inputBaseline.other}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'other', value: v })
                  }
                  height={96}
                />
              </>
            )}

            {/* Custom base URL field */}
            {inputs.type === 8 && (
              <SettingMonacoField
                label={t('channel.edit.proxy_url')}
                hint={t('channel.edit.proxy_url_placeholder')}
                value={inputs.base_url}
                originValue={inputBaseline.base_url}
                onChange={(v) =>
                  handleInputChange(null, { name: 'base_url', value: v })
                }
                height={96}
              />
            )}
            {inputs.type === 50 && (
                <SettingMonacoField
                  label={t('channel.edit.base_url')}
                  hint={t('channel.edit.base_url_placeholder')}
                  value={inputs.base_url}
                  originValue={inputBaseline.base_url}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'base_url', value: v })
                  }
                  height={96}
                />
            )}

            {inputs.type === 18 && (
              <SettingMonacoField
                label={t('channel.edit.spark_version')}
                hint={t('channel.edit.spark_version_placeholder')}
                value={inputs.other}
                originValue={inputBaseline.other}
                onChange={(v) =>
                  handleInputChange(null, { name: 'other', value: v })
                }
                height={88}
              />
            )}
            {inputs.type === 21 && (
              <SettingMonacoField
                label={t('channel.edit.knowledge_id')}
                hint={t('channel.edit.knowledge_id_placeholder')}
                value={inputs.other}
                originValue={inputBaseline.other}
                onChange={(v) =>
                  handleInputChange(null, { name: 'other', value: v })
                }
                height={88}
              />
            )}
            {inputs.type === 17 && (
              <SettingMonacoField
                label={t('channel.edit.plugin_param')}
                hint={t('channel.edit.plugin_param_placeholder')}
                value={inputs.other}
                originValue={inputBaseline.other}
                onChange={(v) =>
                  handleInputChange(null, { name: 'other', value: v })
                }
                height={120}
              />
            )}
            {inputs.type === 34 && (
              <Message>{t('channel.edit.coze_notice')}</Message>
            )}
            {inputs.type === 40 && (
              <Message>
                {t('channel.edit.douban_notice')}
                <a
                  target='_blank'
                  href='https://console.volcengine.com/ark/region:ark+cn-beijing/endpoint'
                >
                  {t('channel.edit.douban_notice_link')}
                </a>
                {t('channel.edit.douban_notice_2')}
              </Message>
            )}
            {inputs.type !== 43 && (
              <Form.Field>
                <Form.Dropdown
                  label={t('channel.edit.models')}
                  placeholder={t('channel.edit.models_placeholder')}
                  name='models'
                  required
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
            )}
            {inputs.type !== 43 && (
              <div style={{ lineHeight: '40px', marginBottom: '12px' }}>
                <Button
                  type={'button'}
                  onClick={() => {
                    handleInputChange(null, {
                      name: 'models',
                      value: basicModels,
                    });
                  }}
                >
                  {t('channel.edit.buttons.fill_models')}
                </Button>
                <Button
                  type={'button'}
                  onClick={() => {
                    handleInputChange(null, {
                      name: 'models',
                      value: fullModels,
                    });
                  }}
                >
                  {t('channel.edit.buttons.fill_all')}
                </Button>
                <Button
                  type={'button'}
                  onClick={() => {
                    handleInputChange(null, { name: 'models', value: [] });
                  }}
                >
                  {t('channel.edit.buttons.clear')}
                </Button>
                <SettingMonacoField
                  label={t('channel.edit.models')}
                  hint={t('channel.edit.buttons.custom_placeholder')}
                  value={customModel}
                  originValue={inputBaseline.customModel}
                  onChange={setCustomModel}
                  height={156}
                />
                <Button type={'button'} onClick={addCustomModel} style={{ marginTop: 8 }}>
                  {t('channel.edit.buttons.add_custom')}
                </Button>
              </div>
            )}
            {inputs.type !== 43 && (
              <>
                <SettingMonacoField
                  label={t('channel.edit.model_mapping')}
                  hint={`${t(
                    'channel.edit.model_mapping_placeholder'
                  )}\n${JSON.stringify(MODEL_MAPPING_EXAMPLE, null, 2)}`}
                  language='json'
                  enableJsonFormat
                  value={inputs.model_mapping}
                  originValue={inputBaseline.model_mapping}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'model_mapping', value: v })
                  }
                  height={272}
                  minimap
                />
                <SettingMonacoField
                  label={t('channel.edit.system_prompt')}
                  hint={t('channel.edit.system_prompt_placeholder')}
                  language='plaintext'
                  value={inputs.system_prompt}
                  originValue={inputBaseline.system_prompt}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'system_prompt', value: v })
                  }
                  height={272}
                  minimap
                />
              </>
            )}
            {inputs.type === 33 && (
              <Form.Field>
                <SettingMonacoField
                  label='Region'
                  hint={t('channel.edit.aws_region_placeholder')}
                  value={config.region}
                  originValue={configBaseline.region}
                  onChange={(v) =>
                    handleConfigChange(null, { name: 'region', value: v })
                  }
                  height={88}
                />
                <SettingMonacoField
                  label='AK'
                  hint={t('channel.edit.aws_ak_placeholder')}
                  value={config.ak}
                  originValue={configBaseline.ak}
                  onChange={(v) =>
                    handleConfigChange(null, { name: 'ak', value: v })
                  }
                  height={88}
                />
                <Form.Input
                  label='SK'
                  name='sk'
                  required
                  placeholder={t('channel.edit.aws_sk_placeholder')}
                  onChange={handleConfigChange}
                  value={config.sk}
                  type='password'
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type === 42 && (
              <Form.Field>
                <SettingMonacoField
                  label='Region'
                  hint={t('channel.edit.vertex_region_placeholder')}
                  value={config.region}
                  originValue={configBaseline.region}
                  onChange={(v) =>
                    handleConfigChange(null, { name: 'region', value: v })
                  }
                  height={88}
                />
                <SettingMonacoField
                  label={t('channel.edit.vertex_project_id')}
                  hint={t('channel.edit.vertex_project_id_placeholder')}
                  value={config.vertex_ai_project_id}
                  originValue={configBaseline.vertex_ai_project_id}
                  onChange={(v) =>
                    handleConfigChange(null, {
                      name: 'vertex_ai_project_id',
                      value: v,
                    })
                  }
                  height={88}
                />
                <SettingMonacoField
                  label={t('channel.edit.vertex_credentials')}
                  hint={t('channel.edit.vertex_credentials_placeholder')}
                  language='json'
                  enableJsonFormat
                  minimap
                  value={config.vertex_ai_adc}
                  originValue={configBaseline.vertex_ai_adc}
                  onChange={(v) =>
                    handleConfigChange(null, {
                      name: 'vertex_ai_adc',
                      value: v,
                    })
                  }
                  height={280}
                />
              </Form.Field>
            )}
            {inputs.type === 34 && (
              <SettingMonacoField
                label={t('channel.edit.user_id')}
                hint={t('channel.edit.user_id_placeholder')}
                value={config.user_id}
                originValue={configBaseline.user_id}
                onChange={(v) =>
                  handleConfigChange(null, { name: 'user_id', value: v })
                }
                height={88}
              />
            )}
            {isEdit &&
              !batch &&
              inputs.type !== 33 &&
              inputs.type !== 42 && (
                <Message info>
                  <p>{t('channel.edit.load_key_hint')}</p>
                  <Button
                    type='button'
                    size='small'
                    onClick={() => void openLoadKeyModal()}
                  >
                    {t('channel.edit.load_key_button')}
                  </Button>
                </Message>
              )}
            {inputs.type !== 33 &&
              inputs.type !== 42 &&
              (batch ? (
                <SettingMonacoField
                  label={t('channel.edit.key')}
                  hint={t('channel.edit.batch_placeholder')}
                  value={inputs.key}
                  originValue={inputBaseline.key}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'key', value: v })
                  }
                  height={220}
                  minimap
                />
              ) : (
                <Form.Field>
                  <Form.Input
                    label={t('channel.edit.key')}
                    name='key'
                    required
                    placeholder={type2secretPrompt(inputs.type, t)}
                    onChange={handleInputChange}
                    value={inputs.key}
                    autoComplete='new-password'
                  />
                </Form.Field>
              ))}
            {inputs.type === 37 && (
              <SettingMonacoField
                label='Account ID'
                hint='请输入 Account ID，例如：d8d7c61dbc334c32d3ced580e4bf42b4'
                value={config.user_id}
                originValue={configBaseline.user_id}
                onChange={(v) =>
                  handleConfigChange(null, { name: 'user_id', value: v })
                }
                height={88}
              />
            )}
            {inputs.type !== 33 && !isEdit && (
              <Form.Checkbox
                checked={batch}
                label={t('channel.edit.batch')}
                name='batch'
                onChange={() => setBatch(!batch)}
              />
            )}
            {inputs.type !== 3 &&
              inputs.type !== 33 &&
              inputs.type !== 8 &&
                inputs.type !== 50 &&
              inputs.type !== 22 && (
                <SettingMonacoField
                  label={t('channel.edit.proxy_url')}
                  hint={t('channel.edit.proxy_url_placeholder')}
                  value={inputs.base_url}
                  originValue={inputBaseline.base_url}
                  onChange={(v) =>
                    handleInputChange(null, { name: 'base_url', value: v })
                  }
                  height={96}
                />
              )}
            {inputs.type === 22 && (
              <SettingMonacoField
                label='私有部署地址'
                hint='请输入私有部署地址，格式为：https://fastgpt.run/api/openapi'
                value={inputs.base_url}
                originValue={inputBaseline.base_url}
                onChange={(v) =>
                  handleInputChange(null, { name: 'base_url', value: v })
                }
                height={96}
              />
            )}
            <Button onClick={handleCancel}>
              {t('channel.edit.buttons.cancel')}
            </Button>
            <Button
              type={isEdit ? 'button' : 'submit'}
              positive
              onClick={submit}
            >
              {t('channel.edit.buttons.submit')}
            </Button>
          </Form>
        </Card.Content>
      </Card>
      <Modal
        open={loadKeyOpen}
        onClose={() => !loadKeyBusy && setLoadKeyOpen(false)}
        size='small'
      >
        <Modal.Header>{t('channel.edit.load_key_modal_title')}</Modal.Header>
        <Modal.Content>
          <Form.Input
            label={t('channel.edit.load_key_modal_code')}
            value={loadKeyCode}
            onChange={(e, { value }) => setLoadKeyCode(value)}
            autoComplete='one-time-code'
          />
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setLoadKeyOpen(false)} disabled={loadKeyBusy}>
            {t('channel.edit.buttons.cancel')}
          </Button>
          <Button
            positive
            loading={loadKeyBusy}
            onClick={() => void submitLoadKey()}
          >
            {t('channel.edit.load_key_modal_submit')}
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
};

export default EditChannel;
