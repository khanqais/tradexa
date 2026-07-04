package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/workers"
	qstash "github.com/upstash/qstash-go"
)

type auctionClosePayload struct {
	ListingID uint `json:"listing_id"`
}

func HandleQStashAuctionClose(c *gin.Context) {

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	if config.QStashReceiver != nil {
		backendURL := os.Getenv("BACKEND_URL")
		if backendURL == "" {
			log.Println("[QStash] ERROR: BACKEND_URL not set but QStashReceiver is configured — cannot verify signature")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server misconfiguration"})
			return
		}
		sig := c.GetHeader("Upstash-Signature")
		verifyErr := config.QStashReceiver.Verify(qstash.VerifyOptions{
			Signature:	sig,
			Body:		string(rawBody),
			Url:		backendURL + "/api/internal/auction-close",
		})
		if verifyErr != nil {
			log.Printf("[QStash] Signature verification failed: %v", verifyErr)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid QStash signature"})
			return
		}
	} else {
		log.Println("[QStash] WARNING: Receiver not configured — skipping signature check (dev mode)")
	}

	var payload auctionClosePayload
	if err := json.Unmarshal(rawBody, &payload); err != nil || payload.ListingID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var listing models.Listing
	if err := config.DB.First(&listing, payload.ListingID).Error; err != nil {
		log.Printf("[QStash] Listing %d not found, dropping job", payload.ListingID)

		c.Status(http.StatusOK)
		return
	}

	if listing.IsSold || (listing.Status != "" && listing.Status != "active") {
		log.Printf("[QStash] Listing %d already closed, dropping job", payload.ListingID)
		c.Status(http.StatusOK)
		return
	}

	log.Printf("[QStash] Received auction-close for listing %d", payload.ListingID)
	workers.ProcessAuctionClose(listing)
	c.Status(http.StatusOK)
}
