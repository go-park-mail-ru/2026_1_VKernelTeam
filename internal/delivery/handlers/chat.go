package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

const (
	opHandleCreateOrder  = "handlers.HandleCreateOrder"
	opHandleConfirmOrder = "handlers.HandleConfirmOrder"
)

// ошибки для handlers чата
const (
	ErrFailedToCreateOrder  = "failed to create order"
	ErrFailedToConfirmOrder = "failed to confirm order"
)

// HandleCreateOrder обрабатывает запрос на создание заказа (запрос покупки).
// @Summary Создать запрос на покупку товара
// @Description Отправляет уведомление продавцу о желании купить товар
// @Tags orders
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "ID объявления (ad_id)"
// @Success 200 {object} map[string]string "запрос на покупку создан успешно"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /ads/{id}/order [post]
func (h *ChatHandlers) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responser.RespondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	// Получаем userID из контекста (авторизация)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// Получаем adID из URL параметра
	adIDStr := r.PathValue("id")
	adID, err := strconv.ParseInt(adIDStr, 10, 64)
	if err != nil {
		h.log.DebugContext(r.Context(), "failed to parse ad id from url",
			slog.String("op", opHandleCreateOrder),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	// Вызвываем usecase CreateOrderRequest
	err = h.services.Chat.CreateOrderRequest(r.Context(), adID, userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to create order request",
			slog.String("op", opHandleCreateOrder),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToCreateOrder)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "order request created successfully",
	})
}

// HandleConfirmOrder обрабатывает подтверждение покупки продавцом.
// @Summary Подтвердить покупку товара
// @Description Продавец подтверждает, что продал товар. Статус объявления меняется на 'sold'
// @Tags orders
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "ID объявления (ad_id)"
// @Success 200 {object} map[string]string "покупка подтверждена успешно"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id / forbidden: не продавец товара"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /ads/{id}/confirm [post]
func (h *ChatHandlers) HandleConfirmOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responser.RespondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	// Получиаем userID из контекста (авторизация)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// Получиаем adID из URL параметра
	adIDStr := r.PathValue("id")
	adID, err := strconv.ParseInt(adIDStr, 10, 64)
	if err != nil {
		h.log.DebugContext(r.Context(), "failed to parse ad id from url",
			slog.String("op", opHandleConfirmOrder),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	// Вызываем usecase ConfirmPurchase
	err = h.services.Chat.ConfirmPurchase(r.Context(), adID, userID)
	if err != nil {
		if err.Error() == "forbidden: not the seller" {
			h.log.WarnContext(r.Context(), "user tried to confirm order not owned",
				slog.String("op", opHandleConfirmOrder),
				slog.Int64("user_id", userID),
				slog.Int64("ad_id", adID),
			)
			responser.RespondWithError(w, http.StatusBadRequest, ErrForbidden)
			return
		}

		h.log.ErrorContext(r.Context(), "failed to confirm purchase",
			slog.String("op", opHandleConfirmOrder),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToConfirmOrder)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "purchase confirmed successfully",
	})
}

// TODO: добавить обработчики для получения чатов и сообщений (HandleGetChats, HandleGetChat)
// и методы в usecase и репозитории для получения данных чата и сообщений.

// TODO: проработать, чтобы при принятии покупки сделка состоялась ТОЛЬКО с конкретным продавцом,
// т.е. запомнить, что купил именно он, чтобы эта сделка была в его истории покупок.
