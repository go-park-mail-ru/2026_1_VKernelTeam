package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleAddToCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().AddToCart(gomock.Any(), int64(1), int64(10)).Return(nil)

		body := map[string]interface{}{
			"product_id": 10,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		
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

	t.Run("Unauthorized", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		body := map[string]interface{}{
			"product_id": 10,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()

		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid Product ID", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		body := map[string]interface{}{
			"product_id": -5, 
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
	
	t.Run("Service Error", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().AddToCart(gomock.Any(), int64(1), int64(10)).Return(errors.New("product not active"))

		body := map[string]interface{}{
			"product_id": 10,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(jsonBody))
		
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleAddToCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleGetCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		expectedResponse := &dto.CartResponse{
			Items: []dto.CartItemResponse{
				{ProductID: 1, Title: "Test", Price: 100},
			},
			TotalPrice: 100,
		}

		mockCart.EXPECT().GetCart(gomock.Any(), int64(1)).Return(expectedResponse, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleGetCart(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response dto.CartResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, int64(100), response.TotalPrice)
		assert.Len(t, response.Items, 1)
	})

	t.Run("Service Error", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().GetCart(gomock.Any(), int64(1)).Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
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

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/10", nil)
		req.SetPathValue("id", "10")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Invalid Params", func(t *testing.T) {
		cartH, _ := setupCartHandlers(t)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/invalid", nil)
		req.SetPathValue("id", "invalid")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleRemoveFromCart(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleCheckout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		expectedResponse := &dto.CheckoutResponse{
			OrderIDs: []int64{101},
			Sellers: map[int64]*dto.SellerContact{
				2: {ID: 2, Name: "Seller 2", Email: "seller2@example.com"},
			},
		}

		mockCart.EXPECT().Checkout(gomock.Any(), int64(1)).Return(expectedResponse, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/checkout", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleCheckout(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response dto.CheckoutResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response.OrderIDs, 1)
		assert.Equal(t, "Seller 2", response.Sellers[2].Name)
	})

	t.Run("Empty Cart Error", func(t *testing.T) {
		cartH, mockCart := setupCartHandlers(t)

		mockCart.EXPECT().Checkout(gomock.Any(), int64(1)).Return(nil, errors.New("cart is empty"))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/checkout", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		cartH.HandleCheckout(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
