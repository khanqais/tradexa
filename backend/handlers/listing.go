package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/tasks"
	"github.com/khanqais/tradexa/utils"
	ws "github.com/khanqais/tradexa/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func maskName(name string, id uint) string {
	words := strings.Fields(name)
	firstWord := name
	if len(words) > 0 {
		firstWord = words[0]
	}

	if len(firstWord) <= 1 {
		firstWord = firstWord + "x"
	}

	firstLetter := string(firstWord[0])
	lastLetter := string(firstWord[len(firstWord)-1])

	hash := sha256.Sum256([]byte(fmt.Sprintf("secret_salt_%d", id)))
	uniqueTag := fmt.Sprintf("%x", hash[0:2])

	return fmt.Sprintf("%s***%s-%s", strings.ToUpper(firstLetter), strings.ToLower(lastLetter), uniqueTag[:4])
}

type CreateListingInput struct {
	Title         string     `json:"title" binding:"required,min=3"`
	Description   string     `json:"description" binding:"required,min=10"`
	Price         float64    `json:"price" binding:"required,gt=0"`
	ReservePrice  float64    `json:"reserve_price"`
	Type          string     `json:"type" binding:"required,oneof=fixed auction"`
	Category      string     `json:"category" binding:"required"`
	ImageURLs     []string   `json:"image_urls"`
	AuctionEndsAt *time.Time `json:"auction_ends_at"`
}

type UpdateListingInput struct {
	Title        string   `json:"title" binding:"omitempty,min=3"`
	Description  string   `json:"description" binding:"omitempty,min=10"`
	Price        float64  `json:"price" binding:"omitempty,gt=0"`
	ReservePrice float64  `json:"reserve_price"`
	Category     string   `json:"category"`
	ImageURLs    []string `json:"image_urls"`
}

type NewBid struct {
	Amount    uint `json:"amount" binding:"required"`
	ListingID uint `json:"listing_id" binding:"required"`
}

func CreateListing(c *gin.Context) {
	var input CreateListingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if input.Type == "auction" && input.AuctionEndsAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction_ends_at is required for auction listings"})
		return
	}

	if input.AuctionEndsAt != nil && input.AuctionEndsAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction_ends_at must be in the future"})
		return
	}
	rawID, _ := c.Get("user_id")
	sellerID := uint(rawID.(float64))
	listing := models.Listing{
		Title:         input.Title,
		Description:   input.Description,
		Price:         input.Price,
		ReservePrice:  input.ReservePrice,
		Type:          models.ListingType(input.Type),
		Category:      input.Category,
		ImageURL:      "",
		SellerID:      sellerID,
		AuctionEndsAt: input.AuctionEndsAt,
	}
	if err := config.DB.WithContext(c.Request.Context()).Create(&listing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create ",
		})
		return
	}
	for _, imageURL := range input.ImageURLs {
		listingImage := models.ListingImage{
			ListingID: listing.ID,
			URL:       imageURL,
		}
		config.DB.WithContext(c.Request.Context()).Create(&listingImage)
	}

	config.DB.WithContext(c.Request.Context()).Preload("Seller").Preload("Images").First(&listing, listing.ID)
	listing.Seller.Password = ""

	if listing.Type == models.ListingTypeAuction && listing.AuctionEndsAt != nil {
		task, err := tasks.NewAuctionCloseTask(listing.ID)
		if err == nil {
			config.AsynqClient.Enqueue(task, asynq.ProcessAt(*listing.AuctionEndsAt))
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"listing": listing,
	})
}

