package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeAuctionClose = "auction:close"

type AuctionClosePayload struct {
	ListingID uint `json:"listing_id"`
}

func NewAuctionCloseTask(listingID uint) (*asynq.Task, error) {
	payload := AuctionClosePayload{ListingID: listingID}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeAuctionClose, payloadBytes), nil
}

const TypeSimulateDelivery = "delivery:simulate"

type DeliveryPayload struct {
	OrderID uint `json:"order_id"`
}

func NewSimulateDeliveryTask(orderID uint) (*asynq.Task, error) {
	payload := DeliveryPayload{OrderID: orderID}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeSimulateDelivery, payloadBytes), nil
}
