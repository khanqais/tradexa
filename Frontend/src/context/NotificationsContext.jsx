import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import { createNotificationSocket } from '../api';
import { useAuth } from './AuthContext';

const NotificationsContext = createContext(null);

export function NotificationsProvider({ children }) {
  const { isAuthenticated } = useAuth();
  const [unreadCount, setUnreadCount] = useState(() => {
    try { return parseInt(localStorage.getItem('tradexa_unread') || '0', 10); }
    catch { return 0; }
  });

  // Track which listingIds have unread messages so badge is meaningful
  const [unreadListings, setUnreadListings] = useState(() => {
    try { return JSON.parse(localStorage.getItem('tradexa_unread_listings') || '[]'); }
    catch { return []; }
  });

  useEffect(() => {
    localStorage.setItem('tradexa_unread', String(unreadCount));
  }, [unreadCount]);

  useEffect(() => {
    localStorage.setItem('tradexa_unread_listings', JSON.stringify(unreadListings));
  }, [unreadListings]);

  const addUnread = useCallback((listingId, listingTitle) => {
    setUnreadCount(n => n + 1);
    setUnreadListings(prev => {
      const existing = prev.find(l => String(l.id) === String(listingId));
      if (existing) {
        return prev.map(l => String(l.id) === String(listingId) ? { ...l, count: l.count + 1 } : l);
      }
      return [...prev, { id: listingId, title: listingTitle || `Listing #${listingId}`, count: 1 }];
    });
  }, []);

  const clearUnread = useCallback(() => {
    setUnreadCount(0);
    setUnreadListings([]);
  }, []);

  const clearUnreadForListing = useCallback((listingId) => {
    setUnreadListings(prev => {
      const listing = prev.find(l => String(l.id) === String(listingId));
      if (!listing) return prev;
      setUnreadCount(c => Math.max(0, c - listing.count));
      return prev.filter(l => String(l.id) !== String(listingId));
    });
  }, []);

  // Global notification socket
  const wsRef = useRef(null);

  useEffect(() => {
    if (!isAuthenticated) {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      return;
    }

    let socket;
    const connect = () => {
      socket = createNotificationSocket();
      if (!socket) return;
      wsRef.current = socket;

      socket.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data);
          if (data.type === 'new_message') {
            addUnread(data.listing_id, data.listing_title);
          }
        } catch (err) { /* ignore */ }
      };

      socket.onclose = () => {
        // Reconnect after 5s if still authenticated
        setTimeout(() => { if (isAuthenticated) connect(); }, 5000);
      };
    };

    connect();
    return () => { 
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [isAuthenticated, addUnread]);

  return (
    <NotificationsContext.Provider value={{ unreadCount, unreadListings, addUnread, clearUnread, clearUnreadForListing }}>
      {children}
    </NotificationsContext.Provider>
  );
}

export function useNotifications() {
  const ctx = useContext(NotificationsContext);
  if (!ctx) throw new Error('useNotifications must be used within NotificationsProvider');
  return ctx;
}
