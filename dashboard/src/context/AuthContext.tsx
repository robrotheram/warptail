import { createContext, useContext, useState, ReactNode } from 'react';

interface AuthContextType {
  token: string | null;
  login: (newToken: string) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext< AuthContextType| undefined>(undefined);

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
  const val = { token, login, logout, isAuthenticated }
  return (
    <AuthContext.Provider value={val}>
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
