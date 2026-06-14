package models

import "time"

type Bid struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ListingID uint      `gorm:"not null;index" json:"listing_id"`
	BidderID  uint      `gorm:"not null" json:"bidder_id"`
	Amount    float64   `gorm:"not null" json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
