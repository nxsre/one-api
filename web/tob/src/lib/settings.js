import { API } from '@/api/client';
import { getStoredStatus, normalizeStatusResponse } from '@/lib/systemStatus';

const QUOTA_DIVISOR = 500_000;

export function formatQuotaDisplay(quota) {
  const v = Number(quota) || 0;
  const amount = v / QUOTA_DIVISOR;
  if (amount >= 1) return amount.toFixed(2);
  if (amount >= 0.001) return amount.toFixed(3);
  return amount.toFixed(4);
}

export function roleLabel(role) {
  const r = Number(role);
  if (r >= 100) return '系统管理员';
  if (r >= 50) return '超级管理员';
  if (r === 20) return '租户管理员';
  if (r >= 10) return '管理员';
  return '普通用户';
}

export async function fetchOAuthState() {
  const res = await API.get('/api/oauth/state');
  if (!res.data?.success) {
    throw new Error(res.data?.message || '获取 OAuth 状态失败');
  }
  return res.data.data;
}

export async function bindWechat(code) {
  const res = await API.get(`/api/oauth/wechat/bind?code=${encodeURIComponent(code.trim())}`);
  if (!res.data?.success) throw new Error(res.data?.message || '绑定失败');
}

export function openGitHubOAuth(clientId) {
  return fetchOAuthState().then((state) => {
    if (!state) return;
    window.open(
      `https://github.com/login/oauth/authorize?client_id=${clientId}&state=${state}&scope=user:email`
    );
  });
}

export function openLarkOAuth(clientId) {
  return fetchOAuthState().then((state) => {
    if (!state) return;
    const redirect_uri = `${window.location.origin}/oauth/lark`;
    window.open(
      `https://open.feishu.cn/open-apis/authen/v1/index?redirect_uri=${encodeURIComponent(redirect_uri)}&app_id=${encodeURIComponent(clientId)}&state=${encodeURIComponent(state)}`
    );
  });
}

export async function fetchSelfProfile() {
  const res = await API.get('/api/user/self');
  if (!res.data?.success) {
    throw new Error(res.data?.message || '加载账户信息失败');
  }
  return res.data.data;
}

export async function updateSelfProfile(payload) {
  const body = { ...payload };
  if (!body.password) delete body.password;
  const res = await API.put('/api/user/self', body);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '保存失败');
  }
  return res.data.data;
}

export async function generateSystemToken() {
  const res = await API.get('/api/user/token');
  if (!res.data?.success) {
    throw new Error(res.data?.message || '生成失败');
  }
  return res.data.data;
}

export async function fetchAffiliateLink() {
  const res = await API.get('/api/user/aff');
  if (!res.data?.success) {
    throw new Error(res.data?.message || '获取失败');
  }
  return `${window.location.origin}/register?aff=${res.data.data}`;
}

export async function deleteSelfAccount() {
  const res = await API.delete('/api/user/self');
  if (!res.data?.success) {
    throw new Error(res.data?.message || '注销失败');
  }
  try {
    await API.get('/api/user/logout');
  } catch {
    /* session may already be gone */
  }
}

export async function submitTenantUpgrade(payload) {
  const res = await API.post('/api/user/tenant_upgrade', payload);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '提交失败');
  }
}

export async function sendEmailVerification(email, turnstileToken = '') {
  const q = new URLSearchParams({ email });
  if (turnstileToken) q.set('turnstile', turnstileToken);
  const res = await API.get(`/api/verification?${q.toString()}`);
  if (!res.data?.success) {
    throw new Error(res.data?.message || '发送失败');
  }
}

export async function bindEmail(email, code) {
  const res = await API.get(
    `/api/oauth/email/bind?email=${encodeURIComponent(email)}&code=${encodeURIComponent(code)}`
  );
  if (!res.data?.success) {
    throw new Error(res.data?.message || '绑定失败');
  }
}

export async function refreshSystemStatus() {
  const res = await API.get('/api/status');
  const payload = normalizeStatusResponse(res.data);
  if (payload && Object.keys(payload).length > 0) {
    localStorage.setItem('status', JSON.stringify(payload));
  }
  return payload;
}

export function getCachedStatus() {
  return getStoredStatus();
}

export async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

/** S3 */
export async function fetchS3Self() {
  const data = await fetchSelfProfile();
  return {
    site: data.s3_site_enabled,
    enabled: data.s3_enabled,
    region: data.s3_region,
    accessKey: data.s3_access_key || '',
  };
}

export async function s3Enable() {
  const res = await API.post('/api/user/s3/enable');
  if (!res.data?.success) throw new Error(res.data?.message || '启用失败');
  return res.data.data;
}

export async function s3Disable() {
  const res = await API.post('/api/user/s3/disable');
  if (!res.data?.success) throw new Error(res.data?.message || '关闭失败');
}

export async function s3RegenerateSecret() {
  const res = await API.post('/api/user/s3/regenerate_secret');
  if (!res.data?.success) throw new Error(res.data?.message || '操作失败');
  return res.data.data;
}

export async function s3RotateKeys() {
  const res = await API.post('/api/user/s3/rotate_keys');
  if (!res.data?.success) throw new Error(res.data?.message || '操作失败');
  return res.data.data;
}

/** 2FA */
export async function fetch2faStatus() {
  const res = await API.get('/api/user/2fa/status');
  if (!res.data?.success) throw new Error(res.data?.message || '加载失败');
  return res.data.data;
}

export async function setup2fa() {
  const res = await API.post('/api/user/2fa/setup');
  if (!res.data?.success) throw new Error(res.data?.message || '初始化失败');
  return res.data.data;
}

export async function enable2fa(code) {
  const res = await API.post('/api/user/2fa/enable', { code: code.trim() });
  if (!res.data?.success) throw new Error(res.data?.message || '启用失败');
}

export async function disable2fa(code) {
  const res = await API.post('/api/user/2fa/disable', { code: code.trim() });
  if (!res.data?.success) throw new Error(res.data?.message || '禁用失败');
}

export async function regenBackupCodes(code) {
  const res = await API.post('/api/user/2fa/backup_codes', { code: code.trim() });
  if (!res.data?.success) throw new Error(res.data?.message || '操作失败');
  return res.data.data?.backup_codes || [];
}

export const qr2faUrl = (otpauth) =>
  `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(otpauth || '')}`;
