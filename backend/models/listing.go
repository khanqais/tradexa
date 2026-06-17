package models

import (
	"time"

	"gorm.io/gorm"
)

type ListingType string

const (
	ListingTypeFixed   ListingType = "fixed"
	ListingTypeAuction ListingType = "auction"
)

type Listing struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"not null" json:"title"`
	Description   string         `json:"description"`
	Price         float64        `gorm:"not null" json:"price"`
	ReservePrice  float64        `json:"reserve_price"`
	Type          ListingType    `gorm:"default:fixed;index:idx_listings_auction_watcher,priority:1" json:"type"`
	Category      string         `gorm:"index" json:"category"`
	ImageURL      string         `json:"image_url"`
	SellerID      uint           `gorm:"not null;index" json:"seller_id"`
	Seller        User           `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Images        []ListingImage `gorm:"foreignKey:ListingID" json:"images"`
	AuctionEndsAt *time.Time     `gorm:"index:idx_listings_auction_watcher,priority:4" json:"auction_ends_at"`
	IsSold        bool           `gorm:"default:false;index:idx_listings_is_sold_created,priority:1;index:idx_listings_auction_watcher,priority:2" json:"is_sold"`
	Status        string         `gorm:"default:active;index:idx_listings_auction_watcher,priority:3" json:"status"`
	CreatedAt     time.Time      `gorm:"index:idx_listings_is_sold_created,priority:2" json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index:idx_listings_is_sold_created,priority:3" json:"-"`
	HighestBid    *float64       `gorm:"-" json:"highest_bid,omitempty"`
}

type ListingImage struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ListingID uint   `gorm:"not null;index" json:"-"`
	URL       string `json:"url"`
	PublicID  string `json:"public_id,omitempty"`
}
