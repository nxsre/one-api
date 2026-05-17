const defaultConfig = {
  input: {
    name: '',
    type: 41,
    key: '',
    base_url: '',
    model_mapping: '',
    system_prompt: '',
    models: [],
    groups: ['default'],
    config: {
      routing_provider: '',
      routing_skip_adaptive: false
    }
  },
  inputLabel: {
    name: '渠道名称',
    type: '渠道类型',
    base_url: '渠道API地址',
    key: '密钥',
    models: '模型',
    model_mapping: '模型映射关系',
    system_prompt: '系统提示词',
    groups: '用户组',
    config: null
  },
  prompt: {
    type: '请选择渠道类型',
    name: '请为渠道命名',
    base_url: '可空，请输入中转API地址，例如通过cloudflare中转',
    key: '请输入渠道对应的鉴权密钥',
    models: '请选择该渠道所支持的模型',
    model_mapping:
      '请输入要修改的模型映射关系，格式为：api请求模型ID:实际转发给渠道的模型ID，使用JSON数组表示，例如：{"gpt-3.5": "gpt-35"}',
    system_prompt:
      '此项可选，用于强制设置给定的系统提示词，请配合自定义模型 & 模型重定向使用，首先创建一个唯一的自定义模型名称并在上面填入，之后将该自定义模型重定向映射到该渠道一个原生支持的模型此项可选，用于强制设置给定的系统提示词，请配合自定义模型 & 模型重定向使用，首先创建一个唯一的自定义模型名称并在上面填入，之后将该自定义模型重定向映射到该渠道一个原生支持的模型',
    groups: '请选择该渠道所支持的用户组',
    config: null
  },
  modelGroup: 'openai'
};

