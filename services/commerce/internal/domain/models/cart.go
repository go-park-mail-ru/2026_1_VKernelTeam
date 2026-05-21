package models

//go:generate easyjson -all $GOFILE

import "time"

// CartItem представляет товар в корзине пользователя.
type CartItem struct {
	UserID    int64     `json:"user_id"`
	ProductID int64     `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Для JOIN'ов
	Product *Ad `json:"product,omitempty"`
}

// Order представляет заказ
type Order struct {
	ID          int64     `json:"id"`
	BuyerID     int64     `json:"buyer_id"`
	TotalAmount int64     `json:"total_amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrderItem представляет товар в заказе
type OrderItem struct {
	OrderID         int64 `json:"order_id"`
	ProductID       int64 `json:"product_id"`
	PriceAtPurchase int64 `json:"price_at_purchase"`
	Quantity        int   `json:"quantity"`
}
