import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  MessageSquare, Send, ArrowLeft
} from 'lucide-react';
import {
  getConversationMessages,
  createConversationSocket,
  getConversations
} from '../api';
import { useAuth } from '../context/AuthContext';
import { useNotifications } from '../context/NotificationsContext';
import { Spinner } from '../components/Spinner';
import './ConversationDetailPage.css';


function formatPrice(price) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD',
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).format(price);
}


function MessageBubble({ msg, currentUserId }) {
  const isOwn = msg.sender_id === currentUserId || msg.sender?.id === currentUserId;

  return (
    <div className={`message ${isOwn ? 'message--sent' : 'message--received'}`}>
      <div className={`message__content ${isOwn ? 'message--sent' : 'message--received'}`}>
        <p style={{ margin: 0 }}>{msg.content}</p>
        <div className="message__time">
          {new Date(msg.created_at || msg.sent_at).toLocaleTimeString([], {
            hour: '2-digit', minute: '2-digit',
          })}
        </div>
      </div>
    </div>
  );
}


export default function ConversationDetailPage() {
  const { conversationId } = useParams();
  const navigate = useNavigate();
  const { user, isAuthenticated } = useAuth();
  const { clearUnreadForConversation } = useNotifications();

  const [conversation, setConversation] = useState(null);
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [msgInput, setMsgInput] = useState('');
  const [wsStatus, setWsStatus] = useState('idle');
  const [chatLoading, setChatLoading] = useState(false);

  const wsRef = useRef(null);
  const chatEndRef = useRef(null);

  // Fetch conversation details
  useEffect(() => {
    const fetchConversation = async () => {
      setLoading(true);
      setError(null);
      try {
        const response = await getConversations();
        const conversations = response.data.conversations || [];
        const foundConversation = conversations.find(
          (c) => c.id.toString() === conversationId
        );
        if (foundConversation) {
          setConversation(foundConversation);
        } else {
          setError('Conversation not found');
        }
      } catch (err) {
        setError('Failed to load conversation');
        console.error('Error fetching conversation:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchConversation();
  }, [conversationId]);

  // Load chat history
  useEffect(() => {
    const loadHistory = async () => {
      if (!conversationId) return;
      setChatLoading(true);
      try {
        const response = await getConversationMessages(conversationId);
        setMessages(response.data.messages || []);
      } catch (err) {
        console.error('Error loading chat history:', err);
      } finally {
        setChatLoading(false);
      }
    };

    loadHistory();
  }, [conversationId]);

  // WebSocket connection
  const connectWs = useCallback(() => {
    if (!isAuthenticated || wsRef.current?.readyState === WebSocket.OPEN) return;

    setWsStatus('connecting');
    const ws = createConversationSocket(conversationId);
    wsRef.current = ws;

    ws.onopen = () => setWsStatus('open');
    ws.onerror = () => setWsStatus('error');
    ws.onclose = () => setWsStatus('closed');

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        const newMsg = {
          sender_id: data.sender_id,
          sender_name: data.sender_name,
          content: data.content,
          sent_at: data.sent_at,
          sender: { id: data.sender_id, name: data.sender_name },
        };
        setMessages((prev) => [...prev, newMsg]);
      } catch (err) {
        console.error('Error parsing message:', err);
      }
    };
  }, [conversationId, isAuthenticated]);

  const disconnectWs = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  // Auto-connect when authenticated and conversation is loaded
  useEffect(() => {
    if (conversation && isAuthenticated) {
      connectWs();
      clearUnreadForConversation(conversationId);
    }
    return disconnectWs;
  }, [conversation, isAuthenticated, connectWs, disconnectWs, clearUnreadForConversation, conversationId]);

  // Auto-scroll
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

  if (loading) {
    return (
      <div className="conversation-detail-loading">
        <Spinner size="lg" />
        <p>Loading conversation...</p>
      </div>
    );
  }

  if (error || !conversation) {
    return (
      <div className="conversation-detail-error container">
        <MessageSquare size={32} strokeWidth={1.5} className="conversation-detail-error__icon" />
        <h3>Conversation unavailable</h3>
        <p>{error || 'This conversation could not be found.'}</p>
        <button onClick={() => navigate('/conversations')} className="btn btn--ghost">
          ← Back to messages
        </button>
      </div>
    );
  }

  return (
    <div className="conversation-detail">

      {/* Header */}
      <div className="conversation-detail__header">
        <button
          onClick={() => navigate('/conversations')}
          className="conversation-detail__back-btn"
        >
          <ArrowLeft size={20} />
        </button>

        <div className="conversation-detail__avatar">
          {(conversation.buyer?.name || conversation.seller?.name || '?')[0].toUpperCase()}
        </div>

        <div className="conversation-detail__info">
          <div className="conversation-detail__title">
            {conversation.listing?.title || 'Conversation'}
          </div>
          <div className="conversation-detail__participants">
            <span className="conversation-detail__participant">
              {conversation.buyer?.name || 'Buyer'}
            </span>
            <span className="conversation-detail__separator">•</span>
            <span className="conversation-detail__participant">
              {conversation.seller?.name || 'Seller'}
            </span>
          </div>
        </div>

        <div className="conversation-detail__status">
          <span className={`ws-status-pill ws-status-pill--${wsStatus}`}>
            {wsStatus === 'open' && <span className="live-dot" />}
            {wsStatus === 'open' ? 'Live' : wsStatus === 'connecting' ? 'Connecting…' : 'Offline'}
          </span>
        </div>
      </div>

      {/* Chat */}

      <div className="conversation-detail__chat">
        <div className="conversation-chat-area">
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
        <div className="conversation-input-area">
          <input
            className="conversation-input"
            type="text"
            placeholder={wsStatus === 'open' ? 'Type a message…' : 'Connecting...'}
            value={msgInput}
            onChange={(e) => setMsgInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendMessage(e);
              }
            }}
            disabled={wsStatus !== 'open'}
            maxLength={1000}
          />
          <button
            className="send-button"
            onClick={sendMessage}
            disabled={wsStatus !== 'open' || !msgInput.trim()}
          >
            <Send size={24} />
          </button>
        </div>
      </div>

    </div>
  );
}
