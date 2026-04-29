package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	supportticketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/support_ticket"
	supportticketUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket"
)

const (
	ErrFailedChangeStatus  = "failed to change ticket status"
	ErrFailedGetAllTickets = "failed to get all tickets"
	ErrFailedGetStats      = "failed to get stats"
)

// HandleChangeStatus меняет статус обращения техподдержки.
func (h *SupportTicketHandlers) HandleChangeStatus(w http.ResponseWriter, r *http.Request) {
	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidTicketID)
		return
	}

	var req dto.ChangeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	resp, err := h.services.SupportTicket.ChangeStatus(r.Context(), ticketID, &req)
	if err != nil {
		if errors.Is(err, supportticketUC.ErrInvalidStatus) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, supportticketRepo.ErrTicketNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrTicketNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to change ticket status",
			slog.Int64("ticket_id", ticketID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedChangeStatus)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, resp)
}

// HandleGetAllTickets возвращает список всех обращений всех пользователей.
func (h *SupportTicketHandlers) HandleGetAllTickets(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.services.SupportTicket.GetAllTickets(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get all tickets", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedGetAllTickets)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, tickets)
}

// HandleGetStats возвращает сводную статистику по обращениям.
func (h *SupportTicketHandlers) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.services.SupportTicket.GetStats(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get tickets stats", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedGetStats)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, stats)
}
