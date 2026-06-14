package handlers

import (
	"sync"
)

type AuctionHub struct {
	sync.RWMutex
	ListingClient map[uint]map[chan []byte]struct{}
}

var StreamHub = &AuctionHub{
	ListingClient: make(map[uint]map[chan []byte]struct{}),
}

func (h *AuctionHub) AddClient(listingID uint, clientChan chan []byte) {
	h.Lock()
	defer h.Unlock()
	if h.ListingClient[listingID] == nil {
		h.ListingClient[listingID] = make(map[chan []byte]struct{})
	}
	h.ListingClient[listingID][clientChan] = struct{}{}
}
func (h *AuctionHub) RemoveClient(listingID uint, clientChan chan []byte) {
	h.Lock()
	defer h.Unlock()
	if client, exist := h.ListingClient[listingID]; exist {
		delete(client, clientChan)

		if len(client) == 0 {
			delete(h.ListingClient, listingID)
		}
	}
}
func (h *AuctionHub) Broadcast(listingID uint, message []byte) {
	h.RLock()
	defer h.RUnlock()

	if clients, exists := h.ListingClient[listingID]; exists {
		for clientChan := range clients {
			select {
			case clientChan <- message:
				// Successfully sent to the client's channel
			default:
				// If the pipe is blocked, skip them so the server doesn't freeze
			}
		}
	}
}
