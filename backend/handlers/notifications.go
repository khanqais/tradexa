package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/khanqais/tradexa/websocket"
)

func NotificationHandler(c *gin.Context) {
	log.Println("[DEBUG] NotificationHandler called")
	log.Printf("[DEBUG] Query: %v\n", c.Request.URL.Query())
	log.Printf("[DEBUG] Headers: %v\n", c.Request.Header)

	rawID, exist := c.Get("user_id")
	if !exist {
		log.Println("[ERROR] user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	log.Printf("[DEBUG] user_id: %v\n", rawID)

	userID := uint(rawID.(float64))
	log.Printf("[DEBUG] Converting userID to uint: %d\n", userID)

	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket upgrade failed: %v\n", err)
		return
	}
	log.Println("[DEBUG] WebSocket upgraded successfully")

	client := &ws.Client{
		Hub:    nil,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	ws.Manager.RegisterClient(userID, client)
	log.Printf("[DEBUG] Client registered: userID=%d\n", userID)

	go client.WritePump()

	go func() {
		defer func() {
			ws.Manager.UnregisterClient(userID, client)
			conn.Close()
			log.Printf("[DEBUG] Client unregistered: userID=%d\n", userID)
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				log.Printf("[DEBUG] ReadMessage error: %v\n", err)
				break
			}
		}
	}()
}
