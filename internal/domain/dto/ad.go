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
