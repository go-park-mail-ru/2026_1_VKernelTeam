package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	supportticketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/support_ticket"
	supportmessageUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/support_message"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleSendMessage(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportMessage(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportMessage: mockSupport},
		)

		expected := &dto.MessageResponse{ID: 1, TicketID: 10, UserID: 1, Text: "hi", CreatedAt: time.Now()}
		mockSupport.EXPECT().SendMessage(gomock.Any(), int64(10), int64(1), gomock.Any()).Return(expected, nil)

		body, _ := json.Marshal(dto.SendMessageRequest{Text: "hi"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets/10/messages", bytes.NewBuffer(body))
		req.SetPathValue("id", "10")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleSendMessage(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.MessageResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expected.Text, resp.Text)
	})

	t.Run("Unauthorized no user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handlers := NewSupportTicketHandlers(newLogger(t), &Services{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets/10/messages", bytes.NewBufferString(`{"text":"hi"}`))
		req.SetPathValue("id", "10")
		rr := httptest.NewRecorder()

		handlers.HandleSendMessage(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Ticket not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportMessage(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportMessage: mockSupport},
		)

		mockSupport.EXPECT().SendMessage(gomock.Any(), int64(10), int64(1), gomock.Any()).Return(nil, supportticketRepo.ErrTicketNotFound)

		body, _ := json.Marshal(dto.SendMessageRequest{Text: "hi"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets/10/messages", bytes.NewBuffer(body))
		req.SetPathValue("id", "10")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleSendMessage(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleGetMessages(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportMessage(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportMessage: mockSupport},
		)

		expected := []dto.MessageResponse{{ID: 1, TicketID: 10, UserID: 1, Text: "hello", CreatedAt: time.Now()}}
		mockSupport.EXPECT().GetMessages(gomock.Any(), int64(10), int64(1)).Return(expected, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/support/tickets/10/messages", nil)
		req.SetPathValue("id", "10")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleGetMessages(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp []dto.MessageResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, expected[0].Text, resp[0].Text)
	})

	t.Run("Forbidden", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSupport := mocks.NewMockSupportMessage(ctrl)
		handlers := NewSupportTicketHandlers(
			newLogger(t),
			&Services{SupportMessage: mockSupport},
		)

		mockSupport.EXPECT().GetMessages(gomock.Any(), int64(10), int64(1)).Return(nil, supportmessageUC.ErrForbidden)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/support/tickets/10/messages", nil)
		req.SetPathValue("id", "10")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlers.HandleGetMessages(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
