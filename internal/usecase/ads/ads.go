package ads

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

type AdsProvider interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
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
	const op = "ads.GetAll"

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
		log.Error("failed to create ad")
		return 0, err
	}
	
	log.Info("ad created successfully", "ad_id", adID)
	return adID, nil
}
