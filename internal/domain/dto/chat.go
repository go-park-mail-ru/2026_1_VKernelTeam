// Package dto содержит объекты передачи данных (Data Transfer Objects)
package dto

// OrderResponse - ответ на POST /ads/{id}/order:
// id созданного/найденного чата и сообщение.
type OrderResponse struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}
