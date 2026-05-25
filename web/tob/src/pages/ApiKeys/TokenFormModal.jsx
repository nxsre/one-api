import { useEffect, useState } from 'react';
import { DatePicker, Form, Input, Modal, Select, Switch, message } from 'antd';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { getApiErrorMessage } from '@/api/client';
import { fetchUserAvailableModelIds } from '@/lib/modelCatalog';
import {
  buildTokenPayload,
  copyText,
  createToken,
  fetchTokenById,
  getCopyKeyValue,
  normalizeTokenForForm,
  SUBNET_FIELD_MESSAGE,
  validateSubnetField,
  updateToken,
} from '@/lib/tokens';

dayjs.locale('zh-cn');

const defaultForm = {
  name: '',
  remain_quota: 500000,
  expired_time: null,
  unlimited_quota: false,
  models: [],
  subnet: '',
};

export default function TokenFormModal({ open, tokenId, initialModels, onClose, onSuccess }) {
  const isEdit = tokenId != null;
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [modelOptions, setModelOptions] = useState([]);
  const [createdKey, setCreatedKey] = useState('');

  useEffect(() => {
    if (!open) {
      setCreatedKey('');
      return;
    }
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const ids = await fetchUserAvailableModelIds();
        if (cancelled) return;
        setModelOptions(ids.map((id) => ({ value: id, label: id })));
        if (isEdit) {
          const data = await fetchTokenById(tokenId);
          if (cancelled) return;
          form.setFieldsValue(normalizeTokenForForm(data));
        } else {
          form.setFieldsValue({
            ...defaultForm,
            models: initialModels?.length ? initialModels : [],
          });
        }
      } catch (e) {
        if (!cancelled) {
          Modal.error({ title: '加载失败', content: getApiErrorMessage(e) });
          onClose();
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, tokenId, initialModels, isEdit, form, onClose]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      const payload = buildTokenPayload(values);
      if (isEdit) {
        await updateToken({ ...payload, id: Number(tokenId) });
        onSuccess({ type: 'update' });
        onClose();
      } else {
        const res = await createToken(payload);
        const key = res.data?.key;
        if (key) setCreatedKey(String(key));
        onSuccess({ type: 'create', key });
        if (!key) onClose();
      }
    } catch (e) {
      if (e?.errorFields) return;
      Modal.error({ title: '保存失败', content: getApiErrorMessage(e) });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title={isEdit ? '编辑令牌' : '创建新 Key'}
      open={open}
      className="apikeys-form-modal"
      onCancel={onClose}
      width={520}
      destroyOnClose
      footer={
        createdKey ? (
          <div className="apikeys-form-footer">
            <button
              type="button"
              className="tob-btn tob-btn-ghost"
              onClick={async () => {
                const ok = await copyText(getCopyKeyValue(createdKey));
                if (ok) message.success('已复制成功');
                else message.error('复制失败');
              }}
            >
              复制密钥
            </button>
            <button type="button" className="tob-btn tob-btn-primary" onClick={onClose}>
              完成
            </button>
          </div>
        ) : (
          <div className="apikeys-form-footer">
            <button type="button" className="tob-btn tob-btn-ghost" onClick={onClose}>
              取消
            </button>
            <button
              type="button"
              className="tob-btn tob-btn-primary"
              disabled={saving}
              onClick={handleSubmit}
            >
              {saving ? '保存中…' : '保存'}
            </button>
          </div>
        )
      }
    >
      {createdKey ? (
        <div className="apikeys-created-key">
          <p>密钥已创建，请立即复制保存（仅显示一次）：</p>
          <code>{getCopyKeyValue(createdKey)}</code>
        </div>
      ) : (
        <Form form={form} layout="vertical" disabled={loading}>
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: !isEdit, message: '请输入名称' }]}
          >
            <Input placeholder="例如：生产环境 Key" />
          </Form.Item>
          <Form.Item name="models" label="可用模型">
            <Select
              mode="multiple"
              allowClear
              placeholder="不限制则留空"
              options={modelOptions}
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item
            name="subnet"
            label="IP 限制（CIDR，可选）"
            rules={[
              {
                validator: async (_, value) => {
                  const msg = validateSubnetField(value);
                  if (msg) throw new Error(msg);
                },
              },
            ]}
            extra={SUBNET_FIELD_MESSAGE}
          >
            <Input.TextArea
              rows={2}
            />
          </Form.Item>
          <Form.Item name="expired_time" label="过期时间">
            <DatePicker
              className="apikeys-expire-picker"
              showTime={{ format: 'HH:mm' }}
              format="YYYY-MM-DD HH:mm"
              placeholder="留空表示永不过期"
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <div className="apikeys-expire-presets">
            <button type="button" onClick={() => form.setFieldValue('expired_time', null)}>
              永不过期
            </button>
            <button
              type="button"
              onClick={() => form.setFieldValue('expired_time', dayjs().add(1, 'month'))}
            >
              1 个月
            </button>
          </div>
          <Form.Item name="unlimited_quota" label="无限额度" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.unlimited_quota !== cur.unlimited_quota}
          >
            {({ getFieldValue }) =>
              !getFieldValue('unlimited_quota') ? (
                <Form.Item name="remain_quota" label="剩余额度">
                  <Input type="number" min={0} />
                </Form.Item>
              ) : null
            }
          </Form.Item>
        </Form>
      )}
    </Modal>
  );
}
