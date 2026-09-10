import { Redirect } from './components/Redirect';
import { ReactNode } from 'react';
import { Loader, useAuth } from './context/AuthContext';
import { useLocation } from '@tanstack/react-router';
import { RequestError } from './components/RequestError';
import { Role } from './lib/api';

export default function ProtectedRoute({ children, admin = false }: { children: ReactNode; admin?: boolean }) {
  const { isAuthenticated, isLoading, isLoggingOut, requiresPasswordReset, error, retry, user } = useAuth();
  const location = useLocation();
  if (isLoading || isLoggingOut) return <Loader label={isLoggingOut ? 'Signing out…' : 'Checking your session…'} />;
  if (error) return <RequestError title="Unable to check your session" error={error} retry={retry} />;
  if (!isAuthenticated) return <Redirect to="/login" next={location.href} />;
  if (requiresPasswordReset) return <Redirect to="/password-reset" next={location.href} />;
  if (admin && user?.role !== Role.ADMIN) return <div role="alert" className="p-8">You need administrator access to view this page.</div>;
  return children;
}
