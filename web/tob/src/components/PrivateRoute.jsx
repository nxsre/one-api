import { Navigate, useLocation } from 'react-router-dom';
import { getStoredUser } from '@/lib/auth';

export default function PrivateRoute({ children }) {
  const location = useLocation();
  const user = getStoredUser();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return children;
}
