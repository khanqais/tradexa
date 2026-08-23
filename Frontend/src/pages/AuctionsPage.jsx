import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { api } from '../api';
import ListingCard from '../components/ListingCard';
import { Spinner } from '../components/Spinner';
import '../components/ListingCard.css';
import './ListingsPage.css';

const PROXY_RULES = [
  {
    title: 'Automatic Bidding',
    body: 'Enter your maximum bid — the most you\'re willing to pay. The system bids on your behalf, only as much as needed to stay ahead.',
  },
  {
    title: '$5 Minimum Increment',
    body: 'Each automatic counter-bid raises the price by at least $5 above the competing bid.',
  },
  {
    title: 'One Active Proxy Per Listing',
    body: 'Only one proxy bid is active at a time. If a challenger\'s max beats yours, they become the new leader.',
  },
  {
    title: 'Upgrade Your Proxy',
    body: 'If you\'re already the top bidder, you can raise your max bid to a higher amount at any time.',
  },
  {
    title: 'No Downgrades',
    body: 'You cannot lower a max bid once placed. A new max must be higher than your current maximum.',
  },
  {
    title: 'No Self-Bidding',
    body: 'You cannot place a bid on your own auction listing.',
  },
  {
    title: 'Expired Auctions',
    body: 'Bids placed after an auction\'s end time are rejected automatically.',
  },
];

function ProxyRulesPanel() {
  const [open, setOpen] = useState(false);

  return (
    <section className="proxy-rules">
      <div className="container">
        <button
          className="proxy-rules__toggle"
          onClick={() => setOpen(o => !o)}
          aria-expanded={open}
        >
          <span className="proxy-rules__toggle-label">
            <span className="proxy-rules__icon">&#9881;</span>
            How Proxy Bidding Works
          </span>
          <span className="proxy-rules__chevron" style={{ transform: open ? 'rotate(180deg)' : 'rotate(0deg)' }}>&#8964;</span>
        </button>
        <AnimatePresence initial={false}>
          {open && (
            <motion.div
              key="rules"
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.32, ease: [0.4, 0, 0.2, 1] }}
              style={{ overflow: 'hidden' }}
            >
              <div className="proxy-rules__body">
                <p className="proxy-rules__intro">
                  Tradexa uses <strong>eBay-style proxy bidding</strong>. You set a max price; the system automatically outbids competitors on your behalf, only spending what's necessary.
                </p>
                <div className="proxy-rules__grid">
                  {PROXY_RULES.map((rule) => (
                    <div key={rule.title} className="proxy-rule-card">
                      <span className="proxy-rule-card__title">{rule.title}</span>
                      <span className="proxy-rule-card__body">{rule.body}</span>
                    </div>
                  ))}
                </div>
                <div className="proxy-rules__example">
                  <span className="proxy-rules__example-label">Example</span>
                  <div className="proxy-rules__example-steps">
                    <div className="proxy-step">
                      <span className="proxy-step__num">1</span>
                      <span className="proxy-step__text">Listing starts at <strong>$100</strong>. You set max <strong>$200</strong> — you lead at $100, your true max stays hidden.</span>
                    </div>
                    <div className="proxy-step">
                      <span className="proxy-step__num">2</span>
                      <span className="proxy-step__text">A challenger bids max <strong>$150</strong> — your proxy auto-counters at <strong>$155</strong>. You stay in the lead.</span>
                    </div>
                    <div className="proxy-step">
                      <span className="proxy-step__num">3</span>
                      <span className="proxy-step__text">Another bidder tops you with max <strong>$300</strong> — they win at <strong>$205</strong>, not $300. You're notified to re-bid.</span>
                    </div>
                  </div>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </section>
  );
}

export default function AuctionsPage() {
  const [listings, setListings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    search: '',
    category: '',
    sortBy: 'newest',
    page: 1,
    limit: 12
  });

  useEffect(() => {
    loadListings();
  }, [filters]);

  const loadListings = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const params = new URLSearchParams({
        ...filters,
        type: 'auction'
      });
      
      const response = await api.get(`/listings?${params}`);
      setListings(response.data.listings ?? []);
    } catch (err) {
      setError('Failed to load auction listings');
      console.error('Error loading auction listings:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (key, value) => {
    setFilters(prev => ({
      ...prev,
      [key]: value,
      page: 1
    }));
  };

  if (loading && listings.length === 0) {
    return (
      <div className="page page--loading">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="page">
     
      <section className="hero">
        <div className="hero__content">
          <motion.h1 
            className="hero__title"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
          >
            Live Auctions
          </motion.h1>
          <motion.p 
            className="hero__subtitle"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
          >
            Bid on unique items in real-time auctions. Find your next treasure.
          </motion.p>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
          >
            <input
              type="text"
              placeholder="Search auctions..."
              className="input input--large"
              value={filters.search}
              onChange={(e) => handleFilterChange('search', e.target.value)}
            />
          </motion.div>
        </div>
      </section>

      <section className="stats-strip">
        <div className="container">
          <div className="stats-strip__content">
            <div className="stat">
              <span className="stat__value">{listings.length}</span>
              <span className="stat__label">Live Auctions</span>
            </div>
            <div className="stat">
              <span className="stat__value">24/7</span>
              <span className="stat__label">Bidding Active</span>
            </div>
            <div className="stat">
              <span className="stat__value">100%</span>
              <span className="stat__label">Secure</span>
            </div>
          </div>
        </div>
      </section>

      <ProxyRulesPanel />

      
      <section className="filters">
        <div className="container">
          <div className="filters__content">
            <select
              className="select"
              value={filters.category}
              onChange={(e) => handleFilterChange('category', e.target.value)}
            >
              <option value="">All Categories</option>
              <option value="art">Art</option>
              <option value="electronics">Electronics</option>
              <option value="fashion">Fashion</option>
              <option value="collectibles">Collectibles</option>
              <option value="vehicles">Vehicles</option>
              <option value="real-estate">Real Estate</option>
            </select>
            <select
              className="select"
              value={filters.sortBy}
              onChange={(e) => handleFilterChange('sortBy', e.target.value)}
            >
              <option value="newest">Newest First</option>
              <option value="oldest">Oldest First</option>
              <option value="price-low">Price: Low to High</option>
              <option value="price-high">Price: High to Low</option>
              <option value="bids">Most Bids</option>
            </select>
          </div>
        </div>
      </section>

      
      <section className="listings-grid">
        <div className="container">
          {error && (
            <div className="error">
              <p>{error}</p>
              <button onClick={loadListings} className="btn btn--primary">Retry</button>
            </div>
          )}
          
          {listings.length === 0 ? (
            <div className="empty-state">
              <h3>No auctions found</h3>
              <p>Try adjusting your search criteria or check back later for new auctions.</p>
              <Link to="/" className="btn btn--primary">Browse All Items</Link>
            </div>
          ) : (
            <motion.div 
              className="listings-grid__content"
              layout
            >
              {listings.map((listing, index) => (
                <motion.div
                  key={listing.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.4, delay: index * 0.05 }}
                >
                  <ListingCard listing={listing} />
                </motion.div>
              ))}
            </motion.div>
          )}
        </div>
      </section>
    </div>
  );
}