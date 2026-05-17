package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/delivery/handlers/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	supportticketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/support_ticket"
	supportticketUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleChangeStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		expected := &dto.TicketStatusResponse{ID: 12, Status: statusClosed, UpdatedAt: time.Now()}
		mockSupport.EXPECT().ChangeStatus(gomock.Any(), int64(12), gomock.Any()).Return(expected, nil)

		body, _ := json.Marshal(dto.ChangeStatusRequest{Status: statusClosed})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/support/tickets/12/status", bytes.NewBuffer(body))
		req.SetPathValue("id", "12")
		rr := httptest.NewRecorder()

		handlers.HandleChangeStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.TicketStatusResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.Status, resp.Status)
		assert.Equal(t, expected.ID, resp.ID)
	})

	t.Run("Invalid status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		mockSupport.EXPECT().ChangeStatus(gomock.Any(), int64(5), gomock.Any()).Return(nil, supportticketUC.ErrInvalidStatus)

		body, _ := json.Marshal(dto.ChangeStatusRequest{Status: "invalid_status"})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/support/tickets/5/status", bytes.NewBuffer(body))
		req.SetPathValue("id", "5")
		rr := httptest.NewRecorder()

		handlers.HandleChangeStatus(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Ticket not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		mockSupport.EXPECT().ChangeStatus(gomock.Any(), int64(100), gomock.Any()).Return(nil, supportticketRepo.ErrTicketNotFound)

		body, _ := json.Marshal(dto.ChangeStatusRequest{Status: statusClosed})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/support/tickets/100/status", bytes.NewBuffer(body))
		req.SetPathValue("id", "100")
		rr := httptest.NewRecorder()

		handlers.HandleChangeStatus(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleGetAllTickets(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		expected := []dto.TicketResponse{{ID: 1, Title: "Test ticket"}}
		mockSupport.EXPECT().GetAllTickets(gomock.Any()).Return(expected, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/all", nil)
		rr := httptest.NewRecorder()

		handlers.HandleGetAllTickets(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp []dto.TicketResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, expected[0].Title, resp[0].Title)
	})

	t.Run("Service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		mockSupport.EXPECT().GetAllTickets(gomock.Any()).Return(nil, assert.AnError)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/all", nil)
		rr := httptest.NewRecorder()

		handlers.HandleGetAllTickets(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleGetStats(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		expected := &dto.StatsResponse{Total: 42}
		mockSupport.EXPECT().GetStats(gomock.Any()).Return(expected, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/stats", nil)
		rr := httptest.NewRecorder()

		handlers.HandleGetStats(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.StatsResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.Total, resp.Total)
	})

	t.Run("Service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportTicket: mockSupport},
		)

		mockSupport.EXPECT().GetStats(gomock.Any()).Return(nil, assert.AnError)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/stats", nil)
		rr := httptest.NewRecorder()

		handlers.HandleGetStats(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
