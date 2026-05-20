import React, {useCallback, useEffect, useRef, useState} from 'react';
import {useTranslation} from 'react-i18next';
import {
  Button,
  Card,
  Form,
  Header,
  Input,
  Message,
  Modal,
  Checkbox,
  Progress,
} from 'semantic-ui-react';
import {useNavigate, useParams} from 'react-router-dom';
import {
  API,
  copy,
  splitModelNameList,
  showError,
  showInfo,
  showSuccess,
  verifyJSON,
  verifyStepUp2FA,
  fetchChannelKeyAfterVerify,
  noAutofillFormProps,
  noAutofillSecretProps,
  noAutofillTextProps,
} from '../../helpers';
import {setChannelTypeOptionsCache} from '../../helpers/helper';
import {renderChannelTip} from '../../helpers/render';
import ChannelModelMappingEditor from './ChannelModelMappingEditor';
import FillCatalogModelsModal from './FillCatalogModelsModal';
import { fetchModelCatalogModelIds } from '../../helpers/modelCatalog';
import {
  applyJobSnapshotToModelState,
  applyStoredModelTestResults,
  buildChannelModelTestPayload,
  buildJobProgressSummary,
  buildStoredModelTestSummary,
  fetchChannelModelTestResults,
  filterOutFailedModels,
  MODEL_TEST_FAIL,
  MODEL_TEST_OK,
  normalizeChannelTestBaseUrl,
  renderChannelModelLabel,
  runChannelModelTestJob,
} from '../../helpers/channelModelTest';
import './ChannelEdit.css';

const DEFAULT_CHANNEL_CONFIG = {
  region: '',
  sk: '',
  ak: '',
  user_id: '',
  vertex_ai_project_id: '',
  vertex_ai_adc: '',
  api_version: '',
  library_id: '',
  plugin: '',
  routing_provider: '',
  routing_skip_adaptive: false,
  deep_research_mode: '',
};

