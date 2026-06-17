import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { GoogleLogin } from '@react-oauth/google';
import { useAuth } from '../context/AuthContext';
import { forgotPasswordSendOtp, forgotPasswordReset } from '../api';
import './AuthPage.css';

export default function AuthPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { login, register, sendOtp, googleLogin, isAuthenticated } = useAuth();

  const [mode, setMode]     = useState(searchParams.get('mode') === 'register' ? 'register' : 'login');
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState('');
  const [success, setSuccess] = useState('');

  const [loginData, setLoginData] = useState({ email: '', password: '' });
  const [regData, setRegData] = useState({ name: '', email: '', password: '', role: 'buyer' });
  const [otpSent, setOtpSent] = useState(false);
  const [otp, setOtp] = useState('');
  const [countdown, setCountdown] = useState(0);

  // Forgot password state
  const [forgotEmail, setForgotEmail] = useState('');
  const [forgotOtpSent, setForgotOtpSent] = useState(false);
  const [forgotOtp, setForgotOtp] = useState('');
  const [forgotNewPass, setForgotNewPass] = useState('');
  const [forgotConfirmPass, setForgotConfirmPass] = useState('');
  const [forgotCountdown, setForgotCountdown] = useState(0);
  const [resetDone, setResetDone] = useState(false);

  useEffect(() => {
    if (countdown > 0) {
      const timer = setTimeout(() => setCountdown(c => c - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [countdown]);

  useEffect(() => {
    if (forgotCountdown > 0) {
      const timer = setTimeout(() => setForgotCountdown(c => c - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [forgotCountdown]);

  useEffect(() => {
    if (isAuthenticated) navigate('/', { replace: true });
  }, [isAuthenticated, navigate]);

  const switchMode = (newMode) => {
    setMode(newMode);
    setError('');
    setSuccess('');
    setForgotOtpSent(false);
    setForgotOtp('');
    setForgotNewPass('');
    setForgotConfirmPass('');
    setForgotEmail('');
    setResetDone(false);
    setOtpSent(false);
    setOtp('');
  };

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

  const handleSendOtp = async (e) => {
    e?.preventDefault();
    setError('');
    setLoading(true);
    try {
      await sendOtp(regData.email);
      setOtpSent(true);
      setCountdown(60);
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to send verification code.');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setError('');
    if (!otpSent) {
      if (regData.password.length < 6) {
        setError('Password must be at least 6 characters.');
        return;
      }
      await handleSendOtp();
      return;
    }
    if (!otp || otp.length !== 6) {
      setError('Please enter a valid 6-digit OTP.');
      return;
    }
    setLoading(true);
    try {
      await register(regData.name, regData.email, regData.password, regData.role, otp);
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

  // ── Forgot password handlers ──────────────────────────────────
  const handleForgotSendOtp = async (e) => {
    e?.preventDefault();
    setError('');
    if (!forgotEmail) {
      setError('Please enter your email address.');
      return;
    }
    setLoading(true);
    try {
      await forgotPasswordSendOtp(forgotEmail);
      setForgotOtpSent(true);
      setForgotCountdown(60);
      setSuccess('Reset code sent! Check your inbox.');
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to send reset code.');
    } finally {
      setLoading(false);
    }
  };

  const handleForgotReset = async (e) => {
    e.preventDefault();
    setError('');
    if (!forgotOtp || forgotOtp.length !== 6) {
      setError('Please enter the 6-digit code from your email.');
      return;
    }
    if (forgotNewPass.length < 6) {
      setError('New password must be at least 6 characters.');
      return;
    }
    if (forgotNewPass !== forgotConfirmPass) {
      setError('Passwords do not match.');
      return;
    }
    setLoading(true);
    try {
      await forgotPasswordReset(forgotEmail, forgotOtp, forgotNewPass);
      setResetDone(true);
      setSuccess('Password reset successfully! You can now sign in.');
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to reset password.');
    } finally {
      setLoading(false);
    }
  };

  const tabVariants = {
    hidden:  { opacity: 0, x: mode === 'login' ? -16 : 16 },
    visible: { opacity: 1, x: 0 },
    exit:    { opacity: 0, x: mode === 'login' ? 16 : -16 },
  };

  const slideVariants = {
    hidden:  { opacity: 0, y: 12 },
    visible: { opacity: 1, y: 0 },
    exit:    { opacity: 0, y: -12 },
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
          {/* Tabs — hidden when in forgot mode */}
          {mode !== 'forgot' && (
            <div className="auth-card__tabs">
              <button
                className={`auth-card__tab ${mode === 'login' ? 'auth-card__tab--active' : ''}`}
                onClick={() => switchMode('login')}
              >
                Sign In
              </button>
              <button
                className={`auth-card__tab ${mode === 'register' ? 'auth-card__tab--active' : ''}`}
                onClick={() => switchMode('register')}
              >
                Create Account
              </button>
              <div
                className="auth-card__tab-indicator"
                style={{ transform: `translateX(${mode === 'login' ? '0%' : '100%'})` }}
              />
            </div>
          )}

          {/* Forgot password header */}
          {mode === 'forgot' && (
            <div className="auth-forgot__header">
              <button className="auth-forgot__back" onClick={() => switchMode('login')}>
                ← Back to Sign In
              </button>
              <h2 className="auth-forgot__title">Reset Password</h2>
              <p className="auth-forgot__subtitle">Enter your email and we'll send you a reset code</p>
            </div>
          )}

          {/* Error banner */}
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

          {/* Success banner */}
          <AnimatePresence>
            {success && (
              <motion.div
                className="auth-card__success"
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.2 }}
              >
                <span>✓</span> {success}
              </motion.div>
            )}
          </AnimatePresence>

          <AnimatePresence mode="wait">

            {/* ── LOGIN ── */}
            {mode === 'login' && (
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
                <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '1rem' }}>
                  <GoogleLogin
                    onSuccess={handleGoogleSuccess}
                    onError={handleGoogleError}
                    useOneTap
                  />
                </div>
                <div className="auth-form__divider">
                  <span>or sign in with email</span>
                </div>

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
                  <label className="form-label" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>Password</span>
                    <button
                      type="button"
                      className="auth-form__switch-btn"
                      style={{ fontSize: '0.8rem' }}
                      onClick={() => switchMode('forgot')}
                    >
                      Forgot password?
                    </button>
                  </label>
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
                <p className="auth-form__switch" style={{ marginTop: '1rem' }}>
                  No account?{' '}
                  <button type="button" className="auth-form__switch-btn" onClick={() => switchMode('register')}>
                    Create one
                  </button>
                </p>
              </motion.form>
            )}

            {/* ── REGISTER ── */}
            {mode === 'register' && (
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
                {!otpSent ? (
                  <>
                    <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '1rem' }}>
                      <GoogleLogin
                        onSuccess={handleGoogleSuccess}
                        onError={handleGoogleError}
                        text="signup_with"
                      />
                    </div>
                    <div className="auth-form__divider">
                      <span>or continue with email</span>
                    </div>

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
                        <><span className="spinner spinner--sm" /> Sending code…</>
                      ) : 'Send Verification Code →'}
                    </button>
                  </>
                ) : (
                  <>
                    <p style={{ marginBottom: '1.25rem', fontSize: '0.95rem', opacity: 0.9, lineHeight: 1.5, textAlign: 'center' }}>
                      We have sent a verification code to <strong>{regData.email}</strong>. Please enter the 6-digit OTP code below.
                    </p>
                    <div className="auth-form__field">
                      <label className="form-label" style={{ textAlign: 'center', display: 'block' }}>OTP Verification Code</label>
                      <input
                        className="form-input"
                        type="text"
                        placeholder="123456"
                        maxLength={6}
                        value={otp}
                        onChange={e => setOtp(e.target.value.replace(/\D/g, ''))}
                        required
                        autoFocus
                        style={{ textAlign: 'center', letterSpacing: '8px', fontSize: '1.5rem', fontFamily: 'monospace' }}
                      />
                    </div>
                    <div style={{ display: 'flex', gap: '1rem', marginTop: '1.5rem' }}>
                      <button
                        type="button"
                        className="btn btn--secondary"
                        onClick={() => setOtpSent(false)}
                        style={{ flex: 1, padding: '0.75rem' }}
                      >
                        ← Back
                      </button>
                      <button
                        type="submit"
                        className="btn btn--primary"
                        disabled={loading}
                        style={{ flex: 2, padding: '0.75rem' }}
                      >
                        {loading ? (
                          <><span className="spinner spinner--sm" /> Verifying…</>
                        ) : 'Verify & Register →'}
                      </button>
                    </div>
                    <div style={{ textAlign: 'center', marginTop: '1.25rem' }}>
                      {countdown > 0 ? (
                        <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>Resend code in {countdown}s</p>
                      ) : (
                        <button
                          type="button"
                          className="auth-form__switch-btn"
                          onClick={handleSendOtp}
                          disabled={loading}
                          style={{ fontSize: '0.9rem', textDecoration: 'underline' }}
                        >
                          Resend Verification Code
                        </button>
                      )}
                    </div>
                  </>
                )}
                <p className="auth-form__switch" style={{ marginTop: '1rem' }}>
                  Already have an account?{' '}
                  <button type="button" className="auth-form__switch-btn" onClick={() => switchMode('login')}>
                    Sign in
                  </button>
                </p>
              </motion.form>
            )}

            {/* ── FORGOT PASSWORD ── */}
            {mode === 'forgot' && (
              <motion.div
                key="forgot"
                className="auth-form"
                variants={slideVariants}
                initial="hidden"
                animate="visible"
                exit="exit"
                transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
              >
                {resetDone ? (
                  /* Step 3 — success */
                  <div style={{ textAlign: 'center', padding: '1rem 0' }}>
                    <div className="auth-forgot__success-icon">✓</div>
                    <h3 style={{ marginBottom: '0.5rem', fontSize: '1.1rem' }}>Password Updated!</h3>
                    <p style={{ opacity: 0.7, fontSize: '0.9rem', marginBottom: '1.5rem' }}>
                      Your password has been reset successfully.
                    </p>
                    <button
                      className="btn btn--primary auth-form__submit"
                      onClick={() => switchMode('login')}
                    >
                      Sign In with New Password →
                    </button>
                  </div>
                ) : !forgotOtpSent ? (
                  /* Step 1 — enter email */
                  <form onSubmit={handleForgotSendOtp}>
                    <div className="auth-form__field" style={{ marginBottom: '1.5rem' }}>
                      <label className="form-label">Your Email Address</label>
                      <input
                        className="form-input"
                        type="email"
                        placeholder="you@example.com"
                        value={forgotEmail}
                        onChange={e => { setForgotEmail(e.target.value); setError(''); setSuccess(''); }}
                        required
                        autoFocus
                      />
                    </div>
                    <button
                      type="submit"
                      className="btn btn--primary auth-form__submit"
                      disabled={loading}
                    >
                      {loading ? (
                        <><span className="spinner spinner--sm" /> Sending…</>
                      ) : 'Send Reset Code →'}
                    </button>
                  </form>
                ) : (
                  /* Step 2 — enter OTP + new password */
                  <form onSubmit={handleForgotReset}>
                    <p style={{ marginBottom: '1.25rem', fontSize: '0.9rem', opacity: 0.8, textAlign: 'center', lineHeight: 1.5 }}>
                      Enter the 6-digit code sent to <strong>{forgotEmail}</strong> and choose a new password.
                    </p>

                    <div className="auth-form__field">
                      <label className="form-label" style={{ textAlign: 'center', display: 'block' }}>Reset Code</label>
                      <input
                        className="form-input"
                        type="text"
                        placeholder="123456"
                        maxLength={6}
                        value={forgotOtp}
                        onChange={e => { setForgotOtp(e.target.value.replace(/\D/g, '')); setError(''); }}
                        required
                        autoFocus
                        style={{ textAlign: 'center', letterSpacing: '8px', fontSize: '1.5rem', fontFamily: 'monospace' }}
                      />
                    </div>

                    <div className="auth-form__field" style={{ marginTop: '1rem' }}>
                      <label className="form-label">New Password</label>
                      <input
                        className="form-input"
                        type="password"
                        placeholder="At least 6 characters"
                        value={forgotNewPass}
                        onChange={e => { setForgotNewPass(e.target.value); setError(''); }}
                        required
                        minLength={6}
                      />
                    </div>

                    <div className="auth-form__field" style={{ marginTop: '0.75rem' }}>
                      <label className="form-label">Confirm New Password</label>
                      <input
                        className="form-input"
                        type="password"
                        placeholder="Repeat your new password"
                        value={forgotConfirmPass}
                        onChange={e => { setForgotConfirmPass(e.target.value); setError(''); }}
                        required
                        minLength={6}
                      />
                    </div>

                    <div style={{ display: 'flex', gap: '1rem', marginTop: '1.5rem' }}>
                      <button
                        type="button"
                        className="btn btn--secondary"
                        onClick={() => { setForgotOtpSent(false); setForgotOtp(''); setError(''); setSuccess(''); }}
                        style={{ flex: 1, padding: '0.75rem' }}
                      >
                        ← Back
                      </button>
                      <button
                        type="submit"
                        className="btn btn--primary"
                        disabled={loading}
                        style={{ flex: 2, padding: '0.75rem' }}
                      >
                        {loading ? (
                          <><span className="spinner spinner--sm" /> Resetting…</>
                        ) : 'Reset Password →'}
                      </button>
                    </div>

                    <div style={{ textAlign: 'center', marginTop: '1.25rem' }}>
                      {forgotCountdown > 0 ? (
                        <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>Resend code in {forgotCountdown}s</p>
                      ) : (
                        <button
                          type="button"
                          className="auth-form__switch-btn"
                          onClick={handleForgotSendOtp}
                          disabled={loading}
                          style={{ fontSize: '0.9rem', textDecoration: 'underline' }}
                        >
                          Resend Code
                        </button>
                      )}
                    </div>
                  </form>
                )}
              </motion.div>
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
