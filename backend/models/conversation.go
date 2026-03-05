package models

import (
	"time"
)

// Conversation represents a 1:1 chat between a buyer and seller for a specific listing
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ListingID uint      `gorm:"not null" json:"listing_id"`
	BuyerID   uint      `gorm:"not null" json:"buyer_id"`
	SellerID  uint      `gorm:"not null" json:"seller_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Listing *Listing `gorm:"foreignKey:ListingID" json:"listing,omitempty"`
	Buyer   *User    `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Seller  *User    `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
}

// TableName overrides the table name used by GORM
func (Conversation) TableName() string {
	return "conversations"
}
