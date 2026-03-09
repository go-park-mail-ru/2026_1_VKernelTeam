// Package models содержит структуры данных, используемые в домене
// приложения.
package models

import "time"

// Ad содержит поля объявления.
type Ad struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Photos      []string  `json:"photos"`
	Tags        []string  `json:"tags"`
	SellerID    int       `json:"seller_id"`
	CreatedAt   time.Time `json:"created_at"`
	Views       int       `json:"views"`
}
