package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	chatRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/chat"
)

const (
	opHandleCreateOrder  = "handlers.HandleCreateOrder"
	opHandleConfirmOrder = "handlers.HandleConfirmOrder"
	opHandleGetAllChats  = "handlers.HandleGetAllChats"
	opHandleGetChat      = "handlers.HandleGetChat"
)

// ошибки для handlers чата
const (
	ErrFailedToCreateOrder  = "failed to create order"
	ErrFailedToConfirmOrder = "failed to confirm order"
	ErrFailedToGetChats     = "failed to get chats"
	ErrFailedToGetChat      = "failed to get chat"
	ErrChatNotFound         = "chat not found"
)

// HandleCreateOrder обрабатывает запрос на создание заказа (запрос покупки).
// @Summary Создать запрос на покупку товара
// @Description Создаёт (или переиспользует существующий) чат между покупателем
// @Description и продавцом по объявлению и отправляет туда сообщение-заказ.
// @Description В ответе возвращается ID чата для редиректа на фронте.
// @Tags orders
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "ID объявления (ad_id)"
// @Success 200 {object} dto.OrderResponse "запрос на покупку создан успешно"
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

	// Получаем ID чата, создавая запрос на покупку (или переиспользуя существующий)
	chatID, err := h.services.Chat.CreateOrderRequest(r.Context(), adID, userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to create order request",
			slog.String("op", opHandleCreateOrder),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToCreateOrder)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, dto.OrderResponse{
		ChatID:  chatID,
		Message: "order request created successfully",
	})
}

// HandleConfirmOrder обрабатывает подтверждение покупки продавцом в чате.
// @Summary Подтвердить покупку товара
// @Description Продавец подтверждает сделку по чату. Создаётся заказ за покупателем,
// @Description объявление переводится в статус 'sold' и удаляется из корзин всех пользователей.
// @Tags orders
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "ID чата"
// @Success 200 {object} dto.SuccessConfirmOrderResponse "покупка подтверждена успешно"
// @Failure 400 {object} dto.ErrorResponse "invalid chat id / forbidden: not the seller / ad is not active"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /chats/{id}/confirm [post]
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

	// Получаем chatID из URL параметра
	chatIDStr := r.PathValue("id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		h.log.DebugContext(r.Context(), "failed to parse chat id from url",
			slog.String("op", opHandleConfirmOrder),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidChatID)
		return
	}

	// Подтверждаем покупку в чате
	err = h.services.Chat.ConfirmPurchase(r.Context(), chatID, userID)
	if err != nil {
		if err.Error() == "forbidden: not the seller" {
			h.log.WarnContext(r.Context(), "user tried to confirm purchase in non-owned chat",
				slog.String("op", opHandleConfirmOrder),
				slog.Int64("user_id", userID),
				slog.Int64("chat_id", chatID),
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

	responser.RespondWithJSON(w, http.StatusOK, dto.SuccessConfirmOrderResponse{
		Message: "purchase confirmed successfully",
	})
}

// HandleGetAllChats возвращает список чатов пользователя.
// @Summary Получить список чатов
// @Description Возвращает чаты, в которых участвует текущий пользователь,
// @Description отсортированные по времени последнего сообщения.
// @Tags chats
// @Produce json
// @Security Bearer
// @Success 200 {object} dto.ChatListResponse "список чатов"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /chats [get]
func (h *ChatHandlers) HandleGetAllChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	resp, err := h.services.Chat.GetAllChats(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get user chats",
			slog.String("op", opHandleGetAllChats),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToGetChats)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleGetChat возвращает шапку и сообщения одного чата.
// @Summary Получить чат
// @Description Возвращает объявление, собеседника и все сообщения чата.
// @Description Доступ только у участников чата — для остальных 400.
// @Tags chats
// @Produce json
// @Security Bearer
// @Param id path int true "ID чата"
// @Success 200 {object} dto.ChatDetailResponse "детали чата"
// @Failure 400 {object} dto.ErrorResponse "invalid chat id"
// @Failure 400 {object} dto.ErrorResponse "chat not found"
// @Failure 401 {object} dto.ErrorResponse "unauthorized"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /chats/{id} [get]
func (h *ChatHandlers) HandleGetChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	chatID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidChatID)
		return
	}

	resp, err := h.services.Chat.GetChat(r.Context(), chatID, userID)
	if err != nil {
		if errors.Is(err, chatRepo.ErrChatNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrChatNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to get chat",
			slog.String("op", opHandleGetChat),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToGetChat)
		return
	}

	// Убедимся, что messages не nil в JSON (фронт может упасть на null)
	if resp.Messages == nil {
		resp.Messages = []dto.MessageItem{}
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}
