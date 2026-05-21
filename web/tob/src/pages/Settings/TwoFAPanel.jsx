import { useCallback, useEffect, useRef, useState } from 'react';
import { Modal } from 'antd';
import { getApiErrorMessage } from '@/api/client';
import {
  copyText,
  disable2fa,
  enable2fa,
  fetch2faStatus,
  qr2faUrl,
  regenBackupCodes,
  setup2fa,
} from '@/lib/settings';

export default function TwoFAPanel({
  forceMode = false,
  onEnabled,
  onCancelLogin,
  cancelLoginBusy = false,
}) {
  const [status, setStatus] = useState({
    enabled: false,
    locked: false,
    backup_codes_remaining: 0,
  });
  const [statusLoaded, setStatusLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [code, setCode] = useState('');
  const [confirmDisable, setConfirmDisable] = useState(false);
  const [setupData, setSetupData] = useState(null);
  const [newBackupCodes, setNewBackupCodes] = useState([]);
  const [setupOpen, setSetupOpen] = useState(false);
  const [enableOpen, setEnableOpen] = useState(false);
  const [disableOpen, setDisableOpen] = useState(false);
  const [backupOpen, setBackupOpen] = useState(false);
  const [error, setError] = useState('');
  const autoSetupStarted = useRef(false);

  const loadStatus = useCallback(async () => {
    try {
      const data = await fetch2faStatus();
      setStatus(data);
    } catch (e) {
      setError(getApiErrorMessage(e));
    } finally {
      setStatusLoaded(true);
    }
  }, []);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const startSetup = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await setup2fa();
      setSetupData(data);
      setSetupOpen(true);
      setCode('');
    } catch (e) {
      setError(getApiErrorMessage(e));
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

  const handleEnable = async () => {
    if (!code.trim()) {
      setError('请输入认证器上的 6 位数字验证码');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await enable2fa(code);
      setEnableOpen(false);
      setSetupOpen(false);
      setCode('');
      setSetupData(null);
      await loadStatus();
      onEnabled?.();
    } catch (e) {
      setError(getApiErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const handleDisable = async () => {
    if (!code.trim()) {
      setError('请输入验证码或 8 位备用码');
      return;
    }
    if (!confirmDisable) {
      setError('请勾选确认了解禁用后果');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await disable2fa(code);
      setDisableOpen(false);
      setCode('');
      setConfirmDisable(false);
      await loadStatus();
    } catch (e) {
      setError(getApiErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const handleRegenBackup = async () => {
    if (!code.trim()) {
      setError('请输入认证器验证码以重新生成备用码');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const codes = await regenBackupCodes(code);
      setNewBackupCodes(codes);
      setCode('');
      await loadStatus();
    } catch (e) {
      setError(getApiErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const copyBackups = async (list) => {
    const ok = await copyText((list || []).join('\n'));
    if (ok) setError('');
  };

  return (
    <div className="settings-2fa-block">
      <div className="settings-alert settings-alert-info">
        使用 TOTP 认证器（如 Google Authenticator）扫描二维码后，按提示输入 6
        位验证码完成绑定。登录时可在验证码与备用码之间二选一。
      </div>
      {forceMode ? (
        <div className="settings-alert settings-alert-warning">
          管理员已开启全员 MFA。完成两步验证配置前，你无法继续使用控制台或 API。
        </div>
      ) : null}

      <div className="settings-2fa-status">
        状态：<strong>{status.enabled ? '已启用' : '未启用'}</strong>
        {status.enabled && status.backup_codes_remaining != null ? (
          <span style={{ marginLeft: 12 }}>
            剩余备用码：<strong>{status.backup_codes_remaining}</strong>
          </span>
        ) : null}
      </div>

      {status.locked ? (
        <div className="settings-alert settings-alert-danger">
          两步验证已锁定，请联系管理员处理。
        </div>
      ) : null}

      {error ? <div className="settings-alert settings-alert-danger">{error}</div> : null}

      <div className="settings-inline-actions">
        {status.locked ? null : !status.enabled ? (
          forceMode ? (
            <button type="button" className="tob-btn tob-btn-primary" disabled>
              正在打开两步验证配置…
            </button>
          ) : (
            <button
              type="button"
              className="tob-btn tob-btn-primary"
              disabled={loading}
              onClick={startSetup}
            >
              启用两步验证
            </button>
          )
        ) : (
          <>
            {!forceMode ? (
              <button
                type="button"
                className="tob-btn tob-btn-danger"
                onClick={() => {
                  setDisableOpen(true);
                  setCode('');
                  setConfirmDisable(false);
                  setError('');
                }}
              >
                禁用两步验证
              </button>
            ) : null}
            <button
              type="button"
              className="tob-btn tob-btn-ghost"
              onClick={() => {
                setBackupOpen(true);
                setNewBackupCodes([]);
                setCode('');
                setError('');
              }}
            >
              重新生成备用码
            </button>
          </>
        )}
        {forceMode && onCancelLogin ? (
          <button
            type="button"
            className="tob-btn tob-btn-ghost"
            disabled={cancelLoginBusy || loading}
            onClick={onCancelLogin}
          >
            取消并返回登录
          </button>
        ) : null}
      </div>

      <Modal
        className="settings-form-modal"
        title="设置两步验证"
        open={setupOpen}
        onCancel={forceMode ? undefined : () => setSetupOpen(false)}
        closable={!forceMode}
        maskClosable={!forceMode}
        footer={
          <div className="settings-modal-actions settings-modal-actions-equal">
            {forceMode && onCancelLogin ? (
              <button
                type="button"
                className="tob-btn tob-btn-ghost"
                disabled={cancelLoginBusy}
                onClick={onCancelLogin}
              >
                返回登录
              </button>
            ) : (
              <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setSetupOpen(false)}>
                取消
              </button>
            )}
            <button
              type="button"
              className="tob-btn tob-btn-primary"
              disabled={!setupData}
              onClick={() => {
                setSetupOpen(false);
                setEnableOpen(true);
                setCode('');
              }}
            >
              下一步：输入验证码启用
            </button>
          </div>
        }
      >
        {setupData ? (
          <>
            <p style={{ fontSize: 13, color: 'var(--text2)' }}>
              请使用认证器扫描以下二维码（或手动输入密钥）：
            </p>
            <img
              className="settings-qr"
              src={qr2faUrl(setupData.qr_code_data)}
              alt="2FA QR"
            />
            <div className="settings-readonly-field" style={{ wordBreak: 'break-all' }}>
              <strong>密钥（勿泄露）：</strong> {setupData.secret}
            </div>
            <p style={{ marginTop: 12, fontSize: 13, fontWeight: 600 }}>备用码（仅显示一次，请保存）</p>
            <pre className="settings-backup-pre">
              {(setupData.backup_codes || []).join('\n')}
            </pre>
            <button
              type="button"
              className="tob-btn tob-btn-ghost"
              style={{ marginTop: 8 }}
              onClick={() => copyBackups(setupData.backup_codes)}
            >
              复制备用码
            </button>
          </>
        ) : null}
      </Modal>

      <Modal
        className="settings-form-modal"
        title="确认启用"
        open={enableOpen}
        onCancel={forceMode ? undefined : () => setEnableOpen(false)}
        closable={!forceMode}
        maskClosable={!forceMode}
        onOk={handleEnable}
        confirmLoading={loading}
        okText="启用"
        cancelText={forceMode ? undefined : '取消'}
        cancelButtonProps={forceMode ? { style: { display: 'none' } } : undefined}
        footer={
          forceMode && onCancelLogin ? (
            <div className="settings-modal-actions settings-modal-actions-equal">
              <button
                type="button"
                className="tob-btn tob-btn-ghost"
                disabled={cancelLoginBusy}
                onClick={onCancelLogin}
              >
                返回登录
              </button>
              <button
                type="button"
                className="tob-btn tob-btn-primary"
                disabled={loading}
                onClick={handleEnable}
              >
                启用
              </button>
            </div>
          ) : undefined
        }
      >
        <div className="settings-form-group">
          <label className="settings-form-label">认证器 6 位验证码</label>
          <input
            className="settings-form-input"
            placeholder="123456"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            autoComplete="one-time-code"
          />
        </div>
      </Modal>

      <Modal
        className="settings-form-modal"
        title="禁用两步验证"
        open={disableOpen}
        onCancel={() => setDisableOpen(false)}
        footer={null}
      >
        <div className="settings-alert settings-alert-warning">
          禁用后账户安全性降低；若管理员开启「全员 2FA」，禁用后可能无法使用部分功能。
        </div>
        <div className="settings-form-group">
          <label className="settings-form-label">验证码或备用码</label>
          <input
            className="settings-form-input"
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
        <label style={{ display: 'flex', gap: 8, fontSize: 13, color: 'var(--text2)' }}>
          <input
            type="checkbox"
            checked={confirmDisable}
            onChange={(e) => setConfirmDisable(e.target.checked)}
          />
          我了解禁用两步验证会降低账户安全性
        </label>
        <div className="settings-modal-actions settings-modal-actions-equal">
          <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setDisableOpen(false)}>
            取消
          </button>
          <button
            type="button"
            className="tob-btn tob-btn-danger"
            disabled={loading}
            onClick={handleDisable}
          >
            确认禁用
          </button>
        </div>
      </Modal>

      <Modal
        className="settings-form-modal"
        title="重新生成备用码"
        open={backupOpen}
        onCancel={() => setBackupOpen(false)}
        footer={null}
      >
        <div className="settings-form-group">
          <label className="settings-form-label">认证器 6 位验证码</label>
          <input
            className="settings-form-input"
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
        {newBackupCodes.length > 0 ? (
          <>
            <pre className="settings-backup-pre">{newBackupCodes.join('\n')}</pre>
            <button
              type="button"
              className="tob-btn tob-btn-ghost"
              onClick={() => copyBackups(newBackupCodes)}
            >
              复制备用码
            </button>
          </>
        ) : null}
        <div className="settings-modal-actions settings-modal-actions-equal">
          <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setBackupOpen(false)}>
            关闭
          </button>
          <button
            type="button"
            className="tob-btn tob-btn-primary"
            disabled={loading}
            onClick={handleRegenBackup}
          >
            生成
          </button>
        </div>
      </Modal>
    </div>
  );
}
