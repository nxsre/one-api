import { useCallback, useEffect, useState } from 'react';
import { Modal } from 'antd';
import { useNavigate } from 'react-router-dom';
import { getApiErrorMessage } from '@/api/client';
import {
  clearUserSession,
  getStoredUser,
  logout,
  saveUserSession,
} from '@/lib/auth';
import {
  bindEmail as submitEmailBind,
  bindWechat,
  copyText,
  deleteSelfAccount,
  fetchAffiliateLink,
  fetchS3Self,
  formatQuotaDisplay,
  generateSystemToken,
  getCachedStatus,
  openGitHubOAuth,
  openLarkOAuth,
  roleLabel,
  s3Disable,
  s3Enable,
  s3RegenerateSecret,
  s3RotateKeys,
  sendEmailVerification,
  submitTenantUpgrade,
  updateSelfProfile,
  fetchSelfProfile,
} from '@/lib/settings';
import TwoFAPanel from './TwoFAPanel';
import './settings.css';

function SaveIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
      <polyline points="17 21 17 13 7 13 7 21" />
      <polyline points="7 3 7 8 15 8" />
    </svg>
  );
}

function avatarLetter(name) {
  const s = String(name || '?').trim();
  return s ? s[0].toUpperCase() : '?';
}

export default function SettingsPage() {
  const navigate = useNavigate();
  const storedUser = getStoredUser();
  const force2fa = !!storedUser?.require_force_2fa_setup;

  const [status, setStatus] = useState(() => getCachedStatus());
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState('');
  const [error, setError] = useState('');

  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');

  const [systemToken, setSystemToken] = useState('');
  const [affLink, setAffLink] = useState('');
  const [s3Info, setS3Info] = useState(null);
  const [s3Secret, setS3Secret] = useState({
    open: false,
    accessKey: '',
    secretKey: '',
    subtitle: '',
  });

  const [emailModal, setEmailModal] = useState(false);
  const [wechatModal, setWechatModal] = useState(false);
  const [deleteModal, setDeleteModal] = useState(false);
  const [upgradeModal, setUpgradeModal] = useState(false);
  const [emailBindAddress, setEmailBindAddress] = useState('');
  const [bindCode, setBindCode] = useState('');
  const [wechatCode, setWechatCode] = useState('');
  const [deleteConfirm, setDeleteConfirm] = useState('');
  const [codeCooldown, setCodeCooldown] = useState(0);
  const [upgradeForm, setUpgradeForm] = useState({ name: '', slug: '', remark: '' });
  const [cancelLoginBusy, setCancelLoginBusy] = useState(false);

  const showToast = (msg) => {
    setToast(msg);
    window.setTimeout(() => setToast(''), 2200);
  };

  const loadProfile = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await fetchSelfProfile();
      setProfile(data);
      setUsername(data.username || '');
      setDisplayName(data.display_name || '');
      setPassword('');
      const user = getStoredUser();
      if (user) {
        const next = {
          ...user,
          username: data.username ?? user.username,
          display_name: data.display_name ?? user.display_name,
          role: data.role ?? user.role,
          group: data.group ?? user.group,
        };
        if (data.require_force_2fa_setup) {
          next.require_force_2fa_setup = true;
        } else {
          delete next.require_force_2fa_setup;
        }
        saveUserSession(next);
      }
    } catch (e) {
      setError(getApiErrorMessage(e));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadS3 = useCallback(async () => {
    try {
      const info = await fetchS3Self();
      setS3Info(info);
    } catch {
      /* optional */
    }
  }, []);

  useEffect(() => {
    loadProfile();
    loadS3();
  }, [loadProfile, loadS3]);

  useEffect(() => {
    if (!codeCooldown) return undefined;
    const t = window.setInterval(() => {
      setCodeCooldown((c) => (c <= 1 ? 0 : c - 1));
    }, 1000);
    return () => window.clearInterval(t);
  }, [codeCooldown]);

  const handleSave = async () => {
    if (!username.trim()) {
      setError('用户名不能为空');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await updateSelfProfile({
        username: username.trim(),
        display_name: displayName.trim(),
        password: password.trim() || undefined,
      });
      setPassword('');
      await loadProfile();
      showToast('保存成功');
    } catch (e) {
      setError(getApiErrorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  const handle2faEnabled = async () => {
    await loadProfile();
    showToast('两步验证已启用');
    if (force2fa) {
      navigate('/overview', { replace: true });
    }
  };

  const handleCancelForceLogin = async () => {
    setCancelLoginBusy(true);
    try {
      await logout();
      navigate('/login', { replace: true });
    } finally {
      setCancelLoginBusy(false);
    }
  };

  const handleGenerateToken = async () => {
    try {
      const token = await generateSystemToken();
      setSystemToken(token);
      setAffLink('');
      await copyText(token);
      showToast('系统令牌已生成并复制');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleAffLink = async () => {
    try {
      const link = await fetchAffiliateLink();
      setAffLink(link);
      setSystemToken('');
      await copyText(link);
      showToast('邀请链接已复制');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleDeleteAccount = async () => {
    const name = profile?.username || storedUser?.username;
    if (deleteConfirm !== name) {
      setError('请输入你的账户名以确认删除');
      return;
    }
    try {
      await deleteSelfAccount();
      clearUserSession();
      navigate('/login', { replace: true });
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const openS3Secret = (accessKey, secretKey, subtitle) => {
    setS3Secret({ open: true, accessKey, secretKey, subtitle });
  };

  const handleS3Enable = async () => {
    try {
      const data = await s3Enable();
      openS3Secret(data.access_key, data.secret_key, '临时 S3 已启用');
      await loadS3();
      showToast('临时 S3 已启用');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleS3Disable = async () => {
    if (!window.confirm('确定关闭并作废当前 S3 密钥？已存储的对象不会自动删除。')) return;
    try {
      await s3Disable();
      await loadS3();
      showToast('S3 已关闭');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleS3Regen = async () => {
    try {
      const data = await s3RegenerateSecret();
      openS3Secret(s3Info?.accessKey || '', data.secret_key, 'Secret 已更新');
      showToast('Secret 已更新');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleS3Rotate = async () => {
    if (!window.confirm('将生成新的 Access Key 与 Secret，旧密钥立即失效。继续？')) return;
    try {
      const data = await s3RotateKeys();
      openS3Secret(data.access_key, data.secret_key, '密钥已轮换');
      await loadS3();
      showToast('密钥已轮换');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const sendCode = async () => {
    if (!emailBindAddress.trim()) return;
    if (status.turnstile_check) {
      setError('当前环境需 Turnstile 验证，请在默认控制台完成邮箱绑定');
      return;
    }
    try {
      await sendEmailVerification(emailBindAddress.trim());
      setCodeCooldown(30);
      showToast('验证码已发送，请查收邮箱');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleBindEmail = async () => {
    try {
      await submitEmailBind(emailBindAddress.trim(), bindCode.trim());
      setEmailModal(false);
      setBindCode('');
      showToast('邮箱绑定成功');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleBindWechat = async () => {
    try {
      await bindWechat(wechatCode);
      setWechatModal(false);
      setWechatCode('');
      showToast('微信绑定成功');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const handleUpgrade = async () => {
    if (!upgradeForm.name.trim() || !upgradeForm.slug.trim()) {
      setError('企业名称和租户标识为必填项');
      return;
    }
    try {
      await submitTenantUpgrade(upgradeForm);
      setUpgradeModal(false);
      showToast('租户升级申请已提交');
    } catch (e) {
      setError(getApiErrorMessage(e));
    }
  };

  const role = profile?.role ?? storedUser?.role;
  const group = profile?.group ?? storedUser?.group ?? 'default';
  const quota = storedUser?.quota;
  const displayLabel = displayName || username || storedUser?.username || '—';

  return (
    <div className="settings-page page-enter">
      {toast ? <div className="settings-toast">{toast}</div> : null}

      <div className="settings-section-header">
        <div>
          <div className="settings-section-title">个人设置</div>
          <div className="settings-section-sub">管理账户信息、安全设置与系统访问</div>
        </div>
        <div className="settings-header-actions">
          <button
            type="button"
            className="tob-btn tob-btn-primary"
            disabled={saving || loading}
            onClick={handleSave}
          >
            <SaveIcon />
            {saving ? '保存中…' : '保存更改'}
          </button>
        </div>
      </div>

      {error ? <div className="settings-alert settings-alert-danger">{error}</div> : null}

      <div className="settings-grid-2-1">
        <div>
          <div className="settings-card">
            <div className="settings-card-title">基本信息</div>
            <div className="settings-profile-head">
              <div className="settings-avatar">{avatarLetter(displayLabel)}</div>
              <div>
                <div className="settings-profile-name">{displayLabel}</div>
                <div className="settings-profile-meta">
                  {roleLabel(role)} · {group}
                </div>
              </div>
            </div>
            <div className="settings-form-row">
              <div className="settings-form-group">
                <label className="settings-form-label" htmlFor="settings-username">
                  用户名
                </label>
                <input
                  id="settings-username"
                  className="settings-form-input"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  disabled={loading}
                />
              </div>
              <div className="settings-form-group">
                <label className="settings-form-label" htmlFor="settings-display-name">
                  显示名称
                </label>
                <input
                  id="settings-display-name"
                  className="settings-form-input"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  disabled={loading}
                />
              </div>
            </div>
            <p className="settings-hint">
              邮箱、GitHub、微信等绑定信息需在下方「账户绑定」中操作；只读字段与 default 控制台一致。
            </p>
          </div>

          <div className="settings-card">
            <div className="settings-card-title">安全设置</div>
            <div className="settings-form-row">
              <div className="settings-form-group">
                <label className="settings-form-label" htmlFor="settings-password">
                  新密码
                </label>
                <input
                  id="settings-password"
                  type="password"
                  className="settings-form-input"
                  placeholder="留空则不修改"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                />
              </div>
            </div>
            <div className="settings-card-title" style={{ marginTop: 8, border: 'none', paddingBottom: 0 }}>
              双因素认证 (2FA)
            </div>
            <TwoFAPanel
              forceMode={force2fa}
              onEnabled={handle2faEnabled}
              onCancelLogin={force2fa ? handleCancelForceLogin : undefined}
              cancelLoginBusy={cancelLoginBusy}
            />
          </div>

          {s3Info?.site ? (
            <div className="settings-card">
              <div className="settings-card-title">临时 S3 存储</div>
              <p className="settings-hint">
                启用后可获得临时对象存储凭证；密钥仅展示一次，请妥善保存。
              </p>
              <div className="settings-s3-meta">
                区域：<code>{s3Info.region}</code>
              </div>
              <div className="settings-s3-meta">
                状态：{s3Info.enabled ? '已启用' : '未启用'}
                {s3Info.accessKey ? ` · Access Key: ${s3Info.accessKey}` : ''}
              </div>
              <div className="settings-inline-actions">
                {!s3Info.enabled ? (
                  <button
                    type="button"
                    className="tob-btn tob-btn-primary"
                    onClick={handleS3Enable}
                  >
                    启用 S3
                  </button>
                ) : (
                  <>
                    <button type="button" className="tob-btn tob-btn-ghost" onClick={handleS3Regen}>
                      重新生成 Secret
                    </button>
                    <button type="button" className="tob-btn tob-btn-ghost" onClick={handleS3Rotate}>
                      轮换密钥
                    </button>
                    <button type="button" className="tob-btn tob-btn-danger" onClick={handleS3Disable}>
                      关闭 S3
                    </button>
                  </>
                )}
              </div>
            </div>
          ) : null}
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div className="settings-accent-card">
            <div className="settings-accent-title">当前账户</div>
            <div className="settings-accent-value">{roleLabel(role)}</div>
            <div className="settings-accent-sub">
              用户组：{group}
              {quota != null ? ` · 余额约 ¥${formatQuotaDisplay(quota)}` : ''}
            </div>
            {Number(role) === 1 ? (
              <button
                type="button"
                className="settings-accent-btn"
                onClick={() => setUpgradeModal(true)}
              >
                升级为企业（租户） →
              </button>
            ) : null}
          </div>

          <div className="settings-card" style={{ marginBottom: 0 }}>
            <div className="settings-card-title">系统访问</div>
            <p className="settings-hint">
              系统令牌用于部分 API 调用；邀请链接用于推广注册。生成后会尝试复制到剪贴板。
            </p>
            <div className="settings-inline-actions">
              <button type="button" className="tob-btn tob-btn-ghost" onClick={handleGenerateToken}>
                生成系统令牌
              </button>
              <button type="button" className="tob-btn tob-btn-ghost" onClick={handleAffLink}>
                复制邀请链接
              </button>
            </div>
            {systemToken ? (
              <input
                className="settings-form-input settings-token-field"
                readOnly
                value={systemToken}
                onClick={(e) => {
                  e.target.select();
                  copyText(systemToken).then((ok) => ok && showToast('已复制'));
                }}
              />
            ) : null}
            {affLink ? (
              <input
                className="settings-form-input settings-token-field"
                readOnly
                value={affLink}
                onClick={(e) => {
                  e.target.select();
                  copyText(affLink).then((ok) => ok && showToast('已复制'));
                }}
              />
            ) : null}
          </div>

          <div className="settings-card" style={{ marginBottom: 0 }}>
            <div className="settings-card-title">账户绑定</div>
            <div className="settings-inline-actions">
              {status.wechat_login ? (
                <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setWechatModal(true)}>
                  绑定微信
                </button>
              ) : null}
              {status.github_oauth && status.github_client_id ? (
                <button
                  type="button"
                  className="tob-btn tob-btn-ghost"
                  onClick={() =>
                    openGitHubOAuth(status.github_client_id).catch((e) =>
                      setError(getApiErrorMessage(e))
                    )
                  }
                >
                  绑定 GitHub
                </button>
              ) : null}
              {status.lark_oauth && status.lark_client_id ? (
                <button
                  type="button"
                  className="tob-btn tob-btn-ghost"
                  onClick={() =>
                    openLarkOAuth(status.lark_client_id).catch((e) =>
                      setError(getApiErrorMessage(e))
                    )
                  }
                >
                  绑定飞书
                </button>
              ) : null}
              <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setEmailModal(true)}>
                绑定邮箱
              </button>
            </div>
          </div>

          <div className="settings-card settings-danger-card" style={{ marginBottom: 0 }}>
            <div className="settings-card-title settings-danger-title">危险操作</div>
            <p className="settings-danger-text">
              注销账户将清除个人数据且不可恢复，请确认已备份重要 API Key 与配置。
            </p>
            <div className="settings-danger-actions">
              <button
                type="button"
                className="tob-btn tob-btn-danger"
                onClick={() => {
                  setDeleteModal(true);
                  setDeleteConfirm('');
                }}
              >
                注销账户
              </button>
            </div>
          </div>
        </div>
      </div>

      <Modal
        className="settings-form-modal"
        title="S3 密钥（仅显示一次）"
        open={s3Secret.open}
        onCancel={() => setS3Secret((s) => ({ ...s, open: false, secretKey: '' }))}
        footer={null}
      >
        <p style={{ fontSize: 13, color: 'var(--text2)' }}>{s3Secret.subtitle}</p>
        {s3Secret.accessKey ? (
          <div className="settings-form-group">
            <label className="settings-form-label">Access Key</label>
            <input className="settings-form-input" readOnly value={s3Secret.accessKey} />
          </div>
        ) : null}
        <div className="settings-form-group">
          <label className="settings-form-label">Secret Key</label>
          <input className="settings-form-input" readOnly value={s3Secret.secretKey} />
        </div>
        <div className="settings-modal-actions settings-modal-actions-equal">
          <button
            type="button"
            className="tob-btn tob-btn-primary"
            onClick={async () => {
              const text = [
                s3Secret.accessKey ? `AK=${s3Secret.accessKey}` : '',
                `SK=${s3Secret.secretKey}`,
              ]
                .filter(Boolean)
                .join('\n');
              if (await copyText(text)) showToast('已复制');
            }}
          >
            复制到剪贴板
          </button>
          <button
            type="button"
            className="tob-btn tob-btn-ghost"
            onClick={() => setS3Secret((s) => ({ ...s, open: false, secretKey: '' }))}
          >
            关闭
          </button>
        </div>
      </Modal>

      <Modal
        className="settings-form-modal"
        title="绑定邮箱"
        open={emailModal}
        onCancel={() => setEmailModal(false)}
        footer={null}
      >
        <div className="settings-form-group">
          <label className="settings-form-label">邮箱</label>
          <div style={{ display: 'flex', gap: 8 }}>
            <input
              className="settings-form-input"
              type="email"
              value={emailBindAddress}
              onChange={(e) => setEmailBindAddress(e.target.value)}
            />
            <button
              type="button"
              className="tob-btn tob-btn-ghost"
              disabled={codeCooldown > 0}
              onClick={sendCode}
            >
              {codeCooldown > 0 ? `${codeCooldown}s` : '获取验证码'}
            </button>
          </div>
        </div>
        <div className="settings-form-group">
          <label className="settings-form-label">验证码</label>
          <input
            className="settings-form-input"
            value={bindCode}
            onChange={(e) => setBindCode(e.target.value)}
          />
        </div>
        <div className="settings-modal-actions settings-modal-actions-equal">
          <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setEmailModal(false)}>
            取消
          </button>
          <button type="button" className="tob-btn tob-btn-primary" onClick={handleBindEmail}>
            绑定
          </button>
        </div>
      </Modal>

      <Modal
        className="settings-form-modal"
        title="绑定微信"
        open={wechatModal}
        onCancel={() => setWechatModal(false)}
        footer={null}
      >
        {status.wechat_qrcode ? (
          <img
            src={status.wechat_qrcode}
            alt="微信二维码"
            style={{ width: '100%', maxWidth: 280, margin: '0 auto', display: 'block' }}
          />
        ) : null}
        <p className="settings-hint" style={{ textAlign: 'center' }}>
          扫码后输入验证码完成绑定
        </p>
        <input
          className="settings-form-input"
          placeholder="验证码"
          value={wechatCode}
          onChange={(e) => setWechatCode(e.target.value)}
        />
        <div className="settings-modal-actions">
          <button type="button" className="tob-btn tob-btn-primary" onClick={handleBindWechat}>
            绑定
          </button>
        </div>
      </Modal>

      <Modal
        className="settings-form-modal"
        title="升级为企业（租户）"
        open={upgradeModal}
        onCancel={() => setUpgradeModal(false)}
        footer={null}
      >
        <div className="settings-form-group">
          <label className="settings-form-label">企业名称</label>
          <input
            className="settings-form-input"
            value={upgradeForm.name}
            onChange={(e) => setUpgradeForm((f) => ({ ...f, name: e.target.value }))}
          />
        </div>
        <div className="settings-form-group">
          <label className="settings-form-label">租户标识 (slug)</label>
          <input
            className="settings-form-input"
            value={upgradeForm.slug}
            onChange={(e) => setUpgradeForm((f) => ({ ...f, slug: e.target.value }))}
          />
        </div>
        <div className="settings-form-group">
          <label className="settings-form-label">备注</label>
          <input
            className="settings-form-input"
            value={upgradeForm.remark}
            onChange={(e) => setUpgradeForm((f) => ({ ...f, remark: e.target.value }))}
          />
        </div>
        <div className="settings-modal-actions settings-modal-actions-equal">
          <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setUpgradeModal(false)}>
            取消
          </button>
          <button type="button" className="tob-btn tob-btn-primary" onClick={handleUpgrade}>
            提交申请
          </button>
        </div>
      </Modal>

      <Modal
        className="settings-form-modal"
        title="注销账户"
        open={deleteModal}
        onCancel={() => setDeleteModal(false)}
        footer={null}
      >
        <p className="settings-danger-text">
          请输入账户名 <strong>{profile?.username || storedUser?.username}</strong> 以确认注销：
        </p>
        <input
          className="settings-form-input"
          value={deleteConfirm}
          onChange={(e) => setDeleteConfirm(e.target.value)}
        />
        <div className="settings-modal-actions settings-modal-actions-equal">
          <button type="button" className="tob-btn tob-btn-ghost" onClick={() => setDeleteModal(false)}>
            取消
          </button>
          <button type="button" className="tob-btn tob-btn-danger" onClick={handleDeleteAccount}>
            确认注销
          </button>
        </div>
      </Modal>
    </div>
  );
}
