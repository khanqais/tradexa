import { useState, useEffect, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { getListings, deleteListing } from '../api';
import { useAuth } from '../context/AuthContext';
import { Spinner } from '../components/Spinner';
import './MyListingsPage.css';

function formatPrice(p) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD',
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).format(p);
}

export default function MyListingsPage() {
  const { user, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  const [listings, setListings] = useState([]);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState('');
  const [deletingId, setDeletingId] = useState(null);

  const fetchMyListings = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      // Fetch all listings — we filter client-side by seller_id
      // (Backend doesn't have a /my-listings endpoint, so we fetch all and filter)
      const res = await getListings({ limit: 50, sold: 'false' });
      const all = res.data.listings || [];
      // Also fetch sold ones
      const soldRes = await getListings({ limit: 50, sold: 'true' });
      const sold = soldRes.data.listings || [];
      const mine = [...all, ...sold].filter(l => l.seller_id === user?.id);
      setListings(mine);
    } catch {
      setError('Failed to load your listings.');
    } finally {
      setLoading(false);
    }
  }, [user]);

  useEffect(() => {
    if (!isAuthenticated) { navigate('/auth'); return; }
    fetchMyListings();
  }, [isAuthenticated, fetchMyListings, navigate]);

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this listing permanently?')) return;
    setDeletingId(id);
    try {
      await deleteListing(id);
      setListings(prev => prev.filter(l => l.id !== id));
    } catch (e) {
      alert(e?.response?.data?.error || 'Delete failed.');
    } finally {
      setDeletingId(null);
    }
  };

  const activeListings = listings.filter(l => !l.is_sold);
  const soldListings   = listings.filter(l => l.is_sold);

  if (loading) {
    return (
      <div className="mylistings-loading">
        <Spinner size="lg" />
        <p>Loading your listings…</p>
      </div>
    );
  }

  return (
    <div className="mylistings container">
      {/* Header */}
      <motion.div
        className="mylistings__header"
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
      >
        <div>
          <h1 className="mylistings__title">My Listings</h1>
          <p className="mylistings__subtitle">
            {listings.length === 0
              ? "You haven't listed anything yet."
              : `${activeListings.length} active · ${soldListings.length} sold`}
          </p>
        </div>
        <Link to="/create" className="btn btn--amber">+ New Listing</Link>
      </motion.div>

      {error && (
        <div className="mylistings__error">
          <span>⚠</span> {error}
          <button className="btn btn--ghost btn--sm" onClick={fetchMyListings}>Retry</button>
        </div>
      )}

      {listings.length === 0 && !error ? (
        <motion.div
          className="mylistings__empty"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.2 }}
        >
          <div className="mylistings__empty-icon">◈</div>
          <h3>No listings yet</h3>
          <p>List your first item and start trading on Tradexa.</p>
          <Link to="/create" className="btn btn--primary" style={{ marginTop: '16px' }}>
            List an Item
          </Link>
        </motion.div>
      ) : (
        <>
          {/* Active listings */}
          {activeListings.length > 0 && (
            <section className="mylistings__section">
              <h2 className="mylistings__section-title">
                <span className="live-dot" />
                Active
                <span className="mylistings__section-count">{activeListings.length}</span>
              </h2>
              <div className="mylistings__table">
                <div className="mylistings__table-head">
                  <span>Item</span>
                  <span>Type</span>
                  <span>Price</span>
                  <span>Created</span>
                  <span>Actions</span>
                </div>
                <AnimatePresence>
                  {activeListings.map((listing, i) => (
                    <motion.div
                      key={listing.id}
                      className="mylistings__row"
                      initial={{ opacity: 0, x: -16 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0, x: 16, height: 0 }}
                      transition={{ duration: 0.3, delay: i * 0.04, ease: [0.16, 1, 0.3, 1] }}
                    >
                      {/* Item */}
                      <div className="mylistings__row-item">
                        <div className="mylistings__row-img">
                          {listing.images?.[0]?.url || listing.image_url ? (
                            <img src={listing.images?.[0]?.url || listing.image_url} alt={listing.title} />
                          ) : (
                            <span className="mylistings__row-img-placeholder">◈</span>
                          )}
                        </div>
                        <div className="mylistings__row-info">
                          <Link to={`/listings/${listing.id}`} className="mylistings__row-title">
                            {listing.title}
                          </Link>
                          {listing.category && (
                            <span className="mylistings__row-cat">{listing.category}</span>
                          )}
                        </div>
                      </div>

                      {/* Type */}
                      <div>
                        {listing.type === 'auction'
                          ? <span className="tag tag--auction">⚡ Auction</span>
                          : <span className="tag tag--fixed">Buy Now</span>
                        }
                      </div>

                      {/* Price */}
                      <div className="price-display" style={{ fontSize: '1rem' }}>
                        {formatPrice(listing.price)}
                      </div>

                      {/* Date */}
                      <div className="mylistings__row-date">
                        {new Date(listing.created_at).toLocaleDateString('en-US', {
                          month: 'short', day: 'numeric', year: 'numeric',
                        })}
                      </div>

                      {/* Actions */}
                      <div className="mylistings__row-actions">
                        <Link
                          to={`/listings/${listing.id}`}
                          className="btn btn--ghost btn--sm"
                        >
                          View
                        </Link>
                        <button
                          className="btn btn--danger btn--sm"
                          onClick={() => handleDelete(listing.id)}
                          disabled={deletingId === listing.id}
                        >
                          {deletingId === listing.id
                            ? <Spinner size="sm" />
                            : '✕'}
                        </button>
                      </div>
                    </motion.div>
                  ))}
                </AnimatePresence>
              </div>
            </section>
          )}

          {/* Sold listings */}
          {soldListings.length > 0 && (
            <section className="mylistings__section">
              <h2 className="mylistings__section-title">
                Sold
                <span className="mylistings__section-count">{soldListings.length}</span>
              </h2>
              <div className="mylistings__table">
                <div className="mylistings__table-head">
                  <span>Item</span>
                  <span>Type</span>
                  <span>Price</span>
                  <span>Created</span>
                  <span>Actions</span>
                </div>
                {soldListings.map((listing, i) => (
                  <motion.div
                    key={listing.id}
                    className="mylistings__row mylistings__row--sold"
                    initial={{ opacity: 0, x: -16 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ duration: 0.3, delay: i * 0.04, ease: [0.16, 1, 0.3, 1] }}
                  >
                    <div className="mylistings__row-item">
                      <div className="mylistings__row-img mylistings__row-img--sold">
                        {listing.images?.[0]?.url || listing.image_url ? (
                          <img src={listing.images?.[0]?.url || listing.image_url} alt={listing.title} />
                        ) : (
                          <span className="mylistings__row-img-placeholder">◈</span>
                        )}
                      </div>
                      <div className="mylistings__row-info">
                        <Link to={`/listings/${listing.id}`} className="mylistings__row-title">
                          {listing.title}
                        </Link>
                        {listing.category && (
                          <span className="mylistings__row-cat">{listing.category}</span>
                        )}
                      </div>
                    </div>
                    <div><span className="tag tag--sold">Sold</span></div>
                    <div className="price-display" style={{ fontSize: '1rem', opacity: 0.6 }}>
                      {formatPrice(listing.price)}
                    </div>
                    <div className="mylistings__row-date">
                      {new Date(listing.created_at).toLocaleDateString('en-US', {
                        month: 'short', day: 'numeric', year: 'numeric',
                      })}
                    </div>
                    <div className="mylistings__row-actions">
                      <Link to={`/listings/${listing.id}`} className="btn btn--ghost btn--sm">
                        View
                      </Link>
                    </div>
                  </motion.div>
                ))}
              </div>
            </section>
          )}
        </>
      )}
    </div>
  );
}
