package handlers

import (
	"errors"
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
	ImageURL      string     `json:"image_url"`
	AuctionEndsAt *time.Time `json:"auction_ends_at"`
}

type UpdateListingInput struct {
	Title        string  `json:"title" binding:"omitempty,min=3"`
	Description  string  `json:"description" binding:"omitempty,min=10"`
	Price        float64 `json:"price" binding:"omitempty,gt=0"`
	ReservePrice float64 `json:"reserve_price"`
	Category     string  `json:"category"`
	ImageURL     string  `json:"image_url"`
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
		ImageURL:      input.ImageURL,
		SellerID:      sellerID,
		AuctionEndsAt: input.AuctionEndsAt,
	}
	if err := config.DB.WithContext(c.Request.Context()).Create(&listing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create ",
		})
		return
	}
	config.DB.WithContext(c.Request.Context()).Preload("Seller").First(&listing, listing.ID)
	listing.Seller.Password = ""

	c.JSON(http.StatusCreated, gin.H{
		"listing": listing,
	})
}

func GetListingByID(c *gin.Context) {
	id := c.Param("id")

	var listing models.Listing
	if err := config.DB.WithContext(c.Request.Context()).Preload("Seller").First(&listing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listing"})
		return
	}

	listing.Seller.Password = ""
	c.JSON(http.StatusOK, gin.H{"listing": listing})
}

func GetListings(c *gin.Context) {
	var listings []models.Listing

	query := config.DB.WithContext(c.Request.Context()).Preload("Seller")

	// ?search=laptop
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			"%"+strings.ToLower(search)+"%",
			"%"+strings.ToLower(search)+"%",
		)
	}

	// ?category=electronics
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("LOWER(category) = ?", strings.ToLower(category))
	}

	// ?type=auction OR ?type=fixed
	if listingType := c.Query("type"); listingType == "auction" || listingType == "fixed" {
		query = query.Where("type = ?", listingType)
	}

	// ?sold=false — hide sold listings by default
	if sold := c.Query("sold"); sold == "true" {
		query = query.Where("is_sold = ?", true)
	} else {
		query = query.Where("is_sold = ?", false)
	}

	// pagination: ?page=1&limit=10
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

	// hide seller passwords
	for i := range listings {
		listings[i].Seller.Password = ""
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

// PUT /listings/:id — update listing (protected, owner only)
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

	// only the owner can update
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

	// only update fields that were actually sent
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
	if input.ImageURL != "" {
		updates["image_url"] = input.ImageURL
	}
	if input.ReservePrice > 0 {
		updates["reserve_price"] = input.ReservePrice
	}

	if err := config.DB.WithContext(c.Request.Context()).Model(&listing).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update listing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"listing": listing})
}

// DELETE /listings/:id — delete listing (protected, owner only)
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
