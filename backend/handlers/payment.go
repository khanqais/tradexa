package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/websocket"
)

type CreatePaymentRequest struct {
	Amount    float64 `json:"amount"` // Ignored by server, calculated server-side
	ListingID uint    `json:"listing_id" binding:"required"`
}

func CreateCashfreeOrder(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	appID := os.Getenv("CASHFREE_APP_ID")
	secretKey := os.Getenv("CASHFREE_SECRET_KEY")

	if appID == "" || secretKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cashfree credentials are not configured"})
		return
	}

	rawID, _ := c.Get("user_id")
	userID := uint(rawID.(float64))
	email, _ := c.Get("email")
	phone := "9999999999"

	var listing models.Listing
	if err := config.DB.First(&listing, req.ListingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	var amount float64
	if listing.Type == models.ListingTypeFixed {
		amount = listing.Price
	} else if listing.Type == models.ListingTypeAuction {
		var highestBid models.Bid
		if err := config.DB.Where("listing_id = ?", listing.ID).Order("amount DESC").First(&highestBid).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No bids found for this auction"})
			return
		}
		if highestBid.BidderID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the winning bidder can pay for this auction"})
			return
		}
		amount = highestBid.Amount
	} else {
		amount = listing.Price
	}

	order := models.Order{
		ListingID: req.ListingID,
		WinnerID:  userID,
		SellerID:  listing.SellerID,
		Amount:    amount,
		Status:    models.OrderStatusPendingPayment,
	}

	if err := config.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	orderID := fmt.Sprintf("txn_%d_%d", order.ID, time.Now().Unix())

	payload := map[string]interface{}{
		"order_amount":   amount,
		"order_currency": "INR",
		"order_id":       orderID,
		"customer_details": map[string]string{
			"customer_id":    fmt.Sprintf("user_%d", userID),
			"customer_phone": phone,
			"customer_email": fmt.Sprintf("%v", email),
		},
		"order_meta": map[string]string{
			"return_url": os.Getenv("FRONTEND_URL") + "/payment-status?order_id={order_id}",
		},
	}

	bodyBytes, _ := json.Marshal(payload)

	request, err := http.NewRequest("POST", "https://sandbox.cashfree.com/pg/orders", bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build request"})
		return
	}

	request.Header.Set("accept", "application/json")
	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-api-version", "2023-08-01")
	request.Header.Set("x-client-id", appID)
	request.Header.Set("x-client-secret", secretKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to Cashfree"})
		return
	}
	defer response.Body.Close()

	resBytes, _ := io.ReadAll(response.Body)
	var resData map[string]interface{}
	json.Unmarshal(resBytes, &resData)

	if response.StatusCode != http.StatusOK {
		c.JSON(response.StatusCode, gin.H{"error": "Payment gateway error", "details": resData})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_session_id": resData["payment_session_id"],
		"order_id":           orderID,
	})
}

type VerifyPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

func VerifyPayment(c *gin.Context) {
	var req VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	parts := strings.Split(req.OrderID, "_")
	if len(parts) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}
	localOrderID, _ := strconv.Atoi(parts[1])

	var order models.Order
	if err := config.DB.First(&order, localOrderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	appID := os.Getenv("CASHFREE_APP_ID")
	secretKey := os.Getenv("CASHFREE_SECRET_KEY")

	request, _ := http.NewRequest("GET", "https://sandbox.cashfree.com/pg/orders/"+req.OrderID, nil)
	request.Header.Set("accept", "application/json")
	request.Header.Set("x-api-version", "2023-08-01")
	request.Header.Set("x-client-id", appID)
	request.Header.Set("x-client-secret", secretKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify with Cashfree"})
		return
	}
	defer response.Body.Close()

	resBytes, _ := io.ReadAll(response.Body)
	var resData map[string]interface{}
	json.Unmarshal(resBytes, &resData)

	status, ok := resData["order_status"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not parse order status"})
		return
	}

	if status == "PAID" {
		order.Status = models.OrderStatusPaidInEscrow
		config.DB.Save(&order)
		var listing models.Listing
		if err := config.DB.First(&listing, order.ListingID).Error; err == nil {
			listing.IsSold = true
			config.DB.Save(&listing)
		}
		websocket.Manager.NotifyUser(order.SellerID, []byte(fmt.Sprintf(`{"type":"payment_received", "message":"Buyer has paid for order %d! Funds are in Escrow. Please ship the item."}`, order.ID)))
		c.JSON(http.StatusOK, gin.H{"status": "paid_in_escrow"})
	} else if status == "ACTIVE" {
		c.JSON(http.StatusOK, gin.H{"status": "pending_payment"})
	} else {
		order.Status = models.OrderStatusFailed
		config.DB.Save(&order)
		c.JSON(http.StatusOK, gin.H{"status": "failed"})
	}
}

func WebhookPayment(c *gin.Context) {
	// Cashfree webhook signature verification
	appSecret := os.Getenv("CASHFREE_SECRET_KEY")
	if appSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cashfree secret key not configured"})
		return
	}

	timestamp := c.GetHeader("x-webhook-timestamp")
	signature := c.GetHeader("x-webhook-signature")
	if timestamp == "" || signature == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing webhook signature or timestamp headers"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Calculate signature: Base64(HMAC_SHA256(timestamp + rawBody, secretKey))
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(timestamp))
	mac.Write(bodyBytes)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		c.Status(http.StatusOK)
		return
	}
	orderMap, ok := data["order"].(map[string]interface{})
	if !ok {
		c.Status(http.StatusOK)
		return
	}

	orderID, ok := orderMap["order_id"].(string)
	if !ok {
		c.Status(http.StatusOK)
		return
	}
	
	status, _ := payload["type"].(string)

	parts := strings.Split(orderID, "_")
	if len(parts) < 2 {
		c.Status(http.StatusOK)
		return
	}
	localOrderID, _ := strconv.Atoi(parts[1])

	var order models.Order
	if err := config.DB.First(&order, localOrderID).Error; err != nil {
		c.Status(http.StatusOK)
		return
	}

	if status == "PAYMENT_SUCCESS_WEBHOOK" {
		order.Status = models.OrderStatusPaidInEscrow
		config.DB.Save(&order)
		var listing models.Listing
		if err := config.DB.First(&listing, order.ListingID).Error; err == nil {
			listing.IsSold = true
			config.DB.Save(&listing)
		}
		websocket.Manager.NotifyUser(order.SellerID, []byte(fmt.Sprintf(`{"type":"payment_received", "message":"Buyer has paid for order %d! Funds are in Escrow. Please ship the item."}`, order.ID)))
	} else if status == "PAYMENT_FAILED_WEBHOOK" {
		order.Status = models.OrderStatusFailed
		config.DB.Save(&order)
	}

	c.Status(http.StatusOK)
}
