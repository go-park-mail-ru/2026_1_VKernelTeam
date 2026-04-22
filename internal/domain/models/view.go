package models

import "time"

// ViewEvent представляет событие просмотра объявления.
type ViewEvent struct {
	MessageID string // ID сообщения в Redis Stream (для XACK)
	ProductID int64
	UserID    *int64
	ViewedAt  time.Time
}
