import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../api';
import ListingCard from '../components/ListingCard';
import { Spinner } from '../components/Spinner';
import '../components/ListingCard.css';
// shared grid styles come from ListingCard.css

export default function BuyProductsPage() {
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
        type: 'fixed'
      });
      
      const response = await api.get(`/listings?${params}`);
      setListings(response.data.listings ?? []);
    } catch (err) {
      setError('Failed to load buy now listings');
      console.error('Error loading buy now listings:', err);
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
      <div className="page">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="page">
      {/* Hero Section */}
      <section className="hero">
        <div className="hero__content">
          <motion.h1 
            className="hero__title"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
          >
            Buy Products
          </motion.h1>
          <motion.p 
            className="hero__subtitle"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
          >
            Shop from thousands of items available for immediate purchase. Fast, secure transactions.
          </motion.p>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
          >
            <input
              type="text"
              placeholder="Search products..."
              className="input input--large"
              value={filters.search}
              onChange={(e) => handleFilterChange('search', e.target.value)}
            />
          </motion.div>
        </div>
      </section>

      {/* Stats Strip */}
      <section className="stats-strip">
        <div className="container">
          <div className="stats-strip__content">
            <div className="stat">
              <span className="stat__value">{listings.length}</span>
              <span className="stat__label">Products Available</span>
            </div>
            <div className="stat">
              <span className="stat__value">24/7</span>
              <span className="stat__label">Order Processing</span>
            </div>
            <div className="stat">
              <span className="stat__value">100%</span>
              <span className="stat__label">Secure</span>
            </div>
          </div>
        </div>
      </section>

      {/* Filters */}
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
              <option value="popularity">Most Popular</option>
            </select>
          </div>
        </div>
      </section>

      {/* Listings Grid */}
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
              <h3>No products found</h3>
              <p>Try adjusting your search criteria or check back later for new products.</p>
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