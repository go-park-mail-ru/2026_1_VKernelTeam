package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	chatRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/chat"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// newChatReq строит запрос с user_id в контексте и path-param "id".
func newChatReq(t *testing.T, method, target, pathID string, userID int64) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	if pathID != "" {
		req.SetPathValue("id", pathID)
	}
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestHandleCreateOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			CreateOrderRequest(gomock.Any(), int64(38), int64(1)).
			Return(int64(77), nil)

		req := newChatReq(t, http.MethodPost, "/api/v1/ads/38/order", "38", 1)
		rr := httptest.NewRecorder()
		h.HandleCreateOrder(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp dto.OrderResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, int64(77), resp.ChatID)
		assert.NotEmpty(t, resp.Message)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/ads/38/order", nil)
		req.SetPathValue("id", "38")
		rr := httptest.NewRecorder()
		h.HandleCreateOrder(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid ad id", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := newChatReq(t, http.MethodPost, "/api/v1/ads/abc/order", "abc", 1)
		rr := httptest.NewRecorder()
		h.HandleCreateOrder(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Usecase error", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			CreateOrderRequest(gomock.Any(), int64(38), int64(1)).
			Return(int64(0), errors.New("boom"))

		req := newChatReq(t, http.MethodPost, "/api/v1/ads/38/order", "38", 1)
		rr := httptest.NewRecorder()
		h.HandleCreateOrder(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Method not allowed", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := newChatReq(t, http.MethodGet, "/api/v1/ads/38/order", "38", 1)
		rr := httptest.NewRecorder()
		h.HandleCreateOrder(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleConfirmOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			ConfirmPurchase(gomock.Any(), int64(5), int64(2)).
			Return(nil)

		req := newChatReq(t, http.MethodPost, "/api/v1/chats/5/confirm", "5", 2)
		rr := httptest.NewRecorder()
		h.HandleConfirmOrder(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/chats/5/confirm", nil)
		req.SetPathValue("id", "5")
		rr := httptest.NewRecorder()
		h.HandleConfirmOrder(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid chat id", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := newChatReq(t, http.MethodPost, "/api/v1/chats/xx/confirm", "xx", 2)
		rr := httptest.NewRecorder()
		h.HandleConfirmOrder(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Forbidden - not the seller", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			ConfirmPurchase(gomock.Any(), int64(5), int64(2)).
			Return(errors.New("forbidden: not the seller"))

		req := newChatReq(t, http.MethodPost, "/api/v1/chats/5/confirm", "5", 2)
		rr := httptest.NewRecorder()
		h.HandleConfirmOrder(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Usecase internal error", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			ConfirmPurchase(gomock.Any(), int64(5), int64(2)).
			Return(errors.New("boom"))

		req := newChatReq(t, http.MethodPost, "/api/v1/chats/5/confirm", "5", 2)
		rr := httptest.NewRecorder()
		h.HandleConfirmOrder(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Method not allowed", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := newChatReq(t, http.MethodGet, "/api/v1/chats/5/confirm", "5", 2)
		rr := httptest.NewRecorder()
		h.HandleConfirmOrder(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleGetAllChats(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		expected := dto.ChatListResponse{
			Chats: []dto.ChatPreview{
				{
					ChatID:  7,
					Ad:      dto.AdPreview{ID: 38, Title: titleIPhone, Price: 1000, Status: "active"},
					Partner: dto.UserPreview{ID: 2, Name: nameIvan},
				},
			},
		}
		mockChat.EXPECT().
			GetAllChats(gomock.Any(), int64(1)).
			Return(expected, nil)

		req := newChatReq(t, http.MethodGet, "/api/v1/chats", "", 1)
		rr := httptest.NewRecorder()
		h.HandleGetAllChats(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp dto.ChatListResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Len(t, resp.Chats, 1)
		assert.Equal(t, int64(7), resp.Chats[0].ChatID)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/chats", nil)
		rr := httptest.NewRecorder()
		h.HandleGetAllChats(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Usecase error", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			GetAllChats(gomock.Any(), int64(1)).
			Return(dto.ChatListResponse{}, errors.New("db down"))

		req := newChatReq(t, http.MethodGet, "/api/v1/chats", "", 1)
		rr := httptest.NewRecorder()
		h.HandleGetAllChats(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleGetChat(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		expected := dto.ChatDetailResponse{
			ChatID:   5,
			Ad:       dto.AdPreview{ID: 38, Title: titleIPhone, Price: 1000, Status: "active"},
			Partner:  dto.UserPreview{ID: 2, Name: nameIvan},
			Messages: []dto.MessageItem{{ID: 1, SenderID: 1, Text: "hi", Type: "order"}},
		}
		mockChat.EXPECT().
			GetChat(gomock.Any(), int64(5), int64(1)).
			Return(expected, nil)

		req := newChatReq(t, http.MethodGet, "/api/v1/chats/5", "5", 1)
		rr := httptest.NewRecorder()
		h.HandleGetChat(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp dto.ChatDetailResponse
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, int64(5), resp.ChatID)
		assert.Len(t, resp.Messages, 1)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/chats/5", nil)
		req.SetPathValue("id", "5")
		rr := httptest.NewRecorder()
		h.HandleGetChat(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid chat id", func(t *testing.T) {
		h, _ := setupChatHandlers(t)

		req := newChatReq(t, http.MethodGet, "/api/v1/chats/xx", "xx", 1)
		rr := httptest.NewRecorder()
		h.HandleGetChat(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not found or not a participant", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			GetChat(gomock.Any(), int64(5), int64(1)).
			Return(dto.ChatDetailResponse{}, chatRepo.ErrChatNotFound)

		req := newChatReq(t, http.MethodGet, "/api/v1/chats/5", "5", 1)
		rr := httptest.NewRecorder()
		h.HandleGetChat(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Usecase internal error", func(t *testing.T) {
		h, mockChat := setupChatHandlers(t)

		mockChat.EXPECT().
			GetChat(gomock.Any(), int64(5), int64(1)).
			Return(dto.ChatDetailResponse{}, errors.New("boom"))

		req := newChatReq(t, http.MethodGet, "/api/v1/chats/5", "5", 1)
		rr := httptest.NewRecorder()
		h.HandleGetChat(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