func GetListingByID(c *gin.Context) {
	id := c.Param("id")

	var listing models.Listing
	if err := config.DB.WithContext(c.Request.Context()).Preload("Seller").Preload("Images").First(&listing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listing"})
		return
	}

	listing.Seller.Password = ""

	rawID, _ := c.Get("user_id")
	if rawID != nil && listing.IsSold {
		userID := uint(rawID.(float64))
		if listing.SellerID == userID {
			var order models.Order
			if err := config.DB.Where("listing_id = ? AND status IN ?", listing.ID, []models.OrderStatus{models.OrderStatusPaidInEscrow, models.OrderStatusShipped, models.OrderStatusDelivered}).First(&order).Error; err == nil {
				listing.Order = &order
			}
		}
	}

	var highestBid models.Bid
	if err := config.DB.WithContext(c.Request.Context()).
		Where("listing_id = ?", listing.ID).
		Order("amount desc").
		First(&highestBid).Error; err == nil {
		listing.HighestBid = &highestBid.Amount

		var bidder models.User
		if err := config.DB.First(&bidder, highestBid.BidderID).Error; err == nil {
			listing.HighestBidder = maskName(bidder.Name, bidder.ID)
		}
	}

	rawID, exist := c.Get("user_id")
	if exist {
		bidderID := uint(rawID.(float64))
		var proxy models.ProxyBid
		if err := config.DB.Where("listing_id = ? AND bidder_id = ?", listing.ID, bidderID).First(&proxy).Error; err == nil {
			listing.UserMaxBid = &proxy.MaxAmount
		}
	}

	c.JSON(http.StatusOK, gin.H{"listing": listing})
}

func GetListings(c *gin.Context) {
	var listings []models.Listing

	query := config.DB.WithContext(c.Request.Context()).Preload("Seller").Preload("Images")

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			"%"+strings.ToLower(search)+"%",
			"%"+strings.ToLower(search)+"%",
		)
	}

	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("LOWER(category) = ?", strings.ToLower(category))
	}

	if listingType := c.Query("type"); listingType == "auction" || listingType == "fixed" {
		query = query.Where("type = ?", listingType)
	}

	if sold := c.Query("sold"); sold == "true" {
		query = query.Where("is_sold = ?", true)
	} else {
		query = query.Where("is_sold = ?", false)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int64
	if err := query.Model(&models.Listing{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count listings"})
		return
	}

	if err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&listings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}

	var auctionIDs []uint
	for i := range listings {
		listings[i].Seller.Password = ""
		if listings[i].Type == models.ListingTypeAuction {
			auctionIDs = append(auctionIDs, listings[i].ID)
		}
	}

	if len(auctionIDs) > 0 {
		type highestBidResult struct {
			ListingID uint
			MaxAmount float64
		}
		var results []highestBidResult
		config.DB.WithContext(c.Request.Context()).
			Model(&models.Bid{}).
			Select("listing_id, MAX(amount) as max_amount").
			Where("listing_id IN ?", auctionIDs).
			Group("listing_id").
			Find(&results)

		bidMap := make(map[uint]float64, len(results))
		for _, r := range results {
			bidMap[r.ListingID] = r.MaxAmount
		}
		for i := range listings {
			if amt, ok := bidMap[listings[i].ID]; ok {
				listings[i].HighestBid = &amt
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"listings": listings,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}

func UpdateListing(c *gin.Context) {
	id := c.Param("id")

	var listing models.Listing
	if err := config.DB.WithContext(c.Request.Context()).First(&listing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listing"})
		return
	}

	rawID, _ := c.Get("user_id")
	sellerID := uint(rawID.(float64))
	if listing.SellerID != sellerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this listing"})
		return
	}

	var input UpdateListingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Title != "" {
		updates["title"] = input.Title
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Price > 0 {
		updates["price"] = input.Price
	}
	if input.Category != "" {
		updates["category"] = input.Category
	}
	if len(input.ImageURLs) > 0 {

		config.DB.WithContext(c.Request.Context()).Where("listing_id = ?", listing.ID).Delete(&models.ListingImage{})
		for _, imageURL := range input.ImageURLs {
			listingImage := models.ListingImage{
				ListingID: listing.ID,
				URL:       imageURL,
			}
			config.DB.WithContext(c.Request.Context()).Create(&listingImage)
		}
	}
	if input.ReservePrice > 0 {
		updates["reserve_price"] = input.ReservePrice
	}

	if err := config.DB.WithContext(c.Request.Context()).Model(&listing).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update listing"})
		return
	}

	config.DB.WithContext(c.Request.Context()).Preload("Images").First(&listing, listing.ID)
	c.JSON(http.StatusOK, gin.H{"listing": listing})
}

func DeleteListing(c *gin.Context) {
	id := c.Param("id")

	var listing models.Listing
	if err := config.DB.WithContext(c.Request.Context()).First(&listing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listing"})
		return
	}

	rawID, _ := c.Get("user_id")
	sellerID := uint(rawID.(float64))
	if listing.SellerID != sellerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this listing"})
		return
	}

	if err := config.DB.WithContext(c.Request.Context()).Delete(&listing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete listing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "listing deleted successfully"})
}

