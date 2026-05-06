package dto

import "time"

// CreateTicketRequest — запрос на создание обращения в техподдержку
type CreateTicketRequest struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTicketRequest — запрос на обновление обращения
type UpdateTicketRequest struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// TicketResponse — ответ с данными обращения
type TicketResponse struct {
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

// RateTicketRequest — запрос на оценку обращения
type RateTicketRequest struct {
	Rating int `json:"rating"`
}
