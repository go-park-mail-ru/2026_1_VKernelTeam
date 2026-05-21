// Package dto содержит объекты передачи данных (Data Transfer Objects)
package dto

//go:generate easyjson -all $GOFILE

import "time"

// OrderResponse - ответ на POST /ads/{id}/order:
// id созданного/найденного чата и сообщение.
type OrderResponse struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}

// SuccessConfirmOrderResponse - ответ на POST /chats/{id}/confirm:
type SuccessConfirmOrderResponse struct {
	Message string `json:"message"`
}

// AdPreview - карточка объявления в контексте чата.
type AdPreview struct {
	ID     int64  `json:"ad_id"`
	Title  string `json:"title"`
	Price  int64  `json:"price"`
	Status string `json:"status"`
	Photo  string `json:"photo,omitempty"`
}

// UserPreview - краткая информация о собеседнике (для шапки и списка чатов).
type UserPreview struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

// LastMessagePreview - последнее сообщение чата для превью в списке.
type LastMessagePreview struct {
	Text      string    `json:"text"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatPreview - элемент списка чатов на странице /chats.
type ChatPreview struct {
	ChatID      int64               `json:"chat_id"`
	Ad          AdPreview           `json:"ad"`
	Partner     UserPreview         `json:"partner"`
	LastMessage *LastMessagePreview `json:"last_message,omitempty"`
}

// ChatListResponse - ответ на GET /chats.
type ChatListResponse struct {
	Chats []ChatPreview `json:"chats"`
}

// MessageItem - одно сообщение в ленте чата.
type MessageItem struct {
	ID        int64     `json:"id"`
	SenderID  int64     `json:"sender_id"`
	Text      string    `json:"text"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatDetailResponse - ответ на GET /chats/{id}.
type ChatDetailResponse struct {
	ChatID   int64         `json:"chat_id"`
	Ad       AdPreview     `json:"ad"`
	Partner  UserPreview   `json:"partner"`
	Messages []MessageItem `json:"messages"`
}
