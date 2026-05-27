// Package models содержит структуры данных, используемые в домене
// приложения.
package models

//go:generate easyjson -all $GOFILE

import "time"

// MessageType определяет тип сообщения в чате
const (
	MessageTypeText   = "text"
	MessageTypeOrder  = "order"
	MessageTypeSystem = "system"
)

// Chat представляет чат между двумя пользователями
type Chat struct {
	ID        int64     `json:"id"`
	BuyerID   int64     `json:"buyer_id"`
	SellerID  int64     `json:"seller_id"`
	AdID      int64     `json:"ad_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message представляет сообщение в чате
type Message struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	Text      string    `json:"text"`
	Type      string    `json:"type"` // 'text' или 'order'
	CreatedAt time.Time `json:"created_at"`
}
