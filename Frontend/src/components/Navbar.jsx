import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { MessageSquare, LogOut, Plus, X, Menu, Camera } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { useNotifications } from '../context/NotificationsContext';
import './Navbar.css';

export default function Navbar() {
  const { user, isAuthenticated, logout, updatePicture } = useAuth();
  const { unreadCount, unreadConversations, clearUnread } = useNotifications();
  const navigate = useNavigate();
  const [msgOpen, setMsgOpen] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [avatarUploading, setAvatarUploading] = useState(false);
  const dropdownRef = useRef(null);
  const avatarInputRef = useRef(null);

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  const handleAvatarChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarUploading(true);
    try {
      await updatePicture(file);
    } catch {
      alert('Failed to upload avatar. Please try again.');
    } finally {
      setAvatarUploading(false);
      e.target.value = '';
    }
  };

  const handleMsgClick = () => {
    setMsgOpen(prev => !prev);
  };

  useEffect(() => {
    const handler = (e) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target)) {
        setMsgOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  return (
    <header className="navbar">
      <div className="navbar__inner container">
        <Link to="/" className="navbar__logo">
          <span className="navbar__logo-mark">T</span>
          <span className="navbar__logo-text">RADEXA</span>
        </Link>

        <nav className="navbar__links">
          <Link to="/" className="navbar__link">Market</Link>
          <Link to="/auctions" className="navbar__link">
            <span className="live-dot" />
            Live Auctions
          </Link>
          <Link to="/buy-products" className="navbar__link">Buy Products</Link>
        </nav>

        <div className="navbar__actions">
          {isAuthenticated ? (
            <>
              <div className="navbar__msg-wrap" ref={dropdownRef}>
                <Link
                  to="/conversations"
                  className="navbar__msg-btn"
                  onClick={handleMsgClick}
                  aria-label="Messages"
                >
                  <MessageSquare size={20} strokeWidth={1.75} />
                  {unreadCount > 0 && (
                    <span className="navbar__msg-badge">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  )}
                </Link>

                {msgOpen && (
                  <div className="navbar__msg-dropdown">
                    <div className="navbar__msg-dropdown-header">
                      <span className="navbar__msg-dropdown-title">Messages</span>
                      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                        {unreadCount > 0 && (
                          <button 
                            className="navbar__msg-dropdown-clear"
                            onClick={() => clearUnread()}
                          >
                            Clear all
                          </button>
                        )}
                        <button
                          className="navbar__msg-dropdown-close"
                          onClick={() => setMsgOpen(false)}
                        >
                          <X size={14} />
                        </button>
                      </div>
                    </div>

                    {unreadConversations.length === 0 ? (
                      <div className="navbar__msg-empty">
                        <MessageSquare size={28} strokeWidth={1} />
                        <p>No new messages</p>
                      </div>
                    ) : (
                      <ul className="navbar__msg-list">
                        {unreadConversations.map(c => (
                          <li key={c.conversationId}>
                            <Link
                              to={`/conversations/${c.conversationId}`}
                              className="navbar__msg-item"
                              onClick={() => setMsgOpen(false)}
                            >
                              <span className="navbar__msg-item-title">{c.title}</span>
                              <span className="navbar__msg-item-count">{c.count} new</span>
                            </Link>
                          </li>
                        ))}
                      </ul>
                    )}

                    <div className="navbar__msg-dropdown-footer">
                      <Link
                        to="/my-listings"
                        className="navbar__msg-view-all"
                        onClick={() => setMsgOpen(false)}
                      >
                        View my listings
                      </Link>
                    </div>
                  </div>
                )}
              </div>

              <Link to="/create" className="btn btn--primary btn--sm">
                <Plus size={14} strokeWidth={2.5} />
                List Item
              </Link>

              <button
                className="navbar__avatar-wrap"
                onClick={() => avatarInputRef.current?.click()}
                title="Change profile picture"
              >
                {user.picture ? (
                  <img
                    src={user.picture}
                    alt={user.name}
                    className="navbar__avatar navbar__avatar--img"
                    referrerPolicy="no-referrer"
                  />
                ) : (
                  <span className="navbar__avatar">{user.name?.[0]?.toUpperCase()}</span>
                )}
                <span className="navbar__avatar-overlay">
                  {avatarUploading ? (
                    <span className="spinner spinner--sm" />
                  ) : (
                    <Camera size={12} strokeWidth={2} />
                  )}
                </span>
                <input
                  ref={avatarInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  style={{ display: 'none' }}
                  onChange={handleAvatarChange}
                />
              </button>

              <Link to="/my-listings" className="navbar__username">
                {user.name}
              </Link>

              <button className="navbar__signout" onClick={handleLogout} title="Sign out">
                <LogOut size={16} strokeWidth={1.75} />
              </button>
            </>
          ) : (
            <>
              <Link to="/auth" className="btn btn--ghost btn--sm">Sign In</Link>
              <Link to="/auth?mode=register" className="btn btn--primary btn--sm">Join</Link>
            </>
          )}
        </div>
        
        <button className="navbar__mobile-toggle" onClick={() => setMobileMenuOpen(!mobileMenuOpen)}>
          {mobileMenuOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
      </div>

      {mobileMenuOpen && (
        <nav className="navbar__mobile-menu">
          <Link to="/" className="navbar__mobile-link" onClick={() => setMobileMenuOpen(false)}>Market</Link>
          <Link to="/auctions" className="navbar__mobile-link" onClick={() => setMobileMenuOpen(false)}>
            Live Auctions
          </Link>
          <Link to="/buy-products" className="navbar__mobile-link" onClick={() => setMobileMenuOpen(false)}>Buy Products</Link>
          {!isAuthenticated && (
            <div className="navbar__mobile-auth">
              <Link to="/auth" className="btn btn--ghost" onClick={() => setMobileMenuOpen(false)}>Sign In</Link>
              <Link to="/auth?mode=register" className="btn btn--primary" onClick={() => setMobileMenuOpen(false)}>Join</Link>
            </div>
          )}
        </nav>
      )}

      <div className="navbar__accent" />
    </header>
  );
}
