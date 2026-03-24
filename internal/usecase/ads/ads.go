package ads

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

type AdsProvider interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
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
