import React, { useContext } from 'react';
import { Navigate } from 'react-router-dom';
import { PrivateRoute } from './PrivateRoute';
import { StatusContext } from '../context/Status';
import { isNacosEnabled } from '../helpers';

const NacosPrivateRoute = ({ children }) => {
  const [statusState] = useContext(StatusContext);
  const enabled =
    statusState.status?.nacos_enabled !== false &&
    statusState.status?.nacos_enabled !== 'false' &&
    isNacosEnabled();

  if (!enabled) {
    return <Navigate to='/' replace />;
  }

  return <PrivateRoute>{children}</PrivateRoute>;
};

export default NacosPrivateRoute;
