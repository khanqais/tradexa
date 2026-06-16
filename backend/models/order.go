package models

import "time"

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusCancelled      OrderStatus = "cancelled"
)

type Order struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	ListingID uint        `gorm:"not null;index" json:"listing_id"`
	Listing   Listing     `gorm:"foreignKey:ListingID" json:"listing,omitempty"`
	WinnerID  uint        `gorm:"not null" json:"winner_id"`
	Winner    User        `gorm:"foreignKey:WinnerID" json:"winner,omitempty"`
	SellerID  uint        `gorm:"not null" json:"seller_id"`
	Seller    User        `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Amount    float64     `gorm:"not null" json:"amount"`
	Status    OrderStatus `gorm:"default:pending_payment" json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
