import { getProfile, User } from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { Loader2 } from 'lucide-react';
import React, { createContext, useState, useContext, useMemo, useCallback, useEffect, useRef } from 'react';

interface AuthContextType {
  token: string | null;
  user?: User;
  isAuthenticated: boolean;
  login: (newToken: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const navigateRef = useRef(navigate);
  navigateRef.current = navigate;

  const [token, setToken] = useState<string | null>(() =>
    sessionStorage.getItem('token')
  );

  const { data: user, isLoading, isFetching } = useQuery({
    queryKey: ['profile', token],
    queryFn: () => getProfile(token),
    enabled: !!token,
    retry: false,
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 10, // 10 minutes
  });

  const logout = useCallback(() => {
    sessionStorage.removeItem('token');
    setToken(null);
    queryClient.removeQueries({ queryKey: ['profile'] });
    navigateRef.current({ to: '/login' });
  }, [queryClient]);

  useEffect(() => {
    if (!token) return;

    const checkToken = () => {
      const decodedToken = parseJwt(token);
      if (decodedToken && decodedToken.exp * 1000 < Date.now()) {
        logout();
      }
    };

    checkToken();
    const interval = setInterval(checkToken, 1000 * 10);
    return () => clearInterval(interval);
  }, [token, logout]);

  const login = useCallback((newToken: string) => {
    sessionStorage.setItem('token', newToken);
    setToken(newToken);
  }, []);

  const isAuthenticated = !!user && !!token;

  const value = useMemo(
    () => ({
      token,
      user,
      isAuthenticated,
      login,
      logout,
    }),
    [token, user, isAuthenticated, login, logout]
  );

  // Only show loader on initial load when we have a token
  if (token && isLoading && !isFetching) {
    return <Loader />;
  }

  // Show loader while fetching profile for first time with token
  if (token && isLoading) {
    return <Loader />;
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};


const parseJwt = (token: string) => {
  try {
    return JSON.parse(atob(token.split(".")[1]));
  } catch (error) {
    return null;
  }
};


export const Loader = () => {
  return <div className="grid place-items-center h-screen w-full">
    <Loader2 className='animate-spin h-16 w-16' />
  </div>
}