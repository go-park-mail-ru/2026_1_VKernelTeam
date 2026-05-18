package dto

import "time"

// PromotionPlanResponse — тариф продвижения для публичной выдачи.
type PromotionPlanResponse struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	Kind         string `json:"kind"`
	DurationDays int    `json:"duration_days"`
	Price        int64  `json:"price"`
}

// PurchasePromotionRequest — тело POST /ads/{id}/promotions.
type PurchasePromotionRequest struct {
	PlanCode       string `json:"plan_code"`
	IdempotencyKey string `json:"idempotency_key"`
}

// PromotionResponse — карточка купленного промо.
type PromotionResponse struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Kind      string    `json:"kind"`
	PlanCode  string    `json:"plan_code"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	PricePaid int64     `json:"price_paid"`
}

// PurchasePromotionResponse — ответ POST /ads/{id}/promotions.
type PurchasePromotionResponse struct {
	Promotion     PromotionResponse `json:"promotion"`
	WalletBalance int64             `json:"wallet_balance"`
}

// PromotionListResponse — ответ GET /ads/{id}/promotions и /profile/promotions.
type PromotionListResponse struct {
	Items      []PromotionResponse `json:"items"`
	NextCursor *int64              `json:"next_cursor,omitempty"`
}
