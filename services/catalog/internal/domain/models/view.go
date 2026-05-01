package models

import "time"

// ViewEvent представляет событие просмотра объявления.
type ViewEvent struct {
	MessageID string
	ProductID int64
	UserID    *int64
	ViewedAt  time.Time
}
