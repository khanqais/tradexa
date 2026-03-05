import axios from 'axios';

// Relative base — Vite proxy routes /api → http://localhost:8080 in dev
const API_BASE = '/api';

// WebSocket: dynamically detect protocol so wss works in production too
const WS_PROTOCOL = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const WS_BASE = typeof window !== 'undefined'
  ? `${WS_PROTOCOL}//${window.location.host}/api`
  : 'ws://localhost:8080/api';

export const api = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
});

// Attach JWT token to every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('tradexa_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Auth
export const login    = (data) => api.post('/login', data);
export const register = (data) => api.post('/register', data);
export const getMe    = ()     => api.get('/me');

// Conversations
export const getConversations = () => api.get('/conversations');
export const createConversation = (data) => api.post('/conversations', data);
export const getConversationMessages = (conversationId) => api.get(`/conversations/${conversationId}/messages`);

// Listings
export const getListings = (params) => api.get('/listings', { params });
export const getListingById = (id)  => api.get(`/listings/${id}`);
export const createListing = (data) => api.post('/listings', data);
export const updateListing = (id, data) => api.put(`/listings/${id}`, data);
export const deleteListing = (id)   => api.delete(`/listings/${id}`);

// Upload
export const uploadImage = (file) => {
  const form = new FormData();
  form.append('image', file);
  return api.post('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};

// Chat history (public)
export const getChatHistory = (listingId) => api.get(`/chat/${listingId}/history`);

// WebSocket factory — token passed as query param (WS can't use headers)
export const createChatSocket = (listingId) => {
  const token = localStorage.getItem('tradexa_token');
  return new WebSocket(`${WS_BASE}/ws/chat/${listingId}?token=${token}`);
};

// WebSocket factory for conversations
export const createConversationSocket = (conversationId) => {
  const token = localStorage.getItem('tradexa_token');
  return new WebSocket(`${WS_BASE}/ws/conversation/${conversationId}?token=${token}`);
};

export const createNotificationSocket = () => {
  const token = localStorage.getItem('tradexa_token');
  if (!token) return null;
  return new WebSocket(`${WS_BASE}/ws/notifications?token=${token}`);
};



export default api;
