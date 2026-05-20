import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Divider, Form, Grid, Header, Icon, Popup } from 'semantic-ui-react';
import {
  API,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
  noAutofillSecretProps,
  noAutofillTextProps,
} from '../helpers';
import PricingEntryEditor from './PricingEntryEditor';
import PricingEntryBlockPanel from './PricingEntryBlockPanel';
import './PricingEntryBlockPanel.css';
import UpstreamRatioSync from './UpstreamRatioSync';
import {
  fetchOperationGroupOptions,
  fetchOperationModelOptions,
} from '../helpers/operationRatioLookup';
import {
  BillingExprHelpContent,
  BillingModeHelpContent,
} from './OperationFieldHelp';
import './OperationFieldHelp.css';

const RATIO_BLOCK_DEFS = [
  {
    blockId: 'model_ratio',
    titleKey: 'setting.operation.ratio.model.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'completion_ratio',
    titleKey: 'setting.operation.ratio.completion.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'model_price',
    titleKey: 'setting.operation.ratio.model_price.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'cache_ratio',
    titleKey: 'setting.operation.ratio.cache.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'create_cache_ratio',
    titleKey: 'setting.operation.ratio.create_cache.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'image_ratio',
    titleKey: 'setting.operation.ratio.image.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'audio_ratio',
    titleKey: 'setting.operation.ratio.audio.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'audio_completion_ratio',
    titleKey: 'setting.operation.ratio.audio_completion.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
  },
  {
    blockId: 'billing_mode',
    titleKey: 'setting.operation.ratio.billing_mode.title',
    valueKind: 'string',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
    helpContent: BillingModeHelpContent,
  },
  {
    blockId: 'billing_expr',
    titleKey: 'setting.operation.ratio.billing_expr.title',
    valueKind: 'string',
    keyLabelKey: 'setting.operation.ratio.editor.col_model',
    keyOptions: 'model',
    helpContent: BillingExprHelpContent,
  },
  {
    blockId: 'group_ratio',
    titleKey: 'setting.operation.ratio.group.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_group',
    keyOptions: 'group',
  },
  {
    blockId: 'group_group_ratio',
    titleKey: 'setting.operation.ratio.group_group.title',
    valueKind: 'group_group',
    keyLabelKey: 'setting.operation.ratio.group_group.col_user_group',
    subKeyLabelKey: 'setting.operation.ratio.group_group.col_use_group',
    keyOptions: 'group',
  },
  {
    blockId: 'topup_group_ratio',
    titleKey: 'setting.operation.ratio.topup_group.title',
    valueKind: 'number',
    keyLabelKey: 'setting.operation.ratio.editor.col_group',
    keyOptions: 'group',
  },
];

function BlockHelpTrigger({ content: HelpContent }) {
  if (!HelpContent) return null;
  return (
    <Popup
      wide
      hoverable
      position='top left'
      className='operation-billing-help-popup-wrap'
      trigger={
        <span
          className='operation-field-help-trigger'
          role='button'
          tabIndex={0}
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') e.stopPropagation();
          }}
        >
          <Icon
            name='question circle outline'
            className='operation-field-help-icon'
            aria-label='help'
          />
        </span>
      }
      content={
        <div className='operation-billing-help-popup'>
          <HelpContent />
        </div>
      }
    />
  );
}

