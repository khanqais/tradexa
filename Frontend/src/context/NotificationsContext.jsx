import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useRef,
} from "react";
import { createNotificationSocket } from "../api";
import { useAuth } from "./AuthContext";
import { motion, AnimatePresence } from "framer-motion";

const NotificationsContext = createContext(null);

function isTokenExpired(token) {
  if (!token) return true;
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.exp * 1000 < Date.now();
  } catch {
    return true;
  }
}

export function NotificationsProvider({ children }) {
  const { isAuthenticated, logout } = useAuth();
  const [unreadCount, setUnreadCount] = useState(() => {
    try {
      return parseInt(localStorage.getItem("tradexa_unread") || "0", 10);
    } catch {
      return 0;
    }
  });

  const [unreadConversations, setUnreadConversations] = useState(() => {
    try {
      return JSON.parse(
        localStorage.getItem("tradexa_unread_conversations") || "[]",
      );
    } catch {
      return [];
    }
  });

  const [toasts, setToasts] = useState([]);

  const showToast = useCallback((id, message) => {
    setToasts((prev) => [...prev, { id, message }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 8000);
  }, []);

  useEffect(() => {
    localStorage.setItem("tradexa_unread", String(unreadCount));
  }, [unreadCount]);

  useEffect(() => {
    localStorage.setItem(
      "tradexa_unread_conversations",
      JSON.stringify(unreadConversations),
    );
  }, [unreadConversations]);

  const addUnread = useCallback((conversationId, listingId, listingTitle) => {
    setUnreadCount((n) => n + 1);
    setUnreadConversations((prev) => {
      const existing = prev.find(
        (c) => String(c.conversationId) === String(conversationId),
      );
      if (existing) {
        return prev.map((c) =>
          String(c.conversationId) === String(conversationId)
            ? { ...c, count: c.count + 1 }
            : c,
        );
      }
      return [
        ...prev,
        {
          conversationId,
          listingId,
          title: listingTitle || `Listing #${listingId}`,
          count: 1,
        },
      ];
    });
  }, []);

  const clearUnread = useCallback(() => {
    setUnreadCount(0);
    setUnreadConversations([]);
  }, []);

  const clearUnreadForConversation = useCallback((conversationId) => {
    setUnreadConversations((prev) => {
      const conversation = prev.find(
        (c) => String(c.conversationId) === String(conversationId),
      );
      if (!conversation) return prev;
      setUnreadCount((c) => Math.max(0, c - conversation.count));
      return prev.filter(
        (c) => String(c.conversationId) !== String(conversationId),
      );
    });
  }, []);

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
          if (data.type === "new_message") {
            addUnread(
              data.conversation_id,
              data.listing_id,
              data.listing_title,
            );
          } else if (data.type === "auction_won") {
            showToast(
              `auction_${data.listing_id}`,
              `🏆 You won "${data.title}"! Pay $${data.amount} within 48 hours.`
            );
          } else if (data.type === "auction_sold") {
            showToast(
              `auction_${data.listing_id}`,
              `✅ Your item "${data.title}" sold for $${data.amount} to ${data.buyer_name}.`
            );
          } else if (data.type === "auction_reserve_not_met") {
            showToast(
              `auction_${data.listing_id}`,
              `❌ Reserve not met for "${data.title}". Item is unsold.`
            );
          }
        } catch (err) {
          console.log(err.Error);
        }
      };

      socket.onclose = () => {
        const currentToken = localStorage.getItem("tradexa_token");
        if (isTokenExpired(currentToken)) {
          logout();
          return;
        }
        setTimeout(() => {
          if (isAuthenticated) connect();
        }, 5000);
      };
    };

    connect();
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [isAuthenticated, addUnread, logout]);

  return (
    <NotificationsContext.Provider
      value={{
        unreadCount,
        unreadConversations,
        addUnread,
        clearUnread,
        clearUnreadForConversation,
      }}
    >
      {children}

      <div style={{
        position: 'fixed',
        bottom: '24px',
        right: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: '12px',
        zIndex: 9999,
        pointerEvents: 'none',
      }}>
        <AnimatePresence>
          {toasts.map((t) => (
            <motion.div
              key={t.id}
              initial={{ opacity: 0, y: 50, scale: 0.9 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9, transition: { duration: 0.2 } }}
              style={{
                background: 'var(--ink-raised)',
                border: '1px solid var(--ink-border)',
                color: 'var(--text-primary)',
                padding: '16px 20px',
                borderRadius: '8px',
                boxShadow: '0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.1)',
                pointerEvents: 'auto',
                maxWidth: '350px',
                fontSize: '0.9rem',
                lineHeight: '1.5',
                fontWeight: '500',
              }}
            >
              {t.message}
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </NotificationsContext.Provider>
  )
}

export function useNotifications() {
  const ctx = useContext(NotificationsContext);
  if (!ctx)
    throw new Error(
      "useNotifications must be used within NotificationsProvider",
    );
  return ctx;
}
