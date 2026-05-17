import React, {useEffect, useState} from 'react';
import {API, isMobile, showError, showInfo, showSuccess, verifyJSON, splitModelNameList, getChannelModels, setChannelModelsCache} from '../../helpers';
import Title from "@douyinfe/semi-ui/lib/es/typography/title";
import {SideSheet, Space, Spin, Button, Input, Typography, Select, TextArea, Checkbox, Banner} from "@douyinfe/semi-ui";

const MODEL_MAPPING_EXAMPLE = {
    'gpt-3.5-turbo-0301': 'gpt-3.5-turbo',
    'gpt-4-0314': 'gpt-4',
    'gpt-4-32k-0314': 'gpt-4-32k'
};

function type2secretPrompt(type) {
    switch (type) {
        case 43:
            return '请输入 AiPPT 渠道所需密钥（格式见类型说明）';
        case 44:
            return '请输入高德渠道所需密钥（格式见类型说明）';
        case 45:
            return '填写深知上游 Bearer Token；Base URL 至 …/deepresearch；客户端须 stream:true，模型 deep-research';
        case 5:
        case 46:
            return '请输入 Anthropic API Key（x-api-key）';
        case 14:
        case 15:
        case 42:
            return '请输入 Gemini API Key';
        default:
            return '请输入渠道对应的鉴权密钥';
    }
}

