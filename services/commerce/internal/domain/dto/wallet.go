package dto

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
type TopupWalletResponse struct {
	Balance   int64 `json:"balance"`
	PaymentID int64 `json:"payment_id"`
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
