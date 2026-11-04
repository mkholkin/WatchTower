import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import * as authApi from '../api/auth';

interface User {
  login: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (login: string, password: string) => Promise<string | null>;
  verifyLogin: (login: string, challenge: string, code: string) => Promise<boolean>;
  rotationToken: string;
  register: (login: string, password: string, email?: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [rotationToken, setRotationToken] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    try {
      const token = localStorage.getItem('access_token');
      if (token) {
        const encoded = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
        const payload = JSON.parse(decodeURIComponent(Array.from(atob(encoded), c => '%' + c.charCodeAt(0).toString(16).padStart(2, '0')).join('')));
        if (payload.exp * 1000 <= Date.now() || (authApi.mfaEnabled && (!payload.mfa || payload.scope !== 'access'))) throw new Error('Expired session');
        setUser({ login: payload.sub || payload.login || 'user' });
      }
    } catch {
      localStorage.removeItem('access_token');
    } finally {
      setLoading(false);
    }
  }, []);

  const loginFn = useCallback(async (login: string, password: string) => {
    localStorage.removeItem('access_token');
    setUser(null);
    setRotationToken('');
    const res = await authApi.login(login, password);
    if (authApi.mfaEnabled) return res.data.challenge_id as string;
    localStorage.setItem('access_token', res.data.access_token);
    setUser({ login });
    return null;
  }, []);

  const verifyLogin = useCallback(async (login: string, challenge: string, code: string) => {
    const res = await authApi.verify(challenge, code);
    if (res.data.password_change_required) { setRotationToken(res.data.access_token); return true; }
    localStorage.setItem('access_token', res.data.access_token);
    setUser({ login });
    return false;
  }, []);

  const registerFn = useCallback(async (login: string, password: string, email?: string) => {
    await authApi.register(login, password, email);
    if (!authApi.mfaEnabled) await loginFn(login, password);
  }, [loginFn]);

  const logout = useCallback(() => {
    localStorage.removeItem('access_token');
    setUser(null);
    setRotationToken('');
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, rotationToken, verifyLogin, login: loginFn, register: registerFn, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
