package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
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

	//infinite loop -> keep reading until connection break
	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("Websocket read error:%v", err)
			}
			break
		}
		var incoming struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(raw, &incoming); err != nil || incoming.Content == "" {
			continue
		}
		listingIDint := parseUnit(listingID)
		msg := models.Message{
			ListingID: listingIDint,
			SenderID:  c.UserID,
			Content:   incoming.Content,
		}
		config.DB.Create(&msg)

		// push to hub's broadcast channel — hub will deliver to all clients
		type outMsg struct {
			SenderID   uint      `json:"sender_id"`
			SenderName string    `json:"sender_name"`
			Content    string    `json:"content"`
			ListingID  string    `json:"listing_id"`
			SentAt     time.Time `json:"sent_at"`
		}
		out := outMsg{
			SenderID:   c.UserID,
			SenderName: c.Name,
			Content:    incoming.Content,
			ListingID:  listingID,
			SentAt:     time.Now(),
		}
		// marshal struct back to JSON bytes
		outBytes, _ := json.Marshal(out)
		// push to hub's broadcast channel — hub will deliver to all clients
		c.Hub.broadcast <- outBytes

	}
}

// WritePump runs in its own goroutine — continuously sends messages TO the browser
// it reads from client.Send channel (messages put there by the hub)
func (c *Client) WritePump() {
	// ticker fires every pingPeriod to send a ping to the browser
	// browser must reply with a pong — this confirms the connection is still alive
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	select {
	// CASE 1 — hub has a message ready for this client
	case message, ok := <-c.Send:
		c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}

	case <-ticker.C:
		c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

func parseUnit(s string) uint {
	var n uint64
	fmt.Sscanf(s, "%d", &n)
	return uint(n)
}