func BidHandler(c *gin.Context) {
	var input NewBid
	err := c.ShouldBindJSON(&input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rawID, exist := c.Get("user_id")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	bidderID := uint(rawID.(float64))

	tx := config.DB.WithContext(c.Request.Context()).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var listing models.Listing
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&listing, input.ListingID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	if listing.Type != models.ListingTypeAuction {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "this item is not up for auction"})
		return
	}

	if listing.IsSold {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "this auction is already closed"})
		return
	}

	if listing.AuctionEndsAt != nil && listing.AuctionEndsAt.Before(time.Now()) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "this auction has expired"})
		return
	}

	if listing.SellerID == bidderID {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "you cannot bid on your own listing"})
		return
	}

	var highestPublicBid models.Bid
	hasPublicBid := true
	if err := tx.Where("listing_id = ?", input.ListingID).Order("amount desc").First(&highestPublicBid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hasPublicBid = false
		} else {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	currentPublicPrice := listing.Price
	if hasPublicBid {
		currentPublicPrice = highestPublicBid.Amount
	}

	bidIncrement := 5.0
	minRequiredBid := listing.Price
	if hasPublicBid {
		minRequiredBid = currentPublicPrice + bidIncrement
	}

	var currentProxy models.ProxyBid
	hasProxy := true
	if err := tx.Where("listing_id = ?", input.ListingID).First(&currentProxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hasProxy = false
		} else {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	isSelfBump := hasProxy && currentProxy.BidderID == bidderID

	if !isSelfBump && float64(input.Amount) < minRequiredBid {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Your max bid must be at least %.2f", minRequiredBid)})
		return
	}

	var finalPublicBidAmount float64
	var winningBidderID uint
	var outbidUserID uint

	if !hasProxy {
		finalPublicBidAmount = currentPublicPrice
		winningBidderID = bidderID

		proxy := models.ProxyBid{
			ListingID: listing.ID,
			BidderID:  bidderID,
			MaxAmount: float64(input.Amount),
		}
		if err := tx.Create(&proxy).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create proxy"})
			return
		}

		newBid := models.Bid{
			ListingID: listing.ID,
			BidderID:  bidderID,
			Amount:    finalPublicBidAmount,
		}
		tx.Create(&newBid)

	} else {
		if currentProxy.BidderID == bidderID {
			if float64(input.Amount) <= currentProxy.MaxAmount {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "new max bid must be higher than your current max bid"})
				return
			}
			currentProxy.MaxAmount = float64(input.Amount)
			tx.Save(&currentProxy)
			finalPublicBidAmount = currentPublicPrice
			winningBidderID = bidderID

		} else {
			newUserMax := float64(input.Amount)
			oldProxyMax := currentProxy.MaxAmount

			if newUserMax > oldProxyMax {
				outbidUserID = currentProxy.BidderID
				fightPrice := oldProxyMax + bidIncrement
				if fightPrice > newUserMax {
					fightPrice = newUserMax
				}

				tx.Create(&models.Bid{ListingID: listing.ID, BidderID: currentProxy.BidderID, Amount: oldProxyMax})
				tx.Create(&models.Bid{ListingID: listing.ID, BidderID: bidderID, Amount: fightPrice})

				currentProxy.BidderID = bidderID
				currentProxy.MaxAmount = newUserMax
				tx.Save(&currentProxy)

				finalPublicBidAmount = fightPrice
				winningBidderID = bidderID

			} else if newUserMax < oldProxyMax {
				fightPrice := newUserMax + bidIncrement
				if fightPrice > oldProxyMax {
					fightPrice = oldProxyMax
				}

				tx.Create(&models.Bid{ListingID: listing.ID, BidderID: bidderID, Amount: newUserMax})
				tx.Create(&models.Bid{ListingID: listing.ID, BidderID: currentProxy.BidderID, Amount: fightPrice})

				finalPublicBidAmount = fightPrice
				winningBidderID = currentProxy.BidderID

			} else {
				fightPrice := oldProxyMax
				tx.Create(&models.Bid{ListingID: listing.ID, BidderID: bidderID, Amount: newUserMax})
				tx.Create(&models.Bid{ListingID: listing.ID, BidderID: currentProxy.BidderID, Amount: fightPrice})

				finalPublicBidAmount = fightPrice
				winningBidderID = currentProxy.BidderID
			}
		}
	}

	tx.Commit()

	if outbidUserID != 0 {
		go sendOutbidNotifications(listing, outbidUserID, finalPublicBidAmount)
	}

	var winner models.User
	config.DB.First(&winner, winningBidderID)

	payload := map[string]interface{}{
		"type":                "new_bid",
		"listing_id":          listing.ID,
		"amount":              finalPublicBidAmount,
		"winning_bidder_name": maskName(winner.Name, winner.ID),
	}
	if listing.AuctionEndsAt != nil {
		payload["auction_ends_at"] = *listing.AuctionEndsAt
	}
	messageBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
	}
	StreamHub.Broadcast(listing.ID, messageBytes)

	c.JSON(http.StatusOK, gin.H{
		"Message":        "bid placed successfully",
		"current_price":  finalPublicBidAmount,
		"winning_bidder": winningBidderID,
	})
}

