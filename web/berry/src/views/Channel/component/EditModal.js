import PropTypes from 'prop-types';
import { useState, useEffect } from 'react';
import { useTheme } from '@mui/material/styles';
import { API } from 'utils/api';
import { showError, showSuccess, getChannelModels } from 'utils/common';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Divider,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  OutlinedInput,
  ButtonGroup,
  Container,
  Autocomplete,
  FormHelperText,
  Switch,
  Checkbox,
  FormControlLabel,
  Stack
} from '@mui/material';

import { Formik } from 'formik';
import * as Yup from 'yup';
import { defaultConfig, typeConfig } from '../type/Config';
import { createFilterOptions } from '@mui/material/Autocomplete';
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank';
import CheckBoxIcon from '@mui/icons-material/CheckBox';

const icon = <CheckBoxOutlineBlankIcon fontSize="small" />;
const checkedIcon = <CheckBoxIcon fontSize="small" />;

const filter = createFilterOptions();
const validationSchema = Yup.object().shape({
  is_edit: Yup.boolean(),
  name: Yup.string().required('名称 不能为空'),
  type: Yup.number().required('渠道 不能为空'),
  key: Yup.string().test('key-req', '密钥 不能为空', function (value) {
    const { is_edit, type } = this.parent;
    if (is_edit) return true;
    if (typeConfig[type]?.inputLabel?.key === '') return true;
    return Boolean(value && String(value).trim());
  }),
  models: Yup.array().min(1, '模型 不能为空'),
  groups: Yup.array().min(1, '用户组 不能为空'),
  base_url: Yup.string(),
  model_mapping: Yup.string().test('is-json', '必须是有效的JSON字符串', function (value) {
    try {
      if (value === '' || value === null || value === undefined) {
        return true;
      }
      const parsedValue = JSON.parse(value);
      if (typeof parsedValue === 'object') {
        return true;
      }
    } catch (e) {
      return false;
    }
    return false;
  })
});

