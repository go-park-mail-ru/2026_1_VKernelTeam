package ads

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/ads/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAds_GetAllAds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	testAds := []models.Ad{{ID: 1, Title: "Test"}}

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().GetAllAds(ctx).Return(testAds, nil)
		ads, err := usecase.GetAllAds(ctx)
		assert.NoError(t, err)
		assert.Equal(t, testAds, ads)
	})

	t.Run("Error", func(t *testing.T) {
		mockStorage.EXPECT().GetAllAds(ctx).Return(nil, errors.New("db error"))
		ads, err := usecase.GetAllAds(ctx)
		assert.Error(t, err)
		assert.Nil(t, ads)
	})
}

func TestAds_CreateAd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	req := &dto.CreateAdRequest{Title: "Title"}

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(1), nil)
		id, err := usecase.CreateAd(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("Error", func(t *testing.T) {
		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(0), errors.New("db error"))
		id, err := usecase.CreateAd(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestAds_GetAdByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	adID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().GetAdByID(ctx, adID).Return(models.Ad{ID: adID}, nil)
		ad, err := usecase.GetAdByID(ctx, adID)
		assert.NoError(t, err)
		assert.Equal(t, adID, ad.ID)
	})

	t.Run("Error", func(t *testing.T) {
		mockStorage.EXPECT().GetAdByID(ctx, adID).Return(models.Ad{}, errors.New("not found"))
		ad, err := usecase.GetAdByID(ctx, adID)
		assert.Error(t, err)
		assert.Equal(t, int64(0), ad.ID)
	})
}

func TestAds_UpdateAd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	req := &dto.UpdateAdRequest{ID: 1}

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().UpdateAd(ctx, req).Return(nil)
		err := usecase.UpdateAd(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockStorage.EXPECT().UpdateAd(ctx, req).Return(errors.New("forbidden"))
		err := usecase.UpdateAd(ctx, req)
		assert.Error(t, err)
	})
}

func TestAds_DeleteAd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	adID := int64(1)
	userID := int64(2)

	t.Run("Success", func(t *testing.T) {
		ad := models.Ad{ID: adID, Photos: []string{"photo1.jpg", "photo2.jpg"}}
		mockStorage.EXPECT().GetAdByID(ctx, adID).Return(ad, nil)
		mockStorage.EXPECT().DeleteAd(ctx, adID, userID).Return(nil)
		mockFileStorage.EXPECT().DeleteFile(ctx, "photo1.jpg").Return(nil)
		mockFileStorage.EXPECT().DeleteFile(ctx, "photo2.jpg").Return(nil)
		mockStorage.EXPECT().DeleteProductImages(ctx, adID).Return(nil)
		err := usecase.DeleteAd(ctx, adID, userID)
		assert.NoError(t, err)
	})

	t.Run("GetAdByID error", func(t *testing.T) {
		mockStorage.EXPECT().GetAdByID(ctx, adID).Return(models.Ad{}, errors.New("not found"))
		err := usecase.DeleteAd(ctx, adID, userID)
		assert.Error(t, err)
	})

	t.Run("DeleteAd error", func(t *testing.T) {
		ad := models.Ad{ID: adID}
		mockStorage.EXPECT().GetAdByID(ctx, adID).Return(ad, nil)
		mockStorage.EXPECT().DeleteAd(ctx, adID, userID).Return(errors.New("forbidden"))
		err := usecase.DeleteAd(ctx, adID, userID)
		assert.Error(t, err)
	})
}

func TestAds_CloseAd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	adID := int64(1)
	userID := int64(2)

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().CloseAd(ctx, adID, userID).Return(nil)
		err := usecase.CloseAd(ctx, adID, userID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockStorage.EXPECT().CloseAd(ctx, adID, userID).Return(errors.New("not found"))
		err := usecase.CloseAd(ctx, adID, userID)
		assert.Error(t, err)
	})
}

func TestAds_GetAdsByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	userID := int64(1)
	testAds := []models.Ad{{ID: 1, SellerID: userID}}

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().GetAdsByUserID(ctx, userID).Return(testAds, nil)
		ads, err := usecase.GetAdsByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testAds, ads)
	})

	t.Run("Error", func(t *testing.T) {
		mockStorage.EXPECT().GetAdsByUserID(ctx, userID).Return(nil, errors.New("db error"))
		ads, err := usecase.GetAdsByUserID(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, ads)
	})
}

func TestAds_AddFavorite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	userID := int64(1)
	adID := int64(10)

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().AddFavorite(ctx, userID, adID).Return(nil)

		err := usecase.AddFavorite(ctx, userID, adID)

		assert.NoError(t, err)
	})

	t.Run("Storage Error", func(t *testing.T) {
		storageErr := errors.New("db error")
		mockStorage.EXPECT().AddFavorite(ctx, userID, adID).Return(storageErr)

		err := usecase.AddFavorite(ctx, userID, adID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "usecase.ads.AddFavorite")
	})
}

func TestAds_RemoveFavorite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	userID := int64(1)
	adID := int64(10)

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().RemoveFavorite(ctx, userID, adID).Return(nil)

		err := usecase.RemoveFavorite(ctx, userID, adID)

		assert.NoError(t, err)
	})

	t.Run("Storage Error", func(t *testing.T) {
		mockStorage.EXPECT().RemoveFavorite(ctx, userID, adID).Return(errors.New("db error"))

		err := usecase.RemoveFavorite(ctx, userID, adID)

		assert.Error(t, err)
	})
}

func TestAds_GetUserFavorites(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	userID := int64(1)
	testAds := []models.Ad{
		{ID: 10, Title: "Fav Ad"},
	}

	t.Run("Success", func(t *testing.T) {
		mockStorage.EXPECT().GetUserFavorites(ctx, userID).Return(testAds, nil)

		ads, err := usecase.GetUserFavorites(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, testAds, ads)
	})

	t.Run("Storage Error", func(t *testing.T) {
		mockStorage.EXPECT().GetUserFavorites(ctx, userID).Return(nil, errors.New("db error"))

		ads, err := usecase.GetUserFavorites(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, ads)
	})
}
