import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import * as authApi from '../api/auth';

interface User {
  login: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (login: string, password: string) => Promise<void>;
  register: (login: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    try {
      const token = localStorage.getItem('access_token');
      if (token) {
        const payload = JSON.parse(atob(token.split('.')[1]));
        setUser({ login: payload.sub || payload.login || 'user' });
      }
    } catch {
      localStorage.removeItem('access_token');
    } finally {
      setLoading(false);
    }
  }, []);

  const loginFn = useCallback(async (login: string, password: string) => {
    const res = await authApi.login(login, password);
    localStorage.setItem('access_token', res.data.access_token);
    setUser({ login });
  }, []);

  const registerFn = useCallback(async (login: string, password: string) => {
    await authApi.register(login, password);
    return loginFn(login, password);
  }, [loginFn]);

  const logout = useCallback(() => {
    localStorage.removeItem('access_token');
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login: loginFn, register: registerFn, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
