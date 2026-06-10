import { useState, useEffect, useCallback } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { getListings } from '../api';
import ListingCard from '../components/ListingCard';
import { Spinner } from '../components/Spinner';
import { useAuth } from '../context/AuthContext';
import './HomePage.css';

const CATEGORIES = ['All', 'Electronics', 'Art', 'Fashion', 'Collectibles', 'Furniture', 'Jewelry', 'Books', 'Sports', 'Other'];

export default function HomePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { isAuthenticated } = useAuth();

  const [listings, setListings] = useState([]);
  const [meta, setMeta]         = useState({ total: 0, page: 1, pages: 1 });
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState(null);

  const search   = searchParams.get('search')   || '';
  const category = searchParams.get('category') || '';
  const type     = searchParams.get('type')     || '';
  const page     = parseInt(searchParams.get('page') || '1', 10);

  const [searchInput, setSearchInput] = useState(search);

  const fetchListings = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = { page, limit: 12 };
      if (search)   params.search   = search;
      if (category) params.category = category;
      if (type)     params.type     = type;

      const res = await getListings(params);
      setListings(res.data.listings || []);
      setMeta(res.data.meta || { total: 0, page: 1, pages: 1 });
    } catch (e) {
      setError('Failed to load listings. Is the backend running?');
    } finally {
      setLoading(false);
    }
  }, [search, category, type, page]);

  useEffect(() => { fetchListings(); }, [fetchListings]);

  const setParam = (key, val) => {
    const p = new URLSearchParams(searchParams);
    if (val) p.set(key, val); else p.delete(key);
    p.delete('page');
    setSearchParams(p);
  };

  const handleSearch = (e) => {
    e.preventDefault();
    setParam('search', searchInput.trim());
  };

  const heroVariants = {
    hidden:  { opacity: 0 },
    visible: { opacity: 1, transition: { staggerChildren: 0.12 } },
  };
  const heroItem = {
    hidden:  { opacity: 0, y: 30 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.6, ease: [0.16, 1, 0.3, 1] } },
  };

  return (
    <div className="home">
      <section className="home__hero">
        <div className="home__hero-grid" aria-hidden="true">
          {[...Array(8)].map((_, i) => (
            <div key={i} className="home__hero-gridline" style={{ '--i': i }} />
          ))}
        </div>

        <motion.div
          className="home__hero-content container"
          variants={heroVariants}
          initial="hidden"
          animate="visible"
        >
          <motion.div className="home__hero-eyebrow" variants={heroItem}>
            <span className="live-dot" />
            <span>Live marketplace · {meta.total} items</span>
          </motion.div>

          <motion.h1 className="home__hero-title" variants={heroItem}>
            Where rare things<br />
            <span className="home__hero-title-accent">find new homes</span>
          </motion.h1>

          <motion.p className="home__hero-subtitle" variants={heroItem}>
            Bid on live auctions or buy instantly. From vintage electronics to rare art 
            every item has a story waiting to continue.
          </motion.p>

          <motion.div className="home__hero-actions" variants={heroItem}>
            {isAuthenticated ? (
              <Link to="/create" className="btn btn--amber btn--lg">
                List an Item →
              </Link>
            ) : (
              <Link to="/auth?mode=register" className="btn btn--amber btn--lg">
                Start Trading →
              </Link>
            )}
            <a href="#listings" className="btn btn--ghost btn--lg">
              Browse Market
            </a>
          </motion.div>
        </motion.div>

        <div className="home__hero-fade" aria-hidden="true" />
      </section>

      <div className="home__stats">
        <div className="container">
          <div className="home__stats-inner">
            <div className="home__stat">
              <span className="home__stat-value price-display">{meta.total}</span>
              <span className="home__stat-label">Active Listings</span>
            </div>
            <div className="home__stat-divider" />
            <div className="home__stat">
              <span className="home__stat-value price-display">24/7</span>
              <span className="home__stat-label">Live Auctions</span>
            </div>
            <div className="home__stat-divider" />
            <div className="home__stat">
              <span className="home__stat-value price-display">0%</span>
              <span className="home__stat-label">Buyer Fees</span>
            </div>
            <div className="home__stat-divider" />
            <div className="home__stat">
              <span className="home__stat-value price-display">∞</span>
              <span className="home__stat-label">Categories</span>
            </div>
          </div>
        </div>
      </div>

      <section className="home__listings" id="listings">
        <div className="container">

          <div className="home__toolbar">
            <form className="home__search" onSubmit={handleSearch}>
              <span className="home__search-icon">⌕</span>
              <input
                className="home__search-input"
                type="text"
                placeholder="Search listings…"
                value={searchInput}
                onChange={e => setSearchInput(e.target.value)}
              />
              {searchInput && (
                <button
                  type="button"
                  className="home__search-clear"
                  onClick={() => { setSearchInput(''); setParam('search', ''); }}
                >×</button>
              )}
            </form>

            <div className="home__filters">
              {['', 'auction', 'fixed'].map(t => (
                <button
                  key={t}
                  className={`home__filter-btn ${type === t ? 'home__filter-btn--active' : ''}`}
                  onClick={() => setParam('type', t)}
                >
                  {t === '' ? 'All' : t === 'auction' ? '⚡ Auctions' : 'Buy Now'}
                </button>
              ))}
            </div>
          </div>

          <div className="home__categories">
            {CATEGORIES.map(cat => {
              const val = cat === 'All' ? '' : cat.toLowerCase();
              const active = (cat === 'All' && !category) || category === val;
              return (
                <button
                  key={cat}
                  className={`home__cat-btn ${active ? 'home__cat-btn--active' : ''}`}
                  onClick={() => setParam('category', val)}
                >
                  {cat}
                </button>
              );
            })}
          </div>

          {(search || category || type) && (
            <div className="home__active-filters">
              <span className="home__active-filters-label">Filtering:</span>
              {search   && <span className="home__filter-pill">"{search}" <button onClick={() => { setSearchInput(''); setParam('search', ''); }}>×</button></span>}
              {category && <span className="home__filter-pill">{category} <button onClick={() => setParam('category', '')}>×</button></span>}
              {type     && <span className="home__filter-pill">{type} <button onClick={() => setParam('type', '')}>×</button></span>}
            </div>
          )}

          {loading ? (
            <div className="home__loading">
              <Spinner size="lg" />
              <p>Loading the market…</p>
            </div>
          ) : error ? (
            <div className="home__error">
              <span className="home__error-icon">⚠</span>
              <p>{error}</p>
              <button className="btn btn--ghost btn--sm" onClick={fetchListings}>Retry</button>
            </div>
          ) : listings.length === 0 ? (
            <div className="home__empty">
              <div className="home__empty-icon">◈</div>
              <h3>Nothing here yet</h3>
              <p>Be the first to list something in this category.</p>
              {isAuthenticated && (
                <Link to="/create" className="btn btn--primary" style={{ marginTop: '16px' }}>
                  List an Item
                </Link>
              )}
            </div>
          ) : (
            <AnimatePresence mode="wait">
              <motion.div
                key={`${search}-${category}-${type}-${page}`}
                className="home__grid"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.2 }}
              >
                {listings.map((listing, i) => (
                  <ListingCard key={listing.id} listing={listing} index={i} />
                ))}
              </motion.div>
            </AnimatePresence>
          )}

          {meta.pages > 1 && (
            <div className="home__pagination">
              <button
                className="btn btn--ghost btn--sm"
                disabled={page <= 1}
                onClick={() => setParam('page', String(page - 1))}
              >
                ← Prev
              </button>
              <span className="home__pagination-info">
                <span className="price-display">{page}</span>
                <span> of {meta.pages}</span>
              </span>
              <button
                className="btn btn--ghost btn--sm"
                disabled={page >= meta.pages}
                onClick={() => setParam('page', String(page + 1))}
              >
                Next →
              </button>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
