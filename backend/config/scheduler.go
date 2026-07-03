package config

import (
	"fmt"
	"log"
	"os"
	"time"

	qstash "github.com/upstash/qstash-go"
)

// ScheduleAuctionClose schedules the auction close job for listingID at time `at`.
// - In production (BACKEND_URL + QSTASH_TOKEN set): publishes to QStash, stores messageID in DB.
// - In dev (no BACKEND_URL or no token): falls back to an in-process time.AfterFunc.
//
// Returns the QStash messageID (empty string in dev fallback mode).
func ScheduleAuctionClose(listingID uint, at time.Time) string {
	backendURL := os.Getenv("BACKEND_URL")

	// --- Production path: QStash ---
	if QStashClient != nil && backendURL != "" {
		callbackURL := fmt.Sprintf("%s/api/internal/auction-close", backendURL)

		res, err := QStashClient.PublishJSON(qstash.PublishJSONOptions{
			Url:       callbackURL,
			Body:      map[string]any{"listing_id": listingID},
			NotBefore: fmt.Sprintf("%d", at.Unix()),
		})
		if err != nil {
			log.Printf("[Scheduler] QStash publish failed for listing %d: %v — falling back to in-process timer", listingID, err)
		} else {
			log.Printf("[Scheduler] QStash job scheduled for listing %d at %v (msgID=%s)", listingID, at, res.MessageId)
			// Save messageID back to DB so we can cancel it on anti-snipe
			DB.Model(&map[string]interface{}{}).
				Table("listings").
				Where("id = ?", listingID).
				Update("qstash_message_id", res.MessageId)
			return res.MessageId
		}
	}

	// --- Dev fallback path: in-process timer ---
	delay := time.Until(at)
	if delay < 0 {
		delay = 0
	}
	log.Printf("[Scheduler] Using in-process timer for listing %d (fires in %v)", listingID, delay)
	time.AfterFunc(delay, func() {
		triggerAuctionClose(listingID)
	})
	return ""
}

// CancelAuctionClose cancels a previously scheduled QStash message.
// Safe to call with an empty messageID (no-op in dev mode).
func CancelAuctionClose(messageID string) {
	if messageID == "" || QStashClient == nil {
		return
	}
	if err := QStashClient.Messages().Cancel(messageID); err != nil {
		log.Printf("[Scheduler] Failed to cancel QStash message %s: %v", messageID, err)
	} else {
		log.Printf("[Scheduler] Cancelled QStash message %s", messageID)
	}
}

// triggerAuctionClose is called by the dev in-process timer.
// It's set by workers package via SetAuctionCloseHandler to avoid import cycles.
var triggerAuctionClose func(listingID uint)

// SetAuctionCloseHandler wires the workers package callback.
func SetAuctionCloseHandler(fn func(listingID uint)) {
	triggerAuctionClose = fn
}

// broadcastSSE is set by handlers.StreamHub to avoid a workers→handlers import cycle.
var broadcastSSE func(listingID uint, payload []byte)

// SetSSEBroadcaster registers the SSE broadcast function from the handlers package.
func SetSSEBroadcaster(fn func(listingID uint, payload []byte)) {
	broadcastSSE = fn
}

// BroadcastSSE sends a server-sent event to all listeners for a listing.
// Safe to call even before the broadcaster is registered (no-op).
func BroadcastSSE(listingID uint, payload []byte) {
	if broadcastSSE != nil {
		broadcastSSE(listingID, payload)
	}
}
