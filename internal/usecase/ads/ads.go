package ads

//go:generate mockgen -source=ads.go -destination=mocks/mock_ads.go -package=mocks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

type AdsProvider interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
	AddProductImages(ctx context.Context, adID int64, photos []string) error
	UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error
	DeleteAd(ctx context.Context, id int64, userID int64) error
	CloseAd(ctx context.Context, id int64, userID int64) error
	GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error)
}

type Ads struct {
	log        *slog.Logger
	adsStorage AdsProvider
}

// New создаёт новый экземпляр Auth с переданными зависимостями.
func New(
	log *slog.Logger,
	userStorage AdsProvider,
) *Ads {
	return &Ads{
		log:        log,
		adsStorage: userStorage,
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

// UpdateAd обновляет объявление.
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

	log.Info("ad updated successfully")
	return nil
}

// DeleteAd удаляет объявление (мягкое удаление).
func (a *Ads) DeleteAd(ctx context.Context, id int64, userID int64) error {
	const op = "ads.DeleteAd"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("ad_id", id),
		slog.Int64("user_id", userID),
	)
	log.Info("deleting ad")

	err := a.adsStorage.DeleteAd(ctx, id, userID)
	if err != nil {
		log.Error("failed to delete ad", "error", err)
		return err
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
