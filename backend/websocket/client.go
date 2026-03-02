package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pongPeriod = (pongWait * 9) / 10
	maxMessage = 1024
)

type Client struct {
	Hub    *Hub            // pointer to the Hub this client belongs to
	Conn   *websocket.Conn // the actual WebSocket connection to this browser
	Send   chan []byte     // buffered channel — messages waiting to be sent to this client
	UserID uint            // from JWT — who is this person
	Name   string          // from JWT — their display name
}

// ReadPump runs in its own goroutine — continuously reads messages FROM the browser
// one ReadPump per client, for the entire lifetime of their connection
func (c *Client) ReadPump(listingID string) {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessage)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	// when we receive a pong (response to our ping), reset the deadline
	// this is how we know the client is still alive
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	

}
