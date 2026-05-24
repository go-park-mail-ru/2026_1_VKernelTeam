package models

//go:generate easyjson -all $GOFILE

import "time"

// Review представляет отзыв одного пользователя о другом по конкретному товару.
type Review struct {
	ID         int64     `json:"id"`
	SenderID   int64     `json:"sender_id"`
	ReceiverID int64     `json:"receiver_id"`
	ProductID  int64     `json:"product_id"`
	Rating     int       `json:"rating"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
