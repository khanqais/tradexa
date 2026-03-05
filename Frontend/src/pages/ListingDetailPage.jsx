import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import {
  Zap, Package, AlertTriangle, MessageSquare, Send,
  RefreshCw, Pencil, Trash2, Clock, ShieldCheck,
} from 'lucide-react';
import { getListingById, getChatHistory, createChatSocket, deleteListing } from '../api';
import { useAuth } from '../context/AuthContext';
import { useNotifications } from '../context/NotificationsContext';
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

function MessageBubble({ msg, currentUserId }) {
  const isOwn = msg.sender_id === currentUserId || msg.sender?.id === currentUserId;
  return (
    <motion.div
      className={`chat-bubble ${isOwn ? 'chat-bubble--own' : ''}`}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
    >
      {!isOwn && (
        <div className="chat-bubble__meta">
          <span className="chat-bubble__avatar">
            {(msg.sender?.name || msg.sender_name || '?')[0].toUpperCase()}
          </span>
          <span className="chat-bubble__name">
            {msg.sender?.name || msg.sender_name}
          </span>
        </div>
      )}
      <div className="chat-bubble__body">
        <p className="chat-bubble__text">{msg.content}</p>
        <span className="chat-bubble__time">
          {new Date(msg.created_at || msg.sent_at).toLocaleTimeString([], {
            hour: '2-digit', minute: '2-digit',
          })}
        </span>
      </div>
    </motion.div>
  );
}

