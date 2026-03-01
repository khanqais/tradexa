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
	Type          ListingType    `gorm:"default:fixed" json:"type"`
	Category      string         `json:"category"`
	ImageURL      string         `json:"image_url"`
	SellerID      uint           `gorm:"not null" json:"seller_id"`
	Seller        User           `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	AuctionEndsAt *time.Time     `json:"auction_ends_at"`
	IsSold        bool           `gorm:"default:false" json:"is_sold"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
