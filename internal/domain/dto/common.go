package dto

// ErrorResponse представляет собой структуру для отправки ошибок в формате JSON
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrors представляет собой структуру для отправки ошибок валидации по полям
type ValidationErrors struct {
	Email       string   `json:"email,omitempty"`
	Password    string   `json:"password,omitempty"`
	Name        string   `json:"name,omitempty"`
	UserID      string   `json:"user_id,omitempty"`
	CategoryID  string   `json:"category_id,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Price       string   `json:"price,omitempty"`
	Status      string   `json:"status,omitempty"`
	Location    string   `json:"location,omitempty"`
	Photos      []string `json:"photos,omitempty"`
}

// HasErrors проверяет, есть ли ошибки валидации
func (v *ValidationErrors) HasErrors() bool {
	return v.Email != "" || v.Password != "" || v.Name != "" || v.UserID != "" ||
		v.CategoryID != "" || v.Title != "" || v.Description != "" || v.Price != "" ||
		v.Status != "" || v.Location != "" || len(v.Photos) > 0
}
