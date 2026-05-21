export function getStoredStatus() {
  try {
    const raw = localStorage.getItem('status');
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

export function isSecurePasswordLoginEnabled() {
  const data = getStoredStatus();
  if (Object.prototype.hasOwnProperty.call(data, 'secure_password_login')) {
    return data.secure_password_login === true || data.secure_password_login === 'true';
  }
  return false;
}

export function isLoginMathCaptchaEnabled(status) {
  const s = status || getStoredStatus();
  return !!s.login_math_captcha && !s.turnstile_check;
}
