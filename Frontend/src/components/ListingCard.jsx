import { Link } from "react-router-dom";
import { motion } from "framer-motion";

import "./ListingCard.css";

function formatPrice(price) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(price);
}

function timeLeft(endDate) {
  if (!endDate) return null;
  const diff = new Date(endDate) - Date.now();
  if (diff <= 0) return "Ended";
  const h = Math.floor(diff / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  if (h > 24) return `${Math.floor(h / 24)}d left`;
  if (h > 0) return `${h}h ${m}m left`;
  return `${m}m left`;
}

export default function ListingCard({ listing, index = 0 }) {
  const isAuction = listing.type === "auction";
  const countdown = isAuction ? timeLeft(listing.auction_ends_at) : null;
  const isUrgent =
    isAuction &&
    countdown &&
    countdown.includes("m left") &&
    !countdown.includes("h");

  return (
    <motion.div
      className={`listing-card ${isAuction ? "listing-card--auction" : ""}`}
      initial={{ opacity: 0, y: 24 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{
        duration: 0.4,
        delay: index * 0.06,
        ease: [0.16, 1, 0.3, 1],
      }}
      whileHover={{ y: -4 }}
    >
      <Link to={`/listings/${listing.id}`} className="listing-card__link">
        <div className="listing-card__img-wrap">
          {listing.images?.[0]?.url || listing.image_url ? (
            <img
              src={listing.images?.[0]?.url || listing.image_url}
              alt={listing.title}
              className="listing-card__img"
              loading="lazy"
            />
          ) : (
            <div className="listing-card__img-placeholder">
              <span className="listing-card__img-icon">◈</span>
            </div>
          )}

          <div className="listing-card__badges">
            {isAuction ? (
              <span className="tag tag--auction">⚡ Auction</span>
            ) : (
              <span className="tag tag--fixed">Buy Now</span>
            )}
            {listing.is_sold && <span className="tag tag--sold">Sold</span>}
          </div>

          {isAuction && countdown && !listing.is_sold && (
            <div
              className={`listing-card__countdown ${isUrgent ? "listing-card__countdown--urgent" : ""}`}
            >
              <span className="live-dot" />
              <span className="listing-card__countdown-text">{countdown}</span>
            </div>
          )}
        </div>

        <div className="listing-card__body">
          {listing.category && (
            <span className="listing-card__category">{listing.category}</span>
          )}
          <h3 className="listing-card__title">{listing.title}</h3>
          <p className="listing-card__desc">{listing.description}</p>

          <div className="listing-card__footer">
            <div>
              <div className="listing-card__price-label">
                {isAuction ? (listing.highest_bid ? "Current bid" : "Starting bid") : "Price"}
              </div>
              <div className="price-display listing-card__price">
                {formatPrice(isAuction && listing.highest_bid ? listing.highest_bid : listing.price)}
              </div>
            </div>
            {listing.seller && (
              <div className="listing-card__seller">
                <span className="listing-card__seller-avatar">
                  {listing.seller.name?.[0]?.toUpperCase()}
                </span>
                <span className="listing-card__seller-name">
                  {listing.seller.name}
                </span>
              </div>
            )}
          </div>
        </div>
      </Link>
    </motion.div>
  );
}