export default function ListingDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user, isAuthenticated } = useAuth();
  const { addUnread } = useNotifications();

  const [listing, setListing]   = useState(null);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState(null);
  const [deleting, setDeleting] = useState(false);

  // Chat state
  const [messages, setMessages]       = useState([]);
  const [msgInput, setMsgInput]       = useState('');
  const [wsStatus, setWsStatus]       = useState('idle'); // idle | connecting | open | closed | error
  const [chatLoading, setChatLoading] = useState(false);

  // Countdown timer
  const [countdown, setCountdown] = useState('');

  const wsRef      = useRef(null);
  const chatEndRef = useRef(null);

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

  // ── Load chat history ──
  useEffect(() => {
    const loadHistory = async () => {
      setChatLoading(true);
      try {
        const res = await getChatHistory(id);
        setMessages(res.data.message || []);
      } catch { /* chat not critical */ }
      finally { setChatLoading(false); }
    };
    loadHistory();
  }, [id]);

  // ── WebSocket ──
  const connectWs = useCallback(() => {
    if (!isAuthenticated || wsRef.current?.readyState === WebSocket.OPEN) return;

    setWsStatus('connecting');
    const ws = createChatSocket(id);
    wsRef.current = ws;

    ws.onopen  = () => setWsStatus('open');
    ws.onerror = () => setWsStatus('error');
    ws.onclose = () => setWsStatus('closed');

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        const newMsg = {
          sender_id:   data.sender_id,
          sender_name: data.sender_name,
          content:     data.content,
          sent_at:     data.sent_at,
          sender: { id: data.sender_id, name: data.sender_name },
        };
        setMessages(prev => [...prev, newMsg]);

        // Notify if message is from someone else (not current user)
        if (data.sender_id !== user?.id) {
          addUnread(id, listing?.title);
        }
      } catch { /* ignore bad frames */ }
    };
  }, [id, isAuthenticated, user?.id, listing?.title, addUnread]);

  const disconnectWs = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  // Auto-connect when user is authenticated and listing loaded
  useEffect(() => {
    if (listing && isAuthenticated) connectWs();
    return disconnectWs;
  }, [listing, isAuthenticated, connectWs, disconnectWs]);

  // Auto-scroll chat
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendMessage = (e) => {
    e.preventDefault();
    const text = msgInput.trim();
    if (!text || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify({ content: text }));
    setMsgInput('');
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

  const isOwner = listing && user && listing.seller_id === user.id;

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
          {/* Image */}
          <div className="detail__img-wrap">
            {listing.image_url ? (
              <img src={listing.image_url} alt={listing.title} className="detail__img" />
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

            {/* CTA */}
            {!listing.is_sold && (
              <div className="detail__cta">
                {isAuction ? (
                  <button
                    className="btn btn--amber btn--lg detail__cta-btn"
                    onClick={() => { if (!isAuthenticated) navigate('/auth'); else connectWs(); }}
                  >
                    <Zap size={16} strokeWidth={2.5} />
                    Place Bid via Chat
                  </button>
                ) : (
                  <button
                    className="btn btn--amber btn--lg detail__cta-btn"
                    onClick={() => { if (!isAuthenticated) navigate('/auth'); else connectWs(); }}
                  >
                    <MessageSquare size={16} strokeWidth={2} />
                    Contact Seller
                  </button>
                )}
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

        {/* ── Right: Chat ── */}
        <motion.div
          className="detail__chat"
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.5, delay: 0.1, ease: [0.16, 1, 0.3, 1] }}
        >
          <div className="chat-panel">
            {/* Chat header */}
            <div className="chat-panel__header">
              <div className="chat-panel__header-left">
                <MessageSquare size={16} strokeWidth={1.75} className="chat-panel__header-icon" />
                <h4 className="chat-panel__title">Live Chat</h4>
                <div className="chat-panel__status">
                  <span className={`chat-panel__status-dot chat-panel__status-dot--${wsStatus}`} />
                  <span className="chat-panel__status-text">
                    {wsStatus === 'open'       ? 'Connected'
                    : wsStatus === 'connecting' ? 'Connecting…'
                    : wsStatus === 'error'      ? 'Error'
                    : wsStatus === 'closed'     ? 'Disconnected'
                    : 'Not connected'}
                  </span>
                </div>
              </div>
              <span className="chat-panel__count">{messages.length} msgs</span>
            </div>

            {/* Messages */}
            <div className="chat-panel__messages">
              {chatLoading ? (
                <div className="chat-panel__empty">
                  <Spinner size="md" />
                </div>
              ) : messages.length === 0 ? (
                <div className="chat-panel__empty">
                  <MessageSquare size={32} strokeWidth={1} className="chat-panel__empty-icon" />
                  <p>No messages yet. Start the conversation!</p>
                </div>
              ) : (
                <>
                  {messages.map((msg, i) => (
                    <MessageBubble
                      key={msg.id || `ws-${i}`}
                      msg={msg}
                      currentUserId={user?.id}
                    />
                  ))}
                  <div ref={chatEndRef} />
                </>
              )}
            </div>

            {/* Input */}
            {isAuthenticated ? (
              <form className="chat-panel__form" onSubmit={sendMessage}>
                <input
                  className="chat-panel__input"
                  type="text"
                  placeholder={
                    wsStatus === 'open'
                      ? 'Type a message…'
                      : 'Connect to start chatting'
                  }
                  value={msgInput}
                  onChange={e => setMsgInput(e.target.value)}
                  disabled={wsStatus !== 'open'}
                  maxLength={1000}
                />
                <button
                  type="submit"
                  className="chat-panel__send"
                  disabled={wsStatus !== 'open' || !msgInput.trim()}
                >
                  <Send size={15} strokeWidth={2} />
                </button>
              </form>
            ) : (
              <div className="chat-panel__login-prompt">
                <Link to="/auth" className="btn btn--primary btn--sm">
                  Sign in to chat
                </Link>
              </div>
            )}

            {/* Reconnect if disconnected */}
            {(wsStatus === 'closed' || wsStatus === 'error') && isAuthenticated && (
              <div className="chat-panel__reconnect">
                <button className="btn btn--ghost btn--sm" onClick={connectWs}>
                  <RefreshCw size={13} strokeWidth={2} /> Reconnect
                </button>
              </div>
            )}
          </div>
        </motion.div>
      </div>
    </div>
  );
}
