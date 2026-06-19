import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Zap, Package, AlertTriangle, MessageSquare,
  Pencil, Trash2, Clock, ShieldCheck,
} from 'lucide-react';
import { API_BASE, getListingById, deleteListing, createConversation, createBid, createPaymentOrder, shipOrder } from '../api';
import { useAuth } from '../context/AuthContext';
import { Spinner } from '../components/Spinner';
import './ListingDetailPage.css';

function formatPrice(price) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(price || 0);
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

  const [listing, setListing] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [chatLoading, setChatLoading] = useState(false);
  const [bidLoading, setBidLoading] = useState(false);
  const [amount, setAmount] = useState('');
  const [currentBid, setCurrentBid] = useState(null);
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const [countdown, setCountdown] = useState('');
  const [bidSuccess, setBidSuccess] = useState('');
  const [auctionStatus, setAuctionStatus] = useState(null); // 'sold' | 'reserve_not_met' | null
  useEffect(() => {
    const fetchListing = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await getListingById(id);
        const fetchedListing = res.data.listing;
        setListing(fetchedListing);

        const initialBid =
          fetchedListing.current_bid ??
          fetchedListing.highest_bid ??
          fetchedListing.latest_bid?.amount ??
          null;

        setCurrentBid(initialBid);
      } catch {
        setError('Listing not found or server unavailable.');
      } finally {
        setLoading(false);
      }
    };

    fetchListing();
  }, [id]);

  useEffect(() => {
    if (!listing?.auction_ends_at || listing.type !== 'auction') return;
    const update = () => setCountdown(timeLeft(listing.auction_ends_at));
    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [listing]);

  // SSE: real-time bid updates
  useEffect(() => {
    if (!listing || listing.type !== 'auction') return;

    const eventSource = new EventSource(`${API_BASE}/stream/${id}`);

    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);

      if (data.type === 'new_bid') {
        setCurrentBid(Number(data.amount));
        setListing((prev) => {
          if (!prev) return prev;
          const updates = { ...prev, highest_bid: Number(data.amount) };
          if (data.winning_bidder_name) {
            updates.highest_bidder = data.winning_bidder_name;
          }
          // Update auction_ends_at if the server sent a new one (anti-snipe extension)
          if (data.auction_ends_at) {
            updates.auction_ends_at = data.auction_ends_at;
          }
          return updates;
        });
      }

      if (data.type === 'timer_extended') {
        // Anti-snipe: timer was extended
        setListing((prev) =>
          prev ? { ...prev, auction_ends_at: data.new_auction_ends_at } : prev
        );
      }

      if (data.type === 'auction_closed') {
        setAuctionStatus(data.status); // 'sold' or 'reserve_not_met'
        setListing((prev) =>
          prev ? { ...prev, is_sold: data.status === 'sold', status: data.status } : prev
        );
        setCountdown('Auction ended');
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [id, listing?.type]);

  const handleChat = async () => {
    if (!isAuthenticated) {
      navigate('/auth');
      return;
    }

    setChatLoading(true);
    try {
      const res = await createConversation({
        listing_id: listing.id,
        buyer_id: user.id,
      });
      navigate(`/conversations/${res.data.conversation.id}`);
    } catch (err) {
      console.error('Error starting conversation:', err);
      alert('Failed to start conversation. Please try again.');
    } finally {
      setChatLoading(false);
    }
  };

  const BidFunc = async () => {
    if (!isAuthenticated) {
      navigate('/auth');
      return;
    }

    if (!amount || Number(amount) <= 0) {
      alert('Please enter a valid bid amount');
      return;
    }

    setBidLoading(true);
    setBidSuccess('');

    try {
      const res = await createBid({
        listing_id: listing.id,
        amount: Number(amount),
      });

      const newPrice = res.data.current_price;
      setCurrentBid(Number(newPrice));
      setListing((prev) =>
        prev
          ? {
            ...prev,
            highest_bid: Number(newPrice),
            user_max_bid: Number(amount),
          }
          : prev
      );
      setBidSuccess(`Your max bid was processed. Current price is now ${formatPrice(newPrice)}.`);
      setAmount('');
      console.log('Bid placed:', res.data);
    } catch (err) {
      const errorMsg = err?.response?.data?.error || 'Failed to place bid. Please try again.';
      alert(errorMsg);
      console.error('Bid error:', err);
    } finally {
      setBidLoading(false);
    }
  };

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

  const handleShipOrder = async () => {
    if (!listing?.order?.id) return;
    try {
      const res = await shipOrder(listing.order.id);
      alert(res.data.message || "Item shipped!");
      // Optionally trigger a reload or update local state
      setListing(prev => ({
        ...prev,
        order: { ...prev.order, status: 'shipped' }
      }));
    } catch (err) {
      console.error(err);
      alert("Failed to mark as shipped");
    }
  };

  if (loading) {
    return (
      <div className="detail-loading">
        <Spinner size="lg" />
        <p>Loading listing…</p>
      </div>
    );
  }  const handlePayment = async () => {
    try {
      const res = await createPaymentOrder({
        amount: currentBid || listing.price,
        listing_id: listing.id
      });
      const sessionId = res.data.payment_session_id;
      if (sessionId && window.Cashfree) {
        const cashfree = window.Cashfree({
          mode: "sandbox",
        });
        cashfree.checkout({
          paymentSessionId: sessionId
        });
      }
    } catch (err) {
      console.error("Payment failed", err);
      alert("Payment initialization failed");
    }
  };


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
  const isOwner = listing && user && listing.seller_id === user.id;
  const liveBid = currentBid ?? null;
  const minimumNextBid = liveBid ? liveBid + 1 : Number(listing.price);
  const hasLiveBid = liveBid !== null;

  return (
    <div className="detail container">
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
        <motion.div
          className="detail__main"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        >
          <div className="detail__gallery">
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

              {listing.images && listing.images.length > 1 && (
                <div className="detail__img-counter">
                  {currentImageIndex + 1} / {listing.images.length}
                </div>
              )}

              {listing.images && listing.images.length > 1 && (
                <>
                  <button
                    className="detail__img-nav detail__img-nav--prev"
                    onClick={() =>
                      setCurrentImageIndex((idx) => (idx - 1 + listing.images.length) % listing.images.length)
                    }
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

          <div className="detail__info">
            {listing.category && (
              <span className="detail__category">{listing.category}</span>
            )}

            <h1 className="detail__title">{listing.title}</h1>

            {isAuction && countdown && !listing.is_sold && !auctionStatus && (
              <div className="detail__countdown">
                <span className="live-dot" />
                <Clock size={13} strokeWidth={2} />
                <span className="detail__countdown-label">Time remaining</span>
                <span className="detail__countdown-value price-display">{countdown}</span>
              </div>
            )}

            <AnimatePresence>
              {auctionStatus === 'sold' && (
                              <motion.div
                  className="detail__auction-banner detail__auction-banner--sold"
                  initial={{ opacity: 0, y: -10, scale: 0.95 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
                >
                  <span style={{ fontSize: '1.25rem' }}>{isOwner ? '📦' : '🏆'}</span>
                  <div style={{ flex: 1 }}>
                    <strong>{isOwner ? 'Auction Ended — Your Item Sold!' : 'Auction Ended — Item Sold!'}</strong>
                    <p style={{ margin: '0.25rem 0 0', opacity: 0.85, fontSize: '0.85rem' }}>
                      The winning bid was {formatPrice(liveBid)}. {isOwner ? 'Waiting for the buyer to complete payment.' : 'Please complete your payment to claim the item.'}
                    </p>
                    {!isOwner && (
                      <button onClick={handlePayment} className="btn btn--primary btn--sm" style={{ marginTop: '0.75rem' }}>
                        Pay Now with Cashfree (Test)
                      </button>
                    )}
                  </div>
                </motion.div>
              )}
              {auctionStatus === 'reserve_not_met' && (
                <motion.div
                  className="detail__auction-banner detail__auction-banner--reserve"
                  initial={{ opacity: 0, y: -10, scale: 0.95 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
                >
                  <span style={{ fontSize: '1.25rem' }}>❌</span>
                  <div>
                    <strong>Reserve Price Not Met</strong>
                    <p style={{ margin: '0.25rem 0 0', opacity: 0.85, fontSize: '0.85rem' }}>
                      The auction has ended but the reserve price was not reached. The item remains unsold.
                    </p>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            <div className="detail__price-block">
              <div className="detail__price-section">
                <span className="detail__price-label">
                  {isAuction ? 'Starting Bid' : 'Price'}
                </span>
                <span className="detail__price price-display">
                  {formatPrice(listing.price)}
                </span>
              </div>

              {isAuction && (
                <motion.div
                  className="detail__price-section"
                  key={liveBid ?? 'no-bid'}
                  initial={{ opacity: 0.6, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.3 }}
                >
                  <span className="detail__price-label">
                    {hasLiveBid ? 'Current Bid' : 'Current Bid'}
                  </span>
                  <span
                    className="detail__price price-display"
                    style={{ color: hasLiveBid ? '#f59e0b' : '#94a3b8' }}
                  >
                    {hasLiveBid ? formatPrice(liveBid) : 'No bids yet'}
                  </span>
                  {hasLiveBid && listing.highest_bidder && !listing.is_sold && !auctionStatus && (
                    <span style={{ fontSize: '0.85rem', color: '#94a3b8', marginTop: '-0.2rem' }}>
                      Winning: {listing.highest_bidder}
                    </span>
                  )}
                </motion.div>
              )}

              {listing.reserve_price > 0 && (
                <div className="detail__price-section">
                  <span className="detail__price-label">Reserve</span>
                  <span className="detail__reserve price-display">
                    {formatPrice(listing.reserve_price)}
                  </span>
                </div>
              )}
            </div>

            {!listing.is_sold && !isOwner && !auctionStatus && listing.type === 'auction' && (
              <div className="detail__bid-section">
                {listing.user_max_bid && (
                  <div style={{ marginBottom: '1rem', padding: '0.75rem', backgroundColor: '#f0fdf4', border: '1px solid #bbf7d0', borderRadius: '8px', color: '#15803d', fontSize: '0.9rem', fontWeight: 500 }}>
                    Your current maximum proxy bid is {formatPrice(listing.user_max_bid)}.
                  </div>
                )}
                <label className="detail__bid-label">
                  Maximum Proxy Bid
                  <span style={{ marginLeft: '0.5rem', color: '#94a3b8', fontWeight: 500 }}>
                    (Min: {formatPrice(minimumNextBid)})
                  </span>
                </label>
                <p style={{ fontSize: '0.8rem', color: '#94a3b8', marginBottom: '0.5rem' }}>
                  We'll automatically bid on your behalf to keep you in the lead, up to your limit.
                </p>

                <div className="detail__bid-input-group">
                  <span className="detail__bid-currency">$</span>
                  <input
                    type="number"
                    className="detail__bid-input"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                    placeholder={`${minimumNextBid}`}
                    min={minimumNextBid}
                  />
                </div>

                <button
                  className="btn btn--amber btn--lg detail__cta-btn"
                  onClick={BidFunc}
                  disabled={bidLoading}
                >
                  {bidLoading && <Zap size={16} strokeWidth={2} />}
                  {bidLoading ? 'Placing Bid...' : 'Set Maximum Bid'}
                </button>

                <AnimatePresence mode="wait">
                  {bidSuccess && (
                    <motion.p
                      key={bidSuccess}
                      initial={{ opacity: 0, y: 8 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -8 }}
                      transition={{ duration: 0.25 }}
                      style={{
                        marginTop: '0.75rem',
                        fontSize: '0.875rem',
                        color: '#10b981',
                        textAlign: 'center',
                        fontWeight: '500',
                      }}
                    >
                      ✓ {bidSuccess}
                    </motion.p>
                  )}
                </AnimatePresence>
              </div>
            )}

            {!listing.is_sold && !isOwner && listing.type !== 'auction' && (
              <div className="detail__cta" style={{ flexDirection: 'column' }}>
                <button
                  className="btn btn--amber btn--lg detail__cta-btn"
                  onClick={handleChat}
                >
                  {chatLoading && <MessageSquare size={16} strokeWidth={2} />}
                  Contact Seller
                </button>
                <button
                  className="btn btn--primary btn--lg detail__cta-btn"
                  onClick={handlePayment}
                >
                  Buy Now with Cashfree (Test)
                </button>
              </div>
            )}

            {isOwner && listing.is_sold && listing.order && (
              <div style={{ marginBottom: '1.5rem', padding: '1rem', backgroundColor: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
                <h4 style={{ margin: '0 0 0.5rem', color: '#0f172a' }}>📦 Seller Dashboard</h4>
                <p style={{ margin: '0 0 1rem', fontSize: '0.9rem', color: '#475569' }}>
                  Order Status: <strong style={{ textTransform: 'uppercase' }}>{listing.order.status.replace(/_/g, ' ')}</strong>
                </p>
                {listing.order.status === 'paid_in_escrow' && (
                  <button onClick={handleShipOrder} className="btn btn--primary btn--sm" style={{ width: '100%' }}>
                    Mark as Shipped
                  </button>
                )}
                {listing.order.status === 'shipped' && (
                  <p style={{ margin: 0, fontSize: '0.85rem', color: '#10b981' }}>Item marked as shipped. Waiting for delivery...</p>
                )}
              </div>
            )}

            <div className="detail__desc-wrap">
              <h4 className="detail__desc-heading">Description</h4>
              <p className="detail__desc">{listing.description}</p>
            </div>

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

            <div className="detail__meta-row">
              <span className="detail__meta-item">
                <span className="detail__meta-key">Listed</span>
                {new Date(listing.created_at).toLocaleDateString('en-US', {
                  year: 'numeric',
                  month: 'short',
                  day: 'numeric',
                })}
              </span>

              {isAuction && listing.auction_ends_at && (
                <span className="detail__meta-item">
                  <span className="detail__meta-key">Ends</span>
                  {new Date(listing.auction_ends_at).toLocaleDateString('en-US', {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })}
                </span>
              )}
            </div>

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