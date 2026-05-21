package models

//go:generate easyjson -all $GOFILE

import "time"

// PaymentStatus* — возможные статусы платежа; PaymentProvider* — идентификаторы провайдеров.
const (
	PaymentStatusPending   = "pending"
	PaymentStatusSucceeded = "succeeded"
	PaymentStatusFailed    = "failed"
	PaymentStatusCancelled = "cancelled"

	PaymentProviderMock = "mock"
)

// Payment — лог пополнения кошелька через платёжного провайдера.
type Payment struct {
	ID          int64
	UserID      int64
	Amount      int64
	Status      string
	Provider    string
	ProviderRef *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
