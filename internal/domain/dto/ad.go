package dto

// CreateAdRequest представляет собой структуру для запроса на создание объявления
type CreateAdRequest struct {
	UserID      int64    `json:"user_id"`
	CategoryID  int64    `json:"category_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	Status      string   `json:"status"`
	Photos      []string `json:"photos"`
}

// UpdateAdRequest представляет собой структуру для запроса на обновление объявления
type UpdateAdRequest struct {
	ID          int64    `json:"-"`
	UserID      int64    `json:"-"`
	CategoryID  int64    `json:"category_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	Status      string   `json:"status"`
}

