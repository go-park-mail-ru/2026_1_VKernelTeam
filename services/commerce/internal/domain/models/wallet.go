package models

//go:generate easyjson -all $GOFILE

import "time"

// WalletTxType* — возможные типы операций по кошельку.
const (
	WalletTxTypeTopup           = "topup"
	WalletTxTypePromotionCharge = "promotion_charge"
	WalletTxTypeRefund          = "refund"
)

// Wallet — баланс пользователя.
type Wallet struct {
	UserID    int64
	Balance   int64
	UpdatedAt time.Time
}

// WalletTransaction — операция по кошельку.
// Amount > 0 — пополнение, Amount < 0 — списание.
// IdempotencyKey уникален и защищает от дублей.
type WalletTransaction struct {
	ID             int64
	UserID         int64
	Amount         int64
	Type           string
	ReferenceID    *int64
	IdempotencyKey *string
	CreatedAt      time.Time
}
