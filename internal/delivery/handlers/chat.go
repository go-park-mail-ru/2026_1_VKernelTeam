package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// const (
// 	opHandleCreateOrder  = "handlers.HandleCreateOrder"
// 	opHandleConfirmOrder = "handlers.HandleConfirmOrder"
// )

// HandleCreateOrder обрабатывает запрос на создание заказа (запрос покупки)
// TODO
func (h *ChatHandlers) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	// TODO
	responser.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "order request created successfully",
	})
}

// HandleConfirmOrder обрабатывает подтверждение заказа продавцом
// TODO
func (h *ChatHandlers) HandleConfirmOrder(w http.ResponseWriter, r *http.Request) {
	// TODO
	responser.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "purchase confirmed successfully",
	})
}
