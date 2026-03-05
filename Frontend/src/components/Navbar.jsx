import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { MessageSquare, LogOut, Plus, X } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { useNotifications } from '../context/NotificationsContext';
import './Navbar.css';

export default function Navbar() {
  const { user, isAuthenticated, logout } = useAuth();
  const { unreadCount, unreadListings, clearUnread } = useNotifications();
  const navigate = useNavigate();
  const [msgOpen, setMsgOpen] = useState(false);
  const dropdownRef = useRef(null);

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  const handleMsgClick = () => {
    setMsgOpen(prev => !prev);
  };

  // Close dropdown on outside click
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
        {/* Logo */}
        <Link to="/" className="navbar__logo">
          <span className="navbar__logo-mark">T</span>
          <span className="navbar__logo-text">RADEXA</span>
        </Link>

        {/* Center nav links */}
        <nav className="navbar__links">
          <Link to="/" className="navbar__link">Market</Link>
          <Link to="/auctions" className="navbar__link">
            <span className="live-dot" />
            Live Auctions
          </Link>
          <Link to="/buy-products" className="navbar__link">Buy Products</Link>
        </nav>

        {/* Right actions */}
        <div className="navbar__actions">
          {isAuthenticated ? (
            <>
              {/* Messages icon with badge */}
              <div className="navbar__msg-wrap" ref={dropdownRef}>
                <button
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
                </button>

                {/* Dropdown */}
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

                    {unreadListings.length === 0 ? (
                      <div className="navbar__msg-empty">
                        <MessageSquare size={28} strokeWidth={1} />
                        <p>No new messages</p>
                      </div>
                    ) : (
                      <ul className="navbar__msg-list">
                        {unreadListings.map(l => (
                          <li key={l.id}>
                            <Link
                              to={`/listings/${l.id}`}
                              className="navbar__msg-item"
                              onClick={() => setMsgOpen(false)}
                            >
                              <span className="navbar__msg-item-title">{l.title}</span>
                              <span className="navbar__msg-item-count">{l.count} new</span>
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

              <Link to="/my-listings" className="navbar__user-btn">
                <span className="navbar__avatar">{user.name?.[0]?.toUpperCase()}</span>
                <span className="navbar__username">{user.name}</span>
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
      </div>

      {/* Bottom accent line */}
      <div className="navbar__accent" />
    </header>
  );
}
