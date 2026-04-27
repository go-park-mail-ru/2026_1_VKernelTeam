package models

import "time"

// SupportMessage представляет сообщение в чате обращения техподдержки.
type SupportMessage struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	UserID    int64     `json:"user_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
