package websocket

import "sync"

// HubManager is a global registry of all active chat rooms
// it maps listingID → Hub, so each listing has exactly one Hub
// think of it as a building directory — room 42 = Hub for listing 42
type HubManager struct {
	hubs map[string]*Hub // key = listingID as string e.g. "42"
	mu   sync.Mutex      // mutex prevents two goroutines creating the same hub simultaneously
}

// Manager is a package-level singleton — one instance shared across the entire app
// initialized once at startup, used by all WebSocket handlers
var Manager = &HubManager{
	hubs: make(map[string]*Hub),
}

// GetOrCreate returns the existing Hub for a listing, or creates a new one
// this is called every time a user opens a chat for a listing
func (m *HubManager) GetOrCreate(listingID string) *Hub {
	// Lock prevents race condition:
	// if two users connect to listing 42 at the exact same millisecond,
	// without the lock, both goroutines might check "does hub exist?" simultaneously,
	// both see "no", and both create a new hub — now you have 2 hubs for listing 42
	// with the lock, only one goroutine runs this block at a time
	m.mu.Lock()
	defer m.mu.Unlock() // unlock runs when this function exits, even on error

	// if hub already exists for this listing, return it
	// all future users join the same hub = same chat room
	if hub, exists := m.hubs[listingID]; exists {
		return hub
	}

	// first user for this listing — create a brand new hub
	hub := NewHub()

	// store it so future users get the same one
	m.hubs[listingID] = hub

	// start the hub's Run() loop in a separate goroutine
	// it will now listen for register/unregister/broadcast events forever
	go hub.Run()

	return hub
}
