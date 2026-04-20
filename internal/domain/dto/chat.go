// Package dto содержит объекты передачи данных (Data Transfer Objects)
package dto

import "time"

// CreateOrderRequest представляет запрос на создание заказа
type CreateOrderRequest struct {
	AdID int64 `json:"ad_id" validate:"required,gt=0"`
}

// OrderResponse представляет ответ при создании заказа
type OrderResponse struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}

// ChatResponse представляет информацию о чате
type ChatResponse struct {
	ID        int64     `json:"id"`
	BuyerID   int64     `json:"buyer_id"`
	SellerID  int64     `json:"seller_id"`
	AdID      int64     `json:"ad_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MessageResponse представляет сообщение в чате
type MessageResponse struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	Text      string    `json:"text"`
	Type      string    `json:"type"` // 'text' или 'order'
	CreatedAt time.Time `json:"created_at"`
}

type ConfirmPurchaseRequest struct {
	// TODO: Добавить нужные поля при необходимости
}
