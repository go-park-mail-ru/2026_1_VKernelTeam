package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	supportticketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/support_ticket"
	supportmessageUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_message"
)

// Сообщения ошибок для обработчиков сообщений обращений техподдержки.
const (
	ErrFailedSendMessage = "failed to send message"
	ErrFailedGetMessages = "failed to get messages"
	ErrEmptyMessageText  = "text is required"
)

// HandleSendMessage отправляет сообщение в чат обращения техподдержки.
// @Summary Отправить сообщение в чат обращения
// @Description Отправляет сообщение в чат обращения. Доступно автору обращения,
// @Description а также пользователям с ролью support/admin.
// @Tags support
// @Accept json
// @Produce json
// @Param id path int true "ID обращения"
// @Param request body dto.SendMessageRequest true "Текст сообщения"
// @Success 200 {object} dto.MessageResponse "сообщение отправлено"
// @Failure 400 {object} dto.ErrorResponse "bad request: пустой текст, неверный ID или нет доступа"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets/{id}/messages [post]
func (h *SupportTicketHandlers) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidTicketID)
		return
	}

	var req dto.SendMessageRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.services.SupportMessage.SendMessage(r.Context(), ticketID, userID, &req)
	if err != nil {
		if errors.Is(err, supportmessageUC.ErrTextRequired) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrEmptyMessageText)
			return
		}
		if errors.Is(err, supportticketRepo.ErrTicketNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrTicketNotFound)
			return
		}
		if errors.Is(err, supportmessageUC.ErrForbidden) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrForbidden)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to send support message",
			slog.Int64("ticket_id", ticketID),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedSendMessage)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleGetMessages возвращает все сообщения чата обращения.
// @Summary Получить сообщения обращения
// @Description Возвращает все сообщения чата обращения. Доступно автору обращения,
// @Description а также пользователям с ролью support/admin.
// @Tags support
// @Produce json
// @Param id path int true "ID обращения"
// @Success 200 {array} dto.MessageResponse "список сообщений"
// @Failure 400 {object} dto.ErrorResponse "bad request: неверный ID или нет доступа"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets/{id}/messages [get]
func (h *SupportTicketHandlers) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok || userID == 0 {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidTicketID)
		return
	}

	messages, err := h.services.SupportMessage.GetMessages(r.Context(), ticketID, userID)
	if err != nil {
		if errors.Is(err, supportticketRepo.ErrTicketNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrTicketNotFound)
			return
		}
		if errors.Is(err, supportmessageUC.ErrForbidden) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrForbidden)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to get support messages",
			slog.Int64("ticket_id", ticketID),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedGetMessages)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, messages)
}
