import { createContext, useContext, useState, useCallback, useEffect } from 'react';

const NotificationsContext = createContext(null);

export function NotificationsProvider({ children }) {
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
      const existing = prev.find(l => l.id === listingId);
      if (existing) {
        return prev.map(l => l.id === listingId ? { ...l, count: l.count + 1 } : l);
      }
      return [...prev, { id: listingId, title: listingTitle || `Listing #${listingId}`, count: 1 }];
    });
  }, []);

  const clearUnread = useCallback(() => {
    setUnreadCount(0);
    setUnreadListings([]);
  }, []);

  return (
    <NotificationsContext.Provider value={{ unreadCount, unreadListings, addUnread, clearUnread }}>
      {children}
    </NotificationsContext.Provider>
  );
}

export function useNotifications() {
  const ctx = useContext(NotificationsContext);
  if (!ctx) throw new Error('useNotifications must be used within NotificationsProvider');
  return ctx;
}
