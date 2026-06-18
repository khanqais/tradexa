package models

import "time"

type ProxyBid struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ListingID uint      `gorm:"not null;uniqueIndex" json:"listing_id"` // Only one reigning proxy per listing
	BidderID  uint      `gorm:"not null;index" json:"bidder_id"`
	MaxAmount float64   `gorm:"not null" json:"max_amount"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
