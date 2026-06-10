import { createContext, useContext, useState, useEffect, useCallback } from 'react';

import { login as apiLogin, register as apiRegister } from '../api';

const AuthContext = createContext(null);

function isTokenExpired(token) {
  if (!token) return true;
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.exp * 1000 < Date.now();
  } catch {
    return true;
  }
}

export function AuthProvider({ children }) {
  const [user, setUser]   = useState(null);
  const [token, setToken] = useState(() => {
    const t = localStorage.getItem('tradexa_token');
    if (isTokenExpired(t)) {
      localStorage.removeItem('tradexa_token');
      localStorage.removeItem('tradexa_user');
      return null;
    }
    return t;
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const storedUser = localStorage.getItem('tradexa_user');
    if (storedUser && token) {
      try { setUser(JSON.parse(storedUser)); } catch { /* ignore */ }
    }
    setLoading(false);
  }, [token]);

  const login = useCallback(async (email, password) => {
    const res = await apiLogin({ email, password });
    const { token: t, user: u } = res.data;
    localStorage.setItem('tradexa_token', t);
    localStorage.setItem('tradexa_user', JSON.stringify(u));
    setToken(t);
    setUser(u);
    return u;
  }, []);

  const register = useCallback(async (name, email, password, role) => {
    const res = await apiRegister({ name, email, password, role });
    return res.data.user;
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('tradexa_token');
    localStorage.removeItem('tradexa_user');
    setToken(null);
    setUser(null);
  }, []);

  const isAuthenticated = !!token && !!user && !isTokenExpired(token);

  return (
    <AuthContext.Provider value={{ user, token, loading, isAuthenticated, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
