import { Navigate } from 'react-router-dom';
import {
  ROLE_TENANT_ADMIN,
  isAdmin,
  isTenantConsoleDelegate,
} from '../helpers/utils';
import { PrivateRoute } from './PrivateRoute';

/** 租户管理员、委派子账号或平台管理员（代管）可访问租户控制台 UI */
export function TenantConsoleRoute({ children }) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    return <Navigate to='/login' replace />;
  }
  try {
    const u = JSON.parse(raw);
    if (Number(u.role) === ROLE_TENANT_ADMIN) {
      return children;
    }
    if (isAdmin()) {
      return children;
    }
    if (isTenantConsoleDelegate()) {
      return children;
    }
  } catch {
    return <Navigate to='/login' replace />;
  }
  return <Navigate to='/' replace />;
}

/** 租户管理员禁止访问平台控制台页（保留兼容导出） */
export function BlockTenantAdminRoute({ children }) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    return <Navigate to='/login' replace />;
  }
  try {
    const u = JSON.parse(raw);
    if (Number(u.role) === ROLE_TENANT_ADMIN) {
      return <Navigate to='/tenant-console' replace />;
    }
  } catch {
    return <Navigate to='/login' replace />;
  }
  return children;
}

/** 租户管理员或租户委派子账号禁止访问平台管理页 */
function BlockTenantScopedRoute({ children }) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    return children;
  }
  try {
    const u = JSON.parse(raw);
    if (Number(u.role) === ROLE_TENANT_ADMIN) {
      return <Navigate to='/tenant-console' replace />;
    }
    if (isTenantConsoleDelegate()) {
      return <Navigate to='/tenant-console' replace />;
    }
  } catch {
    return <Navigate to='/login' replace />;
  }
  return children;
}

/** 已登录且仅限平台侧用户（租户管理员与委派子账号退回租户控制台） */
export function PrivatePlatformRoute({ children }) {
  return (
    <PrivateRoute>
      <BlockTenantScopedRoute>{children}</BlockTenantScopedRoute>
    </PrivateRoute>
  );
}
