package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/tasks"
	"github.com/khanqais/tradexa/websocket"
)

func HandleSimulateDeliveryTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.DeliveryPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	var order models.Order
	if err := config.DB.First(&order, payload.OrderID).Error; err != nil {
		log.Printf("[DeliveryWorker] Order %d not found, dropping task.", payload.OrderID)
		return nil
	}

	if order.Status == models.OrderStatusDelivered || order.Status == models.OrderStatusCancelled {
		return nil
	}

	// Mark as delivered
	order.Status = models.OrderStatusDelivered
	config.DB.Save(&order)

	// Mock Escrow Release
	ReleaseFundsToSeller(order.Amount, order.SellerID)

	// Notify both parties
	buyerMsg := fmt.Sprintf(`{"type":"delivery_update", "message":"Your item (Order %d) has been delivered!"}`, order.ID)
	sellerMsg := fmt.Sprintf(`{"type":"delivery_update", "message":"Item delivered! Funds for Order %d have been released to your bank account."}`, order.ID)
	
	websocket.Manager.NotifyUser(order.WinnerID, []byte(buyerMsg))
	websocket.Manager.NotifyUser(order.SellerID, []byte(sellerMsg))

	log.Printf("[DeliveryWorker] Successfully delivered order %d and released funds.", order.ID)
	return nil
}

func ReleaseFundsToSeller(amount float64, sellerID uint) {
	// In production, use Cashfree Transfers/Payouts API here
	log.Printf("[ESCROW RELEASE] Transferring ₹%.2f to Seller ID %d...", amount, sellerID)
}
