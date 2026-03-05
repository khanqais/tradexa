import { useState, useRef } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { createListing, uploadImage } from '../api';
import { useAuth } from '../context/AuthContext';
import { Spinner } from '../components/Spinner';
import './CreateListingPage.css';

const CATEGORIES = ['Electronics', 'Art', 'Fashion', 'Collectibles', 'Furniture', 'Jewelry', 'Books', 'Sports', 'Other'];

function toLocalDateTimeString(date) {
  const pad = n => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export default function CreateListingPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [form, setForm] = useState({
    title: '',
    description: '',
    price: '',
    reserve_price: '',
    type: 'fixed',
    category: '',
    image_url: '',
    auction_ends_at: '',
  });

  const [imageFile, setImageFile]   = useState(null);
  const [imagePreview, setImagePreview] = useState('');
  const [uploading, setUploading]   = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError]           = useState('');
  const [step, setStep]             = useState(1); // 1: details, 2: preview

  const fileInputRef = useRef(null);

  if (!isAuthenticated) {
    return (
      <div className="create-gate container">
        <h2>Sign in to list an item</h2>
        <p>You need an account to create listings on Tradexa.</p>
        <Link to="/auth" className="btn btn--primary">Sign In</Link>
      </div>
    );
  }

  const handleChange = (key, val) => {
    setForm(f => ({ ...f, [key]: val }));
    setError('');
  };

  const handleImagePick = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setImageFile(file);
    setImagePreview(URL.createObjectURL(file));
  };

  const handleImageDrop = (e) => {
    e.preventDefault();
    const file = e.dataTransfer.files?.[0];
    if (!file || !file.type.startsWith('image/')) return;
    setImageFile(file);
    setImagePreview(URL.createObjectURL(file));
  };

  const handleImageUpload = async () => {
    if (!imageFile) return form.image_url;
    setUploading(true);
    try {
      const res = await uploadImage(imageFile);
      const url = res.data.url;
      setForm(f => ({ ...f, image_url: url }));
      return url;
    } catch {
      setError('Image upload failed. Please try again.');
      return null;
    } finally {
      setUploading(false);
    }
  };

  const validate = () => {
    if (!form.title.trim() || form.title.length < 3)
      return 'Title must be at least 3 characters.';
    if (!form.description.trim() || form.description.length < 10)
      return 'Description must be at least 10 characters.';
    if (!form.price || parseFloat(form.price) <= 0)
      return 'Price must be greater than 0.';
    if (!form.category)
      return 'Please select a category.';
    if (form.type === 'auction' && !form.auction_ends_at)
      return 'Auction listings need an end date.';
    if (form.type === 'auction' && new Date(form.auction_ends_at) <= new Date())
      return 'Auction end date must be in the future.';
    return null;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const validationError = validate();
    if (validationError) { setError(validationError); return; }

    setSubmitting(true);
    setError('');

    try {
      // Upload image first if one was chosen
      let imageUrl = form.image_url;
      if (imageFile) {
        imageUrl = await handleImageUpload();
        if (!imageUrl) { setSubmitting(false); return; }
      }

      const payload = {
        title:        form.title.trim(),
        description:  form.description.trim(),
        price:        parseFloat(form.price),
        type:         form.type,
        category:     form.category,
        image_url:    imageUrl || '',
      };

      if (form.reserve_price && parseFloat(form.reserve_price) > 0)
        payload.reserve_price = parseFloat(form.reserve_price);

      if (form.type === 'auction' && form.auction_ends_at)
        payload.auction_ends_at = new Date(form.auction_ends_at).toISOString();

      const res = await createListing(payload);
      navigate(`/listings/${res.data.listing.id}`);
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to create listing.');
    } finally {
      setSubmitting(false);
    }
  };

  const minDate = toLocalDateTimeString(new Date(Date.now() + 60 * 60 * 1000));

  return (
    <div className="create container">
      {/* Header */}
      <motion.div
        className="create__header"
        initial={{ opacity: 0, y: -16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
      >
        <Link to="/" className="create__back">← Back to market</Link>
        <h1 className="create__title">List an Item</h1>
        <p className="create__subtitle">
          Fill in the details below. Your item will appear live on the marketplace immediately.
        </p>
      </motion.div>

      <motion.form
        className="create__form"
        onSubmit={handleSubmit}
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.05, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="create__layout">
          {/* ── Left column: main fields ── */}
          <div className="create__col-main">

            {/* Listing type */}
            <div className="create__section">
              <h3 className="create__section-title">Listing Type</h3>
              <div className="create__type-picker">
                <button
                  type="button"
                  className={`create__type-btn ${form.type === 'fixed' ? 'create__type-btn--active' : ''}`}
                  onClick={() => handleChange('type', 'fixed')}
                >
                  <span className="create__type-icon">🏷</span>
                  <span className="create__type-label">Fixed Price</span>
                  <span className="create__type-desc">Buyers can purchase immediately at your set price</span>
                </button>
                <button
                  type="button"
                  className={`create__type-btn ${form.type === 'auction' ? 'create__type-btn--active' : ''}`}
                  onClick={() => handleChange('type', 'auction')}
                >
                  <span className="create__type-icon">⚡</span>
                  <span className="create__type-label">Auction</span>
                  <span className="create__type-desc">Buyers bid — highest offer wins when timer ends</span>
                </button>
              </div>
            </div>

            {/* Basic info */}
            <div className="create__section">
              <h3 className="create__section-title">Item Details</h3>

              <div className="create__field">
                <label className="form-label">Title *</label>
                <input
                  className="form-input"
                  type="text"
                  placeholder="e.g. Vintage Sony Walkman TPS-L2 (1979)"
                  value={form.title}
                  onChange={e => handleChange('title', e.target.value)}
                  minLength={3}
                  maxLength={120}
                  required
                />
                <span className="create__char-count">{form.title.length}/120</span>
              </div>

              <div className="create__field">
                <label className="form-label">Description *</label>
                <textarea
                  className="form-input create__textarea"
                  placeholder="Describe condition, provenance, what's included…"
                  value={form.description}
                  onChange={e => handleChange('description', e.target.value)}
                  minLength={10}
                  maxLength={2000}
                  rows={5}
                  required
                />
                <span className="create__char-count">{form.description.length}/2000</span>
              </div>

              <div className="create__field">
                <label className="form-label">Category *</label>
                <div className="create__category-grid">
                  {CATEGORIES.map(cat => (
                    <button
                      key={cat}
                      type="button"
                      className={`create__cat-btn ${form.category === cat.toLowerCase() ? 'create__cat-btn--active' : ''}`}
                      onClick={() => handleChange('category', cat.toLowerCase())}
                    >
                      {cat}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Pricing */}
            <div className="create__section">
              <h3 className="create__section-title">Pricing</h3>

              <div className="create__field-row">
                <div className="create__field">
                  <label className="form-label">
                    {form.type === 'auction' ? 'Starting Bid (USD) *' : 'Price (USD) *'}
                  </label>
                  <div className="create__input-prefix-wrap">
                    <span className="create__input-prefix">$</span>
                    <input
                      className="form-input create__input-prefixed"
                      type="number"
                      placeholder="0"
                      min="0.01"
                      step="0.01"
                      value={form.price}
                      onChange={e => handleChange('price', e.target.value)}
                      required
                    />
                  </div>
                </div>

                {form.type === 'auction' && (
                  <div className="create__field">
                    <label className="form-label">Reserve Price (optional)</label>
                    <div className="create__input-prefix-wrap">
                      <span className="create__input-prefix">$</span>
                      <input
                        className="form-input create__input-prefixed"
                        type="number"
                        placeholder="0"
                        min="0"
                        step="0.01"
                        value={form.reserve_price}
                        onChange={e => handleChange('reserve_price', e.target.value)}
                      />
                    </div>
                    <span className="create__field-hint">Minimum price to sell</span>
                  </div>
                )}
              </div>

              {form.type === 'auction' && (
                <div className="create__field">
                  <label className="form-label">Auction End Date & Time *</label>
                  <input
                    className="form-input"
                    type="datetime-local"
                    min={minDate}
                    value={form.auction_ends_at}
                    onChange={e => handleChange('auction_ends_at', e.target.value)}
                    required
                  />
                </div>
              )}
            </div>
          </div>

          {/* ── Right column: image ── */}
          <div className="create__col-side">
            <div className="create__section">
              <h3 className="create__section-title">Item Photo</h3>

              <div
                className={`create__dropzone ${imagePreview ? 'create__dropzone--has-image' : ''}`}
                onDragOver={e => e.preventDefault()}
                onDrop={handleImageDrop}
                onClick={() => fileInputRef.current?.click()}
              >
                {imagePreview ? (
                  <>
                    <img src={imagePreview} alt="Preview" className="create__dropzone-img" />
                    <div className="create__dropzone-overlay">
                      <span>Click to change</span>
                    </div>
                  </>
                ) : (
                  <div className="create__dropzone-placeholder">
                    <span className="create__dropzone-icon">⊞</span>
                    <span className="create__dropzone-text">Drop image here</span>
                    <span className="create__dropzone-hint">or click to browse</span>
                    <span className="create__dropzone-types">JPG, PNG, WEBP · max 5MB</span>
                  </div>
                )}
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".jpg,.jpeg,.png,.webp"
                  className="create__file-input"
                  onChange={handleImagePick}
                />
              </div>

              {imagePreview && (
                <button
                  type="button"
                  className="btn btn--ghost btn--sm create__remove-img"
                  onClick={() => {
                    setImageFile(null);
                    setImagePreview('');
                    setForm(f => ({ ...f, image_url: '' }));
                  }}
                >
                  ✕ Remove image
                </button>
              )}

              {/* Or URL input */}
              <div className="create__field" style={{ marginTop: '16px' }}>
                <label className="form-label">Or paste image URL</label>
                <input
                  className="form-input"
                  type="url"
                  placeholder="https://…"
                  value={imageFile ? '' : form.image_url}
                  onChange={e => {
                    if (!imageFile) handleChange('image_url', e.target.value);
                  }}
                  disabled={!!imageFile}
                />
              </div>
            </div>

            {/* Preview card */}
            {(form.title || form.price) && (
              <div className="create__section create__preview-section">
                <h3 className="create__section-title">Preview</h3>
                <div className="create__preview-card">
                  {(imagePreview || form.image_url) && (
                    <div className="create__preview-img-wrap">
                      <img
                        src={imagePreview || form.image_url}
                        alt="preview"
                        className="create__preview-img"
                        onError={e => { e.target.style.display = 'none'; }}
                      />
                    </div>
                  )}
                  <div className="create__preview-body">
                    {form.category && (
                      <span className="create__preview-cat">{form.category}</span>
                    )}
                    <p className="create__preview-title">{form.title || 'Your title here'}</p>
                    <div className="create__preview-foot">
                      <span className="price-display" style={{ fontSize: '1rem' }}>
                        {form.price ? `$${parseFloat(form.price).toLocaleString()}` : '$—'}
                      </span>
                      {form.type === 'auction'
                        ? <span className="tag tag--auction">⚡ Auction</span>
                        : <span className="tag tag--fixed">Buy Now</span>
                      }
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Error */}
        {error && (
          <motion.div
            className="create__error"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
          >
            <span>⚠</span> {error}
          </motion.div>
        )}

        {/* Submit */}
        <div className="create__actions">
          <Link to="/" className="btn btn--ghost">Cancel</Link>
          <button
            type="submit"
            className="btn btn--amber btn--lg"
            disabled={submitting || uploading}
          >
            {submitting || uploading ? (
              <><Spinner size="sm" /> {uploading ? 'Uploading image…' : 'Publishing…'}</>
            ) : (
              form.type === 'auction' ? '⚡ Publish Auction' : '🏷 Publish Listing'
            )}
          </button>
        </div>
      </motion.form>
    </div>
  );
}
