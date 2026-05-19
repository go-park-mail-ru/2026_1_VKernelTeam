package dto

//go:generate easyjson -all $GOFILE

import "time"

// RegisterRequest представляет собой структуру для запроса на регистрацию пользователя
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest представляет собой структуру для запроса на вход в систему
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse представляет собой структуру для ответа на запрос входа в систему
type LoginResponse struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

// UpdateProfileRequest представляет собой структуру для запроса на обновление профиля
type UpdateProfileRequest struct {
	Name string `json:"name" validate:"required,min=3,max=50"`
}

// PublicUserResponse представляет собой структуру для ответа на запрос профиля пользователя
type PublicUserResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	AvatarPath   string    `json:"avatar_path"`
	Rating       float64   `json:"rating"`
	ReviewsCount int       `json:"reviews_count"`
	AdsCount     int       `json:"ads_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// ErrorResponse стандартный ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrors содержит ошибки валидации по полям
type ValidationErrors struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}
