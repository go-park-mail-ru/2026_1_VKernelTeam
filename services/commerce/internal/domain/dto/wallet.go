package dto

//go:generate easyjson -all $GOFILE

import "time"

// WalletResponse — баланс кошелька.
type WalletResponse struct {
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
}

// TopupWalletRequest — тело POST /wallet/topup.
type TopupWalletRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

// TopupWalletResponse — ответ POST /wallet/topup.
//
// Когда платёж синхронный (mock или ЮКасса auto-capture): status='succeeded',
// balance — актуальный, confirmation_url пустой.
//
// Когда платёж асинхронный (ЮКасса): status='pending', balance=0, фронт
// должен сделать редирект на confirmation_url. После возврата с return_url
// фронт может опросить GET /wallet/payments/{id} для финального статуса.
type TopupWalletResponse struct {
	Balance         int64  `json:"balance"`
	PaymentID       int64  `json:"payment_id"`
	Status          string `json:"status"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

// PaymentStatusResponse — текущее состояние платежа.
type PaymentStatusResponse struct {
	PaymentID       int64  `json:"payment_id"`
	Status          string `json:"status"`
	Amount          int64  `json:"amount"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

// WalletTransactionResponse — операция в ленте кошелька.
type WalletTransactionResponse struct {
	ID          int64     `json:"id"`
	Amount      int64     `json:"amount"`
	Type        string    `json:"type"`
	ReferenceID *int64    `json:"reference_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// WalletTransactionListResponse — лента операций.
type WalletTransactionListResponse struct {
	Items      []WalletTransactionResponse `json:"items"`
	NextCursor *int64                      `json:"next_cursor,omitempty"`
}
