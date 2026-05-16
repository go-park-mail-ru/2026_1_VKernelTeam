package models

import "time"

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