const EditChannel = (props) => {
    const channelId = props.editingChannel.id;
    const isEdit = channelId !== undefined;
    const [loading, setLoading] = useState(isEdit);
    const handleCancel = () => {
        props.handleClose()
    };
    const originInputs = {
        name: '',
        type: 41,
        key: '',
        openai_organization: '',
        base_url: '',
        model_mapping: '',
        system_prompt: '',
        models: [],
        auto_ban: 1,
        groups: ['default'],
        routing_provider: '',
        routing_skip_adaptive: false,
        config: '{}',
        test_model: '',
        remark: '',
        tag: '',
        status_code_mapping: '',
        param_override: '',
        header_override: '',
        setting: '',
        settings: '',
        other_info: '',
    };
    const [batch, setBatch] = useState(false);
    const [autoBan, setAutoBan] = useState(true);
    // const [autoBan, setAutoBan] = useState(true);
    const [config, setConfig] = useState({
        api_version: '',
        library_id: '',
        plugin: '',
    });
    const [inputs, setInputs] = useState(originInputs);
    const [originModelOptions, setOriginModelOptions] = useState([]);
    const [modelOptions, setModelOptions] = useState([]);
    const [groupOptions, setGroupOptions] = useState([]);
    const [channelTypeOptions, setChannelTypeOptions] = useState([]);
    const [channelTypesLoading, setChannelTypesLoading] = useState(true);
    const [basicModels, setBasicModels] = useState([]);
    const [fullModels, setFullModels] = useState([]);
    const [customModel, setCustomModel] = useState('');
    const handleInputChange = (name, value) => {
        if (name === 'type') {
            const opt = channelTypeOptions.find((o) => o.value === value);
            const defaultUrl =
                opt && opt.default_base_url
                    ? String(opt.default_base_url).trim()
                    : '';
            const localModels = getChannelModels(value);
            setBasicModels(localModels);
            setInputs((prev) => {
                const typeChanged = prev.type !== value;
                const nextModels =
                  typeChanged && prev.models.length === 0 ? localModels : prev.models;
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
                return {...prev, type: value, models: nextModels, base_url};
            });
            setConfig({ api_version: '', library_id: '', plugin: '' });
            return;
        }
        setInputs((inputs) => ({...inputs, [name]: value}));
    };


    const loadChannel = async () => {
        setLoading(true)
        let res = await API.get(`/api/channel/${channelId}`);
        const {success, message, data} = res.data;
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
                data.model_mapping = JSON.stringify(JSON.parse(data.model_mapping), null, 2);
            }
            let cfg = {};
            try {
                if (data.config && typeof data.config === 'string' && data.config.trim()) {
                    cfg = JSON.parse(data.config);
                }
            } catch {
                cfg = {};
            }
            data.routing_provider = cfg.routing_provider || '';
            data.routing_skip_adaptive = !!cfg.routing_skip_adaptive;
            if (data.config) {
                try {
                    const c = JSON.parse(data.config);
                    setConfig({
                        api_version: c.api_version || '',
                        library_id: c.library_id || '',
                        plugin: c.plugin || '',
                    });
                } catch (e) {
                    setConfig({ api_version: '', library_id: '', plugin: '' });
                }
            } else {
                setConfig({ api_version: '', library_id: '', plugin: '' });
            }
            if (data.auto_ban === 0) {
                setAutoBan(false);
            } else {
                setAutoBan(true);
            }
            data.test_model = data.test_model || '';
            data.remark = data.remark || '';
            data.tag = data.tag || '';
            data.status_code_mapping = data.status_code_mapping || '';
            data.param_override = data.param_override || '';
            data.header_override = data.header_override || '';
            data.setting = data.setting || '';
            data.settings = data.settings || '';
            data.other_info = data.other_info || '';
            setInputs(data);
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
                label: model.id,
                value: model.id
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
            setGroupOptions(res.data.data.map((group) => ({
                label: group,
                value: group
            })));
        } catch (error) {
            showError(error.message);
        }
    };

    useEffect(() => {
        let localModelOptions = [...originModelOptions];
        inputs.models.forEach((model) => {
            if (!localModelOptions.find((option) => option.value === model)) {
                localModelOptions.push({
                    label: model,
                    value: model
                });
            }
        });
        setModelOptions(localModelOptions);
    }, [originModelOptions, inputs.models]);

    useEffect(() => {
        (async () => {
            try {
                const res = await API.get('/api/models');
                if (res.data?.success && res.data?.data) {
                    setChannelModelsCache(res.data.data);
                }
            } catch (_) {
                /* ignore */
            }
            try {
                const res = await API.get('/api/model_catalog/editor_options');
                const opts = res.data?.data?.channel_types;
                if (Array.isArray(opts) && opts.length) {
                    setChannelTypeOptions(opts);
                }
            } catch (_) {
                /* ignore */
            } finally {
                setChannelTypesLoading(false);
            }
            await fetchModels();
            await fetchGroups();
            if (isEdit) {
                await loadChannel();
            } else {
                setInputs({ ...originInputs });
                setConfig({ api_version: '', library_id: '', plugin: '' });
                setBasicModels(getChannelModels(originInputs.type));
            }
        })();
    }, [props.editingChannel.id]);


    const submit = async () => {
        if (!isEdit && (inputs.name === '' || inputs.key === '')) {
            showInfo('请填写渠道名称和渠道密钥！');
            return;
        }
        if (inputs.models.length === 0) {
            showInfo('请至少选择一个模型！');
            return;
        }
        if (inputs.model_mapping !== '' && !verifyJSON(inputs.model_mapping)) {
            showInfo('模型映射必须是合法的 JSON 格式！');
            return;
        }
        for (const j of [
            inputs.status_code_mapping,
            inputs.param_override,
            inputs.header_override,
            inputs.setting,
            inputs.settings,
        ]) {
            if (j && String(j).trim() && !verifyJSON(j)) {
                showInfo('扩展 JSON 配置格式无效');
                return;
            }
        }
        let localInputs = {...inputs};
        if (localInputs.base_url && localInputs.base_url.endsWith('/')) {
            localInputs.base_url = localInputs.base_url.slice(0, localInputs.base_url.length - 1);
        }
        let cfg = { ...config };
        if ((localInputs.type === 14 || localInputs.type === 42) && !cfg.api_version) {
            cfg.api_version = 'v1';
        }
        cfg.routing_provider = inputs.routing_provider || '';
        cfg.routing_skip_adaptive = !!inputs.routing_skip_adaptive;
        localInputs.config = JSON.stringify(cfg);
        delete localInputs.other;
        let res;
        if (!Array.isArray(localInputs.models)) {
            showError('提交失败，请勿重复提交！');
            handleCancel();
            return;
        }
        localInputs.auto_ban = autoBan ? 1 : 0;
        localInputs.models = localInputs.models.join(',');
        localInputs.group = localInputs.groups.join(',');
        delete localInputs.groups;
        delete localInputs.routing_provider;
        delete localInputs.routing_skip_adaptive;
        if (isEdit) {
            res = await API.put(`/api/channel/`, {...localInputs, id: parseInt(channelId)});
        } else {
            res = await API.post(`/api/channel/`, localInputs);
        }
        const {success, message} = res.data;
        if (success) {
            if (isEdit) {
                showSuccess('渠道更新成功！');
            } else {
                showSuccess('渠道创建成功！');
                setInputs(originInputs);
            }
            props.refresh();
            props.handleClose();
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
            newOptions.push({
                key: name,
                text: name,
                value: name,
            });
        }
        if (newOptions.length === 0) {
            return showError('输入的模型均已存在于列表中');
        }
        setModelOptions(modelOptions => [...modelOptions, ...newOptions]);
        setCustomModel('');
        handleInputChange('models', localModels);
    };

    const channelTypeSelectList = channelTypeOptions.map((o) => ({
        label: o.text,
        value: o.value,
    }));
    const channelTypeCur = channelTypeOptions.find((o) => o.value === inputs.type);

    return (
        <>
            <SideSheet
                maskClosable={false}
                placement={isEdit ? 'right' : 'left'}
                title={<Title level={3}>{isEdit ? '更新渠道信息' : '创建新的渠道'}</Title>}
                headerStyle={{borderBottom: '1px solid var(--semi-color-border)'}}
                bodyStyle={{borderBottom: '1px solid var(--semi-color-border)'}}
                visible={props.visible}
                footer={
                    <div style={{display: 'flex', justifyContent: 'flex-end'}}>
                        <Space>
                            <Button theme='solid' size={'large'} onClick={submit}>提交</Button>
                            <Button theme='solid' size={'large'} type={'tertiary'} onClick={handleCancel}>取消</Button>
                        </Space>
                    </div>
                }
                closeIcon={null}
                onCancel={() => handleCancel()}
                width={isMobile() ? '100%' : 600}
            >
                <Spin spinning={loading || channelTypesLoading}>
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>类型：</Typography.Text>
                    </div>
                    <Select
                      name='type'
                      required
                      placeholder={channelTypesLoading ? '加载渠道类型…' : undefined}
                      optionList={channelTypeSelectList}
                      value={inputs.type}
                      onChange={value => handleInputChange('type', value)}
                      style={{ width: '50%' }}
                    />
                    {
                      channelTypeCur?.tip ? (
                        <div style={{ marginTop: 10 }}>
                          <Banner type="info" description={
                            <span dangerouslySetInnerHTML={{ __html: channelTypeCur.tip }} />
                          } />
                        </div>
                      ) : null
                    }
                    {
                      channelTypeCur?.description ? (
                        <div style={{ marginTop: 10 }}>
                          <Banner type="info" description={channelTypeCur.description} />
                        </div>
                      ) : null
                    }
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>名称：</Typography.Text>
                    </div>
                    <Input
                      required
                      name='name'
                      placeholder={'请为渠道命名'}
                      onChange={value => {
                          handleInputChange('name', value)
                      }}
                      value={inputs.name}
                      autoComplete='new-password'
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>分组：</Typography.Text>
                    </div>
                    <Select
                      placeholder={'请选择可以使用该渠道的分组'}
                      name='groups'
                      required
                      multiple
                      selection
                      allowAdditions
                      additionLabel={'请在系统设置页面编辑分组倍率以添加新的分组：'}
                      onChange={value => {
                          handleInputChange('groups', value)
                      }}
                      value={inputs.groups}
                      autoComplete='new-password'
                      optionList={groupOptions}
                    />
                    {
                      (inputs.type === 14 || inputs.type === 42) && (
                        <>
                            <div style={{ marginTop: 10 }}>
                                <Typography.Text strong>Gemini API 版本：</Typography.Text>
                            </div>
                            <Input
                              name='gemini_api_version'
                              placeholder={'例如：v1'}
                              onChange={value => {
                                  setConfig((c) => ({ ...c, api_version: value }));
                              }}
                              value={config.api_version}
                              autoComplete='new-password'
                            />
                        </>
                      )
                    }
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>Base URL / 代理：</Typography.Text>
                    </div>
                    <Typography.Paragraph type="tertiary" size="small" style={{ marginTop: 4, marginBottom: 8 }}>
                      可填上游或自定义兼容地址；留空则使用默认。
                    </Typography.Paragraph>
                    <Input
                      name='base_url'
                      placeholder={'可选，例如 https://api.openai.com 或通过代理访问的地址'}
                      onChange={value => handleInputChange('base_url', value)}
                      value={inputs.base_url}
                      autoComplete='new-password'
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>模型：</Typography.Text>
                    </div>
                    <Select
                      placeholder={'请选择该渠道所支持的模型'}
                      name='models'
                      required
                      multiple
                      selection
                      onChange={value => {
                          handleInputChange('models', value)
                      }}
                      value={inputs.models}
                      autoComplete='new-password'
                      optionList={modelOptions}
                    />
                    <div style={{ lineHeight: '40px', marginBottom: '12px' }}>
                        <Space>
                            <Button type='primary' onClick={() => {
                                handleInputChange('models', basicModels);
                            }}>填入基础模型</Button>
                            <Button type='secondary' onClick={() => {
                                handleInputChange('models', fullModels);
                            }}>填入所有模型</Button>
                            <Button type='warning' onClick={() => {
                                handleInputChange('models', []);
                            }}>清除所有模型</Button>
                        </Space>
                        <TextArea
                          placeholder="多条用英文逗号、中文逗号、分号或换行分隔，例如：model-a,model-b"
                          value={customModel}
                          onChange={(value) => setCustomModel(value)}
                          autosize={{ minRows: 2, maxRows: 12 }}
                          style={{
                            width: '100%',
                            marginTop: 8,
                          }}
                        />
                        <Button type='primary' style={{ marginTop: 8 }} onClick={addCustomModel}>填入</Button>
                    </div>
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>模型重定向：</Typography.Text>
                    </div>
                    <TextArea
                      placeholder={`此项可选，用于修改请求体中的模型名称，为一个 JSON 字符串，键为请求中模型名称，值为要替换的模型名称，例如：\n${JSON.stringify(MODEL_MAPPING_EXAMPLE, null, 2)}`}
                      name='model_mapping'
                      onChange={value => {
                          handleInputChange('model_mapping', value)
                      }}
                      autosize
                      value={inputs.model_mapping}
                      autoComplete='new-password'
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>系统提示词：</Typography.Text>
                    </div>
                    <TextArea
                      placeholder={`此项可选，用于强制设置给定的系统提示词，请配合自定义模型 & 模型重定向使用，首先创建一个唯一的自定义模型名称并在上面填入，之后将该自定义模型重定向映射到该渠道一个原生支持的模型`}
                      name='system_prompt'
                      onChange={value => {
                          handleInputChange('system_prompt', value)
                      }}
                      autosize
                      value={inputs.system_prompt}
                      autoComplete='new-password'
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>智能路由（Direction）：</Typography.Text>
                    </div>
                    <Input
                      placeholder="Direction（Provider）标签，例如 azure-east / openai-main"
                      value={inputs.routing_provider}
                      onChange={(value) => handleInputChange('routing_provider', value)}
                      style={{ width: '100%', marginTop: 8 }}
                    />
                    <div style={{ marginTop: 10, display: 'flex' }}>
                        <Space>
                            <Checkbox
                                checked={!!inputs.routing_skip_adaptive}
                                onChange={(checked) => handleInputChange('routing_skip_adaptive', checked)}
                            />
                            <Typography.Text strong>跳过自适应权重调整（手工倍率与熔断仍生效）</Typography.Text>
                        </Space>
                    </div>
                    <Typography.Text style={{
                        color: 'rgba(var(--semi-blue-5), 1)',
                        userSelect: 'none',
                        cursor: 'pointer'
                    }} onClick={
                        () => {
                            handleInputChange('model_mapping', JSON.stringify(MODEL_MAPPING_EXAMPLE, null, 2))
                        }
                    }>
                        填入模板
                    </Typography.Text>
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>密钥：</Typography.Text>
                    </div>
                    {
                        batch ?
                          <TextArea
                            label='密钥'
                            name='key'
                            required
                            placeholder={'请输入密钥，一行一个'}
                            onChange={value => {
                                handleInputChange('key', value)
                            }}
                            value={inputs.key}
                            style={{ minHeight: 150, fontFamily: 'JetBrains Mono, Consolas' }}
                            autoComplete='new-password'
                          />
                          :
                          <Input
                            label='密钥'
                            name='key'
                            required
                            placeholder={type2secretPrompt(inputs.type)}
                            onChange={value => {
                                handleInputChange('key', value)
                            }}
                            value={inputs.key}
                            autoComplete='new-password'
                          />
                    }
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>组织：</Typography.Text>
                    </div>
                    <Input
                      label='组织，可选，不填则为默认组织'
                      name='openai_organization'
                      placeholder='请输入组织org-xxx'
                      onChange={value => {
                          handleInputChange('openai_organization', value)
                      }}
                      value={inputs.openai_organization}
                    />
                    <div style={{ marginTop: 10, display: 'flex' }}>
                        <Space>
                            <Checkbox
                              name='auto_ban'
                              checked={autoBan}
                              onChange={
                                  () => {
                                      setAutoBan(!autoBan);
                                  }
                              }
                              // onChange={handleInputChange}
                            />
                            <Typography.Text
                              strong>是否自动禁用（仅当自动禁用开启时有效），关闭后不会自动禁用该渠道：</Typography.Text>
                        </Space>
                    </div>
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>默认测试模型：</Typography.Text>
                    </div>
                    <Input
                      name='test_model'
                      placeholder='单渠道测试未带 model 参数时使用'
                      onChange={value => handleInputChange('test_model', value)}
                      value={inputs.test_model}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>备注 / 标签：</Typography.Text>
                    </div>
                    <Input
                      name='remark'
                      placeholder='备注'
                      onChange={value => handleInputChange('remark', value)}
                      value={inputs.remark}
                    />
                    <Input
                      style={{ marginTop: 8 }}
                      name='tag'
                      placeholder='标签'
                      onChange={value => handleInputChange('tag', value)}
                      value={inputs.tag}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>状态码映射 JSON：</Typography.Text>
                    </div>
                    <TextArea
                      placeholder='如 {"429":200}'
                      onChange={value => handleInputChange('status_code_mapping', value)}
                      value={inputs.status_code_mapping}
                      style={{ minHeight: 60, fontFamily: 'JetBrains Mono, Consolas' }}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>请求体覆盖 JSON：</Typography.Text>
                    </div>
                    <TextArea
                      onChange={value => handleInputChange('param_override', value)}
                      value={inputs.param_override}
                      style={{ minHeight: 80, fontFamily: 'JetBrains Mono, Consolas' }}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>请求头覆盖 JSON：</Typography.Text>
                    </div>
                    <TextArea
                      onChange={value => handleInputChange('header_override', value)}
                      value={inputs.header_override}
                      style={{ minHeight: 60, fontFamily: 'JetBrains Mono, Consolas' }}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>
                          setting 列（接口字段 setting）— 与下方 settings 为不同字段
                        </Typography.Text>
                    </div>
                    <TextArea
                      placeholder='库列 setting，JSON，一般可留空'
                      onChange={value => handleInputChange('setting', value)}
                      value={inputs.setting}
                      style={{ minHeight: 50, fontFamily: 'JetBrains Mono, Consolas' }}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>
                          settings 列（接口字段 settings）
                        </Typography.Text>
                    </div>
                    <TextArea
                      placeholder='库列 settings，JSON（如 api_version 等）'
                      onChange={value => handleInputChange('settings', value)}
                      value={inputs.settings}
                      style={{ minHeight: 50, fontFamily: 'JetBrains Mono, Consolas' }}
                    />
                    <div style={{ marginTop: 10 }}>
                        <Typography.Text strong>other_info：</Typography.Text>
                    </div>
                    <TextArea
                      onChange={value => handleInputChange('other_info', value)}
                      value={inputs.other_info}
                      style={{ minHeight: 40, fontFamily: 'JetBrains Mono, Consolas' }}
                    />

                    {
                      !isEdit && (
                        <div style={{ marginTop: 10, display: 'flex' }}>
                            <Space>
                                <Checkbox
                                  checked={batch}
                                  label='批量创建'
                                  name='batch'
                                  onChange={() => setBatch(!batch)}
                                />
                                <Typography.Text strong>批量创建</Typography.Text>
                            </Space>
                        </div>
                      )
                    }

                </Spin>
            </SideSheet>
        </>
    );
};

export default EditChannel;
