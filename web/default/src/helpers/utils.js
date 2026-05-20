import {toast} from 'react-toastify';
import {toastConstants} from '../constants';
import React from 'react';
import {API} from './api';
import { clearNacosEmbeddedConsoleLocalSession } from './nacosEmbeddedConsole';
import { clearTenantConsoleActingTenantId } from './tenantConsoleImpersonation';

const HTMLToastContent = ({ htmlContent }) => {
  return <div dangerouslySetInnerHTML={{ __html: htmlContent }} />;
};
export default HTMLToastContent;

/** 租户管理员角色（仅租户控制台，非平台管理员菜单） */
export const ROLE_TENANT_ADMIN = 20;

export function isTenantAdmin() {
  let user = localStorage.getItem('user');
  if (!user) return false;
  user = JSON.parse(user);
  return Number(user.role) === ROLE_TENANT_ADMIN;
}

export function isAdmin() {
  let user = localStorage.getItem('user');
  if (!user) return false;
  user = JSON.parse(user);
  if (Number(user.role) === ROLE_TENANT_ADMIN) return false;
  return user.role >= 10;
}

export function getStoredUserJSON() {
  try {
    const raw = localStorage.getItem('user');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

/** 具备租户控制台委派权限的子账号（普通用户 + tenant_id + tenant_permissions） */
export function isTenantConsoleDelegate() {
  const u = getStoredUserJSON();
  if (!u) return false;
  const tid = u.tenant_id;
  return (
    Number(u.role) === 1 &&
    tid != null &&
    tid !== '' &&
    Array.isArray(u.tenant_permissions) &&
    u.tenant_permissions.length > 0
  );
}

/** 租户管理员视为拥有全部租户内权限；子账号按 tenant_permissions 校验 */
export function hasTenantPermission(perm) {
  if (!perm) return false;
  if (isTenantAdmin()) return true;
  const u = getStoredUserJSON();
  if (!u || !Array.isArray(u.tenant_permissions)) return false;
  return u.tenant_permissions.includes(perm);
}

export function postLoginDefaultPath(userPayload) {
  if (!userPayload) return '/token';
  if (userPayload.require_force_2fa_setup) return '/setting';
  if (Number(userPayload.role) === ROLE_TENANT_ADMIN) return '/tenant-console';
  if (
    Number(userPayload.role) === 1 &&
    userPayload.tenant_id != null &&
    userPayload.tenant_id !== '' &&
    Array.isArray(userPayload.tenant_permissions) &&
    userPayload.tenant_permissions.length > 0
  ) {
    return '/tenant-console';
  }
  return '/token';
}

export function isRoot() {
  let user = localStorage.getItem('user');
  if (!user) return false;
  user = JSON.parse(user);
  return user.role >= 100;
}

/** 与 /api/status 的 secure_password_login 一致；未加载 status 时默认 false。 */
export function isSecurePasswordLoginEnabled() {
  try {
    const raw = localStorage.getItem('status');
    if (!raw) return false;
    const data = JSON.parse(raw);
    if (Object.prototype.hasOwnProperty.call(data, 'secure_password_login')) {
      return (
        data.secure_password_login === true ||
        data.secure_password_login === 'true'
      );
    }
  } catch {
    /* ignore */
  }
  return false;
}

/** 与 /api/status 的 nacos_enabled 一致；未加载 status 时默认 false。 */
export function isNacosEnabled() {
  try {
    const raw = localStorage.getItem('status');
    if (!raw) return false;
    const data = JSON.parse(raw);
    if (data && Object.prototype.hasOwnProperty.call(data, 'nacos_enabled')) {
      return data.nacos_enabled === true || data.nacos_enabled === 'true';
    }
  } catch {
    /* ignore */
  }
  return false;
}

export function getSystemName() {
  let system_name = localStorage.getItem('system_name');
  if (!system_name) return 'One API';
  return system_name;
}

export function getLogo() {
  let logo = localStorage.getItem('logo');
  if (!logo) return '/logo.png';
  return logo;
}

export function getFooterHTML() {
  return localStorage.getItem('footer_html');
}

export async function copy(text) {
  let okay = true;
  try {
    await navigator.clipboard.writeText(text);
  } catch (e) {
    okay = false;
    console.error(e);
  }
  return okay;
}

export function isMobile() {
  return window.innerWidth <= 600;
}

let showErrorOptions = { autoClose: toastConstants.ERROR_TIMEOUT };
let showWarningOptions = { autoClose: toastConstants.WARNING_TIMEOUT };
let showSuccessOptions = { autoClose: toastConstants.SUCCESS_TIMEOUT };
let showInfoOptions = { autoClose: toastConstants.INFO_TIMEOUT };
let showNoticeOptions = { autoClose: false };

if (isMobile()) {
  showErrorOptions.position = 'top-center';
  // showErrorOptions.transition = 'flip';

  showSuccessOptions.position = 'top-center';
  // showSuccessOptions.transition = 'flip';

  showInfoOptions.position = 'top-center';
  // showInfoOptions.transition = 'flip';

  showNoticeOptions.position = 'top-center';
  // showNoticeOptions.transition = 'flip';
}

export function showError(error) {
  if (!error) return;
  console.error(error);
  if (error.message) {
    if (error.name === 'AxiosError') {
      switch (error.response.status) {
        case 401:
          // toast.error('错误：未登录或登录已过期，请重新登录！', showErrorOptions);
          localStorage.removeItem('user');
          clearTenantConsoleActingTenantId();
          clearNacosEmbeddedConsoleLocalSession();
          if (window.location.pathname !== '/login') {
            window.location.replace('/login?expired=true');
          }
          break;
        case 429:
          toast.error('错误：请求次数过多，请稍后再试！', showErrorOptions);
          break;
        case 500:
          toast.error('错误：服务器内部错误，请联系管理员！', showErrorOptions);
          break;
        case 405:
          toast.info('本站仅作演示之用，无服务端！');
          break;
        default:
          toast.error('错误：' + error.message, showErrorOptions);
      }
      return;
    }
    toast.error('错误：' + error.message, showErrorOptions);
  } else {
    toast.error('错误：' + error, showErrorOptions);
  }
}

export function showWarning(message) {
  toast.warn(message, showWarningOptions);
}

export function showSuccess(message) {
  toast.success(message, showSuccessOptions);
}

export function showInfo(message) {
  toast.info(message, showInfoOptions);
}

export function showNotice(message, isHTML = false) {
  if (isHTML) {
    toast(<HTMLToastContent htmlContent={message} />, showNoticeOptions);
  } else {
    toast.info(message, showNoticeOptions);
  }
}

export function openPage(url) {
  window.open(url);
}

export function removeTrailingSlash(url) {
  if (url.endsWith('/')) {
    return url.slice(0, -1);
  } else {
    return url;
  }
}

export function timestamp2string(timestamp) {
  let date = new Date(timestamp * 1000);
  let year = date.getFullYear().toString();
  let month = (date.getMonth() + 1).toString();
  let day = date.getDate().toString();
  let hour = date.getHours().toString();
  let minute = date.getMinutes().toString();
  let second = date.getSeconds().toString();
  if (month.length === 1) {
    month = '0' + month;
  }
  if (day.length === 1) {
    day = '0' + day;
  }
  if (hour.length === 1) {
    hour = '0' + hour;
  }
  if (minute.length === 1) {
    minute = '0' + minute;
  }
  if (second.length === 1) {
    second = '0' + second;
  }
  return (
      year + '-' + month + '-' + day + ' ' + hour + ':' + minute + ':' + second
  );
}

export function downloadTextAsFile(text, filename) {
  let blob = new Blob([text], { type: 'text/plain;charset=utf-8' });
  let url = URL.createObjectURL(blob);
  let a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
}

export const verifyJSON = (str) => {
  try {
    JSON.parse(str);
  } catch (e) {
    return false;
  }
  return true;
};

export function shouldShowPrompt(id) {
  let prompt = localStorage.getItem(`prompt-${id}`);
  return !prompt;
}

export function setPromptShown(id) {
  localStorage.setItem(`prompt-${id}`, 'true');
}

/** 解析渠道「自定义模型」输入：逗号（中英文）、分号、换行分隔。 */
export function splitModelNameList(raw) {
  if (typeof raw !== 'string' || raw.trim() === '') {
    return [];
  }
  return raw
    .split(/[\n\r\t,，;]+/u)
    .map((s) => s.trim())
    .filter(Boolean);
}

let channelModels = undefined;
export async function loadChannelModels() {
  const res = await API.get('/api/models');
  const { success, data } = res.data;
  if (!success) {
    return;
  }
  channelModels = data;
  localStorage.setItem('channel_models', JSON.stringify(data));
}

export function getChannelModels(type) {
  if (channelModels !== undefined && type in channelModels) {
    return channelModels[type];
  }
  let models = localStorage.getItem('channel_models');
  if (!models) {
    return [];
  }
  channelModels = JSON.parse(models);
  if (type in channelModels) {
    return channelModels[type];
  }
  return [];
}