func sendOutbidNotifications(listing models.Listing, outbidUserID uint, newPrice float64) {
	var outbidUser models.User
	if err := config.DB.First(&outbidUser, outbidUserID).Error; err != nil {
		fmt.Printf("[BidHandler] Failed to find outbid user %d: %v\n", outbidUserID, err)
		return
	}

	// 1. WebSocket Notification
	notifPayload, _ := json.Marshal(map[string]interface{}{
		"type":       "outbid",
		"listing_id": listing.ID,
		"title":      listing.Title,
		"new_price":  newPrice,
	})
	ws.Manager.NotifyUser(outbidUserID, notifPayload)

	// 2. Email Notification
	subject := fmt.Sprintf("⚠️ You have been outbid on \"%s\"!", listing.Title)
	body := fmt.Sprintf(`
	<div style="font-family: Arial, sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; border: 1px solid #e0e0e0; border-radius: 5px;">
		<h2 style="color: #333;">You've been outbid, %s!</h2>
		<p>Someone placed a higher bid on <strong>%s</strong>.</p>
		<div style="font-size: 20px; font-weight: bold; background-color: #fef08a; padding: 15px; border-radius: 4px; text-align: center; margin: 20px 0; color: #854d0e;">
			Current Bid: $%.0f
		</div>
		<p>Don't let it get away! Log back in to increase your max bid and stay in the lead.</p>
		<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
		<p style="color: #999; font-size: 12px; text-align: center;">Tradexa &copy; 2026. All rights reserved.</p>
	</div>
	`, outbidUser.Name, listing.Title, newPrice)

	if err := utils.SendEmail(outbidUser.Email, subject, body); err != nil {
		fmt.Printf("[BidHandler] Failed to send outbid email to %s: %v\n", outbidUser.Email, err)
	}
}
