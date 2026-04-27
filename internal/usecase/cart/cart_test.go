package cart_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/cart"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/cart/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestAddToCart(t *testing.T) {
	log := discardLogger()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 10, SellerID: 2, Status: "active"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(10)).Return(ad, nil)
		cartMock.EXPECT().Add(gomock.Any(), int64(1), int64(10)).Return(nil)

		err := uc.AddToCart(context.Background(), 1, 10)
		assert.NoError(t, err)
	})

	t.Run("Own product error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 10, SellerID: 1, Status: "active"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(10)).Return(ad, nil)

		err := uc.AddToCart(context.Background(), 1, 10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, cart.ErrCannotAddOwnProduct)
	})

	t.Run("Product not active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 10, SellerID: 2, Status: "reserved"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(10)).Return(ad, nil)

		err := uc.AddToCart(context.Background(), 1, 10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, cart.ErrProductNotActive)
	})

	t.Run("Product not found in ads storage", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(10)).Return(models.Ad{}, errors.New("not found"))

		err := uc.AddToCart(context.Background(), 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get product")
	})

	t.Run("Cart storage error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 10, SellerID: 2, Status: "active"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(10)).Return(ad, nil)
		cartMock.EXPECT().Add(gomock.Any(), int64(1), int64(10)).Return(errors.New("product already in cart"))

		err := uc.AddToCart(context.Background(), 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already in cart")
	})

	t.Run("Product status draft", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 10, SellerID: 2, Status: "draft"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(10)).Return(ad, nil)

		err := uc.AddToCart(context.Background(), 1, 10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, cart.ErrProductNotActive)
	})
}

func TestRemoveFromCart(t *testing.T) {
	log := discardLogger()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().Remove(gomock.Any(), int64(1), int64(10)).Return(nil)

		err := uc.RemoveFromCart(context.Background(), 1, 10)
		assert.NoError(t, err)
	})

	t.Run("Item not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().Remove(gomock.Any(), int64(1), int64(99)).Return(errors.New("cart item not found"))

		err := uc.RemoveFromCart(context.Background(), 1, 99)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cart item not found")
	})

	t.Run("Storage error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().Remove(gomock.Any(), int64(1), int64(10)).Return(errors.New("db connection lost"))

		err := uc.RemoveFromCart(context.Background(), 1, 10)
		assert.Error(t, err)
	})
}

func TestGetCart(t *testing.T) {
	log := discardLogger()

	t.Run("Success with items", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		items := []dto.CartItemResponse{
			{ProductID: 10, Title: "iPhone", Price: 100000, SellerID: 2, SellerName: "Иван"},
			{ProductID: 20, Title: "MacBook", Price: 200000, SellerID: 3, SellerName: "Петр"},
		}
		cartMock.EXPECT().GetByUserID(gomock.Any(), int64(1)).Return(items, nil)

		result, err := uc.GetCart(context.Background(), 1)
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, int64(300000), result.TotalPrice)
	})

	t.Run("Success empty cart", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().GetByUserID(gomock.Any(), int64(1)).Return([]dto.CartItemResponse{}, nil)

		result, err := uc.GetCart(context.Background(), 1)
		require.NoError(t, err)
		assert.Len(t, result.Items, 0)
		assert.Equal(t, int64(0), result.TotalPrice)
	})

	t.Run("Storage error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().GetByUserID(gomock.Any(), int64(1)).Return(nil, errors.New("db error"))

		result, err := uc.GetCart(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Total price calculated correctly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		items := []dto.CartItemResponse{
			{ProductID: 1, Price: 100},
			{ProductID: 2, Price: 250},
			{ProductID: 3, Price: 350},
		}
		cartMock.EXPECT().GetByUserID(gomock.Any(), int64(5)).Return(items, nil)

		result, err := uc.GetCart(context.Background(), 5)
		require.NoError(t, err)
		assert.Equal(t, int64(700), result.TotalPrice)
	})
}

func TestCheckout(t *testing.T) {
	log := discardLogger()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		orderIDs := []int64{101, 102}
		sellers := map[int64]*dto.SellerContact{
			2: {ID: 2, Name: "Иван", Email: "ivan@mail.ru"},
			3: {ID: 3, Name: "Петр", Email: "petr@mail.ru"},
		}

		cartMock.EXPECT().Checkout(gomock.Any(), int64(1)).Return(orderIDs, sellers, nil)

		result, err := uc.Checkout(context.Background(), 1)
		require.NoError(t, err)
		assert.Len(t, result.OrderIDs, 2)
		assert.Contains(t, result.OrderIDs, int64(101))
		assert.Contains(t, result.OrderIDs, int64(102))
		assert.Len(t, result.Sellers, 2)
		assert.Equal(t, "Иван", result.Sellers[2].Name)
	})

	t.Run("Empty cart error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().Checkout(gomock.Any(), int64(1)).Return(nil, nil, errors.New("cart is empty"))

		result, err := uc.Checkout(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cart is empty")
	})

	t.Run("Products reserved error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)
		uc := cart.New(log, cartMock, adsMock)

		cartMock.EXPECT().Checkout(gomock.Any(), int64(1)).
			Return(nil, nil, errors.New("one or more products are no longer available"))

		result, err := uc.Checkout(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
