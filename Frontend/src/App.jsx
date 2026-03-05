import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import { NotificationsProvider } from './context/NotificationsContext';
import Navbar from './components/Navbar';
import HomePage from './pages/HomePage';
import ListingDetailPage from './pages/ListingDetailPage';
import AuthPage from './pages/AuthPage';
import CreateListingPage from './pages/CreateListingPage';
import MyListingsPage from './pages/MyListingsPage';
import AuctionsPage from './pages/AuctionsPage';
import ConversationsPage from './pages/ConversationsPage';
import ConversationDetailPage from './pages/ConversationDetailPage';
import BuyProductsPage from './pages/BuyProductsPage';
import './App.css';

export default function App() {
  return (
    <AuthProvider>
      <NotificationsProvider>
        <BrowserRouter>
          <div className="app">
            <Navbar />
            <main className="app__main">
              <Routes>
                <Route path="/"              element={<HomePage />} />
                <Route path="/listings/:id"  element={<ListingDetailPage />} />
                <Route path="/auth"          element={<AuthPage />} />
                <Route path="/create"        element={<CreateListingPage />} />
                <Route path="/my-listings"   element={<MyListingsPage />} />
                <Route path="/auctions"      element={<AuctionsPage />} />
                <Route path="/buy-products"  element={<BuyProductsPage />} />
                <Route path="/conversations" element={<ConversationsPage />} />
                <Route path="/conversations/:conversationId" element={<ConversationDetailPage />} />
                {/* 404 */}
                <Route path="*" element={
                  <div style={{
                    minHeight: '60vh',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: '16px',
                    textAlign: 'center',
                    padding: '0 24px',
                  }}>
                    <span style={{ fontFamily: 'var(--font-mono)', fontSize: '4rem', color: 'var(--ink-muted)' }}>404</span>
                    <h2 style={{ fontFamily: 'var(--font-display)', color: 'var(--text-secondary)' }}>Page not found</h2>
                    <a href="/" className="btn btn--ghost">← Back to market</a>
                  </div>
                } />
              </Routes>
            </main>

            {/* Footer */}
            <footer className="app__footer">
              <div className="container">
                <div className="app__footer-inner">
                  <div className="app__footer-logo">
                    <span className="app__footer-logo-mark">T</span>
                    <span className="app__footer-logo-text">RADEXA</span>
                  </div>
                  <p className="app__footer-copy">
                    © {new Date().getFullYear()} Tradexa. All rights reserved.
                  </p>
                  <div className="app__footer-links">
                    <a href="/" className="app__footer-link">Market</a>
                    <a href="/auctions" className="app__footer-link">Auctions</a>
                    <a href="/buy-products" className="app__footer-link">Buy Products</a>
                    <a href="/auth" className="app__footer-link">Sign In</a>
                    <a href="/create" className="app__footer-link">List Item</a>
                  </div>
                </div>
              </div>
            </footer>
          </div>
        </BrowserRouter>
      </NotificationsProvider>
    </AuthProvider>
  );
}
