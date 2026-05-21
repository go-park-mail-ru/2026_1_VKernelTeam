package models

//go:generate easyjson -all $GOFILE

import "time"

const (
	PromotionKindBoost     = "boost"
	PromotionKindHighlight = "highlight"
)

// PromotionPlan — тариф продвижения, настраивается админом.
type PromotionPlan struct {
	ID           int64
	Code         string
	Kind         string
	DurationDays int
	Price        int64
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Promotion — купленная услуга продвижения с заданным сроком действия.
type Promotion struct {
	ID        int64
	ProductID int64
	UserID    int64
	PlanID    int64
	Kind      string
	StartsAt  time.Time
	ExpiresAt time.Time
	PricePaid int64
	CreatedAt time.Time
}
