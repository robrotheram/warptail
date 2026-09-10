import { useEffect, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Loader } from '@/context/AuthContext';

// Keep one navigation per destination while a lazy route is loading.
export function Redirect({ to, next }: { to: string; next?: string }) {
  const navigate = useNavigate();
  const [returnTo] = useState(next);
  useEffect(() => {
    const destination = new URL(to, window.location.origin);
    if (returnTo) destination.searchParams.set('next', returnTo);
    void navigate({ href: destination.pathname + destination.search + destination.hash, replace: true });
  }, [navigate, to, returnTo]);
  return <Loader label="Redirecting…" />;
}
