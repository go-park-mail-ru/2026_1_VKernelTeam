package models

import "time"

// User представляет зарегистрированного пользователя системы.
type User struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	PassHash       []byte    `json:"-"`
	AvatarPath     string    `json:"avatar_path"`
	Rating         float64   `json:"rating"`
	Role           string    `json:"role"`
	ReviewsCount   int       `json:"reviews_count"`
	AdsCount       int       `json:"ads_count"`
	FavoritesCount int       `json:"favorites_count"`
	CartCount      int       `json:"cart_count"`
	MessagesCount  int       `json:"unread_messages_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
