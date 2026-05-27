package models

//go:generate easyjson -all $GOFILE

import "time"

// PricePoint - одна точка истории цены объявления.
type PricePoint struct {
	Price     int64     `json:"price"`
	ChangedAt time.Time `json:"changed_at"`
}
