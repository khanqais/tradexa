package models

import "time"

type Bid struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ListingID uint      `gorm:"not null;index;index:idx_bids_listing_amount,priority:1" json:"listing_id"`
	BidderID  uint      `gorm:"not null;index" json:"bidder_id"`
	Amount    float64   `gorm:"not null;index:idx_bids_listing_amount,priority:2" json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
