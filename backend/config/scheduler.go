package config

import (
	"fmt"
	"log"
	"os"
	"time"

	qstash "github.com/upstash/qstash-go"
)

func ScheduleAuctionClose(listingID uint, at time.Time) string {
	backendURL := os.Getenv("BACKEND_URL")

	if QStashClient != nil && backendURL != "" {
		callbackURL := fmt.Sprintf("%s/api/internal/auction-close", backendURL)

		res, err := QStashClient.PublishJSON(qstash.PublishJSONOptions{
			Url:		callbackURL,
			Body:		map[string]any{"listing_id": listingID},
			NotBefore:	fmt.Sprintf("%d", at.Unix()),
			Retries:	qstash.RetryCount(3),
		})
		if err != nil {
			log.Printf("[Scheduler] QStash publish failed for listing %d: %v — falling back to in-process timer", listingID, err)
		} else {
			log.Printf("[Scheduler] QStash job scheduled for listing %d at %v (msgID=%s)", listingID, at, res.MessageId)

			if dbErr := DB.Table("listings").Where("id = ?", listingID).Update("qstash_message_id", res.MessageId).Error; dbErr != nil {
				log.Printf("[Scheduler] Warning: failed to save QStash msgID for listing %d: %v", listingID, dbErr)
			}
			return res.MessageId
		}
	}

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

var triggerAuctionClose func(listingID uint)

func SetAuctionCloseHandler(fn func(listingID uint)) {
	triggerAuctionClose = fn
}

var broadcastSSE func(listingID uint, payload []byte)

func SetSSEBroadcaster(fn func(listingID uint, payload []byte)) {
	broadcastSSE = fn
}

func BroadcastSSE(listingID uint, payload []byte) {
	if broadcastSSE != nil {
		broadcastSSE(listingID, payload)
	}
}