const OperationSetting = () => {
  const { t } = useTranslation();
  let now = new Date();
  let [inputs, setInputs] = useState({
    QuotaForNewUser: 0,
    QuotaForInviter: 0,
    QuotaForInvitee: 0,
    QuotaRemindThreshold: 0,
    PreConsumedQuota: 0,
    ModelRatio: '',
    ModelPrice: '',
    CompletionRatio: '',
    CacheRatio: '',
    CreateCacheRatio: '',
    ImageRatio: '',
    AudioRatio: '',
    AudioCompletionRatio: '',
    BillingMode: '',
    BillingExpr: '',
    GroupRatio: '',
    GroupGroupRatio: '',
    TopupGroupRatio: '',
    ExposeRatioEnabled: '',
    TopUpLink: '',
    ChatLink: '',
    QuotaPerUnit: 0,
    DefaultTenantChannelPricePer1k: 0,
    AutomaticDisableChannelEnabled: '',
    AutomaticEnableChannelEnabled: '',
    ChannelDisableThreshold: 0,
    LogConsumeEnabled: '',
    ErrorLogEnabled: 'true',
    DisplayInCurrencyEnabled: '',
    DisplayTokenStatEnabled: '',
    ApproximateTokenEnabled: '',
    RetryTimes: 0,
  });
  const [originInputs, setOriginInputs] = useState({});
  const [modelOptions, setModelOptions] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [lookupReady, setLookupReady] = useState(false);
  const [expandedBlocks, setExpandedBlocks] = useState({});
  const [blockCounts, setBlockCounts] = useState({});

  const loadBlockCounts = useCallback(async () => {
    try {
      const res = await API.get('/api/pricing_entries/blocks');
      const body = res.data || {};
      if (!body.success) return;
      const rows = Array.isArray(body.data) ? body.data : [];
      const next = {};
      rows.forEach((row) => {
        if (row?.block_id != null) {
          next[row.block_id] = Number(row.entry_count) || 0;
        }
      });
      setBlockCounts(next);
    } catch {
      /* ignore */
    }
  }, []);

  const ratioBlockIds = useMemo(
    () => RATIO_BLOCK_DEFS.map((b) => b.blockId),
    []
  );

  const toggleBlock = (blockId) => {
    setExpandedBlocks((prev) => ({
      ...prev,
      [blockId]: !prev[blockId],
    }));
  };

  const expandAllBlocks = () => {
    const next = {};
    ratioBlockIds.forEach((id) => {
      next[id] = true;
    });
    setExpandedBlocks(next);
  };

  const collapseAllBlocks = () => {
    setExpandedBlocks({});
  };

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [models, groups] = await Promise.all([
          fetchOperationModelOptions(),
          fetchOperationGroupOptions(),
        ]);
        if (!cancelled) {
          setModelOptions(models);
          setGroupOptions(groups);
          setLookupReady(true);
        }
      } catch (e) {
        if (!cancelled) {
          showError(e.message || 'failed to load model/group options');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const mergedModelOptions = modelOptions;
  const mergedGroupOptions = groupOptions;
  let [loading, setLoading] = useState(false);
  let [historyTimestamp, setHistoryTimestamp] = useState(
    timestamp2string(now.getTime() / 1000 - 30 * 24 * 3600)
  ); // a month ago

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.value === '{}') {
          item.value = '';
        }
        newInputs[item.key] = item.value;
      });
      const ratioDefaults = {
        ExposeRatioEnabled: 'false',
      };
      setInputs({ ...ratioDefaults, ...newInputs });
      setOriginInputs({ ...ratioDefaults, ...newInputs });
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions().then();
    loadBlockCounts().then();
  }, []);

  useEffect(() => {
    const onPricingApplied = () => {
      loadBlockCounts().then();
    };
    window.addEventListener('one-api-pricing-applied', onPricingApplied);
    return () =>
      window.removeEventListener('one-api-pricing-applied', onPricingApplied);
  }, [loadBlockCounts]);

  const updateOption = async (key, value) => {
    setLoading(true);
    if (key.endsWith('Enabled')) {
      value = inputs[key] === 'true' ? 'false' : 'true';
    }
    try {
      const res = await API.put('/api/option/', {
        key,
        value,
      });
      const { success, message } = res.data;
      if (success) {
        setInputs((prev) => ({ ...prev, [key]: value }));
        setOriginInputs((prev) => ({ ...prev, [key]: value }));
        return true;
      }
      showError(message);
      return false;
    } catch (e) {
      showError(e.message || '保存失败');
      return false;
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = async (e, { name, value }) => {
    if (name.endsWith('Enabled')) {
      await updateOption(name, value);
    } else {
      setInputs((inputs) => ({ ...inputs, [name]: value }));
    }
  };

  const submitConfig = async (group) => {
    switch (group) {
      case 'monitor':
        if (
          originInputs['ChannelDisableThreshold'] !==
          inputs.ChannelDisableThreshold
        ) {
          await updateOption(
            'ChannelDisableThreshold',
            inputs.ChannelDisableThreshold
          );
        }
        if (
          originInputs['QuotaRemindThreshold'] !== inputs.QuotaRemindThreshold
        ) {
          await updateOption(
            'QuotaRemindThreshold',
            inputs.QuotaRemindThreshold
          );
        }
        break;
      case 'ratio': {
        let savedCount = 0;
        if (originInputs.ExposeRatioEnabled !== inputs.ExposeRatioEnabled) {
          const ok = await updateOption(
            'ExposeRatioEnabled',
            inputs.ExposeRatioEnabled === 'true' ? 'true' : 'false'
          );
          if (!ok) {
            return;
          }
          savedCount += 1;
        }
        if (savedCount > 0) {
          showSuccess(t('setting.operation.ratio.save_success'));
        } else {
          showWarning(t('setting.operation.ratio.save_no_changes'));
        }
        break;
      }
      case 'quota':
        if (originInputs['QuotaForNewUser'] !== inputs.QuotaForNewUser) {
          await updateOption('QuotaForNewUser', inputs.QuotaForNewUser);
        }
        if (originInputs['QuotaForInvitee'] !== inputs.QuotaForInvitee) {
          await updateOption('QuotaForInvitee', inputs.QuotaForInvitee);
        }
        if (originInputs['QuotaForInviter'] !== inputs.QuotaForInviter) {
          await updateOption('QuotaForInviter', inputs.QuotaForInviter);
        }
        if (originInputs['PreConsumedQuota'] !== inputs.PreConsumedQuota) {
          await updateOption('PreConsumedQuota', inputs.PreConsumedQuota);
        }
        break;
      case 'general':
        if (originInputs['TopUpLink'] !== inputs.TopUpLink) {
          await updateOption('TopUpLink', inputs.TopUpLink);
        }
        if (originInputs['ChatLink'] !== inputs.ChatLink) {
          await updateOption('ChatLink', inputs.ChatLink);
        }
        if (originInputs['QuotaPerUnit'] !== inputs.QuotaPerUnit) {
          await updateOption('QuotaPerUnit', inputs.QuotaPerUnit);
        }
        if (originInputs['DefaultTenantChannelPricePer1k'] !== inputs.DefaultTenantChannelPricePer1k) {
          await updateOption('DefaultTenantChannelPricePer1k', inputs.DefaultTenantChannelPricePer1k);
        }
        if (originInputs['RetryTimes'] !== inputs.RetryTimes) {
          await updateOption('RetryTimes', inputs.RetryTimes);
        }
        break;
    }
  };

  const deleteHistoryLogs = async () => {
    console.log(inputs);
    const res = await API.delete(
      `/api/log/?target_timestamp=${Date.parse(historyTimestamp) / 1000}`
    );
    const { success, message, data } = res.data;
    if (success) {
      showSuccess(`${data} 条日志已清理！`);
      return;
    }
    showError('日志清理失败：' + message);
  };

  return (
    <Grid columns={1}>
      <Grid.Column>
        <Form
          loading={loading}
          onSubmit={(e) => {
            e.preventDefault();
          }}
        >
          <Header as='h3'>{t('setting.operation.quota.title')}</Header>
          <Form.Group widths='equal'>
            <Form.Input
              label={t('setting.operation.quota.new_user')}
              name='QuotaForNewUser'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.QuotaForNewUser}
              type='number'
              min='0'
              placeholder={t('setting.operation.quota.new_user_placeholder')}
            />
            <Form.Input
              label={t('setting.operation.quota.pre_consume')}
              name='PreConsumedQuota'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.PreConsumedQuota}
              type='number'
              min='0'
              placeholder={t('setting.operation.quota.pre_consume_placeholder')}
            />
            <Form.Input
              label={t('setting.operation.quota.inviter_reward')}
              name='QuotaForInviter'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.QuotaForInviter}
              type='number'
              min='0'
              placeholder={t(
                'setting.operation.quota.inviter_reward_placeholder'
              )}
            />
            <Form.Input
              label={t('setting.operation.quota.invitee_reward')}
              name='QuotaForInvitee'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.QuotaForInvitee}
              type='number'
              min='0'
              placeholder={t(
                'setting.operation.quota.invitee_reward_placeholder'
              )}
            />
          </Form.Group>
          <Form.Button
            type='button'
            onClick={() => {
              submitConfig('quota').then();
            }}
          >
            {t('setting.operation.quota.buttons.save')}
          </Form.Button>
          <Divider />
          <Header as='h3'>{t('setting.operation.ratio.title')}</Header>
          <div className='pricing-entry-blocks-toolbar'>
            <Button type='button' size='small' basic onClick={expandAllBlocks}>
              {t('setting.operation.ratio.editor.expand_all', '全部展开')}
            </Button>
            <Button type='button' size='small' basic onClick={collapseAllBlocks}>
              {t('setting.operation.ratio.editor.collapse_all', '全部折叠')}
            </Button>
          </div>
          {RATIO_BLOCK_DEFS.map((block) => {
            const keyOptions =
              block.keyOptions === 'group' ? mergedGroupOptions : mergedModelOptions;
            const expanded = !!expandedBlocks[block.blockId];
            return (
              <PricingEntryBlockPanel
                key={block.blockId}
                title={t(block.titleKey)}
                help={
                  block.helpContent ? (
                    <BlockHelpTrigger content={block.helpContent} />
                  ) : null
                }
                expanded={expanded}
                onToggle={() => toggleBlock(block.blockId)}
                entryCount={blockCounts[block.blockId]}
              >
                <PricingEntryEditor
                  blockId={block.blockId}
                  valueKind={block.valueKind}
                  keyColumnLabel={t(block.keyLabelKey)}
                  subKeyColumnLabel={
                    block.subKeyLabelKey ? t(block.subKeyLabelKey) : undefined
                  }
                  keyOptions={keyOptions}
                  keyDropdown={lookupReady}
                  onMutate={loadBlockCounts}
                />
              </PricingEntryBlockPanel>
            );
          })}
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.ExposeRatioEnabled === 'true'}
              label={t('setting.operation.ratio.expose_enabled')}
              name='ExposeRatioEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Button
            type='button'
            onClick={() => {
              submitConfig('ratio').then();
            }}
          >
            {t('setting.operation.ratio.buttons.save_expose')}
          </Form.Button>
          <UpstreamRatioSync />
          <Divider />
          <Header as='h3'>{t('setting.operation.log.title')}</Header>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.LogConsumeEnabled === 'true'}
              label={t('setting.operation.log.enable_consume')}
              name='LogConsumeEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.ErrorLogEnabled === 'true'}
              label={t('setting.operation.log.enable_error')}
              name='ErrorLogEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Group widths={4}>
            <Form.Input
              label={t('setting.operation.log.target_time')}
              value={historyTimestamp}
              type='datetime-local'
              name='history_timestamp'
              onChange={(e, { name, value }) => {
                setHistoryTimestamp(value);
              }}
            />
          </Form.Group>
          <Form.Button
            type='button'
            onClick={() => {
              deleteHistoryLogs().then();
            }}
          >
            {t('setting.operation.log.buttons.clean')}
          </Form.Button>

          <Divider />
          <Header as='h3'>{t('setting.operation.monitor.title')}</Header>
          <Form.Group widths={3}>
            <Form.Input
              label={t('setting.operation.monitor.max_response_time')}
              name='ChannelDisableThreshold'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.ChannelDisableThreshold}
              type='number'
              min='0'
              placeholder={t(
                'setting.operation.monitor.max_response_time_placeholder'
              )}
            />
            <Form.Input
              label={t('setting.operation.monitor.quota_reminder')}
              name='QuotaRemindThreshold'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.QuotaRemindThreshold}
              type='number'
              min='0'
              placeholder={t(
                'setting.operation.monitor.quota_reminder_placeholder'
              )}
            />
          </Form.Group>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.AutomaticDisableChannelEnabled === 'true'}
              label={t('setting.operation.monitor.auto_disable')}
              name='AutomaticDisableChannelEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.AutomaticEnableChannelEnabled === 'true'}
              label={t('setting.operation.monitor.auto_enable')}
              name='AutomaticEnableChannelEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Button
            type='button'
            onClick={() => {
              submitConfig('monitor').then();
            }}
          >
            {t('setting.operation.monitor.buttons.save')}
          </Form.Button>

          <Divider />
          <Header as='h3'>{t('setting.operation.general.title')}</Header>
          <Form.Group widths={4}>
            <Form.Input
              label={t('setting.operation.general.topup_link')}
              name='TopUpLink'
              placeholder={t('setting.operation.general.topup_link_placeholder')}
              value={inputs.TopUpLink}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.operation.general.chat_link')}
              name='ChatLink'
              placeholder={t('setting.operation.general.chat_link_placeholder')}
              value={inputs.ChatLink}
              onChange={handleInputChange}
              {...noAutofillTextProps}
            />
            <Form.Input
              label={t('setting.operation.general.quota_per_unit')}
              name='QuotaPerUnit'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.QuotaPerUnit}
              type='number'
              step='0.01'
              placeholder={t(
                'setting.operation.general.quota_per_unit_placeholder'
              )}
            />
            <Form.Input
              label='租户私有渠道默认千次调用单价'
              name='DefaultTenantChannelPricePer1k'
              onChange={handleInputChange}
              value={inputs.DefaultTenantChannelPricePer1k}
              type='number'
              step='0.001'
              placeholder='未配置专属单价时，系统默认千次调用单价'
            />
            <Form.Input
              label={t('setting.operation.general.retry_times')}
              name='RetryTimes'
              type={'number'}
              step='1'
              min='0'
              onChange={handleInputChange}
              {...noAutofillSecretProps}
              value={inputs.RetryTimes}
              placeholder={t(
                'setting.operation.general.retry_times_placeholder'
              )}
            />
          </Form.Group>
          <Form.Group inline>
            <Form.Checkbox
              checked={inputs.DisplayInCurrencyEnabled === 'true'}
              label={t('setting.operation.general.display_in_currency')}
              name='DisplayInCurrencyEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.DisplayTokenStatEnabled === 'true'}
              label={t('setting.operation.general.display_token_stat')}
              name='DisplayTokenStatEnabled'
              onChange={handleInputChange}
            />
            <Form.Checkbox
              checked={inputs.ApproximateTokenEnabled === 'true'}
              label={t('setting.operation.general.approximate_token')}
              name='ApproximateTokenEnabled'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Button
            type='button'
            onClick={() => {
              submitConfig('general').then();
            }}
          >
            {t('setting.operation.general.buttons.save')}
          </Form.Button>
        </Form>
      </Grid.Column>
    </Grid>
  );
};

export default OperationSetting;
