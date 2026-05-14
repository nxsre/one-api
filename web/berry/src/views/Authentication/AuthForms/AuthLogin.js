import { useState, useRef, useCallback, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { Link } from 'react-router-dom';

// material-ui
import { useTheme } from '@mui/material/styles';
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  FormHelperText,
  Grid,
  IconButton,
  InputAdornment,
  InputLabel,
  OutlinedInput,
  Stack,
  TextField,
  Typography,
  useMediaQuery
} from '@mui/material';

// third party
import * as Yup from 'yup';
import { Formik } from 'formik';

// project imports
import useLogin from 'hooks/useLogin';
import AnimateButton from 'ui-component/extended/AnimateButton';
import WechatModal from 'views/Authentication/AuthForms/WechatModal';

// assets
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';

import Github from 'assets/images/icons/github.svg';
import Wechat from 'assets/images/icons/wechat.svg';
import Lark from 'assets/images/icons/lark.svg';
import OIDC from 'assets/images/icons/oidc.svg';
import { onGitHubOAuthClicked, onLarkOAuthClicked, onOidcClicked, showInfo, showWarning } from 'utils/common';
import { API } from 'utils/api';

// ============================|| FIREBASE - LOGIN ||============================ //

const LoginForm = ({ ...others }) => {
  const theme = useTheme();
  const { login, wechatLogin, verify2FALogin } = useLogin();
  const [openWechat, setOpenWechat] = useState(false);
  const matchDownSM = useMediaQuery(theme.breakpoints.down('md'));
  const customization = useSelector((state) => state.customization);
  const siteInfo = useSelector((state) => state.siteInfo);

  const [captchaMasterSrc, setCaptchaMasterSrc] = useState('');
  const [captchaThumbSrc, setCaptchaThumbSrc] = useState('');
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [captchaLoadError, setCaptchaLoadError] = useState('');
  const [captchaDotNum, setCaptchaDotNum] = useState(0);
  const [captchaChallengeId, setCaptchaChallengeId] = useState('');
  const [captchaClicks, setCaptchaClicks] = useState([]);
  const [captchaMasterNaturalSize, setCaptchaMasterNaturalSize] = useState({ w: 0, h: 0 });
  const loginRequestProofRef = useRef(null);
  const [showTwoFA, setShowTwoFA] = useState(false);
  const [twoFACode, setTwoFACode] = useState('');

  const turnstileOn = !!siteInfo.turnstile_check;

  const loadLoginCaptcha = useCallback(async () => {
    if (!siteInfo.login_math_captcha || turnstileOn) return;
    setCaptchaLoadError('');
    setCaptchaLoading(true);
    try {
      const res = await API.get('/api/user/login/captcha');
      const d = res.data?.data;
      if (res.data?.success && d?.master_image && d?.thumb_image) {
        setCaptchaMasterNaturalSize({ w: 0, h: 0 });
        setCaptchaMasterSrc(d.master_image);
        setCaptchaThumbSrc(d.thumb_image);
        setCaptchaDotNum(Number(d.dot_num) || 0);
        setCaptchaChallengeId(d.captcha_id || '');
        setCaptchaClicks([]);
        setCaptchaLoadError('');
        if (d.login_request_id && d.login_request_sig != null && d.login_request_ts != null) {
          loginRequestProofRef.current = {
            id: d.login_request_id,
            ts: Number(d.login_request_ts),
            sig: d.login_request_sig
          };
        } else {
          loginRequestProofRef.current = null;
        }
      } else {
        setCaptchaMasterSrc('');
        setCaptchaThumbSrc('');
        setCaptchaDotNum(0);
        setCaptchaChallengeId('');
        setCaptchaClicks([]);
        loginRequestProofRef.current = null;
        setCaptchaLoadError(
          (res.data?.message && String(res.data.message).trim()) || '验证码加载失败，请稍后重试'
        );
      }
    } catch {
      setCaptchaLoadError('验证码加载失败，请稍后重试');
    } finally {
      setCaptchaLoading(false);
    }
  }, [siteInfo.login_math_captcha, turnstileOn]);

  useEffect(() => {
    if (siteInfo.login_math_captcha && !turnstileOn) {
      void loadLoginCaptcha();
    }
  }, [siteInfo.login_math_captcha, turnstileOn, loadLoginCaptcha]);

  const onMasterCaptchaClick = (e) => {
    if (!siteInfo.login_math_captcha || turnstileOn) return;
    if (!captchaDotNum || captchaClicks.length >= captchaDotNum) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    setCaptchaClicks((prev) => [...prev, { x, y }]);
  };

  let tripartiteLogin = false;
  if (siteInfo.github_oauth || siteInfo.wechat_login || siteInfo.lark_client_id || siteInfo.oidc) {
    tripartiteLogin = true;
  }

  const handleWechatOpen = () => {
    setOpenWechat(true);
  };

  const handleWechatClose = () => {
    setOpenWechat(false);
  };

  const [showPassword, setShowPassword] = useState(false);
  const handleClickShowPassword = () => {
    setShowPassword(!showPassword);
  };

  const handleMouseDownPassword = (event) => {
    event.preventDefault();
  };

  return (
    <>
      {tripartiteLogin && (
        <Grid container direction="column" justifyContent="center" spacing={2}>
          {siteInfo.github_oauth && (
            <Grid item xs={12}>
              <AnimateButton>
                <Button
                  disableElevation
                  fullWidth
                  onClick={() => onGitHubOAuthClicked(siteInfo.github_client_id)}
                  size="large"
                  variant="outlined"
                  sx={{
                    color: 'grey.700',
                    backgroundColor: theme.palette.grey[50],
                    borderColor: theme.palette.grey[100]
                  }}
                >
                  <Box sx={{ mr: { xs: 1, sm: 2, width: 20 }, display: 'flex', alignItems: 'center' }}>
                    <img src={Github} alt="github" width={25} height={25} style={{ marginRight: matchDownSM ? 8 : 16 }} />
                  </Box>
                  使用 GitHub 登录
                </Button>
              </AnimateButton>
            </Grid>
          )}
          {siteInfo.wechat_login && (
            <Grid item xs={12}>
              <AnimateButton>
                <Button
                  disableElevation
                  fullWidth
                  onClick={handleWechatOpen}
                  size="large"
                  variant="outlined"
                  sx={{
                    color: 'grey.700',
                    backgroundColor: theme.palette.grey[50],
                    borderColor: theme.palette.grey[100]
                  }}
                >
                  <Box sx={{ mr: { xs: 1, sm: 2, width: 20 }, display: 'flex', alignItems: 'center' }}>
                    <img src={Wechat} alt="Wechat" width={25} height={25} style={{ marginRight: matchDownSM ? 8 : 16 }} />
                  </Box>
                  使用微信登录
                </Button>
              </AnimateButton>
              <WechatModal open={openWechat} handleClose={handleWechatClose} wechatLogin={wechatLogin} qrCode={siteInfo.wechat_qrcode} />
            </Grid>
          )}
          {siteInfo.lark_client_id && (
            <Grid item xs={12}>
              <AnimateButton>
                <Button
                  disableElevation
                  fullWidth
                  onClick={() => onLarkOAuthClicked(siteInfo.lark_client_id)}
                  size="large"
                  variant="outlined"
                  sx={{
                    color: 'grey.700',
                    backgroundColor: theme.palette.grey[50],
                    borderColor: theme.palette.grey[100]
                  }}
                >
                  <Box sx={{ mr: { xs: 1, sm: 2, width: 20 }, display: 'flex', alignItems: 'center' }}>
                    <img src={Lark} alt="Lark" width={25} height={25} style={{ marginRight: matchDownSM ? 8 : 16 }} />
                  </Box>
                  使用飞书登录
                </Button>
              </AnimateButton>
            </Grid>
          )}
          {siteInfo.oidc && (
            <Grid item xs={12}>
              <AnimateButton>
                <Button
                  disableElevation
                  fullWidth
                  onClick={() => onOidcClicked(siteInfo.oidc_authorization_endpoint,siteInfo.oidc_client_id)}
                  size="large"
                  variant="outlined"
                  sx={{
                    color: 'grey.700',
                    backgroundColor: theme.palette.grey[50],
                    borderColor: theme.palette.grey[100]
                  }}
                >
                  <Box sx={{ mr: { xs: 1, sm: 2, width: 20 }, display: 'flex', alignItems: 'center' }}>
                    <img src={OIDC} alt="Lark" width={25} height={25} style={{ marginRight: matchDownSM ? 8 : 16 }} />
                  </Box>
                  使用 OIDC 登录
                </Button>
              </AnimateButton>
            </Grid>
          )}
          <Grid item xs={12}>
            <Box
              sx={{
                alignItems: 'center',
                display: 'flex'
              }}
            >
              <Divider sx={{ flexGrow: 1 }} orientation="horizontal" />

              <Button
                variant="outlined"
                sx={{
                  cursor: 'unset',
                  m: 2,
                  py: 0.5,
                  px: 7,
                  borderColor: `${theme.palette.grey[100]} !important`,
                  color: `${theme.palette.grey[900]}!important`,
                  fontWeight: 500,
                  borderRadius: `${customization.borderRadius}px`
                }}
                disableRipple
                disabled
              >
                OR
              </Button>

              <Divider sx={{ flexGrow: 1 }} orientation="horizontal" />
            </Box>
          </Grid>
        </Grid>
      )}

      <Formik
        initialValues={{
          username: '',
          password: '',
          submit: null
        }}
        validationSchema={Yup.object().shape({
          username: Yup.string().max(255).required('Username is required'),
          password: Yup.string().max(255).required('Password is required')
        })}
        onSubmit={async (values, { setErrors, setStatus, setSubmitting }) => {
          if (siteInfo.login_math_captcha && !turnstileOn) {
            if (!captchaMasterSrc) {
              showInfo(captchaLoading ? '加载验证码中…' : captchaLoadError || '请先加载验证码');
              setSubmitting(false);
              return;
            }
            if (!captchaDotNum || captchaClicks.length !== captchaDotNum) {
              showInfo('请按顺序完成图形验证码');
              setSubmitting(false);
              return;
            }
          }
          let proof = loginRequestProofRef.current;
          if (!(siteInfo.login_math_captcha && !turnstileOn)) {
            proof = null;
          }
          const captcha =
            siteInfo.login_math_captcha && !turnstileOn && captchaMasterSrc
              ? { captcha_id: captchaChallengeId, captcha_clicks: captchaClicks }
              : undefined;
          const { success, message, require2FA } = await login(
            values.username,
            values.password,
            captcha,
            proof
          );
          if (require2FA) {
            setShowTwoFA(true);
            setTwoFACode('');
            setSubmitting(false);
            return;
          }
          if (success) {
            setStatus({ success: true });
          } else {
            setStatus({ success: false });
            if (message) {
              setErrors({ submit: message });
            }
            if (siteInfo.login_math_captcha && !turnstileOn) {
              void loadLoginCaptcha();
            }
          }
          setSubmitting(false);
        }}
      >
        {({ errors, handleBlur, handleChange, handleSubmit, isSubmitting, touched, values }) => (
          <form noValidate onSubmit={handleSubmit} {...others}>
            <FormControl fullWidth error={Boolean(touched.username && errors.username)} sx={{ ...theme.typography.customInput }}>
              <InputLabel htmlFor="outlined-adornment-username-login">用户名 / 邮箱</InputLabel>
              <OutlinedInput
                id="outlined-adornment-username-login"
                type="text"
                value={values.username}
                name="username"
                onBlur={handleBlur}
                onChange={handleChange}
                label="用户名"
                inputProps={{ autoComplete: 'username' }}
              />
              {touched.username && errors.username && (
                <FormHelperText error id="standard-weight-helper-text-username-login">
                  {errors.username}
                </FormHelperText>
              )}
            </FormControl>

            <FormControl fullWidth error={Boolean(touched.password && errors.password)} sx={{ ...theme.typography.customInput }}>
              <InputLabel htmlFor="outlined-adornment-password-login">密码</InputLabel>
              <OutlinedInput
                id="outlined-adornment-password-login"
                type={showPassword ? 'text' : 'password'}
                value={values.password}
                name="password"
                onBlur={handleBlur}
                onChange={handleChange}
                endAdornment={
                  <InputAdornment position="end">
                    <IconButton
                      aria-label="toggle password visibility"
                      onClick={handleClickShowPassword}
                      onMouseDown={handleMouseDownPassword}
                      edge="end"
                      size="large"
                    >
                      {showPassword ? <Visibility /> : <VisibilityOff />}
                    </IconButton>
                  </InputAdornment>
                }
                label="Password"
              />
              {touched.password && errors.password && (
                <FormHelperText error id="standard-weight-helper-text-password-login">
                  {errors.password}
                </FormHelperText>
              )}
            </FormControl>

            {siteInfo.login_math_captcha && !turnstileOn && (
              <Box sx={{ mt: 2 }}>
                <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                  图形验证：按缩略图顺序点击主图
                </Typography>
                {captchaThumbSrc ? (
                  <Box sx={{ mb: 1 }}>
                    <img src={captchaThumbSrc} alt="" style={{ maxHeight: 56 }} />
                  </Box>
                ) : null}
                {captchaMasterSrc ? (
                  <Box sx={{ position: 'relative', display: 'inline-block' }}>
                    <img
                      src={captchaMasterSrc}
                      alt=""
                      style={{ maxWidth: '100%', cursor: 'crosshair' }}
                      onLoad={(ev) => {
                        setCaptchaMasterNaturalSize({
                          w: ev.target.naturalWidth,
                          h: ev.target.naturalHeight
                        });
                      }}
                      onClick={onMasterCaptchaClick}
                    />
                    {captchaMasterNaturalSize.w > 0 &&
                      captchaMasterNaturalSize.h > 0 &&
                      captchaClicks.map((p, i) => (
                        <Box
                          key={i}
                          sx={{
                            position: 'absolute',
                            left: `${(p.x / captchaMasterNaturalSize.w) * 100}%`,
                            top: `${(p.y / captchaMasterNaturalSize.h) * 100}%`,
                            transform: 'translate(-50%, -50%)',
                            width: 22,
                            height: 22,
                            borderRadius: '50%',
                            border: '2px solid',
                            borderColor: 'primary.main',
                            bgcolor: 'rgba(25,118,210,0.2)',
                            color: 'common.white',
                            fontSize: 12,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontWeight: 700
                          }}
                        >
                          {i + 1}
                        </Box>
                      ))}
                  </Box>
                ) : (
                  <Typography variant="caption" color="textSecondary">
                    {captchaLoading ? '加载验证码中…' : captchaLoadError || '点击刷新加载验证码'}
                  </Typography>
                )}
                <Typography variant="caption" display="block" sx={{ mt: 1 }}>
                  已点击 {captchaClicks.length}/{captchaDotNum || '—'}
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                  <Button size="small" variant="outlined" onClick={() => setCaptchaClicks([])}>
                    清除点击
                  </Button>
                  <Button size="small" variant="contained" onClick={() => void loadLoginCaptcha()}>
                    刷新题目
                  </Button>
                </Stack>
              </Box>
            )}

            <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={1}>
              {/* <FormControlLabel
                control={
                  <Checkbox checked={checked} onChange={(event) => setChecked(event.target.checked)} name="checked" color="primary" />
                }
                label="记住我"
              /> */}
              <Typography
                component={Link}
                to="/reset"
                variant="subtitle1"
                color="primary"
                sx={{ textDecoration: 'none', cursor: 'pointer' }}
              >
                忘记密码?
              </Typography>
            </Stack>
            {errors.submit && (
              <Box sx={{ mt: 3 }}>
                <FormHelperText error>{errors.submit}</FormHelperText>
              </Box>
            )}

            <Box sx={{ mt: 2 }}>
              <AnimateButton>
                <Button disableElevation disabled={isSubmitting} fullWidth size="large" type="submit" variant="contained" color="primary">
                  登录
                </Button>
              </AnimateButton>
            </Box>
          </form>
        )}
      </Formik>
      <Dialog open={showTwoFA} onClose={() => setShowTwoFA(false)} maxWidth="xs" fullWidth>
        <DialogTitle>两步验证</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ mb: 2 }}>
            请输入认证器 6 位验证码或 8 位备用码
          </Typography>
          <TextField
            fullWidth
            autoFocus
            value={twoFACode}
            onChange={(e) => setTwoFACode(e.target.value)}
            placeholder="验证码 / 备用码"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowTwoFA(false)}>取消</Button>
          <Button
            variant="contained"
            onClick={async () => {
              if (!twoFACode.trim()) {
                showWarning('请输入验证码或备用码');
                return;
              }
              const { success, message } = await verify2FALogin(twoFACode);
              if (!success && message) {
                showInfo(message);
              }
              if (success) {
                setShowTwoFA(false);
              }
            }}
          >
            确认登录
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default LoginForm;
