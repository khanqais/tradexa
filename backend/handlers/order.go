package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/workers"
)

func MarkOrderShipped(c *gin.Context) {
	orderID := c.Param("id")
	rawID, _ := c.Get("user_id")
	userID := uint(rawID.(float64))

	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.SellerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the seller can mark this order as shipped"})
		return
	}

	if order.Status != models.OrderStatusPaidInEscrow {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is not paid or already shipped"})
		return
	}

	order.Status = models.OrderStatusShipped
	config.DB.Save(&order)

	go func(orderID uint) {
		time.Sleep(2 * time.Minute)
		workers.SimulateDelivery(orderID)
	}(order.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Item shipped! Delivery expected in 2 minutes."})
}
