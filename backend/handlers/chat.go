package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
)

// GET /api/chat/:listingId/history — public
// returns the last 100 messages for a listing's chat thread (for backward compatibility)
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
