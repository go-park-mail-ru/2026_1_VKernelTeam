package ads

//go:generate mockgen -source=ads.go -destination=mocks/mock_ads.go -package=mocks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
)

const (
	opGetAllAds                 = "usecase.ads.GetAllAds"
	opUploadAdPhotos            = "usecase.ads.UploadAdPhotos"
	opCreateAd                  = "usecase.ads.CreateAd"
	opGetAdByID                 = "usecase.ads.GetAdByID"
	opUpdateAd                  = "usecase.ads.UpdateAd"
	opDeleteAd                  = "usecase.ads.DeleteAd"
	opCloseAd                   = "usecase.ads.CloseAd"
	opGetAdsByUserID            = "usecase.ads.GetAdsByUserID"
	opAddFavorite               = "usecase.ads.AddFavorite"
	opRemoveFavorite            = "usecase.ads.RemoveFavorite"
	opGetUserFavorites          = "usecase.ads.GetUserFavorites"
	opGetCategoryCharacteristics = "usecase.ads.GetCategoryCharacteristics"
)

type AdsProvider interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
	AddProductImages(ctx context.Context, adID int64, photos []string) error
	DeleteProductImages(ctx context.Context, adID int64) error
	UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error
	DeleteAd(ctx context.Context, id int64, userID int64) error
	CloseAd(ctx context.Context, id int64, userID int64) error
	GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error)
	AddFavorite(ctx context.Context, userID int64, adID int64) error
	RemoveFavorite(ctx context.Context, userID int64, adID int64) error
	GetUserFavorites(ctx context.Context, userID int64) ([]models.Ad, error)
	SetProductCharacteristics(ctx context.Context, productID int64, inputs []dto.CharacteristicInput) error
	SetProductCustomCharacteristics(ctx context.Context, productID int64, inputs []dto.CustomCharacteristicInput) error
	GetCategoryCharacteristics(ctx context.Context, categoryID int64) ([]models.CategoryCharacteristic, error)
}

