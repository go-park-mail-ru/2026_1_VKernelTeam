// Package handlers — HTTP-обработчики catalog-сервиса.
package handlers

//go:generate mockgen -source=handlers.go -destination=mocks/mock_handlers.go -package=mocks

import (
	"context"
	"log/slog"
	"mime/multipart"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
)

// Общие константы ошибок.
const (
	// MaxUploadSize — лимит multipart-загрузок (фото объявления).
	MaxUploadSize = 50 << 20

	ErrInvalidRequestBody   = "invalid request body"
	ErrInternalError        = "internal error"
	ErrMethodNotAllowed     = "Method not allowed"
	ErrAdNotFound           = "ad not found"
	ErrAdForbidden          = "ad forbidden"
	ErrInvalidAdID          = "invalid ad id"
	ErrForbidden            = "forbidden"
	ErrInvalidUserID        = "invalid user id"
	ErrFailedToGetUserAds   = "failed to get user ads"
	ErrUnauthorized         = "unauthorized"
	ErrFileTooBig           = "file too big"
	ErrFailedToUploadPhotos = "failed to upload photos"

	// adsKey — ключ верхнего уровня в JSON-ответах со списком объявлений.
	adsKey = "ads"
)

// Ads — методы usecase объявлений, нужные хендлерам.
type Ads interface {
	GetAllAds(ctx context.Context, limit, offset int32) ([]models.Ad, error)
	SearchAds(ctx context.Context, query string, categoryID int64) ([]models.Ad, error)
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
	UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error
	DeleteAd(ctx context.Context, id int64, userID int64) error
	CloseAd(ctx context.Context, id int64, userID int64) error
	GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error)
	AddFavorite(ctx context.Context, userID int64, adID int64) error
	RemoveFavorite(ctx context.Context, userID int64, adID int64) error
	GetUserFavorites(ctx context.Context, userID int64) ([]models.Ad, error)
	UploadAdPhotos(ctx context.Context, files []multipart.File, filenames []string) ([]string, error)
	GetCategoryCharacteristics(ctx context.Context, categoryID int64) ([]models.CategoryCharacteristic, error)
	GetPriceHistory(ctx context.Context, adID int64) ([]models.PricePoint, error)

	AdminDeleteAd(ctx context.Context, adID, adminID int64) error
	ApproveAd(ctx context.Context, adID, adminID int64) error
	RejectAd(ctx context.Context, adID, adminID int64, reason string) error
	GetModerationQueue(ctx context.Context) ([]models.Ad, error)
	GetUserAdsByStatus(ctx context.Context, userID int64, status string) ([]models.Ad, error)
	IsModerationEnabled(ctx context.Context) bool
	SetModerationEnabled(ctx context.Context, enabled bool, adminID int64) error
}

// Views — методы usecase просмотров.
type Views interface {
	RecordView(ctx context.Context, productID int64, userID *int64, deviceID string) (int64, error)
}

// Services — агрегатор зависимостей хендлеров. Поле Ads оставлено для совместимости
// с портированным из монолита кодом ads.go (h.services.Ads.*).
type Services struct {
	Ads Ads
}

// AdsHandlers содержит обработчики объявлений.
type AdsHandlers struct {
	log      *slog.Logger
	services Services
}

// NewAdsHandlers создаёт AdsHandlers.
func NewAdsHandlers(log *slog.Logger, ads Ads) *AdsHandlers {
	return &AdsHandlers{
		log:      log,
		services: Services{Ads: ads},
	}
}

// ViewsHandlers содержит обработчики просмотров.
type ViewsHandlers struct {
	log      *slog.Logger
	viewsSvc Views
}

// NewViewsHandlers создаёт ViewsHandlers.
func NewViewsHandlers(log *slog.Logger, viewsSvc Views) *ViewsHandlers {
	return &ViewsHandlers{
		log:      log,
		viewsSvc: viewsSvc,
	}
}