function type2secretPrompt(type, t) {
  switch (type) {
    case 43:
      return t('channel.edit.key_prompts.aippt');
    case 44:
      return t('channel.edit.key_prompts.amap_poi');
    case 45:
      return t('channel.edit.key_prompts.deep_research');
    case 5:
    case 46:
      return t('channel.edit.key_prompts.anthropic');
    case 14:
    case 15:
    case 42:
      return t('channel.edit.key_prompts.gemini');
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
    type: 41,
    key: '',
    base_url: '',
    model_mapping: '',
    system_prompt: '',
    models: [],
    groups: ['default'],
    openai_organization: '',
    test_model: '',
    auto_ban: 1,
    remark: '',
    tag: '',
    status_code_mapping: '',
    param_override: '',
    header_override: '',
    setting: '',
    settings: '',
    other_info: '',
  };
  const [channelTypeOptions, setChannelTypeOptions] = useState([]);
  const [channelTypesLoading, setChannelTypesLoading] = useState(true);
  const [batch, setBatch] = useState(false);
  const [loadKeyOpen, setLoadKeyOpen] = useState(false);
  const [loadKeyCode, setLoadKeyCode] = useState('');
  const [loadKeyBusy, setLoadKeyBusy] = useState(false);
  const [fetchUpstreamBusy, setFetchUpstreamBusy] = useState(false);
  const [fillCatalogOpen, setFillCatalogOpen] = useState(false);
  const [fillAllBusy, setFillAllBusy] = useState(false);
  const [testModelsBusy, setTestModelsBusy] = useState(false);
  const [testModelsProgress, setTestModelsProgress] = useState({
    total: 0,
    completed: 0,
    currentModel: '',
    concurrency: 3,
    running: false,
  });
  const [testModelsConcurrency, setTestModelsConcurrency] = useState(3);
  const [modelTestStatus, setModelTestStatus] = useState({});
  const [modelTestMessages, setModelTestMessages] = useState({});
  const [modelTestMeta, setModelTestMeta] = useState({});
  const [modelTestSummary, setModelTestSummary] = useState('');
  const testModelsPollRef = useRef(false);
  const [inputs, setInputs] = useState(defaultChannelInputs);
  const [originModelOptions, setOriginModelOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [customModel, setCustomModel] = useState('');
  const [config, setConfig] = useState({ ...DEFAULT_CHANNEL_CONFIG });
  const handleInputChange = (e, { name, value }) => {
    if (name === 'models') {
      const keep = (obj) => {
        const allowed = new Set((value || []).map((m) => String(m)));
        return Object.fromEntries(
          Object.entries(obj || {}).filter(([key]) => allowed.has(key))
        );
      };
      setModelTestStatus((prev) => keep(prev));
      setModelTestMessages((prev) => keep(prev));
      setModelTestMeta((prev) => keep(prev));
    }
    if (name === 'type') {
      const opt = channelTypeOptions.find((o) => o.value === value);
      const defaultUrl =
        opt && opt.default_base_url
          ? String(opt.default_base_url).trim()
          : '';
      setInputs((prev) => {
        const typeChanged = prev.type !== value;
        let base_url = prev.base_url;
        if (typeChanged && defaultUrl) {
          const prevTrim = String(prev.base_url || '').trim();
          const prevOpt = channelTypeOptions.find((o) => o.value === prev.type);
          const prevDefault =
            prevOpt && prevOpt.default_base_url
              ? String(prevOpt.default_base_url).trim()
              : '';
          if (!prevTrim || prevTrim === prevDefault) {
            base_url = defaultUrl;
          }
        }
        return { ...prev, type: value, base_url };
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
      const { other: _legacyOther, ...channelData } = data;
      if (channelData.auto_ban === undefined || channelData.auto_ban === null) {
        channelData.auto_ban = 1;
      }
      channelData.openai_organization = channelData.openai_organization ?? '';
      channelData.test_model = channelData.test_model ?? '';
      channelData.remark = channelData.remark ?? '';
      channelData.tag = channelData.tag ?? '';
      channelData.status_code_mapping = channelData.status_code_mapping ?? '';
      channelData.param_override = channelData.param_override ?? '';
      channelData.header_override = channelData.header_override ?? '';
      channelData.setting = channelData.setting ?? '';
      channelData.settings = channelData.settings ?? '';
      channelData.other_info = channelData.other_info ?? '';
      setInputs(channelData);
      if (data.config !== '' && data.config != null) {
        const cfg = JSON.parse(data.config);
        const merged = { ...DEFAULT_CHANNEL_CONFIG, ...cfg };
        setConfig(merged);
      } else {
        const empty = { ...DEFAULT_CHANNEL_CONFIG };
        setConfig(empty);
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const applyJobSnapshot = useCallback(
    (jobData, models) => {
      const applied = applyJobSnapshotToModelState(jobData, models);
      setModelTestStatus(applied.status);
      setModelTestMessages(applied.messages);
      setModelTestMeta(applied.meta);
      setTestModelsProgress(applied.progress);
      setModelTestSummary(buildJobProgressSummary(t, jobData));
    },
    [t]
  );

  const pollRunningJob = useCallback(
    async (payload, models) => {
      if (testModelsPollRef.current) return;
      testModelsPollRef.current = true;
      setTestModelsBusy(true);
      try {
        const summary = await runChannelModelTestJob(payload, models, {
          onJobSnapshot: (data) => applyJobSnapshot(data, models),
        });
        if (summary.fail === 0 && summary.ok > 0) {
          showSuccess(t('channel.edit.messages.test_models_all_ok', { count: summary.ok }));
        } else if (summary.fail > 0) {
          showInfo(
            t('channel.edit.messages.test_models_partial', {
              ok: summary.ok,
              fail: summary.fail,
            })
          );
        }
      } catch (e) {
        showError(e.message || t('channel.edit.messages.test_models_fail'));
      } finally {
        setTestModelsBusy(false);
        setTestModelsProgress((prev) => ({ ...prev, running: false, currentModel: '' }));
        testModelsPollRef.current = false;
      }
    },
    [applyJobSnapshot, t]
  );

  const loadStoredModelTestResults = useCallback(async () => {
    if (!isEdit || !channelId) return;
    try {
      const base = normalizeChannelTestBaseUrl(
        inputs.base_url,
        inputs.type,
        channelTypeOptions
      );
      const { results, job } = await fetchChannelModelTestResults({
        channelId,
        baseUrl: base,
        channelType: inputs.type,
        channelTypeOptions,
      });
      if (job?.running) {
        applyJobSnapshot(job, inputs.models);
        const payload = buildChannelModelTestPayload({
          channelId,
          type: inputs.type,
          baseUrl: inputs.base_url,
          key: '',
          config,
          modelMapping: inputs.model_mapping,
          channelTypeOptions,
          concurrency: testModelsConcurrency,
        });
        void pollRunningJob(payload, inputs.models);
        return;
      }
      const applied = applyStoredModelTestResults(results);
      setModelTestStatus(applied.status);
      setModelTestMessages(applied.messages);
      setModelTestMeta(applied.meta);
      setTestModelsProgress({ total: 0, completed: 0, currentModel: '', running: false });
      const visible = (inputs.models || []).filter((m) => applied.status[m]);
      const ok = visible.filter((m) => applied.status[m] === MODEL_TEST_OK).length;
      const fail = visible.filter((m) => applied.status[m] === MODEL_TEST_FAIL).length;
      setModelTestSummary(
        buildStoredModelTestSummary(t, ok, fail, visible.length)
      );
    } catch {
      // 忽略历史加载失败
    }
  }, [
    isEdit,
    channelId,
    inputs.base_url,
    inputs.type,
    inputs.models,
    inputs.model_mapping,
    channelTypeOptions,
    config,
    applyJobSnapshot,
    pollRunningJob,
    testModelsConcurrency,
    t,
  ]);

  useEffect(() => {
    if (!loading && isEdit) {
      void loadStoredModelTestResults();
    }
  }, [loading, isEdit, loadStoredModelTestResults]);

  const fetchModels = async () => {
    try {
      const ids = await fetchModelCatalogModelIds({});
      setOriginModelOptions(
        ids.map((id) => ({
          key: id,
          text: id,
          value: id,
        }))
      );
    } catch (error) {
      showError(error.message);
    }
  };

  const fillAllCatalogModels = async () => {
    setFillAllBusy(true);
    try {
      const ids = await fetchModelCatalogModelIds({});
      if (!ids.length) {
        showInfo(t('channel.edit.fill_catalog_empty'));
        return;
      }
      handleInputChange(null, { name: 'models', value: ids });
      showSuccess(t('channel.edit.fill_all_ok', { count: ids.length }));
    } catch (e) {
      showError(e.message || t('channel.edit.fill_catalog_fail'));
    } finally {
      setFillAllBusy(false);
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
    (async () => {
      try {
        const res = await API.get('/api/model_catalog/editor_options');
        const opts = res.data?.data?.channel_types;
        if (Array.isArray(opts) && opts.length) {
          setChannelTypeOptions(opts);
          setChannelTypeOptionsCache(opts);
        }
      } catch (_) {
        /* 忽略 */
      } finally {
        setChannelTypesLoading(false);
      }
    })();
    if (isEdit) {
      loadChannel().then();
    }
    fetchModels().then();
    fetchGroups().then();
  }, []);

  const submit = async () => {
    if (inputs.key === '') {
      if (config.ak !== '' && config.sk !== '' && config.region !== '') {
        inputs.key = `${config.ak}|${config.sk}|${config.region}`;
      }
    }
    if (!isEdit && (inputs.name === '' || inputs.key === '')) {
      showInfo(t('channel.edit.messages.name_required'));
      return;
    }
    if (inputs.models.length === 0) {
      showInfo(t('channel.edit.messages.models_required'));
      return;
    }
    if (inputs.model_mapping !== '' && !verifyJSON(inputs.model_mapping)) {
      showInfo(t('channel.edit.messages.model_mapping_invalid'));
      return;
    }
    const jsonFields = [
      ['status_code_mapping', inputs.status_code_mapping],
      ['param_override', inputs.param_override],
      ['header_override', inputs.header_override],
      ['setting', inputs.setting],
      ['settings', inputs.settings],
    ];
    for (const [label, val] of jsonFields) {
      if (val && String(val).trim() && !verifyJSON(val)) {
        showInfo(`${label}: ${t('channel.edit.messages.model_mapping_invalid')}`);
        return;
      }
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
    let cfg = { ...config };
    if ((localInputs.type === 14 || localInputs.type === 42) && !cfg.api_version) {
      cfg.api_version = 'v1';
    }
    let res;
    localInputs.models = localInputs.models.join(',');
    localInputs.group = localInputs.groups.join(',');
    localInputs.config = JSON.stringify(cfg);
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
        setCustomModel('');
        setConfig({ ...DEFAULT_CHANNEL_CONFIG });
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

  const resolveKeyForUpstreamFetch = () => {
    if (batch) {
      const first = String(inputs.key || '')
        .split(/\r?\n/)
        .map((s) => s.trim())
        .find(Boolean);
      return first || '';
    }
    let k = String(inputs.key || '').trim();
    if (!k && config.ak && config.sk && config.region) {
      k = `${config.ak}|${config.sk}|${config.region}`;
    }
    if (
      !k &&
      config.region &&
      config.vertex_ai_project_id &&
      config.vertex_ai_adc
    ) {
      k = `${config.region}|${config.vertex_ai_project_id}|${config.vertex_ai_adc}`;
    }
    return k;
  };

  const mergeFetchedUpstreamModels = (ids) => {
    if (!Array.isArray(ids) || ids.length === 0) {
      showInfo(t('channel.edit.messages.upstream_fetch_empty'));
      return;
    }
    const dedup = [
      ...new Set(ids.map((x) => String(x).trim()).filter(Boolean)),
    ];
    const localModels = [...new Set([...inputs.models, ...dedup])];
    const optKeys = new Set(originModelOptions.map((o) => o.key));
    const newOpts = [...originModelOptions];
    for (const id of dedup) {
      if (!optKeys.has(id)) {
        optKeys.add(id);
        newOpts.push({ key: id, text: id, value: id });
      }
    }
    setOriginModelOptions(newOpts);
    handleInputChange(null, { name: 'models', value: localModels });
    showSuccess(
      t('channel.edit.messages.upstream_fetch_success', {
        count: dedup.length,
      })
    );
  };

  const fetchUpstreamModelsList = async () => {
    const keyFromForm = resolveKeyForUpstreamFetch();
    if (!keyFromForm && !(isEdit && channelId)) {
      showInfo(t('channel.edit.messages.upstream_fetch_need_key'));
      return;
    }
    setFetchUpstreamBusy(true);
    try {
      let cfg = { ...config };
      if ((inputs.type === 14 || inputs.type === 42) && !cfg.api_version) {
        cfg.api_version = 'v1';
      }
      let baseUrl = String(inputs.base_url || '').trim();
      if (baseUrl.endsWith('/')) {
        baseUrl = baseUrl.slice(0, baseUrl.length - 1);
      }
      const res = await API.post('/api/channel/fetch_upstream_models_preview', {
        type: inputs.type,
        base_url: baseUrl,
        key: keyFromForm,
        config: JSON.stringify(cfg),
        channel_id: isEdit ? Number(channelId) || 0 : 0,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || '请求失败');
        return;
      }
      mergeFetchedUpstreamModels(data);
    } catch (e) {
      showError(e.message || '请求失败');
    } finally {
      setFetchUpstreamBusy(false);
    }
  };

  const testAllModels = async () => {
    if (testModelsBusy || testModelsProgress.running) {
      showInfo(t('channel.edit.messages.test_models_already_running'));
      return;
    }
    const models = (inputs.models || []).map((m) => String(m).trim()).filter(Boolean);
    if (!models.length) {
      showInfo(t('channel.edit.messages.test_models_empty'));
      return;
    }
    const keyFromForm = resolveKeyForUpstreamFetch();
    if (!keyFromForm && !(isEdit && channelId)) {
      showInfo(t('channel.edit.messages.upstream_fetch_need_key'));
      return;
    }
    setModelTestStatus({});
    setModelTestMessages({});
    setModelTestMeta({});
    setTestModelsProgress({
      total: models.length,
      completed: 0,
      currentModel: '',
      running: true,
    });
    const payload = buildChannelModelTestPayload({
      channelId: isEdit ? channelId : 0,
      type: inputs.type,
      baseUrl: inputs.base_url,
      key: keyFromForm,
      config,
      modelMapping: inputs.model_mapping,
      channelTypeOptions,
      concurrency: testModelsConcurrency,
    });
    await pollRunningJob(payload, models);
  };

  const clearInvalidModels = () => {
    const { failed, remaining } = filterOutFailedModels(inputs.models, modelTestStatus);
    if (!failed.length) {
      showInfo(t('channel.edit.messages.clear_failed_empty'));
      return;
    }
    handleInputChange(null, { name: 'models', value: remaining });
    showSuccess(t('channel.edit.messages.clear_failed_done', { count: failed.length }));
  };

  const failedModelCount = (inputs.models || []).filter(
    (m) => modelTestStatus[String(m)] === MODEL_TEST_FAIL
  ).length;

  return (
    <div className='dashboard-container channel-edit-page'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {isEdit
              ? t('channel.edit.title_edit')
              : t('channel.edit.title_create')}
          </Card.Header>
          <Form loading={loading || channelTypesLoading} {...noAutofillFormProps}>
            <div className='channel-edit-section'>
              <Header as='h4' className='channel-edit-section__title'>
                {t('channel.edit.section_basic')}
              </Header>
              <Form.Field>
                <Form.Select
                  label={t('channel.edit.type')}
                  name='type'
                  required
                  search
                  options={channelTypeOptions}
                  placeholder={
                    channelTypesLoading
                      ? t('channel.edit.types_loading')
                      : undefined
                  }
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
                  {...noAutofillTextProps}
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
                  {...noAutofillSecretProps}
                  options={groupOptions}
                />
              </Form.Field>
              {renderChannelTip(inputs.type)}
            </div>

            <div className='channel-edit-section'>
              <Header as='h4' className='channel-edit-section__title'>
                {t('channel.edit.section_upstream')}
              </Header>
              {(() => {
                const cur = channelTypeOptions.find(
                  (o) => o.value === inputs.type
                );
                return cur?.description ? (
                  <Message info size='small'>
                    {cur.description}
                  </Message>
                ) : null;
              })()}
              {(inputs.type === 14 || inputs.type === 42) && (
                <Form.Field>
                  <Form.Input
                    label='Gemini API 版本'
                    name='api_version'
                    placeholder='例如：v1'
                    onChange={handleConfigChange}
                    value={config.api_version || ''}
                    {...noAutofillSecretProps}
                  />
                </Form.Field>
              )}
              <div className='channel-edit-upstream-row'>
                <Form.Field>
                  <label>{t('channel.edit.base_url')}</label>
                  <p className='channel-edit-field-hint'>
                    {t('channel.edit.base_url_all_hint')}
                  </p>
                  <Form.Input
                    name='base_url'
                    value={inputs.base_url}
                    onChange={handleInputChange}
                    {...noAutofillSecretProps}
                  />
                </Form.Field>
                {!batch ? (
                  <Form.Field>
                    <label>{t('channel.edit.key')}</label>
                    <p className='channel-edit-field-hint'>
                      {type2secretPrompt(inputs.type, t)}
                    </p>
                    <Form.Input
                      name='key'
                      required
                      placeholder={type2secretPrompt(inputs.type, t)}
                      onChange={handleInputChange}
                      value={inputs.key}
                      {...noAutofillSecretProps}
                    />
                  </Form.Field>
                ) : null}
              </div>
              {batch ? (
                <Form.Field>
                  <label>{t('channel.edit.key')}</label>
                  <p className='channel-edit-field-hint'>
                    {t('channel.edit.batch_placeholder')}
                  </p>
                  <Form.TextArea
                    name='key'
                    rows={10}
                    value={inputs.key}
                    onChange={handleInputChange}
                    {...noAutofillSecretProps}
                  />
                </Form.Field>
              ) : null}
              {!isEdit && (
                <Form.Checkbox
                  checked={batch}
                  label={t('channel.edit.batch')}
                  name='batch'
                  onChange={() => setBatch(!batch)}
                />
              )}
            </div>

            <div className='channel-edit-section'>
                <Header as='h4' className='channel-edit-section__title'>
                  {t('channel.edit.section_models')}
                </Header>
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
                    renderLabel={(item) =>
                      renderChannelModelLabel(
                        item,
                        modelTestStatus,
                        modelTestMessages,
                        modelTestMeta
                      )
                    }
                    selection
                    onChange={handleInputChange}
                    value={inputs.models}
                    {...noAutofillSecretProps}
                    options={modelOptions}
                  />
                </Form.Field>
                {modelTestSummary || ((testModelsBusy || testModelsProgress.running) && testModelsProgress.total > 0) ? (
                  <div className='channel-edit-model-test-block'>
                    {modelTestSummary ? (
                      <p className='channel-edit-model-test-summary'>{modelTestSummary}</p>
                    ) : null}
                    {(testModelsBusy || testModelsProgress.running) && testModelsProgress.total > 0 ? (
                      <Progress
                        className='channel-edit-model-test-progress'
                        percent={Math.min(
                          100,
                          Math.round(
                            (testModelsProgress.completed / testModelsProgress.total) * 100
                          )
                        )}
                        progress
                        indicating={testModelsProgress.running}
                        active={testModelsProgress.running}
                      />
                    ) : null}
                  </div>
                ) : null}
                <div className='channel-edit-models-toolbar'>
                  <Button
                    type='button'
                    loading={testModelsBusy}
                    disabled={testModelsBusy || testModelsProgress.running || fetchUpstreamBusy}
                    onClick={() => void testAllModels()}
                  >
                    {t('channel.edit.buttons.test_models')}
                  </Button>
                  <div className='channel-edit-model-test-concurrency'>
                    <label
                      className='channel-edit-model-test-concurrency__label'
                      htmlFor='channel-test-models-concurrency'
                    >
                      {t('channel.edit.buttons.test_models_concurrency')}
                    </label>
                    <Input
                      id='channel-test-models-concurrency'
                      type='number'
                      min={1}
                      max={10}
                      step={1}
                      value={testModelsConcurrency}
                      disabled={testModelsBusy || testModelsProgress.running}
                      onChange={(_, { value }) => {
                        const n = parseInt(String(value), 10);
                        setTestModelsConcurrency(
                          Number.isFinite(n) ? Math.min(10, Math.max(1, n)) : 3
                        );
                      }}
                    />
                  </div>
                  <Button
                    type='button'
                    loading={fetchUpstreamBusy}
                    disabled={fetchUpstreamBusy}
                    onClick={() => void fetchUpstreamModelsList()}
                  >
                    {t('channel.edit.buttons.fetch_upstream_models')}
                  </Button>
                  <Button
                    type='button'
                    onClick={() => setFillCatalogOpen(true)}
                  >
                    {t('channel.edit.buttons.fill_models')}
                  </Button>
                  <Button
                    type='button'
                    loading={fillAllBusy}
                    disabled={fillAllBusy}
                    onClick={() => void fillAllCatalogModels()}
                  >
                    {t('channel.edit.buttons.fill_all')}
                  </Button>
                  <Button
                    type='button'
                    disabled={
                      testModelsBusy ||
                      testModelsProgress.running ||
                      failedModelCount === 0
                    }
                    onClick={clearInvalidModels}
                  >
                    {t('channel.edit.buttons.clear_failed')}
                  </Button>
                  <Button
                    type='button'
                    onClick={() => {
                      handleInputChange(null, { name: 'models', value: [] });
                    }}
                  >
                    {t('channel.edit.buttons.clear')}
                  </Button>
                </div>
                <Form.Field>
                  <label>{t('channel.edit.custom_models_bulk')}</label>
                  <p className='channel-edit-field-hint'>
                    {t('channel.edit.buttons.custom_placeholder')}
                  </p>
                  <Form.TextArea
                    rows={4}
                    value={customModel}
                    onChange={(e, { value }) => setCustomModel(value)}
                  />
                  <Button
                    type='button'
                    onClick={addCustomModel}
                    style={{ marginTop: 8 }}
                  >
                    {t('channel.edit.buttons.add_custom')}
                  </Button>
                </Form.Field>
                <Form.Field>
                  <label>{t('channel.edit.model_mapping')}</label>
                  <ChannelModelMappingEditor
                    value={inputs.model_mapping}
                    upstreamModelOptions={inputs.models}
                    onChange={(v) =>
                      handleInputChange(null, {
                        name: 'model_mapping',
                        value: v,
                      })
                    }
                  />
                </Form.Field>
                <Form.Field>
                  <label>{t('channel.edit.system_prompt')}</label>
                  <p className='channel-edit-field-hint'>
                    {t('channel.edit.system_prompt_placeholder')}
                  </p>
                  <Form.TextArea
                    name='system_prompt'
                    rows={10}
                    value={inputs.system_prompt}
                    onChange={handleInputChange}
                  />
                </Form.Field>
                <div className='channel-edit-section channel-edit-section--extended'>
                  <Header as='h4' className='channel-edit-section__title'>
                    {t('channel.edit.newapi_section')}
                  </Header>
                  <Form.Field>
                    <label>{t('channel.edit.openai_organization')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.openai_organization_hint')}
                    </p>
                    <Form.Input
                      name='openai_organization'
                      placeholder={t(
                        'channel.edit.openai_organization_placeholder'
                      )}
                      value={inputs.openai_organization}
                      onChange={handleInputChange}
                      {...noAutofillTextProps}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.test_model')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.test_model_hint')}
                    </p>
                    <Form.Input
                      name='test_model'
                      placeholder={t('channel.edit.test_model_placeholder')}
                      value={inputs.test_model}
                      onChange={handleInputChange}
                      {...noAutofillTextProps}
                    />
                  </Form.Field>
                  <Form.Field>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.auto_ban_hint')}
                    </p>
                    <Checkbox
                      toggle
                      label={t('channel.edit.auto_ban')}
                      checked={inputs.auto_ban !== 0}
                      onChange={() =>
                        setInputs((prev) => ({
                          ...prev,
                          auto_ban: prev.auto_ban === 0 ? 1 : 0,
                        }))
                      }
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.remark')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.remark_hint')}
                    </p>
                    <Form.Input
                      name='remark'
                      placeholder={t('channel.edit.remark_placeholder')}
                      value={inputs.remark}
                      onChange={handleInputChange}
                      {...noAutofillTextProps}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.tag')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.tag_hint')}
                    </p>
                    <Form.Input
                      name='tag'
                      placeholder={t('channel.edit.tag_placeholder')}
                      value={inputs.tag}
                      onChange={handleInputChange}
                      {...noAutofillTextProps}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.status_code_mapping')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.status_code_mapping_hint')}
                    </p>
                    <Form.TextArea
                      name='status_code_mapping'
                      rows={4}
                      placeholder={t(
                        'channel.edit.status_code_mapping_placeholder'
                      )}
                      value={inputs.status_code_mapping}
                      onChange={handleInputChange}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.param_override')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.param_override_hint')}
                    </p>
                    <Form.TextArea
                      name='param_override'
                      rows={6}
                      placeholder={t(
                        'channel.edit.param_override_placeholder'
                      )}
                      value={inputs.param_override}
                      onChange={handleInputChange}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.header_override')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.header_override_hint')}
                    </p>
                    <Form.TextArea
                      name='header_override'
                      rows={4}
                      placeholder={t(
                        'channel.edit.header_override_placeholder'
                      )}
                      value={inputs.header_override}
                      onChange={handleInputChange}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.setting_json')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.setting_json_hint')}
                    </p>
                    <Form.TextArea
                      name='setting'
                      rows={4}
                      placeholder={t('channel.edit.setting_json_placeholder')}
                      value={inputs.setting}
                      onChange={handleInputChange}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.settings_json')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.settings_json_hint')}
                    </p>
                    <Form.TextArea
                      name='settings'
                      rows={4}
                      placeholder={t('channel.edit.settings_json_placeholder')}
                      value={inputs.settings}
                      onChange={handleInputChange}
                    />
                  </Form.Field>
                  <Form.Field>
                    <label>{t('channel.edit.other_info')}</label>
                    <p className='channel-edit-field-hint'>
                      {t('channel.edit.other_info_hint')}
                    </p>
                    <Form.TextArea
                      name='other_info'
                      rows={3}
                      placeholder={t('channel.edit.other_info_placeholder')}
                      value={inputs.other_info}
                      onChange={handleInputChange}
                    />
                  </Form.Field>
                </div>
              </div>
            <div className='channel-edit-section'>
                <Header as='h4' className='channel-edit-section__title'>
                  {t('channel.edit.routing_advanced')}
                </Header>
                <Form.Field>
                  <label>{t('channel.edit.routing_provider')}</label>
                  <Form.Input
                    name='routing_provider'
                    placeholder={t(
                      'channel.edit.routing_provider_placeholder'
                    )}
                    value={config.routing_provider}
                    onChange={(e, { value }) =>
                      handleConfigChange(null, {
                        name: 'routing_provider',
                        value,
                      })
                    }
                  />
                </Form.Field>
                <Form.Field>
                  <Checkbox
                    toggle
                    label={t('channel.edit.routing_skip_adaptive')}
                    checked={!!config.routing_skip_adaptive}
                    onChange={(e, { checked }) =>
                      handleConfigChange(null, {
                        name: 'routing_skip_adaptive',
                        value: checked,
                      })
                    }
                  />
                </Form.Field>
              </div>
            {isEdit && !batch && (
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
            <div className='channel-edit-actions'>
              <Button type='button' onClick={handleCancel}>
                {t('channel.edit.buttons.cancel')}
              </Button>
              <Button
                type={isEdit ? 'button' : 'submit'}
                positive
                onClick={submit}
              >
                {t('channel.edit.buttons.submit')}
              </Button>
            </div>
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
            {...noAutofillSecretProps}
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
      <FillCatalogModelsModal
        open={fillCatalogOpen}
        onClose={() => setFillCatalogOpen(false)}
        onConfirm={(ids) => {
          handleInputChange(null, { name: 'models', value: ids });
          showSuccess(t('channel.edit.fill_catalog_ok', { count: ids.length }));
        }}
      />
    </div>
  );
};

export default EditChannel;
