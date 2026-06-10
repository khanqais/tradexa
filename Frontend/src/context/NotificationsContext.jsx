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
    </NotificationsContext.Provider>
  );
}

export function useNotifications() {
  const ctx = useContext(NotificationsContext);
  if (!ctx)
    throw new Error(
      "useNotifications must be used within NotificationsProvider",
    );
  return ctx;
}
