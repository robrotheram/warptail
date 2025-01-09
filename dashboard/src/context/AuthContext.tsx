import { getProfile, User } from '@/lib/api';
import { LoginPage } from '@/LoginPage';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { Loader2 } from 'lucide-react';
import React, { createContext, useState, useContext, useMemo, useCallback, useEffect } from 'react';

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
  const queryClient = useQueryClient()

  const [token, setToken] = useState<string | null>(() =>
    sessionStorage.getItem('token')
  );
  const { data: user, isLoading } = useQuery({
    queryKey: ['profile'],
    queryFn: () => getProfile(token),
    retry: false
  })


  useEffect(() => {
    const checkToken = () => {
      if (token) {
        const decodedToken = parseJwt(token);
        if (decodedToken.exp * 1000 < Date.now()) {
          logout();
        }
      }
    };
    const interval = setInterval(checkToken, 1000 * 10);
    checkToken();
    return () => clearInterval(interval);
  }, [token, navigate]);


  const login = useCallback((newToken: string) => {
    sessionStorage.setItem('token', newToken);
    setToken(newToken);
    queryClient.invalidateQueries({ queryKey: ['profile'] });

  }, []);

  const logout = useCallback(() => {
    sessionStorage.removeItem('token');
    setToken(null);
    navigate({ to: '/login' })
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

  if (isLoading) {
    return <Loader/>
  }
  return (
    <AuthContext.Provider value={value}>
      {isAuthenticated ? children : <LoginPage />}
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