// FileStorage описывает интерфейс для работы с файлами в объектном хранилище
type FileStorage interface {
	UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

var allowedImageTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

type Ads struct {
	log         *slog.Logger
	adsStorage  AdsProvider
	fileStorage FileStorage
}

// New создаёт новый экземпляр Ads с переданными зависимостями.
func New(
	log *slog.Logger,
	adsStorage AdsProvider,
	fileStorage FileStorage,
) *Ads {
	return &Ads{
		log:         log,
		adsStorage:  adsStorage,
		fileStorage: fileStorage,
	}
}

// GetAllAds возвращает все объявления.
func (a *Ads) GetAllAds(ctx context.Context) ([]models.Ad, error) {
	a.log.InfoContext(ctx, "getting all ads",
		slog.String("op", opGetAllAds),
	)

	ads, err := a.adsStorage.GetAllAds(ctx)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get all ads",
			slog.String("op", opGetAllAds),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	a.log.InfoContext(ctx, "all ads fetched",
		slog.String("op", opGetAllAds),
		slog.Int("count", len(ads)),
	)
	return ads, nil
}

// UploadAdPhotos валидирует и загружает файлы фотографий в S3, возвращает список URL.
func (a *Ads) UploadAdPhotos(ctx context.Context, files []multipart.File, filenames []string) ([]string, error) {
	a.log.DebugContext(ctx, "uploading ad photos",
		slog.String("op", opUploadAdPhotos),
		slog.Int("files_count", len(files)),
	)

	urls := make([]string, 0, len(files))
	for _, file := range files {
		buf := make([]byte, 512)
		if _, err := file.Read(buf); err != nil {
			return nil, fmt.Errorf("%s: failed to read file header: %w", opUploadAdPhotos, err)
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("%s: failed to seek file: %w", opUploadAdPhotos, err)
		}

		var mimeToExt = map[string]string{
			"image/jpeg": ".jpg",
			"image/png":  ".png",
			"image/webp": ".webp",
			"image/gif":  ".gif",
		}

		contentType := http.DetectContentType(buf)
		if _, ok := allowedImageTypes[contentType]; !ok {
			return nil, fmt.Errorf("%s: unsupported file type: %s", opUploadAdPhotos, contentType)
		}

		ext := mimeToExt[contentType]

		if len(files) != len(filenames) {
			return nil, fmt.Errorf("%s: files and filenames count mismatch", opUploadAdPhotos)
		}

		url, err := a.fileStorage.UploadFile(ctx, file, "ads", ext)
		if err != nil {
			for _, uploadedURL := range urls {
				_ = a.fileStorage.DeleteFile(ctx, uploadedURL)
			}
			return nil, fmt.Errorf("%s: failed to upload photo: %w", opUploadAdPhotos, err)
		}
		urls = append(urls, url)
	}

	a.log.DebugContext(ctx, "ad photos uploaded",
		slog.String("op", opUploadAdPhotos),
		slog.Int("uploaded_count", len(urls)),
	)
	return urls, nil
}

func (a *Ads) CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error) {
	a.log.InfoContext(ctx, "creating new ad",
		slog.String("op", opCreateAd),
		slog.Int64("user_id", req.UserID),
		slog.Int64("category_id", req.CategoryID),
		slog.String("title", req.Title),
	)

	adID, err := a.adsStorage.CreateAd(ctx, req)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to create ad",
			slog.String("op", opCreateAd),
			slog.String("error", err.Error()),
		)
		return 0, err
	}

	if len(req.Photos) > 0 {
		if err := a.adsStorage.AddProductImages(ctx, adID, req.Photos); err != nil {
			a.log.ErrorContext(ctx, "failed to add product images",
				slog.String("op", opCreateAd),
				slog.String("error", err.Error()),
			)
			return 0, fmt.Errorf("%s: %w", opCreateAd, err)
		}
	}

	if len(req.CategoryCharacteristics) > 0 {
		defs, err := a.adsStorage.GetCategoryCharacteristics(ctx, req.CategoryID)
		if err != nil {
			a.log.ErrorContext(ctx, "failed to get category characteristics",
				slog.String("op", opCreateAd),
				slog.String("error", err.Error()),
			)
			return 0, fmt.Errorf("%s: %w", opCreateAd, err)
		}
		if err := validator.ValidateCharacteristics(req.CategoryCharacteristics, defs); err != nil {
			return 0, err
		}
		if err := a.adsStorage.SetProductCharacteristics(ctx, adID, req.CategoryCharacteristics); err != nil {
			a.log.ErrorContext(ctx, "failed to set product characteristics",
				slog.String("op", opCreateAd),
				slog.String("error", err.Error()),
			)
			return 0, fmt.Errorf("%s: %w", opCreateAd, err)
		}
	}

	if len(req.CustomCharacteristics) > 0 {
		if err := validator.ValidateCustomCharacteristics(req.CustomCharacteristics); err != nil {
			return 0, err
		}
		if err := a.adsStorage.SetProductCustomCharacteristics(ctx, adID, req.CustomCharacteristics); err != nil {
			a.log.ErrorContext(ctx, "failed to set custom characteristics",
				slog.String("op", opCreateAd),
				slog.String("error", err.Error()),
			)
			return 0, fmt.Errorf("%s: %w", opCreateAd, err)
		}
	}

	a.log.InfoContext(ctx, "ad created successfully",
		slog.String("op", opCreateAd),
		slog.Int64("ad_id", adID),
	)
	return adID, nil
}

// GetAdByID возвращает объявление по ID.
func (a *Ads) GetAdByID(ctx context.Context, id int64) (models.Ad, error) {
	a.log.DebugContext(ctx, "getting ad by id",
		slog.String("op", opGetAdByID),
		slog.Int64("ad_id", id),
	)

	ad, err := a.adsStorage.GetAdByID(ctx, id)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get ad by id",
			slog.String("op", opGetAdByID),
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		return models.Ad{}, err
	}

	a.log.DebugContext(ctx, "ad fetched",
		slog.String("op", opGetAdByID),
		slog.Int64("ad_id", id),
	)
	return ad, nil
}

