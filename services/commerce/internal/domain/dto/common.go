package dto

//go:generate easyjson -all $GOFILE

// ErrorResponse — структура для отправки ошибок в формате JSON.
type ErrorResponse struct {
	Error string `json:"error"`
}
