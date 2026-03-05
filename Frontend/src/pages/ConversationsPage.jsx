import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { MessageSquare, Package, User } from 'lucide-react';
import { getConversations, createConversation } from '../api';
import { Spinner } from '../components/Spinner';
import './ConversationsPage.css';

export default function ConversationsPage() {
  const [conversations, setConversations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchConversations = async () => {
      try {
        const response = await getConversations();
        setConversations(response.data.conversations || []);
      } catch (err) {
        setError('Failed to load conversations');
        console.error('Error fetching conversations:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchConversations();
  }, []);

  const startConversation = async (listingId, buyerId) => {
    try {
      const response = await createConversation({ listing_id: listingId, buyer_id: buyerId });
      const conversationId = response.data.conversation.id;
      navigate(`/conversations/${conversationId}`);
    } catch (err) {
      console.error('Error starting conversation:', err);
    }
  };

  if (loading) {
    return (
      <div className="conversations-loading">
        <Spinner size="lg" />
        <p>Loading conversations...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="conversations-error">
        <MessageSquare size={32} />
        <h3>Unable to load conversations</h3>
        <p>{error}</p>
        <button onClick={() => window.location.reload()} className="btn btn--primary">
          Try again
        </button>
      </div>
    );
  }

  return (
    <div className="conversations container">
      <div className="conversations-header">
        <h1>Messages</h1>
        {conversations.length === 0 && (
          <p className="conversations-empty-text">No conversations yet. Start chatting with sellers!</p>
        )}
      </div>

      {conversations.length > 0 && (
        <div className="conversations-list">
          {conversations.map(conversation => (
            <Link 
              key={conversation.id} 
              to={`/conversations/${conversation.id}`}
              className="conversation-item"
            >
              <div className="conversation-item__avatar">
                {conversation.buyer?.name?.[0]?.toUpperCase() || conversation.seller?.name?.[0]?.toUpperCase() || '?'}
              </div>
              <div className="conversation-item__info">
                <div className="conversation-item__title">
                  {conversation.listing?.title || `Listing #${conversation.listing_id}`}
                </div>
                <div className="conversation-item__participants">
                  <span className="conversation-item__participant">
                    {conversation.buyer?.name || 'Buyer'}
                  </span>
                  <span className="conversation-item__separator">•</span>
                  <span className="conversation-item__participant">
                    {conversation.seller?.name || 'Seller'}
                  </span>
                </div>
              </div>
              <div className="conversation-item__listing">
                {conversation.listing?.image_url ? (
                  <img src={conversation.listing.image_url} alt={conversation.listing.title} />
                ) : (
                  <Package size={24} />
                )}
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}