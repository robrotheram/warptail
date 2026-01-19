import { ReactNode, useEffect } from 'react';
import { useAuth } from './context/AuthContext';
import { useNavigate } from '@tanstack/react-router';

const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate({ to: '/login' });
    }
  }, [isAuthenticated, navigate]);

  if (!isAuthenticated) {
    return null;
  }
  
  return <>{children}</>;
};

export default ProtectedRoute;
