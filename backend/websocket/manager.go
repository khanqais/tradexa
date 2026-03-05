package websocket

import "sync"

// HubManager is a global registry of all active chat rooms
type HubManager struct {
	conversationHubs map[string]*Hub           // key = conversationID as string e.g. "42"
	userClients      map[uint]map[*Client]bool // key = userID, value = set of active clients for that user
	mu               sync.Mutex                // mutex prevents two goroutines creating the same hub simultaneously
}

// Manager is a package-level singleton — one instance shared across the entire app
var Manager = &HubManager{
	conversationHubs: make(map[string]*Hub),
	userClients:      make(map[uint]map[*Client]bool),
}

// GetOrCreateConversation returns the existing Hub for a conversation, or creates a new one
func (m *HubManager) GetOrCreateConversation(conversationID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	// if hub already exists for this conversation, return it
	if hub, exists := m.conversationHubs[conversationID]; exists {
		return hub
	}

	// first user for this conversation — create a brand new hub
	hub := NewHub()

	// store it so future users get the same one
	m.conversationHubs[conversationID] = hub

	// start the hub's Run() loop in a separate goroutine
	// it will now listen for register/unregister/broadcast events forever
	go hub.Run()

	return hub
}

// RegisterClient adds a client to the global user registry
func (m *HubManager) RegisterClient(userID uint, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.userClients[userID] == nil {
		m.userClients[userID] = make(map[*Client]bool)
	}
	m.userClients[userID][client] = true
}

// UnregisterClient removes a client from the global user registry
func (m *HubManager) UnregisterClient(userID uint, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if clients, ok := m.userClients[userID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(m.userClients, userID)
		}
	}
}

// NotifyUser sends a message to all active clients of a user
func (m *HubManager) NotifyUser(userID uint, message []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if clients, ok := m.userClients[userID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				// if send fails, we'll let the hub's Run loop or WritePump handle cleanup if needed
				// but here we just skip to avoid blocking
			}
		}
	}
}
