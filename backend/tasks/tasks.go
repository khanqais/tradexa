package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// Task name constant
const TypeAuctionClose = "auction:close"

// Payload struct
type AuctionClosePayload struct {
	ListingID uint `json:"listing_id"`
}

// NewAuctionCloseTask formats the task and its payload
func NewAuctionCloseTask(listingID uint) (*asynq.Task, error) {
	payload := AuctionClosePayload{ListingID: listingID}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// Create the task, allowing Asynq to manage it
	return asynq.NewTask(TypeAuctionClose, payloadBytes), nil
}
