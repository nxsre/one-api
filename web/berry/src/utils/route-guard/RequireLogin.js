import { Navigate } from 'react-router-dom';

const RequireLogin = ({ children }) => {
  if (!localStorage.getItem('user')) {
    return <Navigate to="/login" replace />;
  }
  return children;
};

export default RequireLogin;
