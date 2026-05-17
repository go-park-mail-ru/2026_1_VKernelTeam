package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAddToCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().AddToCart(gomock.Any(), int64(1), int64(10)).Return(nil)

		body := map[string]interface{}{
			keyProductID: 10,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]string
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "added", response["status"])
	})

	t.Run("Unauthorized - no user in context", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		body := map[string]interface{}{keyProductID: 10}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()

		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid request body - broken JSON", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBufferString("{invalid"))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid product ID - negative", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		body := map[string]interface{}{keyProductID: -5}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid product ID - zero", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		body := map[string]interface{}{keyProductID: 0}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service error - product not active", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().AddToCart(gomock.Any(), int64(1), int64(10)).Return(errors.New("product is not active"))

		body := map[string]interface{}{keyProductID: 10}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service error - own product", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().AddToCart(gomock.Any(), int64(1), int64(10)).Return(errors.New("cannot add own product to cart"))

		body := map[string]interface{}{keyProductID: 10}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var errResp dto.ErrorResponse
		err := json.NewDecoder(rr.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "own product")
	})

	t.Run("Service error - already in cart", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().AddToCart(gomock.Any(), int64(1), int64(10)).Return(errors.New("product already in cart"))

		body := map[string]interface{}{keyProductID: 10}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleGetCart(t *testing.T) {
	t.Run("Success with items", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		expectedResponse := &dto.CartResponse{
			Items: []dto.CartItemResponse{
				{ProductID: 1, Title: titleIPhone, Price: 100000, SellerID: 2, SellerName: nameIvan},
				{ProductID: 2, Title: "MacBook", Price: 200000, SellerID: 3, SellerName: "Петр"},
			},
			TotalPrice: 300000,
		}

		mockCart.EXPECT().GetCart(gomock.Any(), int64(1)).Return(expectedResponse, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/cart", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleGetCart(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response dto.CartResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, int64(300000), response.TotalPrice)
		assert.Len(t, response.Items, 2)
		assert.Equal(t, titleIPhone, response.Items[0].Title)
	})

	t.Run("Success empty cart", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		expectedResponse := &dto.CartResponse{
			Items:      []dto.CartItemResponse{},
			TotalPrice: 0,
		}

		mockCart.EXPECT().GetCart(gomock.Any(), int64(1)).Return(expectedResponse, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/cart", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleGetCart(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response dto.CartResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response.Items, 0)
		assert.Equal(t, int64(0), response.TotalPrice)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/cart", nil)
		rr := httptest.NewRecorder()

		cartH.HandleGetCart(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().GetCart(gomock.Any(), int64(1)).Return(nil, errors.New("db error"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/cart", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleGetCart(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleRemoveFromCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().RemoveFromCart(gomock.Any(), int64(1), int64(10)).Return(nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/cart/10", nil)
		req.SetPathValue("id", "10")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]string
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "removed", response["status"])
	})

	t.Run("Unauthorized", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/cart/10", nil)
		req.SetPathValue("id", "10")
		rr := httptest.NewRecorder()

		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid path param - non numeric", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/cart/invalid", nil)
		req.SetPathValue("id", "invalid")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid path param - negative ID", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/cart/-5", nil)
		req.SetPathValue("id", "-5")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service error - item not found", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().RemoveFromCart(gomock.Any(), int64(1), int64(99)).Return(errors.New("cart item not found"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/cart/99", nil)
		req.SetPathValue("id", "99")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
