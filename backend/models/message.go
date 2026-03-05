package models

import "time"

type Message struct {
	ID             uint         `gorm:"primaryKey" json:"id"`
	ListingID      uint         `gorm:"not null" json:"listing_id"`
	ConversationID uint         `gorm:"not null" json:"conversation_id"`
	SenderID       uint         `gorm:"not null" json:"sender_id"`
	Sender         User         `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Content        string       `gorm:"not null" json:"content"`
	CreatedAt      time.Time    `json:"created_at"`
}
