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
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID uint
	Name   string
}

func (c *Client) ReadPump(listingID string) {
	defer func() {
		if c.Hub != nil {
			c.Hub.unregister <- c
		} else {
			Manager.UnregisterClient(c.UserID, c)
		}
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessage)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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

		var listing models.Listing
		if err := config.DB.Select("seller_id, title").First(&listing, listingIDint).Error; err == nil {
			if listing.SellerID != c.UserID {
				notification := map[string]interface{}{
					"type":          "new_message",
					"listing_id":    listingID,
					"listing_title": listing.Title,
					"sender_id":     c.UserID,
					"sender_name":   c.Name,
					"content":       incoming.Content,
					"sent_at":       time.Now(),
				}
				notifBytes, _ := json.Marshal(notification)
				Manager.NotifyUser(listing.SellerID, notifBytes)
			}
		}

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
		outBytes, _ := json.Marshal(out)
		if c.Hub != nil {
			c.Hub.broadcast <- outBytes
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
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
}

func (c *Client) ReadPumpForConversation(conversationID string) {
	defer func() {
		if c.Hub != nil {
			c.Hub.unregister <- c
		} else {
			Manager.UnregisterClient(c.UserID, c)
		}
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessage)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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

		conversationIDint := parseUnit(conversationID)
		var conversation models.Conversation
		if err := config.DB.First(&conversation, conversationIDint).Error; err != nil {
			continue
		}

		msg := models.Message{
			ListingID:      conversation.ListingID,
			ConversationID: conversationIDint,
			SenderID:       c.UserID,
			Content:        incoming.Content,
		}
		config.DB.Create(&msg)

		var recipientID uint
		if conversation.BuyerID == c.UserID {
			recipientID = conversation.SellerID
		} else {
			recipientID = conversation.BuyerID
		}

		recipientInHub := false
		if c.Hub != nil {
			for hubClient := range c.Hub.clients {
				if hubClient.UserID == recipientID {
					recipientInHub = true
					break
				}
			}
		}
		if !recipientInHub {
			notification := map[string]interface{}{
				"type":            "new_message",
				"conversation_id": conversationID,
				"listing_id":      conversation.ListingID,
				"sender_id":       c.UserID,
				"sender_name":     c.Name,
				"content":         incoming.Content,
				"sent_at":         time.Now(),
			}
			notifBytes, _ := json.Marshal(notification)
			Manager.NotifyUser(recipientID, notifBytes)
		}

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
			ListingID:  fmt.Sprintf("%d", conversation.ListingID),
			SentAt:     time.Now(),
		}
		outBytes, _ := json.Marshal(out)
		if c.Hub != nil {
			c.Hub.broadcast <- outBytes
		}
	}
}

func parseUnit(s string) uint {
	var n uint64
	fmt.Sscanf(s, "%d", &n)
	return uint(n)
}
