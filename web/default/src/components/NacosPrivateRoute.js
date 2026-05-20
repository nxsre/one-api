import React, { useContext } from 'react';
import { Navigate } from 'react-router-dom';
import { PrivateRoute } from './PrivateRoute';
import { StatusContext } from '../context/Status';
import {
  hasTenantPermission,
  isAdmin,
  isNacosEnabled,
  isTenantAdmin,
  isTenantConsoleDelegate,
} from '../helpers';

const NacosPrivateRoute = ({ children }) => {
  const [statusState] = useContext(StatusContext);
  const enabled =
    statusState.status?.nacos_enabled !== false &&
    statusState.status?.nacos_enabled !== 'false' &&
    isNacosEnabled();

  if (!enabled) {
    return <Navigate to='/' replace />;
  }

  return (
    <PrivateRoute>
      <NacosRoleGate>{children}</NacosRoleGate>
    </PrivateRoute>
  );
};

/** 平台管理员、租户管理员，或具备 manage_nacos 的委派子账号可访问 Nacos 页面 */
function NacosRoleGate({ children }) {
  if (isAdmin() || isTenantAdmin()) {
    return children;
  }
  if (isTenantConsoleDelegate() && hasTenantPermission('manage_nacos')) {
    return children;
  }
  return <Navigate to='/' replace />;
}

export default NacosPrivateRoute;
