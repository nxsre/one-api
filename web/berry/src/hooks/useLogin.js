import { API } from 'utils/api';
import { buildLoginPayload } from 'utils/secureLogin';
import { useDispatch } from 'react-redux';
import { LOGIN } from 'store/actions';
import { useNavigate } from 'react-router';
import { showSuccess } from 'utils/common';

/**
 * @param {{ captcha_id?: string, captcha_clicks?: {x:number,y:number}[] }} [captcha]
 * @param {{ id: string, ts: number, sig: string } | null} [loginProof]
 */
const useLogin = () => {
  const dispatch = useDispatch();
  const navigate = useNavigate();

  const login = async (username, password, captcha, loginProof) => {
    try {
      const body = await buildLoginPayload(
        username,
        password,
        captcha,
        loginProof
      );
      const res = await API.post(`/api/user/login`, body);
      const { success, message, data } = res.data;
      if (success && data?.require_2fa) {
        return { success: false, require2FA: true, message: '' };
      }
      if (success) {
        localStorage.setItem('user', JSON.stringify(data));
        dispatch({ type: LOGIN, payload: data });
        navigate(data?.require_force_2fa_setup ? '/setting' : '/panel');
        return { success: true, message: '', require2FA: false };
      }
      return { success, message: message || '', require2FA: false };
    } catch (err) {
      return {
        success: false,
        message: err?.message || '',
        require2FA: false,
      };
    }
  };

  const verify2FALogin = async (code) => {
    try {
      const res = await API.post(`/api/user/login/2fa`, { code: code.trim() });
      const { success, message, data } = res.data;
      if (success) {
        localStorage.setItem('user', JSON.stringify(data));
        dispatch({ type: LOGIN, payload: data });
        showSuccess('登录成功！');
        navigate(data?.require_force_2fa_setup ? '/setting' : '/panel');
      }
      return { success, message };
    } catch {
      return { success: false, message: '验证失败' };
    }
  };

  const githubLogin = async (code, state) => {
    try {
      const res = await API.get(`/api/oauth/github?code=${code}&state=${state}`);
      const { success, message, data } = res.data;
      if (success) {
        if (message === 'bind') {
          showSuccess('绑定成功！');
          navigate('/panel');
        } else {
          dispatch({ type: LOGIN, payload: data });
          localStorage.setItem('user', JSON.stringify(data));
          showSuccess('登录成功！');
          navigate('/panel');
        }
      }
      return { success, message };
    } catch (err) {
      return { success: false, message: '' };
    }
  };

  const larkLogin = async (code, state) => {
    try {
      const res = await API.get(`/api/oauth/lark?code=${code}&state=${state}`);
      const { success, message, data } = res.data;
      if (success) {
        if (message === 'bind') {
          showSuccess('绑定成功！');
          navigate('/panel');
        } else {
          dispatch({ type: LOGIN, payload: data });
          localStorage.setItem('user', JSON.stringify(data));
          showSuccess('登录成功！');
          navigate('/panel');
        }
      }
      return { success, message };
    } catch (err) {
      return { success: false, message: '' };
    }
  };

  const oidcLogin = async (code, state) => {
    try {
      const res = await API.get(`/api/oauth/oidc?code=${code}&state=${state}`);
      const { success, message, data } = res.data;
      if (success) {
        if (message === 'bind') {
          showSuccess('绑定成功！');
          navigate('/panel');
        } else {
          dispatch({ type: LOGIN, payload: data });
          localStorage.setItem('user', JSON.stringify(data));
          showSuccess('登录成功！');
          navigate('/panel');
        }
      }
      return { success, message };
    } catch (err) {
      return { success: false, message: '' };
    }
  };

  const wechatLogin = async (code) => {
    try {
      const res = await API.get(`/api/oauth/wechat?code=${code}`);
      const { success, message, data } = res.data;
      if (success) {
        dispatch({ type: LOGIN, payload: data });
        localStorage.setItem('user', JSON.stringify(data));
        showSuccess('登录成功！');
        navigate('/panel');
      }
      return { success, message };
    } catch (err) {
      return { success: false, message: '' };
    }
  };

  const logout = async () => {
    await API.get('/api/user/logout');
    localStorage.removeItem('user');
    dispatch({ type: LOGIN, payload: null });
    navigate('/');
  };

  return {
    login,
    verify2FALogin,
    logout,
    githubLogin,
    wechatLogin,
    larkLogin,
    oidcLogin,
  };
};

export default useLogin;
