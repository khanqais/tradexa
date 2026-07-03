package workers

import (
	"log"
	"time"

	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
)

// StartAuctionSweeper runs a background goroutine that every 5 minutes
// finds any active auctions that have expired but weren't closed (e.g., the
// server restarted exactly when the QStash event fired and missed it).
//
// 99% of the time this query returns 0 rows — QStash already handled them.
// It's a cheap DB query that acts as a safety net.
func StartAuctionSweeper() {
	log.Println("[Sweeper] Auction sweeper started (interval: 5 minutes)")
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run once immediately on startup to catch any auctions missed while the server was down.
	runSweep()

	for range ticker.C {
		runSweep()
	}
}

func runSweep() {
	var expiredListings []models.Listing
	err := config.DB.
		Where("type = ? AND status = ? AND is_sold = ? AND auction_ends_at <= ?",
			models.ListingTypeAuction, "active", false, time.Now()).
		Find(&expiredListings).Error

	if err != nil {
		log.Printf("[Sweeper] DB query failed: %v", err)
		return
	}

	if len(expiredListings) == 0 {
		return // nothing to do — the happy path
	}

	log.Printf("[Sweeper] Found %d expired auction(s) that need closing", len(expiredListings))
	for _, listing := range expiredListings {
		log.Printf("[Sweeper] Closing missed auction listing_id=%d", listing.ID)
		ProcessAuctionClose(listing)
	}
}
