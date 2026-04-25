package models

import "time"

type SupportTicket struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
