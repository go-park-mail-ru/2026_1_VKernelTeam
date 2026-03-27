package dto

// ErrorResponse представляет собой структуру для отправки ошибок в формате JSON
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrors представляет собой структуру для отправки ошибок валидации по полям
type ValidationErrors struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}
