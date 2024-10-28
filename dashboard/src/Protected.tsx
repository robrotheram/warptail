import { ReactNode } from 'react';
import { useAuth } from './context/AuthContext';
import {  useNavigate } from '@tanstack/react-router';

const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();

  if (!isAuthenticated) {
    navigate({ to: '/login' });
    return null;
  }
  return <>{children}</>
};

export default ProtectedRoute;
