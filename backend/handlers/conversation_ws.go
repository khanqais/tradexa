package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/khanqais/tradexa/websocket"
)

func ConversationHandler(c *gin.Context) {
	conversationID := c.Param("conversationId")

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

	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "websocket upgrade failed",
		})
		return
	}

	hub := ws.Manager.GetOrCreateConversation(conversationID)

	client := &ws.Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Name:   userName,
	}

	hub.Register <- client

	ws.Manager.RegisterClient(userID, client)

	go client.WritePump()
	go client.ReadPumpForConversation(conversationID)
}
