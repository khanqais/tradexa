import { createContext, useContext, useState, useEffect, useCallback } from 'react';

import { login as apiLogin, register as apiRegister, sendOtp as apiSendOtp, loginWithGoogle as apiLoginWithGoogle, uploadAvatar as apiUploadAvatar } from '../api';

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

  const googleLogin = useCallback(async (googleToken) => {
    const res = await apiLoginWithGoogle(googleToken);
    const { token: t, user: u } = res.data;
    localStorage.setItem('tradexa_token', t);
    localStorage.setItem('tradexa_user', JSON.stringify(u));
    setToken(t);
    setUser(u);
    return u;
  }, []);

  const register = useCallback(async (name, email, password, role, otp) => {
    const res = await apiRegister({ name, email, password, role, otp });
    return res.data.user;
  }, []);

  const sendOtp = useCallback(async (email) => {
    await apiSendOtp(email);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('tradexa_token');
    localStorage.removeItem('tradexa_user');
    setToken(null);
    setUser(null);
  }, []);

  const updatePicture = async (file) => {
    const res = await apiUploadAvatar(file);
    const picture = res.data.picture;
    const updatedUser = { ...JSON.parse(localStorage.getItem('tradexa_user') || '{}'), picture };
    localStorage.setItem('tradexa_user', JSON.stringify(updatedUser));
    setUser(updatedUser);
    return picture;
  };

  const isAuthenticated = !!token && !!user && !isTokenExpired(token);

  return (
    <AuthContext.Provider value={{ user, token, loading, isAuthenticated, login, register, sendOtp, logout, googleLogin, updatePicture }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