// UpdateAd обновляет объявление, включая замену фотографий.
func (a *Ads) UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error {
	a.log.InfoContext(ctx, "updating ad",
		slog.String("op", opUpdateAd),
		slog.Int64("ad_id", req.ID),
		slog.Int64("user_id", req.UserID),
	)

	hasBaseFields := req.CategoryID != nil || req.Title != nil || req.Description != nil ||
		req.Price != nil || req.Status != nil || req.Location != nil
	if hasBaseFields {
		err := a.adsStorage.UpdateAd(ctx, req)
		if err != nil {
			a.log.ErrorContext(ctx, "failed to update ad",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return err
		}
	}

	if len(req.Photos) > 0 {
		if err := a.adsStorage.DeleteProductImages(ctx, req.ID); err != nil {
			a.log.ErrorContext(ctx, "failed to delete old images",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: %w", opUpdateAd, err)
		}
		if err := a.adsStorage.AddProductImages(ctx, req.ID, req.Photos); err != nil {
			a.log.ErrorContext(ctx, "failed to add new images",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: %w", opUpdateAd, err)
		}
	}

	if len(req.CategoryCharacteristics) > 0 {
		ad, err := a.adsStorage.GetAdByID(ctx, req.ID)
		if err != nil {
			a.log.ErrorContext(ctx, "failed to get ad for characteristics validation",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: %w", opUpdateAd, err)
		}
		defs, err := a.adsStorage.GetCategoryCharacteristics(ctx, ad.CategoryID)
		if err != nil {
			a.log.ErrorContext(ctx, "failed to get category characteristics",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: %w", opUpdateAd, err)
		}
		if err := validator.ValidateCharacteristics(req.CategoryCharacteristics, defs); err != nil {
			return err
		}
		if err := a.adsStorage.SetProductCharacteristics(ctx, req.ID, req.CategoryCharacteristics); err != nil {
			a.log.ErrorContext(ctx, "failed to set product characteristics",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: %w", opUpdateAd, err)
		}
	}

	if len(req.CustomCharacteristics) > 0 {
		if err := validator.ValidateCustomCharacteristics(req.CustomCharacteristics); err != nil {
			return err
		}
		if err := a.adsStorage.SetProductCustomCharacteristics(ctx, req.ID, req.CustomCharacteristics); err != nil {
			a.log.ErrorContext(ctx, "failed to set custom characteristics",
				slog.String("op", opUpdateAd),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: %w", opUpdateAd, err)
		}
	}

	a.log.InfoContext(ctx, "ad updated successfully",
		slog.String("op", opUpdateAd),
		slog.Int64("ad_id", req.ID),
	)
	return nil
}

// DeleteAd удаляет объявление (мягкое удаление) и его фотографии из S3.
func (a *Ads) DeleteAd(ctx context.Context, id int64, userID int64) error {
	a.log.InfoContext(ctx, "deleting ad",
		slog.String("op", opDeleteAd),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)

	ad, err := a.adsStorage.GetAdByID(ctx, id)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get ad before deletion",
			slog.String("op", opDeleteAd),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", opDeleteAd, err)
	}

	err = a.adsStorage.DeleteAd(ctx, id, userID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to delete ad",
			slog.String("op", opDeleteAd),
			slog.String("error", err.Error()),
		)
		return err
	}

	for _, photoURL := range ad.Photos {
		if err := a.fileStorage.DeleteFile(ctx, photoURL); err != nil {
			a.log.WarnContext(ctx, "failed to delete photo from s3",
				slog.String("op", opDeleteAd),
				slog.String("photo_url", photoURL),
				slog.String("error", err.Error()),
			)
		}
	}

	if len(ad.Photos) > 0 {
		if err := a.adsStorage.DeleteProductImages(ctx, id); err != nil {
			a.log.WarnContext(ctx, "failed to delete product images from db",
				slog.String("op", opDeleteAd),
				slog.String("error", err.Error()),
			)
		}
	}

	a.log.InfoContext(ctx, "ad deleted successfully",
		slog.String("op", opDeleteAd),
		slog.Int64("ad_id", id),
	)
	return nil
}

// CloseAd закрывает объявление.
func (a *Ads) CloseAd(ctx context.Context, id int64, userID int64) error {
	a.log.InfoContext(ctx, "closing ad",
		slog.String("op", opCloseAd),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)

	err := a.adsStorage.CloseAd(ctx, id, userID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to close ad",
			slog.String("op", opCloseAd),
			slog.String("error", err.Error()),
		)
		return err
	}

	a.log.InfoContext(ctx, "ad archived successfully",
		slog.String("op", opCloseAd),
		slog.Int64("ad_id", id),
	)
	return nil
}

// GetAdsByUserID возвращает все объявления пользователя по его ID.
func (a *Ads) GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error) {
	a.log.DebugContext(ctx, "getting ads by user id",
		slog.String("op", opGetAdsByUserID),
		slog.Int64("user_id", userID),
	)

	ads, err := a.adsStorage.GetAdsByUserID(ctx, userID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get ads by user id",
			slog.String("op", opGetAdsByUserID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", opGetAdsByUserID, err)
	}

	a.log.DebugContext(ctx, "ads by user id fetched",
		slog.String("op", opGetAdsByUserID),
		slog.Int("count", len(ads)),
	)
	return ads, nil
}

// AddFavorite добавляет объявление в избранное.
func (a *Ads) AddFavorite(ctx context.Context, userID int64, adID int64) error {
	a.log.InfoContext(ctx, "adding ad to favorites",
		slog.String("op", opAddFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)

	err := a.adsStorage.AddFavorite(ctx, userID, adID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to add ad to favorites",
			slog.String("op", opAddFavorite),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", opAddFavorite, err)
	}

	a.log.InfoContext(ctx, "ad added to favorites",
		slog.String("op", opAddFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)
	return nil
}

// RemoveFavorite удаляет объявление из избранного.
func (a *Ads) RemoveFavorite(ctx context.Context, userID int64, adID int64) error {
	a.log.InfoContext(ctx, "removing ad from favorites",
		slog.String("op", opRemoveFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)

	err := a.adsStorage.RemoveFavorite(ctx, userID, adID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to remove ad from favorites",
			slog.String("op", opRemoveFavorite),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", opRemoveFavorite, err)
	}

	a.log.InfoContext(ctx, "ad removed from favorites",
		slog.String("op", opRemoveFavorite),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)
	return nil
}

// GetUserFavorites возвращает избранное.
func (a *Ads) GetUserFavorites(ctx context.Context, userID int64) ([]models.Ad, error) {
	a.log.DebugContext(ctx, "getting favorites",
		slog.String("op", opGetUserFavorites),
		slog.Int64("user_id", userID),
	)

	favorites, err := a.adsStorage.GetUserFavorites(ctx, userID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get favorites",
			slog.String("op", opGetUserFavorites),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", opGetUserFavorites, err)
	}

	a.log.DebugContext(ctx, "favorites fetched",
		slog.String("op", opGetUserFavorites),
		slog.Int("count", len(favorites)),
	)
	return favorites, nil
}

// GetCategoryCharacteristics возвращает определения характеристик для категории.
func (a *Ads) GetCategoryCharacteristics(ctx context.Context, categoryID int64) ([]models.CategoryCharacteristic, error) {
	a.log.DebugContext(ctx, "getting category characteristics",
		slog.String("op", opGetCategoryCharacteristics),
		slog.Int64("category_id", categoryID),
	)

	chars, err := a.adsStorage.GetCategoryCharacteristics(ctx, categoryID)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to get category characteristics",
			slog.String("op", opGetCategoryCharacteristics),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", opGetCategoryCharacteristics, err)
	}

	a.log.DebugContext(ctx, "category characteristics fetched",
		slog.String("op", opGetCategoryCharacteristics),
		slog.Int("count", len(chars)),
	)
	return chars, nil
}
