package dto

// CreateAdRequest представляет собой структуру для запроса на создание объявления
type CreateAdRequest struct {
	UserID      int64    `json:"-"`
	CategoryID  int64    `json:"category_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	Status      string   `json:"status"`
	Photos      []string `json:"photos"`
	Location    string   `json:"location"`
}

// UpdateAdRequest представляет собой структуру для запроса на обновление объявления
type UpdateAdRequest struct {
	ID          int64  `json:"-"`
	UserID      int64  `json:"-"`
	CategoryID  int64  `json:"category_id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Price       int64  `json:"price,omitempty"`
	Status      string `json:"status,omitempty"`
	Location    string `json:"location,omitempty"`
}

// FavoriteRequest представляет собой структуру для запроса на добавление объявления в избранное
type FavoriteRequest struct {
	AdID int64 `json:"ad_id"`
}