// key 与 relay/channeltype 紧凑枚举 type 对齐（与 channels.type、BuiltinEditorTypes 一致）
const typeConfig = {
  41: {
    modelGroup: 'openai'
  },
  15: {
    input: {
      models: ['gemini-2.0-flash']
    },
    prompt: {
      base_url: '可留空使用 Google 官方 …/v1beta/openai'
    },
    modelGroup: 'google gemini openai'
  },
  42: {
    inputLabel: {
      config: { api_version: 'API 版本' }
    },
    input: {
      models: ['gemini-2.0-flash']
    },
    prompt: {
      config: { api_version: '例如 v1 或 v1beta（与上游 OpenAI 兼容前缀一致）' }
    },
    modelGroup: 'google gemini openai'
  },
  46: {
    input: {
      models: ['claude-instant-1', 'claude-2', 'claude-2.0', 'claude-2.1']
    },
    modelGroup: 'anthropic'
  },
  43: {
    input: {
      models: []
    },
    prompt: {
      key: '请输入 AiPPT 渠道密钥（格式见类型说明）'
    },
    modelGroup: 'aippt'
  },
  2: {
    inputLabel: {
      base_url: 'AZURE_OPENAI_ENDPOINT',
      config: {
        api_version: '默认 API 版本'
      }
    },
    prompt: {
      base_url: '请填写AZURE_OPENAI_ENDPOINT',
      config: {
        api_version: '请输入默认API版本，例如：2024-03-01-preview'
      }
    }
  },
  4: {
    input: {
      models: ['PaLM-2']
    },
    modelGroup: 'google palm'
  },
  5: {
    input: {
      models: ['claude-instant-1', 'claude-2', 'claude-2.0', 'claude-2.1']
    },
    modelGroup: 'anthropic'
  },
  6: {
    input: {
      models: ['ERNIE-Bot', 'ERNIE-Bot-turbo', 'ERNIE-Bot-4', 'Embedding-V1']
    },
    prompt: {
      key: '按照如下格式输入：APIKey|SecretKey'
    },
    modelGroup: 'baidu'
  },
  7: {
    input: {
      models: ['glm-4', 'glm-4v', 'glm-3-turbo', 'chatglm_turbo', 'chatglm_pro', 'chatglm_std', 'chatglm_lite']
    },
    modelGroup: 'zhipu'
  },
  8: {
    inputLabel: {
      config: {
        plugin: '插件参数'
      }
    },
    input: {
      models: ['qwen-turbo', 'qwen-plus', 'qwen-max', 'qwen-max-longcontext', 'text-embedding-v1']
    },
    prompt: {
      config: {
        plugin: '请输入插件参数，即 X-DashScope-Plugin 请求头的取值'
      }
    },
    modelGroup: 'ali'
  },
  9: {
    inputLabel: {
      config: { api_version: '版本号' }
    },
    input: {
      models: ['SparkDesk', 'SparkDesk-v1.1', 'SparkDesk-v2.1', 'SparkDesk-v3.1', 'SparkDesk-v3.1-128K', 'SparkDesk-v3.5', 'SparkDesk-v3.5-32K', 'SparkDesk-v4.0']
    },
    prompt: {
      key: '按照如下格式输入：APPID|APISecret|APIKey',
      config: { api_version: '请输入版本号，例如：v3.1' }
    },
    modelGroup: 'xunfei'
  },
  10: {
    input: {
      models: ['360GPT_S2_V9', 'embedding-bert-512-v1', 'embedding_s1_v1', 'semantic_similarity_s1_v1']
    },
    modelGroup: '360'
  },
  12: {
    inputLabel: {
      config: { library_id: '知识库 ID' }
    },
    prompt: {
      config: { library_id: '请输入知识库 ID，例如：123456' }
    }
  },
  13: {
    input: {
      models: ['hunyuan']
    },
    prompt: {
      key: '按照如下格式输入：AppId|SecretId|SecretKey'
    },
    modelGroup: 'tencent'
  },
  14: {
    inputLabel: {
      config: { api_version: '版本号' }
    },
    input: {
      models: ['gemini-pro']
    },
    prompt: {
      config: { api_version: '请输入版本号，例如：v1' }
    },
    modelGroup: 'google gemini'
  },
  16: {
    input: {
      models: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k']
    },
    modelGroup: 'moonshot'
  },
  17: {
    input: {
      models: ['Baichuan2-Turbo', 'Baichuan2-Turbo-192k', 'Baichuan-Text-Embedding']
    },
    modelGroup: 'baichuan'
  },
  18: {
    input: {
      models: ['abab5.5s-chat', 'abab5.5-chat', 'abab6-chat']
    },
    modelGroup: 'minimax'
  },
  19: {
    modelGroup: 'mistral'
  },
  20: {
    modelGroup: 'groq'
  },
  21: {
    modelGroup: 'ollama'
  },
  22: {
    modelGroup: 'lingyiwanwu'
  },
  24: {
    inputLabel: {
      key: '',
      config: {
        region: 'Region',
        ak: 'Access Key',
        sk: 'Secret Key'
      }
    },
    prompt: {
      key: '',
      config: {
        region: 'region，e.g. us-west-2',
        ak: 'AWS IAM Access Key',
        sk: 'AWS IAM Secret Key'
      }
    },
    modelGroup: 'anthropic'
  },
  28: {
    inputLabel: {
      config: {
        user_id: 'Account ID'
      }
    },
    prompt: {
      config: {
        user_id: '请输入 Account ID，例如：d8d7c61dbc334c32d3ced580e4bf42b4'
      }
    },
    modelGroup: 'Cloudflare'
  },
  25: {
    inputLabel: {
      config: {
        user_id: 'User ID'
      }
    },
    prompt: {
      models: '对于 Coze 而言，模型名称即 Bot ID，你可以添加一个前缀 `bot-`，例如：`bot-123456`',
      config: {
        user_id: '生成该密钥的用户 ID'
      }
    },
    modelGroup: 'Coze'
  },
  33: {
    inputLabel: {
      key: '',
      config: {
        region: 'Vertex AI Region',
        vertex_ai_project_id: 'Vertex AI Project ID',
        vertex_ai_adc: 'Google Cloud Application Default Credentials JSON'
      }
    },
    prompt: {
      key: '',
      config: {
        region: 'Vertex AI Region.g. us-east5',
        vertex_ai_project_id: 'Vertex AI Project ID',
        vertex_ai_adc: 'Google Cloud Application Default Credentials JSON: https://cloud.google.com/docs/authentication/application-default-credentials'
      }
    },
    modelGroup: 'anthropic'
  },
  36: {
    modelGroup: 'xai'
  },
  44: {
    input: {
      models: ['amap']
    },
    prompt: {
      key: '请输入高德 Web 服务 API Key'
    },
    modelGroup: 'amap'
  },
  45: {
    input: {
      models: ['deep-research']
    },
    prompt: {
      key: '深知上游 Bearer Token',
      base_url: '完整前缀至 …/deepresearch；客户端须 stream:true',
      config: {
        deep_research_mode: '可选：general | academic | user_files'
      }
    },
    modelGroup: 'deepresearch'
  }
};

export { defaultConfig, typeConfig };
