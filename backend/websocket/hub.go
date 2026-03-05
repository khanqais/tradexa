package websocket

// Hub is the central message router for one chat room (one listing)
// Think of it as a post office — clients register themselves,
// and when a message arrives, the hub delivers it to everyone
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	Register   chan *Client
	unregister chan *Client
}

// NewHub creates a fresh Hub with all channels initialized
// called once per listing when the first user connects to that listing's chat
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run is the heart of the Hub — it runs forever in its own goroutine
// it listens on all three channels using select (like a switch but for channels)
// select blocks until one of the cases has data ready, then executes that case
func (h *Hub) Run() {
	for {
		select {
		// CASE 1 — a new client just connected
		case client := <-h.Register:
			// add client to the map — now they'll receive all future broadcasts
			h.clients[client] = true

		// CASE 2 — a client just disconnected
		case client := <-h.unregister:
			// check if client actually exists (might have already been removed)
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client) // remove from map
				close(client.Send)        // closing the Send channel signals WritePump to stop
				// remove from global registry too
				Manager.UnregisterClient(client.UserID, client)
			}
		case client := <-h.unregister:
			// check if client actually exists (might have already been removed)
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client) // remove from map
				close(client.Send)        // closing the Send channel signals WritePump to stop
			}

		// CASE 3 — a new message arrived, send it to everyone
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				// try to put message into client's Send channel
				case client.Send <- message:

				// if client's Send channel is full (they're too slow / disconnected)
				// close their channel and remove them to avoid memory leak
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
	}
}
