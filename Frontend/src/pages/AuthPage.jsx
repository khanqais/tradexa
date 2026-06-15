import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { GoogleLogin } from '@react-oauth/google';
import { useAuth } from '../context/AuthContext';
import './AuthPage.css';

export default function AuthPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { login, register, googleLogin, isAuthenticated } = useAuth();

  const [mode, setMode]     = useState(searchParams.get('mode') === 'register' ? 'register' : 'login');
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState('');

  const [loginData, setLoginData] = useState({ email: '', password: '' });
  const [regData, setRegData] = useState({ name: '', email: '', password: '', role: 'buyer' });

  useEffect(() => {
    if (isAuthenticated) navigate('/', { replace: true });
  }, [isAuthenticated, navigate]);

  const handleLogin = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(loginData.email, loginData.password);
      navigate('/');
    } catch (err) {
      setError(err?.response?.data?.error || 'Login failed. Check your credentials.');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setError('');
    if (regData.password.length < 6) {
      setError('Password must be at least 6 characters.');
      return;
    }
    setLoading(true);
    try {
      await register(regData.name, regData.email, regData.password, regData.role);
      await login(regData.email, regData.password);
      navigate('/');
    } catch (err) {
      setError(err?.response?.data?.error || 'Registration failed.');
    } finally {
      setLoading(false);
    }
  };

  const handleGoogleSuccess = async (credentialResponse) => {
    setError('');
    setLoading(true);
    try {
      await googleLogin(credentialResponse.credential);
      navigate('/');
    } catch (err) {
      setError(err?.response?.data?.error || 'Google login failed.');
    } finally {
      setLoading(false);
    }
  };

  const handleGoogleError = () => {
    setError('Google Login Failed');
  };

  const tabVariants = {
    hidden:  { opacity: 0, x: mode === 'login' ? -16 : 16 },
    visible: { opacity: 1, x: 0 },
    exit:    { opacity: 0, x: mode === 'login' ? 16 : -16 },
  };

  return (
    <div className="auth-page">
      <div className="auth-page__bg" aria-hidden="true">
        <div className="auth-page__bg-circle auth-page__bg-circle--1" />
        <div className="auth-page__bg-circle auth-page__bg-circle--2" />
        <div className="auth-page__bg-grid">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="auth-page__bg-line" style={{ '--i': i }} />
          ))}
        </div>
      </div>

      <div className="auth-page__content">
        <Link to="/" className="auth-page__logo">
          <span className="auth-page__logo-mark">T</span>
          <span className="auth-page__logo-text">RADEXA</span>
        </Link>

        <motion.div
          className="auth-card"
          initial={{ opacity: 0, y: 24, scale: 0.97 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        >
          <div className="auth-card__tabs">
            <button
              className={`auth-card__tab ${mode === 'login' ? 'auth-card__tab--active' : ''}`}
              onClick={() => { setMode('login'); setError(''); }}
            >
              Sign In
            </button>
            <button
              className={`auth-card__tab ${mode === 'register' ? 'auth-card__tab--active' : ''}`}
              onClick={() => { setMode('register'); setError(''); }}
            >
              Create Account
            </button>
            <div
              className="auth-card__tab-indicator"
              style={{ transform: `translateX(${mode === 'login' ? '0%' : '100%'})` }}
            />
          </div>

          <AnimatePresence>
            {error && (
              <motion.div
                className="auth-card__error"
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.2 }}
              >
                <span>⚠</span> {error}
              </motion.div>
            )}
          </AnimatePresence>

          <AnimatePresence mode="wait">
            {mode === 'login' ? (
              <motion.form
                key="login"
                className="auth-form"
                onSubmit={handleLogin}
                variants={tabVariants}
                initial="hidden"
                animate="visible"
                exit="exit"
                transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
              >
                <div className="auth-form__field">
                  <label className="form-label">Email</label>
                  <input
                    className="form-input"
                    type="email"
                    placeholder="you@example.com"
                    value={loginData.email}
                    onChange={e => setLoginData(d => ({ ...d, email: e.target.value }))}
                    required
                    autoFocus
                  />
                </div>
                <div className="auth-form__field">
                  <label className="form-label">Password</label>
                  <input
                    className="form-input"
                    type="password"
                    placeholder="Your password"
                    value={loginData.password}
                    onChange={e => setLoginData(d => ({ ...d, password: e.target.value }))}
                    required
                    minLength={6}
                  />
                </div>
                <button
                  type="submit"
                  className="btn btn--primary auth-form__submit"
                  disabled={loading}
                >
                  {loading ? (
                    <><span className="spinner spinner--sm" /> Signing in…</>
                  ) : 'Sign In →'}
                </button>
                <div className="auth-form__divider">
                  <span>or</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '1rem' }}>
                  <GoogleLogin
                    onSuccess={handleGoogleSuccess}
                    onError={handleGoogleError}
                    useOneTap
                  />
                </div>
                <p className="auth-form__switch">
                  No account?{' '}
                  <button type="button" className="auth-form__switch-btn" onClick={() => setMode('register')}>
                    Create one
                  </button>
                </p>
              </motion.form>
            ) : (
              <motion.form
                key="register"
                className="auth-form"
                onSubmit={handleRegister}
                variants={tabVariants}
                initial="hidden"
                animate="visible"
                exit="exit"
                transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
              >
                <div className="auth-form__field">
                  <label className="form-label">Full Name</label>
                  <input
                    className="form-input"
                    type="text"
                    placeholder="Your name"
                    value={regData.name}
                    onChange={e => setRegData(d => ({ ...d, name: e.target.value }))}
                    required
                    minLength={2}
                    autoFocus
                  />
                </div>
                <div className="auth-form__field">
                  <label className="form-label">Email</label>
                  <input
                    className="form-input"
                    type="email"
                    placeholder="you@example.com"
                    value={regData.email}
                    onChange={e => setRegData(d => ({ ...d, email: e.target.value }))}
                    required
                  />
                </div>
                <div className="auth-form__field">
                  <label className="form-label">Password</label>
                  <input
                    className="form-input"
                    type="password"
                    placeholder="At least 6 characters"
                    value={regData.password}
                    onChange={e => setRegData(d => ({ ...d, password: e.target.value }))}
                    required
                    minLength={6}
                  />
                </div>
                <div className="auth-form__field">
                  <label className="form-label">I want to</label>
                  <div className="auth-role-picker">
                    {[
                      { val: 'buyer',  label: '🛒 Buy items',  desc: 'Browse & bid' },
                      { val: 'seller', label: '🏷 Sell items', desc: 'List & auction' },
                    ].map(opt => (
                      <button
                        key={opt.val}
                        type="button"
                        className={`auth-role-btn ${regData.role === opt.val ? 'auth-role-btn--active' : ''}`}
                        onClick={() => setRegData(d => ({ ...d, role: opt.val }))}
                      >
                        <span className="auth-role-btn__label">{opt.label}</span>
                        <span className="auth-role-btn__desc">{opt.desc}</span>
                      </button>
                    ))}
                  </div>
                </div>
                <button
                  type="submit"
                  className="btn btn--primary auth-form__submit"
                  disabled={loading}
                >
                  {loading ? (
                    <><span className="spinner spinner--sm" /> Creating account…</>
                  ) : 'Create Account →'}
                </button>
                <div className="auth-form__divider">
                  <span>or</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '1rem' }}>
                  <GoogleLogin
                    onSuccess={handleGoogleSuccess}
                    onError={handleGoogleError}
                    text="signup_with"
                  />
                </div>
                <p className="auth-form__switch">
                  Already have an account?{' '}
                  <button type="button" className="auth-form__switch-btn" onClick={() => setMode('login')}>
                    Sign in
                  </button>
                </p>
              </motion.form>
            )}
          </AnimatePresence>
        </motion.div>

        <p className="auth-page__footer">
          By continuing, you agree to Tradexa's terms of service.
        </p>
      </div>
    </div>
  );
}
