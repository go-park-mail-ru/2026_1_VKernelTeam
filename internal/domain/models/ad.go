package models

import "time"

// структура объявления
type Ad struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Photos      []string  `json:"photos"`
	Tags        []string  `json:"tags"`
	SellerID    int       `json:"seller_id"`
	CreatedAt   time.Time `json:"created_at"`
	Views       int       `json:"views"`
}
