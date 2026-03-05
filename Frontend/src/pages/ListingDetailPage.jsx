import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import {
  Zap, Package, AlertTriangle, MessageSquare,
  Pencil, Trash2, Clock, ShieldCheck,
} from 'lucide-react';
import { getListingById, deleteListing, createConversation } from '../api';
import { useAuth } from '../context/AuthContext';
import { Spinner } from '../components/Spinner';
import './ListingDetailPage.css';

function formatPrice(price) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD',
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).format(price);
}

function timeLeft(endDate) {
  if (!endDate) return null;
  const diff = new Date(endDate) - Date.now();
  if (diff <= 0) return 'Auction ended';
  const d = Math.floor(diff / 86400000);
  const h = Math.floor((diff % 86400000) / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  const s = Math.floor((diff % 60000) / 1000);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  return `${m}m ${s}s`;
}

export default function ListingDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user, isAuthenticated } = useAuth();

  const [listing, setListing]   = useState(null);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [chatLoading, setChatLoading] = useState(false);
  const [currentImageIndex, setCurrentImageIndex] = useState(0);

  // Countdown timer
  const [countdown, setCountdown] = useState('');

  // ── Fetch listing ──
  useEffect(() => {
    const fetchListing = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await getListingById(id);
        setListing(res.data.listing);
      } catch {
        setError('Listing not found or server unavailable.');
      } finally {
        setLoading(false);
      }
    };
    fetchListing();
  }, [id]);

  // ── Countdown timer ──
  useEffect(() => {
    if (!listing?.auction_ends_at || listing.type !== 'auction') return;
    const update = () => setCountdown(timeLeft(listing.auction_ends_at));
    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [listing]);

  // ── Start / open conversation ──
  const handleChat = async () => {
    if (!isAuthenticated) { navigate('/auth'); return; }
    setChatLoading(true);
    try {
      const res = await createConversation({ listing_id: listing.id, buyer_id: user.id });
      navigate(`/conversations/${res.data.conversation.id}`);
    } catch (err) {
      console.error('Error starting conversation:', err);
      alert('Failed to start conversation. Please try again.');
    } finally {
      setChatLoading(false);
    }
  };

  // ── Delete listing ──
  const handleDelete = async () => {
    if (!window.confirm('Delete this listing? This cannot be undone.')) return;
    setDeleting(true);
    try {
      await deleteListing(id);
      navigate('/my-listings');
    } catch (e) {
      alert(e?.response?.data?.error || 'Delete failed.');
      setDeleting(false);
    }
  };

  // ── Render ──
  if (loading) {
    return (
      <div className="detail-loading">
        <Spinner size="lg" />
        <p>Loading listing…</p>
      </div>
    );
  }

  if (error || !listing) {
    return (
      <div className="detail-error container">
        <AlertTriangle size={32} strokeWidth={1.5} className="detail-error__icon" />
        <h3>Listing unavailable</h3>
        <p>{error}</p>
        <Link to="/" className="btn btn--ghost">← Back to market</Link>
      </div>
    );
  }

  const isAuction = listing.type === 'auction';
  const isOwner   = listing && user && listing.seller_id === user.id;

  return (
    <div className="detail container">
      {/* Breadcrumb */}
      <div className="detail__breadcrumb">
        <Link to="/" className="detail__breadcrumb-link">Market</Link>
        <span>›</span>
        {listing.category && (
          <>
            <Link to={`/?category=${listing.category.toLowerCase()}`} className="detail__breadcrumb-link">
              {listing.category}
            </Link>
            <span>›</span>
          </>
        )}
        <span className="detail__breadcrumb-current">{listing.title}</span>
      </div>

      <div className="detail__layout">
        {/* ── Left: Listing info ── */}
        <motion.div
          className="detail__main"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        >
          {/* Image Gallery */}
          <div className="detail__gallery">
            {/* Main Image */}
            <div className="detail__img-wrap">
              {listing.images?.[currentImageIndex]?.url || listing.images?.[0]?.url || listing.image_url ? (
                <img 
                  src={listing.images?.[currentImageIndex]?.url || listing.images?.[0]?.url || listing.image_url} 
                  alt={listing.title} 
                  className="detail__img" 
                />
              ) : (
                <div className="detail__img-placeholder">
                  <Package size={48} strokeWidth={1} />
                </div>
              )}
              
              {/* Type badge */}
              <div className="detail__img-badge">
                {isAuction ? (
                  <span className="tag tag--auction">
                    <Zap size={11} strokeWidth={2.5} /> Live Auction
                  </span>
                ) : (
                  <span className="tag tag--fixed">
                    <ShieldCheck size={11} strokeWidth={2} /> Buy Now
                  </span>
                )}
                {listing.is_sold && <span className="tag tag--sold">Sold</span>}
              </div>

              {/* Image counter */}
              {listing.images && listing.images.length > 1 && (
                <div className="detail__img-counter">
                  {currentImageIndex + 1} / {listing.images.length}
                </div>
              )}

              {/* Navigation arrows */}
              {listing.images && listing.images.length > 1 && (
                <>
                  <button
                    className="detail__img-nav detail__img-nav--prev"
                    onClick={() => setCurrentImageIndex((idx) => (idx - 1 + listing.images.length) % listing.images.length)}
                    aria-label="Previous image"
                  >
                    ‹
                  </button>
                  <button
                    className="detail__img-nav detail__img-nav--next"
                    onClick={() => setCurrentImageIndex((idx) => (idx + 1) % listing.images.length)}
                    aria-label="Next image"
                  >
                    ›
                  </button>
                </>
              )}
            </div>

            {/* Thumbnails */}
            {listing.images && listing.images.length > 1 && (
              <div className="detail__thumbnails">
                {listing.images.map((image, idx) => (
                  <button
                    key={idx}
                    className={`detail__thumbnail ${idx === currentImageIndex ? 'detail__thumbnail--active' : ''}`}
                    onClick={() => setCurrentImageIndex(idx)}
                    aria-label={`View image ${idx + 1}`}
                  >
                    <img src={image.url} alt={`${listing.title} - ${idx + 1}`} />
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Info */}
          <div className="detail__info">
            {listing.category && (
              <span className="detail__category">{listing.category}</span>
            )}
            <h1 className="detail__title">{listing.title}</h1>

            {/* Auction countdown */}
            {isAuction && countdown && !listing.is_sold && (
              <div className="detail__countdown">
                <span className="live-dot" />
                <Clock size={13} strokeWidth={2} />
                <span className="detail__countdown-label">Time remaining</span>
                <span className="detail__countdown-value price-display">{countdown}</span>
              </div>
            )}

            {/* Price block */}
            <div className="detail__price-block">
              <div className="detail__price-section">
                <span className="detail__price-label">
                  {isAuction ? 'Starting Bid' : 'Price'}
                </span>
                <span className="detail__price price-display">
                  {formatPrice(listing.price)}
                </span>
              </div>
              {listing.reserve_price > 0 && (
                <div className="detail__price-section">
                  <span className="detail__price-label">Reserve</span>
                  <span className="detail__reserve price-display">
                    {formatPrice(listing.reserve_price)}
                  </span>
                </div>
              )}
            </div>

            {/* CTA — only shown to non-owners */}
            {!listing.is_sold && !isOwner && (
              <div className="detail__cta">
                <button
                  className="btn btn--amber btn--lg detail__cta-btn"
                  onClick={handleChat}
                  disabled={chatLoading}
                >
                  {chatLoading ? (
                    <Spinner size="sm" />
                  ) : isAuction ? (
                    <><Zap size={16} strokeWidth={2.5} /> Place Bid via Chat</>
                  ) : (
                    <><MessageSquare size={16} strokeWidth={2} /> Contact Seller</>
                  )}
                </button>
              </div>
            )}

            {/* Description */}
            <div className="detail__desc-wrap">
              <h4 className="detail__desc-heading">Description</h4>
              <p className="detail__desc">{listing.description}</p>
            </div>

            {/* Seller */}
            {listing.seller && (
              <div className="detail__seller">
                <span className="detail__seller-avatar">
                  {listing.seller.name?.[0]?.toUpperCase()}
                </span>
                <div>
                  <div className="detail__seller-label">Listed by</div>
                  <div className="detail__seller-name">{listing.seller.name}</div>
                </div>
              </div>
            )}

            {/* Meta */}
            <div className="detail__meta-row">
              <span className="detail__meta-item">
                <span className="detail__meta-key">Listed</span>
                {new Date(listing.created_at).toLocaleDateString('en-US', {
                  year: 'numeric', month: 'short', day: 'numeric',
                })}
              </span>
              {isAuction && listing.auction_ends_at && (
                <span className="detail__meta-item">
                  <span className="detail__meta-key">Ends</span>
                  {new Date(listing.auction_ends_at).toLocaleDateString('en-US', {
                    year: 'numeric', month: 'short', day: 'numeric',
                  })}
                </span>
              )}
            </div>

            {/* Owner actions */}
            {isOwner && (
              <div className="detail__owner-actions">
                <div className="divider" />
                <div className="detail__owner-actions-row">
                  <Link to={`/listings/${id}/edit`} className="btn btn--ghost btn--sm">
                    <Pencil size={13} strokeWidth={2} /> Edit Listing
                  </Link>
                  <button
                    className="btn btn--danger btn--sm"
                    onClick={handleDelete}
                    disabled={deleting}
                  >
                    {deleting ? <Spinner size="sm" /> : <><Trash2 size={13} strokeWidth={2} /> Delete</>}
                  </button>
                </div>
              </div>
            )}
          </div>
        </motion.div>
      </div>
    </div>
  );
}
