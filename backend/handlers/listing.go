package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"gorm.io/gorm"
)

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

	var highestBid models.Bid
	if err := config.DB.WithContext(c.Request.Context()).
		Where("listing_id = ?", listing.ID).
		Order("amount desc").
		First(&highestBid).Error; err == nil {
		listing.HighestBid = &highestBid.Amount
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

	// ?sold=false — hide sold listings by default
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
	query.Model(&models.Listing{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&listings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}

	for i := range listings {
		listings[i].Seller.Password = ""
		if listings[i].Type == models.ListingTypeAuction {
			var highestBid models.Bid
			if err := config.DB.WithContext(c.Request.Context()).
				Where("listing_id = ?", listings[i].ID).
				Order("amount desc").
				First(&highestBid).Error; err == nil {
				listings[i].HighestBid = &highestBid.Amount
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	rawID, exist := c.Get("user_id")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}
	bidderID := uint(rawID.(float64))

	tx := config.DB.WithContext(c.Request.Context()).Begin()
	defer func() {
		// Something wrong happen then this func will work
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var listing models.Listing
	if err := tx.First(&listing, input.ListingID).Error; err != nil {
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

	var highestBid models.Bid
	errr := tx.Where("listing_id = ?", input.ListingID).Order("amount desc").First(&highestBid).Error

	if errr != nil {
		if errors.Is(errr, gorm.ErrRecordNotFound) {
			//No bids exist yet. First bid must be >= starting price.
			if input.Amount < uint(listing.Price) {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "first bid must be at least the starting price"})
				return
			}
		} else {
			// A real database connection error occurred.
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": errr.Error()})
			return
		}
	} else {
		// Bids already exist. New bid must be strictly > highest bid.
		if input.Amount <= uint(highestBid.Amount) {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "bid must be higher than the current highest bid"})
			return
		}
	}
	newBid := models.Bid{
		ListingID: input.ListingID,
		BidderID:  bidderID,
		Amount:    float64(input.Amount),
	}
	if err := tx.Create(&newBid).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to place bid"})
		return
	}
	tx.Commit()

	payload := map[string]interface{}{
		"type":       "new_bid",
		"listing_id": newBid.ListingID,
		"amount":     newBid.Amount,
	}
	messageBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
	}
	StreamHub.Broadcast(newBid.ListingID, messageBytes)

	c.JSON(http.StatusOK, gin.H{
		"Message": "bid placed successfully",
		"bid":     newBid,
	})

}
