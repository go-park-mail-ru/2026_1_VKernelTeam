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
	title := "Updated"
	req := &dto.UpdateAdRequest{ID: 1, Title: &title}

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

func TestAds_CreateAd_WithCharacteristics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()

	t.Run("Success with category and custom characteristics", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			Title:      "Title",
			CategoryID: 10,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Красный"},
			},
			CustomCharacteristics: []dto.CustomCharacteristicInput{
				{Name: "Материал", Value: "Дерево"},
			},
		}

		defs := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: 10, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
		}

		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(1), nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(defs, nil)
		mockStorage.EXPECT().SetProductCharacteristics(ctx, int64(1), req.CategoryCharacteristics).Return(nil)
		mockStorage.EXPECT().SetProductCustomCharacteristics(ctx, int64(1), req.CustomCharacteristics).Return(nil)

		id, err := usecase.CreateAd(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("Fails on invalid category characteristic", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			Title:      "Title",
			CategoryID: 10,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Жёлтый"}, // not in enum
			},
		}

		defs := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: 10, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
		}

		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(2), nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(defs, nil)

		id, err := usecase.CreateAd(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("Fails on GetCategoryCharacteristics error", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			Title:      "Title",
			CategoryID: 10,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Красный"},
			},
		}

		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(3), nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(nil, errors.New("db error"))

		id, err := usecase.CreateAd(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("Fails on SetProductCharacteristics error", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			Title:      "Title",
			CategoryID: 10,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Красный"},
			},
		}

		defs := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: 10, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
		}

		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(4), nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(defs, nil)
		mockStorage.EXPECT().SetProductCharacteristics(ctx, int64(4), req.CategoryCharacteristics).Return(errors.New("db error"))

		id, err := usecase.CreateAd(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("Fails on SetProductCustomCharacteristics error", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			Title:      "Title",
			CategoryID: 10,
			CustomCharacteristics: []dto.CustomCharacteristicInput{
				{Name: "Материал", Value: "Дерево"},
			},
		}

		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(5), nil)
		mockStorage.EXPECT().SetProductCustomCharacteristics(ctx, int64(5), req.CustomCharacteristics).Return(errors.New("db error"))

		id, err := usecase.CreateAd(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("Fails on invalid custom characteristics", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			Title:      "Title",
			CategoryID: 10,
			CustomCharacteristics: []dto.CustomCharacteristicInput{
				{Name: "", Value: "Дерево"}, // empty name
			},
		}

		mockStorage.EXPECT().CreateAd(ctx, req).Return(int64(6), nil)

		id, err := usecase.CreateAd(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestAds_UpdateAd_WithCharacteristics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()

	t.Run("Success with category characteristics merge", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     1,
			UserID: 2,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Синий"},
			},
		}

		defs := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: 10, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
		}

		// Нет основных полей — UpdateAd в репозитории не вызывается
		mockStorage.EXPECT().GetAdByID(ctx, int64(1)).Return(models.Ad{ID: 1, CategoryID: 10}, nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(defs, nil)
		mockStorage.EXPECT().SetProductCharacteristics(ctx, int64(1), req.CategoryCharacteristics).Return(nil)

		err := usecase.UpdateAd(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Success with custom characteristics merge", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     1,
			UserID: 2,
			CustomCharacteristics: []dto.CustomCharacteristicInput{
				{Name: "Материал", Value: "Металл"},
			},
		}

		// Нет основных полей — UpdateAd в репозитории не вызывается
		mockStorage.EXPECT().SetProductCustomCharacteristics(ctx, int64(1), req.CustomCharacteristics).Return(nil)

		err := usecase.UpdateAd(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Fails on GetAdByID error during characteristics update", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     1,
			UserID: 2,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Синий"},
			},
		}

		mockStorage.EXPECT().GetAdByID(ctx, int64(1)).Return(models.Ad{}, errors.New("not found"))

		err := usecase.UpdateAd(ctx, req)
		assert.Error(t, err)
	})

	t.Run("Fails on validation error during characteristics update", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     1,
			UserID: 2,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 999, Value: "Синий"}, // unknown ID
			},
		}

		defs := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: 10, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
		}

		mockStorage.EXPECT().GetAdByID(ctx, int64(1)).Return(models.Ad{ID: 1, CategoryID: 10}, nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(defs, nil)

		err := usecase.UpdateAd(ctx, req)
		assert.Error(t, err)
	})

	t.Run("Fails on SetProductCharacteristics error during update", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     1,
			UserID: 2,
			CategoryCharacteristics: []dto.CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Синий"},
			},
		}

		defs := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: 10, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
		}

		mockStorage.EXPECT().GetAdByID(ctx, int64(1)).Return(models.Ad{ID: 1, CategoryID: 10}, nil)
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, int64(10)).Return(defs, nil)
		mockStorage.EXPECT().SetProductCharacteristics(ctx, int64(1), req.CategoryCharacteristics).Return(errors.New("db error"))

		err := usecase.UpdateAd(ctx, req)
		assert.Error(t, err)
	})

	t.Run("Fails on invalid custom characteristics during update", func(t *testing.T) {
		req := &dto.UpdateAdRequest{
			ID:     1,
			UserID: 2,
			CustomCharacteristics: []dto.CustomCharacteristicInput{
				{Name: "", Value: "value"}, // empty name
			},
		}

		// Нет основных полей — UpdateAd не вызывается, но валидация ловит ошибку
		err := usecase.UpdateAd(ctx, req)
		assert.Error(t, err)
	})
}

func TestAds_GetCategoryCharacteristics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockAdsProvider(ctrl)
	mockFileStorage := mocks.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	usecase := New(logger, mockStorage, mockFileStorage)

	ctx := context.Background()
	categoryID := int64(10)

	t.Run("Success", func(t *testing.T) {
		expected := []models.CategoryCharacteristic{
			{ID: 1, CategoryID: categoryID, Name: "Цвет", AllowedValues: []string{"Красный", "Синий"}},
			{ID: 2, CategoryID: categoryID, Name: "Размер", AllowedValues: nil},
		}

		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, categoryID).Return(expected, nil)

		chars, err := usecase.GetCategoryCharacteristics(ctx, categoryID)
		assert.NoError(t, err)
		assert.Equal(t, expected, chars)
	})

	t.Run("Storage error", func(t *testing.T) {
		mockStorage.EXPECT().GetCategoryCharacteristics(ctx, categoryID).Return(nil, errors.New("db error"))

		chars, err := usecase.GetCategoryCharacteristics(ctx, categoryID)
		assert.Error(t, err)
		assert.Nil(t, chars)
	})
}
