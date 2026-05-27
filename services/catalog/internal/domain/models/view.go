package models

//go:generate easyjson -all $GOFILE

import "time"

// ViewEvent представляет событие просмотра объявления.
type ViewEvent struct {
	MessageID string
	ProductID int64
	UserID    *int64
	ViewedAt  time.Time
}
