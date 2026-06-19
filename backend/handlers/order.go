package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/tasks"
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

	// 1. Update the database
	order.Status = models.OrderStatusShipped
	config.DB.Save(&order)

	// 2. Schedule mock delivery!
	task, err := tasks.NewSimulateDeliveryTask(order.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery task"})
		return
	}

	// Enqueue with a 2-minute delay
	_, err = config.AsynqClient.Enqueue(task, asynq.ProcessIn(2*time.Minute))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule delivery"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item shipped! Delivery expected in 2 minutes."})
}
