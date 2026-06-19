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
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
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
    image_urls: [],
    auction_ends_at: '',
  });

  const [imageFiles, setImageFiles] = useState([]);
  const [imagePreviews, setImagePreviews] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [step, setStep] = useState(1);

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
    const files = Array.from(e.target.files);
    if (!files.length) return;
    const newFiles = [...imageFiles, ...files];
    setImageFiles(newFiles);

    const newPreviews = newFiles.map(file => URL.createObjectURL(file));
    setImagePreviews(newPreviews);

  };
  const handleImageDrop = (e) => {
    e.preventDefault();
    const files = Array.from(e.dataTransfer.files);
    if (!files.length || !files.every(file => file.type.startsWith('image/'))) return;
    const newFiles = [...imageFiles, ...files];
    setImageFiles(newFiles);

    const newPreviews = newFiles.map(file => URL.createObjectURL(file));
    setImagePreviews(newPreviews);
  };
  const handleImageUpload = async () => {
    if (!imageFiles.length) return form.image_urls;
    setUploading(true);
    try {
      const uploadPromises = imageFiles.map(file => uploadImage(file));
      const results = await Promise.all(uploadPromises);
      const urls = results.map(res => res.data.url);
      setForm(f => ({ ...f, image_urls: urls }));
      return urls;

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
      let imageUrls = form.image_urls;
      if (imageFiles.length > 0) {
        imageUrls = await handleImageUpload();
        if (!imageUrls) { setSubmitting(false); return; }
      }

      const payload = {
        title: form.title.trim(),
        description: form.description.trim(),
        price: parseFloat(form.price),
        type: form.type,
        category: form.category,
        image_urls: imageUrls || [],
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

  const minDate = toLocalDateTimeString(new Date(Date.now() + 2 * 60 * 1000));

  return (
    <div className="create container">
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
          <div className="create__col-main">

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

          <div className="create__col-side">
            <div className="create__section">
              <h3 className="create__section-title">Item Photo</h3>

              <div
                className={'create__dropzone ' + (imagePreviews.length > 0 ? 'create__dropzone--has-image' : '')}
                onDragOver={e => e.preventDefault()}
                onDrop={handleImageDrop}
                onClick={() => fileInputRef.current?.click()}
              >
                {imagePreviews.length > 0 ? (
                  <div className="create__dropzone-multiple-preview">
                    {imagePreviews.map((preview, index) => (
                      <div key={index} className="create__preview-thumb-wrapper">
                        <img src={preview} alt={`Preview ${index + 1}`} className="create__preview-thumb" />
                      </div>
                    ))}
                    <div className="create__dropzone-overlay">
                      <span>Add more images</span>
                    </div>
                  </div>
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
                  multiple
                  className="create__file-input"
                  onChange={handleImagePick}
                />
              </div>

              {imagePreviews.length > 0 && (
                <button
                  type="button"
                  className="btn btn--ghost btn--sm create__remove-img"
                  onClick={() => {
                    setImageFiles([]);
                    setImagePreviews([]);
                    setForm(f => ({ ...f, image_urls: [] }));
                  }}
                >
                  ✕ Remove all images
                </button>
              )}

              <div className="create__field" style={{ marginTop: '16px' }}>
                <label className="form-label">Or paste image URL</label>
                <input
                  className="form-input"
                  type="url"
                  placeholder="https://…"
                  value={imageFiles.length > 0 ? '' : form.image_urls[0] || ''}
                  onChange={e => {
                    if (imageFiles.length === 0) handleChange('image_urls', [e.target.value]);
                  }}
                  disabled={imageFiles.length > 0}
                />
              </div>
            </div>

            {(form.title || form.price) && (
              <div className="create__section create__preview-section">
                <h3 className="create__section-title">Preview</h3>
                <div className="create__preview-card">
                  {(imagePreviews.length > 0 || form.image_urls.length > 0) && (
                    <div className="create__preview-img-wrap">
                      <img
                        src={imagePreviews[0] || form.image_urls[0]}
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

        {error && (
          <motion.div
            className="create__error"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
          >
            <span>⚠</span> {error}
          </motion.div>
        )}

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
