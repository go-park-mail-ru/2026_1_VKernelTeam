package cart_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/cart"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/cart/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestAddToCart(t *testing.T) {
	log := setupLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Success", func(t *testing.T) {
		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)

		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 1, SellerID: 2, Status: "active"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(ad, nil)
		cartMock.EXPECT().Add(gomock.Any(), int64(1), int64(1)).Return(nil)

		err := uc.AddToCart(context.Background(), 1, 1)

		assert.NoError(t, err)
	})

	t.Run("Fail Own Product", func(t *testing.T) {
		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)

		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 1, SellerID: 1, Status: "active"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(ad, nil)

		err := uc.AddToCart(context.Background(), 1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "own product")
	})

	t.Run("Fail Not Active", func(t *testing.T) {
		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)

		uc := cart.New(log, cartMock, adsMock)

		ad := models.Ad{ID: 1, SellerID: 2, Status: "reserved"}
		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(ad, nil)

		err := uc.AddToCart(context.Background(), 1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("Fail Ad Not Found", func(t *testing.T) {
		cartMock := mocks.NewMockCartProvider(ctrl)
		adsMock := mocks.NewMockAdsProvider(ctrl)

		uc := cart.New(log, cartMock, adsMock)

		adsMock.EXPECT().GetAdByID(gomock.Any(), int64(1)).Return(models.Ad{}, errors.New("not found"))

		err := uc.AddToCart(context.Background(), 1, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "get product")
	})
}