const EditModal = ({ open, channelId, onCancel, onOk, channelTypesList = [] }) => {
  const theme = useTheme();
  // const [loading, setLoading] = useState(false);
  const [initialInput, setInitialInput] = useState(defaultConfig.input);
  const [inputLabel, setInputLabel] = useState(defaultConfig.inputLabel); //
  const [inputPrompt, setInputPrompt] = useState(defaultConfig.prompt);
  const [groupOptions, setGroupOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [batchAdd, setBatchAdd] = useState(false);
  const [fetchUpstreamBusy, setFetchUpstreamBusy] = useState(false);
  const [basicModels, setBasicModels] = useState([]);

  const initChannel = (typeValue) => {
    if (typeConfig[typeValue]?.inputLabel) {
      setInputLabel({
        ...defaultConfig.inputLabel,
        ...typeConfig[typeValue].inputLabel
      });
    } else {
      setInputLabel(defaultConfig.inputLabel);
    }

    if (typeConfig[typeValue]?.prompt) {
      setInputPrompt({
        ...defaultConfig.prompt,
        ...typeConfig[typeValue].prompt
      });
    } else {
      setInputPrompt(defaultConfig.prompt);
    }

    return typeConfig[typeValue]?.input;
  };
  const handleTypeChange = (setFieldValue, typeValue, values) => {
    initChannel(typeValue);
    let localModels = getChannelModels(typeValue);
    setBasicModels(localModels);
    if (localModels.length > 0 && Array.isArray(values['models']) && values['models'].length == 0) {
      setFieldValue('models', initialModel(localModels));
    }

    setFieldValue('config', { routing_provider: '', routing_skip_adaptive: false });
  };

  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      setGroupOptions(res.data.data);
    } catch (error) {
      showError(error.message);
    }
  };

  const fetchModels = async () => {
    try {
      let res = await API.get(`/api/channel/models`);
      const { data } = res.data;
      data.forEach((item) => {
        if (!item.owned_by) {
          item.owned_by = '未知';
        }
      });
      // 先对data排序
      data.sort((a, b) => {
        const ownedByComparison = a.owned_by.localeCompare(b.owned_by);
        if (ownedByComparison === 0) {
          return a.id.localeCompare(b.id);
        }
        return ownedByComparison;
      });

      setModelOptions(
        data.map((model) => {
          return {
            id: model.id,
            group: model.owned_by
          };
        })
      );
    } catch (error) {
      showError(error.message);
    }
  };

  const submit = async (values, { setErrors, setStatus, setSubmitting }) => {
    setSubmitting(true);
    if (values.base_url && values.base_url.endsWith('/')) {
      values.base_url = values.base_url.slice(0, values.base_url.length - 1);
    }
    const cfg = { ...values.config };
    if (values.type === 24 && !cfg.api_version) {
      cfg.api_version = 'v1';
    }
    if (values.key === '') {
      if (values.config.ak && values.config.sk && values.config.region) {
        values.key = `${values.config.ak}|${values.config.sk}|${values.config.region}`;
      } else if (values.config.region && values.config.vertex_ai_project_id && values.config.vertex_ai_adc) {
        values.key = `${values.config.region}|${values.config.vertex_ai_project_id}|${values.config.vertex_ai_adc}`;
      }
    }

    let res;
    const modelsStr = values.models.map((model) => model.id).join(',');
    const configStr = JSON.stringify(cfg);
    values.group = values.groups.join(',');
    if (channelId) {
      res = await API.put(`/api/channel/`, {
        ...values,
        id: parseInt(channelId),
        models: modelsStr,
        config: configStr
      });
    } else {
      res = await API.post(`/api/channel/`, { ...values, models: modelsStr, config: configStr });
    }
    const { success, message } = res.data;
    if (success) {
      if (channelId) {
        showSuccess('渠道更新成功！');
      } else {
        showSuccess('渠道创建成功！');
      }
      setSubmitting(false);
      setStatus({ success: true });
      onOk(true);
    } else {
      setStatus({ success: false });
      showError(message);
      setErrors({ submit: message });
    }
  };

  function initialModel(channelModel) {
    if (!channelModel) {
      return [];
    }

    // 如果 channelModel 是一个字符串
    if (typeof channelModel === 'string') {
      channelModel = channelModel.split(',');
    }
    let modelList = channelModel.map((model) => {
      const modelOption = modelOptions.find((option) => option.id === model);
      if (modelOption) {
        return modelOption;
      }
      return { id: model, group: '自定义：点击或回车输入' };
    });
    return modelList;
  }

  const loadChannel = async () => {
    let res = await API.get(`/api/channel/${channelId}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.models === '') {
        data.models = [];
      } else {
        data.models = initialModel(data.models);
      }
      if (data.group === '') {
        data.groups = [];
      } else {
        data.groups = data.group.split(',');
      }
      if (data.model_mapping !== '') {
        data.model_mapping = JSON.stringify(JSON.parse(data.model_mapping), null, 2);
      }
      if (data.config !== '') {
        data.config = JSON.parse(data.config);
      } else {
        data.config = {};
      }
      data.config = {
        routing_provider: '',
        routing_skip_adaptive: false,
        ...data.config
      };

      data.base_url = data.base_url ?? '';
      delete data.other;
      data.is_edit = true;
      initChannel(data.type);
      setInitialInput(data);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    fetchGroups().then();
    fetchModels().then();
  }, []);

  useEffect(() => {
    setBatchAdd(false);
    if (channelId) {
      loadChannel().then();
    } else {
      initChannel(defaultConfig.input.type);
      setInitialInput({ ...defaultConfig.input, is_edit: false });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [channelId]);

  return (
    <Dialog open={open} onClose={onCancel} fullWidth maxWidth={'md'}>
      <DialogTitle
        sx={{
          margin: '0px',
          fontWeight: 700,
          lineHeight: '1.55556',
          padding: '24px',
          fontSize: '1.125rem'
        }}
      >
        {channelId ? '编辑渠道' : '新建渠道'}
      </DialogTitle>
      <Divider />
      <DialogContent>
        <Formik initialValues={initialInput} enableReinitialize validationSchema={validationSchema} onSubmit={submit}>
          {({ errors, handleBlur, handleChange, handleSubmit, isSubmitting, touched, values, setFieldValue }) => (
            <form noValidate onSubmit={handleSubmit}>
              <FormControl fullWidth error={Boolean(touched.type && errors.type)} sx={{ ...theme.typography.otherInput }}>
                <InputLabel htmlFor="channel-type-label">{inputLabel.type}</InputLabel>
                <Select
                  id="channel-type-label"
                  label={inputLabel.type}
                  value={values.type}
                  name="type"
                  onBlur={handleBlur}
                  onChange={(e) => {
                    handleChange(e);
                    handleTypeChange(setFieldValue, e.target.value, values);
                  }}
                  MenuProps={{
                    PaperProps: {
                      style: {
                        maxHeight: 200
                      }
                    }
                  }}
                >
                  {channelTypesList.map((option) => (
                        <MenuItem key={option.value} value={option.value}>
                          {option.text}
                        </MenuItem>
                      ))}
                </Select>
                {touched.type && errors.type ? (
                  <FormHelperText error id="helper-tex-channel-type-label">
                    {errors.type}
                  </FormHelperText>
                ) : (
                  <FormHelperText id="helper-tex-channel-type-label"> {inputPrompt.type} </FormHelperText>
                )}
              </FormControl>

              <FormControl fullWidth error={Boolean(touched.name && errors.name)} sx={{ ...theme.typography.otherInput }}>
                <InputLabel htmlFor="channel-name-label">{inputLabel.name}</InputLabel>
                <OutlinedInput
                  id="channel-name-label"
                  label={inputLabel.name}
                  type="text"
                  value={values.name}
                  name="name"
                  onBlur={handleBlur}
                  onChange={handleChange}
                  inputProps={{ autoComplete: 'name' }}
                  aria-describedby="helper-text-channel-name-label"
                />
                {touched.name && errors.name ? (
                  <FormHelperText error id="helper-tex-channel-name-label">
                    {errors.name}
                  </FormHelperText>
                ) : (
                  <FormHelperText id="helper-tex-channel-name-label"> {inputPrompt.name} </FormHelperText>
                )}
              </FormControl>

              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ width: '100%', ...theme.typography.otherInput }}>
                <FormControl fullWidth sx={{ flex: 1 }} error={Boolean(touched.base_url && errors.base_url)}>
                  <InputLabel htmlFor="channel-base_url-label">{inputLabel.base_url}</InputLabel>
                  <OutlinedInput
                    id="channel-base_url-label"
                    label={inputLabel.base_url}
                    type="text"
                    value={values.base_url}
                    name="base_url"
                    onBlur={handleBlur}
                    onChange={handleChange}
                    inputProps={{}}
                    aria-describedby="helper-text-channel-base_url-label"
                  />
                  {touched.base_url && errors.base_url ? (
                    <FormHelperText error id="helper-tex-channel-base_url-label">
                      {errors.base_url}
                    </FormHelperText>
                  ) : (
                    <FormHelperText id="helper-tex-channel-base_url-label"> {inputPrompt.base_url} </FormHelperText>
                  )}
                </FormControl>
                {inputLabel.key && !batchAdd ? (
                  <FormControl fullWidth sx={{ flex: 1 }} error={Boolean(touched.key && errors.key)}>
                    <InputLabel htmlFor="channel-key-label">{inputLabel.key}</InputLabel>
                    <OutlinedInput
                      id="channel-key-label"
                      label={inputLabel.key}
                      type="text"
                      value={values.key}
                      name="key"
                      onBlur={handleBlur}
                      onChange={handleChange}
                      inputProps={{}}
                      aria-describedby="helper-text-channel-key-label"
                    />
                    {touched.key && errors.key ? (
                      <FormHelperText error id="helper-tex-channel-key-label">
                        {errors.key}
                      </FormHelperText>
                    ) : (
                      <FormHelperText id="helper-tex-channel-key-label"> {inputPrompt.key} </FormHelperText>
                    )}
                  </FormControl>
                ) : null}
              </Stack>

              {inputLabel.key && batchAdd ? (
                <FormControl fullWidth error={Boolean(touched.key && errors.key)} sx={{ ...theme.typography.otherInput }}>
                  <TextField
                    multiline
                    id="channel-key-label-batch"
                    label={inputLabel.key}
                    value={values.key}
                    name="key"
                    onBlur={handleBlur}
                    onChange={handleChange}
                    aria-describedby="helper-text-channel-key-label"
                    minRows={5}
                    placeholder={inputPrompt.key + '，一行一个密钥'}
                  />
                  {touched.key && errors.key ? (
                    <FormHelperText error id="helper-tex-channel-key-label">
                      {errors.key}
                    </FormHelperText>
                  ) : (
                    <FormHelperText id="helper-tex-channel-key-label"> {inputPrompt.key} </FormHelperText>
                  )}
                </FormControl>
              ) : null}

              {channelId === 0 && inputLabel.key ? (
                <Container
                  sx={{
                    textAlign: 'right'
                  }}
                >
                  <Switch checked={batchAdd} onChange={(e) => setBatchAdd(e.target.checked)} />
                  批量添加
                </Container>
              ) : null}

              <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
                <Autocomplete
                  multiple
                  id="channel-groups-label"
                  options={groupOptions}
                  value={values.groups}
                  onChange={(e, value) => {
                    const event = {
                      target: {
                        name: 'groups',
                        value: value
                      }
                    };
                    handleChange(event);
                  }}
                  onBlur={handleBlur}
                  filterSelectedOptions
                  renderInput={(params) => <TextField {...params} name="groups" error={Boolean(errors.groups)} label={inputLabel.groups} />}
                  aria-describedby="helper-text-channel-groups-label"
                />
                {errors.groups ? (
                  <FormHelperText error id="helper-tex-channel-groups-label">
                    {errors.groups}
                  </FormHelperText>
                ) : (
                  <FormHelperText id="helper-tex-channel-groups-label"> {inputPrompt.groups} </FormHelperText>
                )}
              </FormControl>

              <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
                <Autocomplete
                  multiple
                  freeSolo
                  id="channel-models-label"
                  options={modelOptions}
                  value={values.models}
                  onChange={(e, value) => {
                    const event = {
                      target: {
                        name: 'models',
                        value: value.map((item) => (typeof item === 'string' ? { id: item, group: '自定义：点击或回车输入' } : item))
                      }
                    };
                    handleChange(event);
                  }}
                  onBlur={handleBlur}
                  // filterSelectedOptions
                  disableCloseOnSelect
                  renderInput={(params) => <TextField {...params} name="models" error={Boolean(errors.models)} label={inputLabel.models} />}
                  groupBy={(option) => option.group}
                  getOptionLabel={(option) => {
                    if (typeof option === 'string') {
                      return option;
                    }
                    if (option.inputValue) {
                      return option.inputValue;
                    }
                    return option.id;
                  }}
                  filterOptions={(options, params) => {
                    const filtered = filter(options, params);
                    const { inputValue } = params;
                    const isExisting = options.some((option) => inputValue === option.id);
                    if (inputValue !== '' && !isExisting) {
                      filtered.push({
                        id: inputValue,
                        group: '自定义：点击或回车输入'
                      });
                    }
                    return filtered;
                  }}
                  renderOption={(props, option, { selected }) => (
                    <li {...props}>
                      <Checkbox icon={icon} checkedIcon={checkedIcon} style={{ marginRight: 8 }} checked={selected} />
                      {option.id}
                    </li>
                  )}
                />
                {errors.models ? (
                  <FormHelperText error id="helper-tex-channel-models-label">
                    {errors.models}
                  </FormHelperText>
                ) : (
                  <FormHelperText id="helper-tex-channel-models-label"> {inputPrompt.models} </FormHelperText>
                )}
              </FormControl>
              <Container
                sx={{
                  textAlign: 'right'
                }}
              >
                <ButtonGroup variant="outlined" aria-label="small outlined primary button group">
                  <Button
                    disabled={fetchUpstreamBusy}
                    onClick={() => {
                      void (async () => {
                        setFetchUpstreamBusy(true);
                        try {
                          const cfg = values.config || {};
                          let keyFromForm = '';
                          if (batchAdd) {
                            keyFromForm =
                              String(values.key || '')
                                .split(/\r?\n/)
                                .map((s) => s.trim())
                                .find(Boolean) || '';
                          } else {
                            keyFromForm = String(values.key || '').trim();
                            if (!keyFromForm && cfg.ak && cfg.sk && cfg.region) {
                              keyFromForm = `${cfg.ak}|${cfg.sk}|${cfg.region}`;
                            }
                            if (
                              !keyFromForm &&
                              cfg.region &&
                              cfg.vertex_ai_project_id &&
                              cfg.vertex_ai_adc
                            ) {
                              keyFromForm = `${cfg.region}|${cfg.vertex_ai_project_id}|${cfg.vertex_ai_adc}`;
                            }
                          }
                          let ids;
                          const mergedCfg = { ...cfg };
                          if ((values.type === 14 || values.type === 42) && !mergedCfg.api_version) {
                            mergedCfg.api_version = 'v1';
                          }
                          let baseUrl = String(values.base_url || '').trim();
                          if (baseUrl.endsWith('/')) {
                            baseUrl = baseUrl.slice(0, baseUrl.length - 1);
                          }
                          if (!keyFromForm && !channelId) {
                            showError('请先填写密钥');
                            return;
                          }
                          const res = await API.post('/api/channel/fetch_upstream_models_preview', {
                            type: values.type,
                            base_url: baseUrl,
                            key: keyFromForm,
                            config: JSON.stringify(mergedCfg),
                            channel_id: channelId ? Number(channelId) || 0 : 0,
                          });
                          const { success, message, data } = res.data;
                          if (!success) {
                            showError(message || '请求失败');
                            return;
                          }
                          ids = data;
                          if (!Array.isArray(ids) || ids.length === 0) {
                            showError('上游未返回可用模型');
                            return;
                          }
                          const dedup = [...new Set(ids.map((x) => String(x).trim()).filter(Boolean))];
                          const existing = new Set(
                            values.models.map((m) => (typeof m === 'string' ? m : m.id))
                          );
                          const mergedModels = [...values.models];
                          for (const id of dedup) {
                            if (!existing.has(id)) {
                              existing.add(id);
                              mergedModels.push({ id, group: '自定义：点击或回车输入' });
                            }
                          }
                          setModelOptions((prev) => {
                            const next = [...prev];
                            for (const id of dedup) {
                              if (!next.some((o) => o.id === id)) {
                                next.push({ id, group: '上游' });
                              }
                            }
                            return next;
                          });
                          setFieldValue('models', mergedModels);
                          showSuccess(`已从上游合并 ${dedup.length} 个模型`);
                        } catch (e) {
                          showError(e.message || '请求失败');
                        } finally {
                          setFetchUpstreamBusy(false);
                        }
                      })();
                    }}
                  >
                    从上游获取模型
                  </Button>
                  <Button
                    onClick={() => {
                      setFieldValue('models', initialModel(basicModels));
                    }}
                  >
                    填入相关模型
                  </Button>
                  <Button
                    onClick={() => {
                      setFieldValue('models', modelOptions);
                    }}
                  >
                    填入所有模型
                  </Button>
                </ButtonGroup>
              </Container>
              {inputLabel.config &&
                Object.keys(inputLabel.config).map((configName) => {
                  return (
                    <FormControl key={'config.' + configName} fullWidth sx={{ ...theme.typography.otherInput }}>
                      <TextField
                        multiline
                        key={'config.' + configName}
                        name={'config.' + configName}
                        value={values.config?.[configName] || ''}
                        label={inputLabel.config[configName] || configName}
                        placeholder={inputPrompt.config[configName]}
                        onChange={handleChange}
                      />
                      <FormHelperText id={`helper-tex-config.${configName}-label`}> {inputPrompt.config[configName]} </FormHelperText>
                    </FormControl>
                  );
                })}

              <FormControl fullWidth error={Boolean(touched.model_mapping && errors.model_mapping)} sx={{ ...theme.typography.otherInput }}>
                {/* <InputLabel htmlFor="channel-model_mapping-label">{inputLabel.model_mapping}</InputLabel> */}
                <TextField
                  multiline
                  id="channel-model_mapping-label"
                  label={inputLabel.model_mapping}
                  value={values.model_mapping}
                  name="model_mapping"
                  onBlur={handleBlur}
                  onChange={handleChange}
                  aria-describedby="helper-text-channel-model_mapping-label"
                  minRows={5}
                  placeholder={inputPrompt.model_mapping}
                />
                {touched.model_mapping && errors.model_mapping ? (
                  <FormHelperText error id="helper-tex-channel-model_mapping-label">
                    {errors.model_mapping}
                  </FormHelperText>
                ) : (
                  <FormHelperText id="helper-tex-channel-model_mapping-label"> {inputPrompt.model_mapping} </FormHelperText>
                )}
              </FormControl>
              <FormControl fullWidth error={Boolean(touched.system_prompt && errors.system_prompt)} sx={{ ...theme.typography.otherInput }}>
                {/* <InputLabel htmlFor="channel-model_mapping-label">{inputLabel.model_mapping}</InputLabel> */}
                <TextField
                  multiline
                  id="channel-system_prompt-label"
                  label={inputLabel.system_prompt}
                  value={values.system_prompt}
                  name="system_prompt"
                  onBlur={handleBlur}
                  onChange={handleChange}
                  aria-describedby="helper-text-channel-system_prompt-label"
                  minRows={5}
                  placeholder={inputPrompt.system_prompt}
                />
                {touched.system_prompt && errors.system_prompt ? (
                  <FormHelperText error id="helper-tex-channel-system_prompt-label">
                    {errors.system_prompt}
                  </FormHelperText>
                ) : (
                  <FormHelperText id="helper-tex-channel-system_prompt-label"> {inputPrompt.system_prompt} </FormHelperText>
                )}
              </FormControl>
              <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
                <TextField
                  label="Direction（Provider）标签"
                  value={values.config?.routing_provider ?? ''}
                  onChange={(e) => setFieldValue('config.routing_provider', e.target.value)}
                  placeholder="例如 azure-east / openai-main"
                />
                <FormHelperText>用于双层打分与探测分组；留空则沿用渠道默认行为。</FormHelperText>
              </FormControl>
              <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
                <FormControlLabel
                  control={
                    <Checkbox
                      checked={!!values.config?.routing_skip_adaptive}
                      onChange={(e) => setFieldValue('config.routing_skip_adaptive', e.target.checked)}
                    />
                  }
                  label="跳过自适应权重调整（手工倍率与熔断仍生效）"
                />
              </FormControl>
              <DialogActions>
                <Button onClick={onCancel}>取消</Button>
                <Button disableElevation disabled={isSubmitting} type="submit" variant="contained" color="primary">
                  提交
                </Button>
              </DialogActions>
            </form>
          )}
        </Formik>
      </DialogContent>
    </Dialog>
  );
};

export default EditModal;

EditModal.propTypes = {
  open: PropTypes.bool,
  channelId: PropTypes.number,
  onCancel: PropTypes.func,
  onOk: PropTypes.func,
  channelTypesList: PropTypes.array
};
