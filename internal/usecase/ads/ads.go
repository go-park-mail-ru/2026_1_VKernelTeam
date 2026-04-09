package ads

//go:generate mockgen -source=ads.go -destination=mocks/mock_ads.go -package=mocks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
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
	const op = "usecase.ads.GetAll"

	log := a.log.With(
		slog.String("op", op),
	)
	log.Info("getting all ads")
	ads, err := a.adsStorage.GetAllAds(ctx)
	if err != nil {
		log.Error("failed to get all ads")
		return nil, err
	}
	log.Info("got all ads")
	return ads, nil

}

// UploadAdPhotos валидирует и загружает файлы фотографий в S3, возвращает список URL.
func (a *Ads) UploadAdPhotos(ctx context.Context, files []multipart.File, filenames []string) ([]string, error) {
	const op = "ads.UploadAdPhotos"

	urls := make([]string, 0, len(files))
	for i, file := range files {
		// Валидация содержимого файла
		buf := make([]byte, 512)
		if _, err := file.Read(buf); err != nil {
			return nil, fmt.Errorf("%s: failed to read file header: %w", op, err)
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("%s: failed to seek file: %w", op, err)
		}

		contentType := http.DetectContentType(buf)
		if _, ok := allowedImageTypes[contentType]; !ok {
			return nil, fmt.Errorf("%s: unsupported file type: %s", op, contentType)
		}

		ext := filepath.Ext(filenames[i])

		url, err := a.fileStorage.UploadFile(ctx, file, "ads", ext)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to upload photo: %w", op, err)
		}
		urls = append(urls, url)
	}

	return urls, nil
}

func (a *Ads) CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error) {
	const op = "ads.CreateAd"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", req.UserID),
		slog.Int64("category_id", req.CategoryID),
		slog.String("title", req.Title),
		slog.String("description", req.Description),
		slog.Int64("price", req.Price),
	)
	log.Info("creating new ad")

	adID, err := a.adsStorage.CreateAd(ctx, req)
	if err != nil {
		log.Error("failed to create ad", "error", err)
		return 0, err
	}

	if len(req.Photos) > 0 {
		if err := a.adsStorage.AddProductImages(ctx, adID, req.Photos); err != nil {
			log.Error("failed to add product images", "error", err)
			return 0, fmt.Errorf("%s: %w", op, err)
		}
	}

	log.Info("ad created successfully", "ad_id", adID)
	return adID, nil
}

// GetAdByID возвращает объявление по ID.
func (a *Ads) GetAdByID(ctx context.Context, id int64) (models.Ad, error) {
	const op = "ads.GetAdByID"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("id", id),
	)
	log.Info("getting ad by id")

	ad, err := a.adsStorage.GetAdByID(ctx, id)
	if err != nil {
		log.Error("failed to get ad by id", "error", err)
		return models.Ad{}, err
	}

	log.Info("got ad by id")
	return ad, nil
}

// UpdateAd обновляет объявление, включая замену фотографий.
func (a *Ads) UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error {
	const op = "ads.UpdateAd"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("ad_id", req.ID),
		slog.Int64("user_id", req.UserID),
	)
	log.Info("updating ad")

	err := a.adsStorage.UpdateAd(ctx, req)
	if err != nil {
		log.Error("failed to update ad", "error", err)
		return err
	}

	// Если переданы новые фото — удаляем старые и сохраняем новые
	if len(req.Photos) > 0 {
		if err := a.adsStorage.DeleteProductImages(ctx, req.ID); err != nil {
			log.Error("failed to delete old images", "error", err)
			return fmt.Errorf("%s: %w", op, err)
		}
		if err := a.adsStorage.AddProductImages(ctx, req.ID, req.Photos); err != nil {
			log.Error("failed to add new images", "error", err)
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	log.Info("ad updated successfully")
	return nil
}

// DeleteAd удаляет объявление (мягкое удаление) и его фотографии из S3.
func (a *Ads) DeleteAd(ctx context.Context, id int64, userID int64) error {
	const op = "ads.DeleteAd"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)
	log.Info("deleting ad")

	ad, err := a.adsStorage.GetAdByID(ctx, id)
	if err != nil {
		log.Error("failed to get ad before deletion", "error", err)
		return fmt.Errorf("%s: %w", op, err)
	}

	err = a.adsStorage.DeleteAd(ctx, id, userID)
	if err != nil {
		log.Error("failed to delete ad", "error", err)
		return err
	}

	// Удаляем фотографии из S3
	for _, photoURL := range ad.Photos {
		if err := a.fileStorage.DeleteFile(ctx, photoURL); err != nil {
			log.Error("failed to delete photo from s3", "photo_url", photoURL, "error", err)
		}
	}

	if len(ad.Photos) > 0 {
		if err := a.adsStorage.DeleteProductImages(ctx, id); err != nil {
			log.Error("failed to delete product images from db", "error", err)
		}
	}

	log.Info("ad deleted successfully")
	return nil
}

// CloseAd закрывает объявление.
func (a *Ads) CloseAd(ctx context.Context, id int64, userID int64) error {
	const op = "ads.CloseAd"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)
	log.Info("closing ad")

	err := a.adsStorage.CloseAd(ctx, id, userID)
	if err != nil {
		log.Error("failed to close ad", "error", err)
		return err
	}

	log.Info("ad archived successfully")
	return nil
}

// GetAdsByUserID возвращает все объявления пользователя по его ID.
func (a *Ads) GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error) {
	const op = "usecase.ads.GetAdsByUserID"

	log := a.log.With(
		slog.String("op", op),
	)
	log.Info("getting all ads by user ID")

	// вызов метода репозитория
	ads, err := a.adsStorage.GetAdsByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get all ads by user ID")
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("got all ads by user ID")

	return ads, nil
}

// AddFavorite добавляет объявление в избранное.
func (a *Ads) AddFavorite(ctx context.Context, userID int64, adID int64) error {
	const op = "usecase.ads.AddFavorite"
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)

	log.Info("attempting to add ad to favorites")

	err := a.adsStorage.AddFavorite(ctx, userID, adID)
	if err != nil {
		log.Error("failed to add ad to favorite", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully added ad to favorites")
	return nil
}

// RemoveFavorite удаляет объявление из избранного.
func (a *Ads) RemoveFavorite(ctx context.Context, userID int64, adID int64) error {
	const op = "usecase.ads.RemoveFavorite"
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
		slog.Int64("ad_id", adID),
	)

	log.Info("attempting to remove ad from favorites")

	err := a.adsStorage.RemoveFavorite(ctx, userID, adID)
	if err != nil {
		log.Error("failed to remove ad from favorite", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully removed ad from favorites")
	return nil
}

// GetUserFavorites возвращает избранное.
func (a *Ads) GetUserFavorites(ctx context.Context, userID int64) ([]models.Ad, error) {
	const op = "usecase.ads.GetUserFavorites"
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)

	log.Info("getting favorites ads list")

	favorites, err := a.adsStorage.GetUserFavorites(ctx, userID)
	if err != nil {
		log.Error("failed to get favorites", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully retrieved favorites", slog.Int("count", len(favorites)))
	return favorites, nil
}
