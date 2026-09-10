import { getProfile, isUnauthorized, logout as endSession, onSessionExpired, token as credentials, User } from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader2 } from 'lucide-react';
import { createContext, useState, useContext, useMemo, useCallback, useEffect, useRef, ReactNode } from 'react';
import { toast } from 'sonner';

interface AuthContextType {
  user?: User;
  isAuthenticated: boolean;
  isLoading: boolean;
  isLoggingOut: boolean;
  error: unknown;
  requiresPasswordReset: boolean;
  login: (token: string) => Promise<void>;
  logout: () => Promise<void>;
  retry: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);
const privateQuery = { predicate: (query: { queryKey: readonly unknown[] }) => query.queryKey[0] !== 'config' };

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [token, setToken] = useState(credentials.get);
  const [session, setSession] = useState(0);
  const [isLoggingOut, setLoggingOut] = useState(false);
  const changing = useRef(false);
  const generation = useRef(0);
  const channel = useRef<BroadcastChannel | null>(null);

  const clearSession = useCallback(() => {
    generation.current++;
    credentials.remove();
    setToken(null);
    // Cancel before removal so old requests cannot repopulate another account's cache.
    void queryClient.cancelQueries(privateQuery);
    queryClient.removeQueries(privateQuery);
    queryClient.getMutationCache().clear();
    setSession(value => value + 1);
  }, [queryClient]);

  useEffect(() => onSessionExpired(clearSession), [clearSession]);
  useEffect(() => {
    if (typeof BroadcastChannel === 'undefined') return;
    const updates = new BroadcastChannel('warptail-session');
    channel.current = updates;
    updates.onmessage = event => { if (event.data === 'logout') clearSession(); };
    return () => { updates.close(); channel.current = null; };
  }, [clearSession]);

  const profile = useQuery({
    // Never put bearer tokens in query keys or devtools.
    queryKey: ['profile', session],
    queryFn: ({ signal }) => getProfile(token, signal),
    enabled: !!token && !isLoggingOut,
    retry: false,
    staleTime: 60_000,
    refetchOnWindowFocus: true,
    refetchInterval: token ? 60_000 : false,
  });

  useEffect(() => {
    if (token && isUnauthorized(profile.error)) clearSession();
  }, [token, profile.error, clearSession]);

  const login = useCallback(async (newToken: string) => {
    if (changing.current || !newToken || credentials.get() === newToken) return;
    clearSession();
    credentials.set(newToken);
    setToken(newToken);
  }, [clearSession]);

  const logout = useCallback(async () => {
    if (changing.current) return;
    changing.current = true;
    setLoggingOut(true);
    const currentGeneration = generation.current;
    await queryClient.cancelQueries(privateQuery);
    try {
      await endSession();
      if (generation.current === currentGeneration) {
        channel.current?.postMessage('logout');
        clearSession();
      }
    } catch {
      toast.error('Could not end your server session. Please try signing out again.');
    } finally {
      changing.current = false;
      setLoggingOut(false);
    }
  }, [queryClient, clearSession]);

  const { refetch } = profile;
  const user = token ? profile.data : undefined;
  const value = useMemo(() => ({
    user,
    isAuthenticated: !!user && !!token,
    isLoading: !!token && profile.isPending,
    isLoggingOut,
    error: token && !user ? profile.error : null,
    requiresPasswordReset: !!user?.password_reset,
    login, logout,
    retry: () => { void refetch(); },
  }), [user, token, profile.isPending, profile.error, refetch, isLoggingOut, login, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within an AuthProvider');
  return context;
}

export function Loader({ label = 'Loading…' }: { label?: string }) {
  return <div role="status" className="flex min-h-48 w-full items-center justify-center gap-3 p-8 text-muted-foreground">
    <Loader2 aria-hidden="true" className="h-5 w-5 animate-spin" />{label}
  </div>;
}
