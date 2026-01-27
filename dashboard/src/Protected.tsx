import { ReactNode, useEffect } from 'react';
import { useAuth } from './context/AuthContext';
import { useNavigate } from '@tanstack/react-router';

const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const { isAuthenticated, requiresPasswordReset } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate({ to: '/login' });
    } else if (requiresPasswordReset) {
      navigate({ to: '/password-reset' });
    }
  }, [isAuthenticated, requiresPasswordReset, navigate]);

  if (!isAuthenticated) {
    return null;
  }

  if (requiresPasswordReset) {
    return null;
  }
  
  return <>{children}</>;
};

export default ProtectedRoute;
