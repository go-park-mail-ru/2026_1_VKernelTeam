// Package models содержит структуры данных, используемые в домене
// приложения.
package models

import "time"

// Ad содержит поля объявления.
const (
	AdStatusActive   = "active"
	AdStatusDraft    = "draft"
	AdStatusReserved = "reserved"
	AdStatusSold     = "sold"
	AdStatusArchived = "archived"
)

type Ad struct {
	ID             int64     `json:"id"`
	SellerID       int64     `json:"seller_id"`
	CategoryID     int64     `json:"category_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Price          int64     `json:"price"`
	Status         string    `json:"status"`
	ViewsCount     int64     `json:"views_count"`
	FavoritesCount int64     `json:"favorites_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DeletedAt      time.Time `json:"deleted_at"`
	Photos         []string  `json:"photos"`
	Location       string    `json:"location"`
}
