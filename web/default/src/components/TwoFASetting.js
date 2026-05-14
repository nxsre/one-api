import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Button,
  Checkbox,
  Divider,
  Form,
  Header,
  Image,
  Message,
  Modal,
} from 'semantic-ui-react';
import { API, copy, showError, showSuccess, showWarning } from '../helpers';

const qrImgUrl = (otpauth) =>
  `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(
    otpauth || ''
  )}`;

const TwoFASetting = ({ forceMode = false, onEnabled }) => {
  const [status, setStatus] = useState({
    enabled: false,
    locked: false,
    backup_codes_remaining: 0,
  });
  const [setupOpen, setSetupOpen] = useState(false);
  const [enableOpen, setEnableOpen] = useState(false);
  const [disableOpen, setDisableOpen] = useState(false);
  const [backupOpen, setBackupOpen] = useState(false);
  const [setupData, setSetupData] = useState(null);
  const [code, setCode] = useState('');
  const [confirmDisable, setConfirmDisable] = useState(false);
  const [newBackupCodes, setNewBackupCodes] = useState([]);
  const [loading, setLoading] = useState(false);
  const [statusLoaded, setStatusLoaded] = useState(false);
  const autoSetupStarted = useRef(false);

  const fetchStatus = useCallback(async () => {
    try {
      const res = await API.get('/api/user/2fa/status');
      if (res.data?.success && res.data.data) {
        setStatus(res.data.data);
      }
    } catch {
      showError('获取两步验证状态失败');
    } finally {
      setStatusLoaded(true);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  const startSetup = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.post('/api/user/2fa/setup');
      if (res.data?.success) {
        setSetupData(res.data.data);
        setSetupOpen(true);
        setCode('');
      } else {
        showError(res.data?.message || '初始化失败');
      }
    } catch {
      showError('初始化两步验证失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (
      !forceMode ||
      !statusLoaded ||
      status.enabled ||
      status.locked ||
      autoSetupStarted.current
    ) {
      return;
    }
    autoSetupStarted.current = true;
    startSetup();
  }, [forceMode, status.enabled, status.locked, statusLoaded, startSetup]);

  const enable2fa = async () => {
    if (!code.trim()) {
      showWarning('请输入认证器上的 6 位数字验证码');
      return;
    }
    setLoading(true);
    try {
      const res = await API.post('/api/user/2fa/enable', { code: code.trim() });
      if (res.data?.success) {
        showSuccess('两步验证已启用');
        setEnableOpen(false);
        setSetupOpen(false);
        setCode('');
        setSetupData(null);
        fetchStatus();
        if (typeof onEnabled === 'function') {
          onEnabled();
        }
      } else {
        showError(res.data?.message || '启用失败');
      }
    } catch {
      showError('启用失败');
    } finally {
      setLoading(false);
    }
  };

  const disable2fa = async () => {
    if (!code.trim()) {
      showWarning('请输入验证码或 8 位备用码');
      return;
    }
    if (!confirmDisable) {
      showWarning('请勾选确认了解禁用后果');
      return;
    }
    setLoading(true);
    try {
      const res = await API.post('/api/user/2fa/disable', { code: code.trim() });
      if (res.data?.success) {
        showSuccess('两步验证已禁用');
        setDisableOpen(false);
        setCode('');
        setConfirmDisable(false);
        fetchStatus();
      } else {
        showError(res.data?.message || '禁用失败');
      }
    } catch {
      showError('禁用失败');
    } finally {
      setLoading(false);
    }
  };

  const regenBackup = async () => {
    if (!code.trim()) {
      showWarning('请输入认证器验证码以重新生成备用码');
      return;
    }
    setLoading(true);
    try {
      const res = await API.post('/api/user/2fa/backup_codes', {
        code: code.trim(),
      });
      if (res.data?.success && res.data.data?.backup_codes) {
        setNewBackupCodes(res.data.data.backup_codes);
        showSuccess('备用码已重新生成，请妥善保存');
        setCode('');
        fetchStatus();
      } else {
        showError(res.data?.message || '操作失败');
      }
    } catch {
      showError('操作失败');
    } finally {
      setLoading(false);
    }
  };

  const copyBackups = async () => {
    const text = newBackupCodes.join('\n');
    await copy(text);
    showSuccess('已复制到剪贴板');
  };

  return (
    <div style={{ marginTop: 8 }}>
      <Message info>
        使用 TOTP
        认证器（如 Google Authenticator）扫描二维码后，按提示输入 6
        位验证码完成绑定。登录时可在验证码与备用码之间二选一。
      </Message>
      {forceMode ? (
        <Message warning>
          管理员已开启全员 MFA。完成两步验证配置前，你无法继续使用控制台或 API。
        </Message>
      ) : null}
      <p>
        状态：
        <strong>{status.enabled ? '已启用' : '未启用'}</strong>
        {status.enabled && status.backup_codes_remaining != null ? (
          <span style={{ marginLeft: 12 }}>
            剩余备用码：<strong>{status.backup_codes_remaining}</strong>
          </span>
        ) : null}
      </p>
      {status.locked ? (
        <Message negative>两步验证已锁定，请联系管理员处理。</Message>
      ) : !status.enabled ? (
        forceMode ? (
          <Button primary loading={loading} disabled>
            正在打开两步验证配置
          </Button>
        ) : (
          <Button primary loading={loading} onClick={startSetup}>
            启用两步验证
          </Button>
        )
      ) : (
        <>
          {!forceMode ? (
            <Button
              negative
              onClick={() => {
                setDisableOpen(true);
                setCode('');
                setConfirmDisable(false);
              }}
            >
              禁用两步验证
            </Button>
          ) : null}
          <Button
            style={{ marginLeft: 8 }}
            onClick={() => {
              setBackupOpen(true);
              setNewBackupCodes([]);
              setCode('');
            }}
          >
            重新生成备用码
          </Button>
        </>
      )}

      <Modal
        open={setupOpen}
        onClose={forceMode ? undefined : () => setSetupOpen(false)}
        closeOnDimmerClick={!forceMode}
        closeOnEscape={!forceMode}
        size='small'
      >
        <Modal.Header>设置两步验证</Modal.Header>
        <Modal.Content>
          {setupData && (
            <>
              <p>请使用认证器扫描以下二维码（或手动输入密钥）：</p>
              <Image
                src={qrImgUrl(setupData.qr_code_data)}
                alt='2FA QR'
                centered
                style={{ maxWidth: 220 }}
              />
              <Message>
                <div style={{ wordBreak: 'break-all' }}>
                  <strong>密钥（勿泄露）：</strong> {setupData.secret}
                </div>
              </Message>
              <Message warning>
                <Header as='h5'>备用码（仅显示一次，请保存）</Header>
                <pre style={{ whiteSpace: 'pre-wrap' }}>
                  {(setupData.backup_codes || []).join('\n')}
                </pre>
                <Button
                  size='small'
                  onClick={() => copy((setupData.backup_codes || []).join('\n'))}
                >
                  复制备用码
                </Button>
              </Message>
              <Button
                primary
                onClick={() => {
                  setSetupOpen(false);
                  setEnableOpen(true);
                  setCode('');
                }}
              >
                下一步：输入验证码启用
              </Button>
            </>
          )}
        </Modal.Content>
      </Modal>

      <Modal
        open={enableOpen}
        onClose={forceMode ? undefined : () => setEnableOpen(false)}
        closeOnDimmerClick={!forceMode}
        closeOnEscape={!forceMode}
        size='tiny'
      >
        <Modal.Header>确认启用</Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Input
              label='认证器 6 位验证码'
              placeholder='123456'
              value={code}
              onChange={(e, { value }) => setCode(value)}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          {!forceMode ? (
            <Button onClick={() => setEnableOpen(false)}>取消</Button>
          ) : null}
          <Button primary loading={loading} onClick={enable2fa}>
            启用
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={disableOpen} onClose={() => setDisableOpen(false)} size='tiny'>
        <Modal.Header>禁用两步验证</Modal.Header>
        <Modal.Content>
          <Message warning>
            禁用后账户安全性降低；若管理员开启「全员 2FA」，禁用后可能无法使用部分功能。
          </Message>
          <Form>
            <Form.Input
              label='验证码或备用码'
              value={code}
              onChange={(e, { value }) => setCode(value)}
            />
            <Form.Field>
              <Checkbox
                label='我已了解禁用后果'
                checked={confirmDisable}
                onChange={(e, { checked }) => setConfirmDisable(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDisableOpen(false)}>取消</Button>
          <Button negative loading={loading} onClick={disable2fa}>
            确认禁用
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={backupOpen} onClose={() => setBackupOpen(false)} size='small'>
        <Modal.Header>重新生成备用码</Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Input
              label='当前认证器 6 位验证码'
              value={code}
              onChange={(e, { value }) => setCode(value)}
            />
          </Form>
          {newBackupCodes.length > 0 && (
            <>
              <Divider />
              <Message success>
                <pre style={{ whiteSpace: 'pre-wrap' }}>
                  {newBackupCodes.join('\n')}
                </pre>
                <Button size='small' onClick={copyBackups}>
                  复制新备用码
                </Button>
              </Message>
            </>
          )}
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setBackupOpen(false)}>关闭</Button>
          <Button primary loading={loading} onClick={regenBackup}>
            生成
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
};

export default TwoFASetting;
