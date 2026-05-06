package dto

import "time"

// SendMessageRequest — запрос на отправку сообщения в чате обращения
type SendMessageRequest struct {
	Text string `json:"text"`
}

// MessageResponse — ответ с данными сообщения
type MessageResponse struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	UserID    int64     `json:"user_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ChangeStatusRequest — запрос на смену статуса обращения
type ChangeStatusRequest struct {
	Status string `json:"status"`
}

// TicketStatusResponse — ответ со сменённым статусом обращения
type TicketStatusResponse struct {
	ID        int64     `json:"id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StatsResponse — сводная статистика по обращениям
type StatsResponse struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByCategory map[string]int `json:"by_category"`
}
