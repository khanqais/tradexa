package workers

import (
	"fmt"
	"log"

	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/websocket"
)

// SimulateDelivery marks an order as delivered and releases funds.
// Called directly (no longer via Asynq) from handlers/order.go when seller marks shipped.
func SimulateDelivery(orderID uint) {
	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		log.Printf("[DeliveryWorker] Order %d not found", orderID)
		return
	}

	if order.Status == models.OrderStatusDelivered || order.Status == models.OrderStatusCancelled {
		return
	}

	order.Status = models.OrderStatusDelivered
	config.DB.Save(&order)

	ReleaseFundsToSeller(order.Amount, order.SellerID)

	buyerMsg := fmt.Sprintf(`{"type":"delivery_update", "message":"Your item (Order %d) has been delivered!"}`, order.ID)
	sellerMsg := fmt.Sprintf(`{"type":"delivery_update", "message":"Item delivered! Funds for Order %d have been released to your bank account."}`, order.ID)

	websocket.Manager.NotifyUser(order.WinnerID, []byte(buyerMsg))
	websocket.Manager.NotifyUser(order.SellerID, []byte(sellerMsg))

	log.Printf("[DeliveryWorker] Successfully delivered order %d and released funds.", order.ID)
}

func ReleaseFundsToSeller(amount float64, sellerID uint) {
	// In production, use Cashfree Transfers/Payouts API here
	log.Printf("[ESCROW RELEASE] Transferring ₹%.2f to Seller ID %d...", amount, sellerID)
}
