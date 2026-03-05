package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/khanqais/tradexa/websocket"
)

// GET /api/ws/notifications
// Global notification socket for logged in users
func NotificationHandler(c *gin.Context) {
	rawID, exist := c.Get("user_id")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uint(rawID.(float64))

	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Hub:    nil, // Global client doesn't belong to a specific listing hub
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	// Register globally
	ws.Manager.RegisterClient(userID, client)

	// Start write pump (to send notifications to client)
	go client.WritePump()

	// Read pump is still needed to handle pings/pongs and close events
	// but we don't expect messages FROM the client here
	go func() {
		defer func() {
			ws.Manager.UnregisterClient(userID, client)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}
