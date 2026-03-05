package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/khanqais/tradexa/websocket"
)

// GET /api/ws/conversation/:conversationId — protected
// upgrades the HTTP connection to WebSocket and starts read/write pumps for a specific conversation
func ConversationHandler(c *gin.Context) {
	conversationID := c.Param("conversationId")

	// Read user info injected by JWT middleware
	rawID, exist := c.Get("user_id")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}
	rawName, _ := c.Get("name")
	userID := uint(rawID.(float64))
	userName, _ := rawName.(string)

	// Upgrade HTTP → WebSocket
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "websocket upgrade failed",
		})
		return
	}

	// Get existing hub for this conversation, or create a new one if first user
	hub := ws.Manager.GetOrCreateConversation(conversationID)

	// Create a new Client representing this specific connected user
	client := &ws.Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Name:   userName,
	}

	// Register this client with the hub
	hub.Register <- client

	// Also register globally for notifications
	ws.Manager.RegisterClient(userID, client)

	// Start WritePump in a goroutine — sends messages FROM hub TO this browser
	go client.WritePump()
	go client.ReadPumpForConversation(conversationID)
}
