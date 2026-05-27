// Package models содержит структуры данных, используемые в домене catalog.
package models

//go:generate easyjson -all $GOFILE

import "time"

// Возможные значения статуса объявления.
const (
	AdStatusActive            = "active"
	AdStatusDraft             = "draft"
	AdStatusReserved          = "reserved"
	AdStatusSold              = "sold"
	AdStatusArchived          = "archived"
	AdStatusAdminDeleted      = "admin_deleted"
	AdStatusPendingModeration = "pending_moderation"
	AdStatusRejected          = "rejected"
)

// Ad содержит поля объявления.
type Ad struct {
	ID                      int64                         `json:"id"`
	SellerID                int64                         `json:"seller_id"`
	CategoryID              int64                         `json:"category_id"`
	Title                   string                        `json:"title"`
	Description             string                        `json:"description"`
	Price                   int64                         `json:"price"`
	Status                  string                        `json:"status"`
	ViewsCount              int64                         `json:"views_count"`
	FavoritesCount          int64                         `json:"favorites_count"`
	CreatedAt               time.Time                     `json:"created_at"`
	UpdatedAt               time.Time                     `json:"updated_at"`
	DeletedAt               time.Time                     `json:"deleted_at"`
	Photos                  []string                      `json:"photos"`
	Location                string                        `json:"location"`
	Lat                     *float64                      `json:"lat,omitempty"`
	Lon                     *float64                      `json:"lon,omitempty"`
	IsBoosted               bool                          `json:"is_boosted"`
	IsHighlighted           bool                          `json:"is_highlighted"`
	CategoryCharacteristics []ProductCharacteristic       `json:"category_characteristics"`
	CustomCharacteristics   []ProductCustomCharacteristic `json:"custom_characteristics"`
}
