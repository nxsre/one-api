import { API } from './api';

/** 已登录用户二次验证（写入 session，供后续敏感接口使用） */
export async function verifyStepUp2FA(code) {
  const res = await API.post('/api/verify', {
    method: '2fa',
    code: (code || '').trim(),
  });
  if (!res?.data?.success) {
    throw new Error(res?.data?.message || '验证失败');
  }
}

/** 须已通过 verifyStepUp2FA（或 5 分钟内验证仍有效） */
export async function fetchChannelKeyAfterVerify(channelId) {
  const res = await API.post(`/api/channel/${channelId}/key`, {});
  if (!res?.data?.success) {
    const msg = res?.data?.message || '获取失败';
    const c = res?.data?.code;
    throw new Error(c ? `${msg} (${c})` : msg);
  }
  return res.data.data?.key ?? '';
}
