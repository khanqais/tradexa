package websocket

import "sync"

type HubManager struct {
	conversationHubs map[string]*Hub
	userClients      map[uint]map[*Client]bool
	mu               sync.Mutex
}

var Manager = &HubManager{
	conversationHubs: make(map[string]*Hub),
	userClients:      make(map[uint]map[*Client]bool),
}

func (m *HubManager) GetOrCreateConversation(conversationID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, exists := m.conversationHubs[conversationID]; exists {
		return hub
	}

	hub := NewHub()

	m.conversationHubs[conversationID] = hub

	go hub.Run()

	return hub
}

func (m *HubManager) RegisterClient(userID uint, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.userClients[userID] == nil {
		m.userClients[userID] = make(map[*Client]bool)
	}
	m.userClients[userID][client] = true
}

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

func (m *HubManager) NotifyUser(userID uint, message []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if clients, ok := m.userClients[userID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
			}
		}
	}
}
