package dto

// ErrorResponse — структура для отправки ошибок в формате JSON.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrors — ошибки валидации полей объявления.
// Поля Email/Password/Name живут в auth-сервисе, здесь только catalog-специфичные.
type ValidationErrors struct {
	UserID                  string   `json:"user_id,omitempty"`
	CategoryID              string   `json:"category_id,omitempty"`
	Title                   string   `json:"title,omitempty"`
	Description             string   `json:"description,omitempty"`
	Price                   string   `json:"price,omitempty"`
	Status                  string   `json:"status,omitempty"`
	Location                string   `json:"location,omitempty"`
	Coords                  string   `json:"coords,omitempty"`
	Photos                  []string `json:"photos,omitempty"`
	CategoryCharacteristics string   `json:"category_characteristics,omitempty"`
	CustomCharacteristics   string   `json:"custom_characteristics,omitempty"`
}

// HasErrors проверяет, есть ли ошибки валидации.
func (v *ValidationErrors) HasErrors() bool {
	return v.UserID != "" ||
		v.CategoryID != "" || v.Title != "" || v.Description != "" || v.Price != "" ||
		v.Status != "" || v.Location != "" || v.Coords != "" || len(v.Photos) > 0 ||
		v.CategoryCharacteristics != "" || v.CustomCharacteristics != ""
}
