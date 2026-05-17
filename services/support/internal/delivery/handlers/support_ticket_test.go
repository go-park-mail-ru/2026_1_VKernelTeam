package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/delivery/handlers/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	supportticketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/support_ticket"
	supportticketUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testTicketTitle = "Test ticket"
	statusClosed    = "closed"
	categoryGeneral = "general"
	titleUpdated    = "Updated"
)

func newLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandleCreateTicket(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		expected := &dto.TicketResponse{ID: 1, Title: testTicketTitle, Category: categoryGeneral}
		mockSupport.EXPECT().CreateTicket(gomock.Any(), int64(1), gomock.Any()).Return(expected, nil)

		body, _ := json.Marshal(dto.CreateTicketRequest{Title: testTicketTitle, Description: "Test desc", Category: categoryGeneral})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/support/tickets", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleCreateTicket(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp dto.TicketResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.ID, resp.ID)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/support/tickets", bytes.NewBufferString(`{"title":"Test"}`))
		rr := httptest.NewRecorder()

		handlers.HandleCreateTicket(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Validation error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		mockSupport.EXPECT().CreateTicket(gomock.Any(), int64(1), gomock.Any()).Return(nil, supportticketUC.ErrTitleRequired)

		body, _ := json.Marshal(dto.CreateTicketRequest{Title: "", Description: "Test desc", Category: categoryGeneral})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/support/tickets", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleCreateTicket(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleGetTicket(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		expected := &dto.TicketResponse{ID: 2, Title: "Ticket"}
		mockSupport.EXPECT().GetTicket(gomock.Any(), int64(2), int64(1)).Return(expected, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/2", nil)
		req.SetPathValue("id", "2")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleGetTicket(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.TicketResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.ID, resp.ID)
	})

	t.Run("Invalid ticket ID", func(t *testing.T) {
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/abc", nil)
		req.SetPathValue("id", "abc")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handlers.HandleGetTicket(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Forbidden", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		mockSupport.EXPECT().GetTicket(gomock.Any(), int64(2), int64(1)).Return(nil, supportticketUC.ErrForbidden)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/support/tickets/2", nil)
		req.SetPathValue("id", "2")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleGetTicket(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

func TestHandleUpdateTicket(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		expected := &dto.TicketResponse{ID: 3, Title: titleUpdated}
		mockSupport.EXPECT().UpdateTicket(gomock.Any(), int64(3), int64(1), gomock.Any()).Return(expected, nil)

		body, _ := json.Marshal(dto.UpdateTicketRequest{Title: titleUpdated, Description: "New desc"})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/v1/support/tickets/3", bytes.NewBuffer(body))
		req.SetPathValue("id", "3")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleUpdateTicket(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.TicketResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.Title, resp.Title)
	})

	t.Run("Ticket not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		mockSupport.EXPECT().UpdateTicket(gomock.Any(), int64(5), int64(1), gomock.Any()).Return(nil, supportticketRepo.ErrTicketNotFound)

		body, _ := json.Marshal(dto.UpdateTicketRequest{Title: titleUpdated, Description: "New desc"})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/v1/support/tickets/5", bytes.NewBuffer(body))
		req.SetPathValue("id", "5")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleUpdateTicket(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandleRateTicket(t *testing.T) {
	t.Run("Invalid rating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		mockSupport.EXPECT().RateTicket(gomock.Any(), int64(1), int64(6), int(0)).Return(nil, supportticketUC.ErrInvalidRating)

		body, _ := json.Marshal(dto.RateTicketRequest{Rating: 0})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/support/tickets/6/rate", bytes.NewBuffer(body))
		req.SetPathValue("id", "6")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleRateTicket(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportTicket(ctrl)
		handlers := NewSupportTicketHandlers(newLogger(t), &Services{SupportTicket: mockSupport})

		expected := &dto.TicketResponse{ID: 7, Title: "Rated"}
		mockSupport.EXPECT().RateTicket(gomock.Any(), int64(1), int64(7), int(5)).Return(expected, nil)

		body, _ := json.Marshal(dto.RateTicketRequest{Rating: 5})
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/support/tickets/7/rate", bytes.NewBuffer(body))
		req.SetPathValue("id", "7")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleRateTicket(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.TicketResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.ID, resp.ID)
	})
}
