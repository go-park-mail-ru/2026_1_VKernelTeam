package dto

// ErrorResponse — структура для отправки ошибок в формате JSON.
type ErrorResponse struct {
	Error string `json:"error"`
}
