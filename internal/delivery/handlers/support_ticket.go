package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	supportticketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/support_ticket"
	supportticketUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/support_ticket"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

const (
	ErrInvalidTicketID    = "invalid ticket id"
	ErrTicketNotFound     = "ticket not found"
	ErrFailedCreateTicket = "failed to create ticket"
	ErrFailedGetTickets   = "failed to get tickets"
	ErrFailedGetTicket    = "failed to get ticket"
	ErrFailedUpdateTicket = "failed to update ticket"
	ErrFailedRateTicket   = "failed to rate ticket"
)

// SupportTicketHandlers обрабатывает запросы техподдержки
type SupportTicketHandlers struct {
	log      *slog.Logger
	services *Services
}

// NewSupportTicketHandlers создает новый экземпляр SupportTicketHandlers
func NewSupportTicketHandlers(log *slog.Logger, services *Services) *SupportTicketHandlers {
	return &SupportTicketHandlers{
		log:      log,
		services: services,
	}
}

// HandleCreateTicket создаёт новое обращение в техподдержку
// @Summary Создать обращение
// @Description Создаёт новое обращение в техподдержку
// @Tags support
// @Accept json
// @Produce json
// @Param request body dto.CreateTicketRequest true "Данные обращения"
// @Success 201 {object} dto.TicketResponse "обращение успешно создано"
// @Failure 400 {object} dto.ErrorResponse "bad request: Ошибка валидации или неверный JSON"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets [post]
func (h *SupportTicketHandlers) HandleCreateTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	var req dto.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.services.SupportTicket.CreateTicket(r.Context(), userID, &req)
	if err != nil {
		if isValidationError(err) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, supportticketRepo.ErrUserNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		h.log.ErrorContext(r.Context(), "failed to create ticket",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedCreateTicket)
		return
	}

	responser.RespondWithJSON(w, http.StatusCreated, resp)
}

// HandleGetMyTickets возвращает список обращений текущего пользователя
// @Summary Мои обращения
// @Description Возвращает список обращений текущего пользователя
// @Tags support
// @Produce json
// @Success 200 {array} dto.TicketResponse "список обращений получен"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets [get]
func (h *SupportTicketHandlers) HandleGetMyTickets(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	tickets, err := h.services.SupportTicket.GetMyTickets(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get tickets",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedGetTickets)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, tickets)
}

// HandleGetTicket возвращает детали одного обращения
// @Summary Получить обращение
// @Description Возвращает детали одного обращения по ID
// @Tags support
// @Produce json
// @Param id path int true "ID обращения"
// @Success 200 {object} dto.TicketResponse "обращение получено"
// @Failure 400 {object} dto.ErrorResponse "bad request: Некорректный ID"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 403 {object} dto.ErrorResponse "forbidden: Нет доступа к чужому обращению"
// @Failure 404 {object} dto.ErrorResponse "not found: Обращение не найдено"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets/{id} [get]
func (h *SupportTicketHandlers) HandleGetTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidTicketID)
		return
	}

	resp, err := h.services.SupportTicket.GetTicket(r.Context(), ticketID, userID)
	if err != nil {
		if errors.Is(err, supportticketRepo.ErrTicketNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrTicketNotFound)
			return
		}
		if errors.Is(err, supportticketUC.ErrForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to get ticket",
			slog.Int64("ticket_id", ticketID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedGetTicket)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleUpdateTicket обновляет обращение (только своё, только в статусе open)
// @Summary Обновить обращение
// @Description Обновляет данные обращения (только в статусе open)
// @Tags support
// @Accept json
// @Produce json
// @Param id path int true "ID обращения"
// @Param request body dto.UpdateTicketRequest true "Новые данные обращения"
// @Success 200 {object} dto.TicketResponse "обращение успешно обновлено"
// @Failure 400 {object} dto.ErrorResponse "bad request: Ошибка валидации, неверный статус или JSON"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 403 {object} dto.ErrorResponse "forbidden: Нет доступа к чужому обращению"
// @Failure 404 {object} dto.ErrorResponse "not found: Обращение не найдено"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets/{id} [put]
func (h *SupportTicketHandlers) HandleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidTicketID)
		return
	}

	var req dto.UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.services.SupportTicket.UpdateTicket(r.Context(), ticketID, userID, &req)
	if err != nil {
		if isValidationError(err) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, supportticketRepo.ErrTicketNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrTicketNotFound)
			return
		}
		if errors.Is(err, supportticketUC.ErrForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		if errors.Is(err, supportticketUC.ErrTicketNotOpen) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.log.ErrorContext(r.Context(), "failed to update ticket",
			slog.Int64("ticket_id", ticketID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedUpdateTicket)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleRateTicket выставляет оценку обращению в техподдержку
// @Summary Оценить обращение
// @Description Позволяет автору обращения выставить оценку от 1 до 5 для закрытого тикета
// @Tags support
// @Accept json
// @Produce json
// @Param id path int true "ID обращения"
// @Param request body dto.RateTicketRequest true "Оценка"
// @Success 200 {object} dto.TicketResponse "оценка выставлена"
// @Failure 400 {object} dto.ErrorResponse "bad request: Некорректная оценка, тикет не закрыт или уже оценён"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 403 {object} dto.ErrorResponse "forbidden: Пользователь не является автором"
// @Failure 404 {object} dto.ErrorResponse "not found: Обращение не найдено"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /support/tickets/{id}/rate [post]
func (h *SupportTicketHandlers) HandleRateTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidTicketID)
		return
	}

	var req dto.RateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.services.SupportTicket.RateTicket(r.Context(), userID, ticketID, req.Rating)
	if err != nil {
		if errors.Is(err, supportticketUC.ErrInvalidRating) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, supportticketRepo.ErrTicketNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrTicketNotFound)
			return
		}
		if errors.Is(err, supportticketUC.ErrForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		if errors.Is(err, supportticketUC.ErrTicketNotClosed) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, supportticketUC.ErrAlreadyRated) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.log.ErrorContext(r.Context(), "failed to rate ticket",
			slog.Int64("ticket_id", ticketID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedRateTicket)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}

func isValidationError(err error) bool {
	return errors.Is(err, supportticketUC.ErrInvalidCategory) ||
		errors.Is(err, supportticketUC.ErrTitleTooLong) ||
		errors.Is(err, supportticketUC.ErrTitleRequired) ||
		errors.Is(err, supportticketUC.ErrDescRequired) ||
		errors.Is(err, supportticketUC.ErrCategoryRequired)
}
