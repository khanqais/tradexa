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

// HandleQStashAuctionClose is the webhook endpoint QStash calls when an auction ends.
// It verifies the request signature then delegates to the existing auction close logic.
func HandleQStashAuctionClose(c *gin.Context) {
	// Must read raw body BEFORE any parsing — signature verification needs the exact bytes.
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// --- Signature verification ---
	if config.QStashReceiver != nil {
		sig := c.GetHeader("Upstash-Signature")
		backendURL := os.Getenv("BACKEND_URL")
		verifyErr := config.QStashReceiver.Verify(qstash.VerifyOptions{
			Signature: sig,
			Body:      string(rawBody),
			Url:       backendURL + "/api/internal/auction-close",
		})
		if verifyErr != nil {
			log.Printf("[QStash] Signature verification failed: %v", verifyErr)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid QStash signature"})
			return
		}
	} else {
		log.Println("[QStash] WARNING: Receiver not configured — skipping signature check (dev mode)")
	}

	// --- Parse payload ---
	var payload auctionClosePayload
	if err := json.Unmarshal(rawBody, &payload); err != nil || payload.ListingID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// --- Load listing and process ---
	var listing models.Listing
	if err := config.DB.First(&listing, payload.ListingID).Error; err != nil {
		log.Printf("[QStash] Listing %d not found, dropping job", payload.ListingID)
		// Return 200 so QStash doesn't retry a listing that doesn't exist
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
