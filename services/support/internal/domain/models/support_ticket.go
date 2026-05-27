package models

//go:generate easyjson -all $GOFILE

import "time"

// SupportTicket описывает обращение пользователя в поддержку.
type SupportTicket struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Rating      *int      `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
