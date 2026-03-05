package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	ws "github.com/khanqais/tradexa/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GET /api/ws/chat/:listingId — protected
// upgrades the HTTP connection to WebSocket and starts read/write pumps

func ChatHandler(c *gin.Context) {
	listingID := c.Param("listingId")
	//read user info injected by JWT middleware
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
	// upgrade HTTP → WebSocket
	// after this line, normal HTTP request/response is gone
	// communication is now full-duplex (both sides can send anytime)
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "websocket upgrade failed",
		})
		return
	}
	// get existing hub for this listing, or create a new one if first user
	hub := ws.Manager.GetOrCreate(listingID)
	// create a new Client representing this specific connected user
	client := &ws.Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Name:   userName,
	}
	// register this client with the hub
	// hub.Run() will pick this up and add client to its map
	hub.Register <- client

	// Also register globally for notifications
	ws.Manager.RegisterClient(userID, client)
	// hub.Run() will pick this up and add client to its map
	hub.Register <- client
	// start WritePump in a goroutine — sends messages FROM hub TO this browser
	go client.WritePump()
	go client.ReadPump(listingID)

}

// GET /api/chat/:listingId/history — public
// returns the last 100 messages for a listing's chat thread
func GetChatHistory(c *gin.Context) {
	listingID := c.Param("listingId")
	var message []models.Message
	err := config.DB.WithContext(c.Request.Context()).Where("listing_id=?", listingID).Preload("Sender").Order("created_at ASC").Limit(100).Find(&message).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch message",
		})
		return
	}
	// hide password field from every sender object in the response
	for i := range message {
		message[i].Sender.Password = ""
	}
	c.JSON(http.StatusOK, gin.H{
		"listing_id": listingID,
		"message":    message,
	})
}
