package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"gorm.io/gorm"
)

func GetOrCreateConversation(c *gin.Context) {
	var input struct {
		ListingID uint `json:"listing_id" binding:"required"`
		BuyerID   uint `json:"buyer_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var listing models.Listing
	if err := config.DB.First(&listing, input.ListingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	sellerID := listing.SellerID

	var conversation models.Conversation
	err := config.DB.Where("listing_id = ? AND buyer_id = ? AND seller_id = ?",
		input.ListingID, input.BuyerID, sellerID).First(&conversation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {

			conversation = models.Conversation{
				ListingID: input.ListingID,
				BuyerID:   input.BuyerID,
				SellerID:  sellerID,
			}
			if err := config.DB.Create(&conversation).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create conversation"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": conversation,
	})
}

func GetConversationsForUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := uint(userID.(float64))

	var conversations []models.Conversation
	err := config.DB.Where("buyer_id = ? OR seller_id = ?", uid, uid).
		Preload("Listing").
		Preload("Buyer").
		Preload("Seller").
		Order("created_at DESC").
		Find(&conversations).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
	})
}

func GetMessagesForConversation(c *gin.Context) {
	conversationID := c.Param("conversationId")

	var messages []models.Message
	err := config.DB.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at ASC").
		Find(&messages).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}

	for i := range messages {
		messages[i].Sender.Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": conversationID,
		"messages":        messages,
	})
}
