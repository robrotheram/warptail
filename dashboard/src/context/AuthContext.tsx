import { createContext, useContext, useState, ReactNode, useCallback } from 'react';

interface AuthContextType {
  token: string | null;
  login: (newToken: string) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

type AuthContextMemo = () => AuthContextType 
const AuthContext = createContext< AuthContextMemo| undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }):ReactNode => {
  const [token, setToken] = useState<string | null>(sessionStorage.getItem('token'));

  const login = (newToken: string) => {
    sessionStorage.setItem('token', newToken);
    setToken(newToken);
  };

  const logout = () => {
    sessionStorage.removeItem('token');
    setToken(null);
  };

  const isAuthenticated = !!token;
  const val = useCallback(() =>({ token, login, logout, isAuthenticated }),[])
  return (
    <AuthContext.Provider value={val}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextMemo => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
