import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { motion } from 'framer-motion';
import './Navbar.css';

export default function Navbar() {
  const { user, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/');
  };

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
              <Link to="/create" className="btn btn--primary btn--sm">
                + List Item
              </Link>
              <Link to="/my-listings" className="navbar__user-btn">
                <span className="navbar__avatar">{user.name?.[0]?.toUpperCase()}</span>
                <span className="navbar__username">{user.name}</span>
              </Link>
              <button className="btn btn--ghost btn--sm" onClick={handleLogout}>
                Sign Out
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
