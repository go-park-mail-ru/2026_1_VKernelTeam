package handlers

// Package handlers содержит HTTP-обработчики для всех маршрутов приложения
import (
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

//go:generate mockgen -source=handlers.go -destination=mocks/mock_handlers.go -package=mocks

// ошибки HTTP-обработчиков
const (
	// Максимальный вес аватарки - 5 MB
	MaxUploadSize = 5 << 20

	ErrInvalidRequestBody   = "invalid request body"
	ErrUserAlreadyExists    = "user already exists"
	ErrFailedToRegisterUser = "failed to register user"
	ErrAutoLoginFailed      = "registered, but failed to login"
	ErrFailedToLogin        = "failed to login"
	ErrInternalError        = "internal error"
	ErrFailedToLogout       = "failed to logout"
	ErrMethodNotAllowed     = "Method not allowed"
	ErrAdNotFound           = "ad not found"
	ErrInvalidAdID          = "invalid ad id"
	ErrForbidden            = "forbidden"
	ErrInvalidUserID        = "invalid user id"
	ErrFailedToGetUserAds   = "failed to get user ads"
	ErrUnauthorized         = "unauthorized"
	ErrInvalidProductID     = "invalid product id"
	ErrFileTooBig           = "file too big"
	ErrFailedToGetFile      = "failed to get file"
)

// Services объединяет все бизнес-сервисы приложения, необходимые хендлерам
type Services struct {
	Ads  Ads
	Auth Auth
	Cart Cart
}

// Cart описывает методы сервиса корзины
type Cart interface {
	AddToCart(ctx context.Context, userID, productID int64) error
	RemoveFromCart(ctx context.Context, userID, productID int64) error
	GetCart(ctx context.Context, userID int64) (*dto.CartResponse, error)
	Checkout(ctx context.Context, userID int64) (*dto.CheckoutResponse, error)
}

// Ads описывает методы сервиса объявлений
type Ads interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
	UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error
	DeleteAd(ctx context.Context, id int64, userID int64) error
	CloseAd(ctx context.Context, id int64, userID int64) error
	GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error)
	UploadAdPhotos(ctx context.Context, files []multipart.File, filenames []string) ([]string, error)
}

// Auth описывает минимальный набор методов сервиса аутентификации
type Auth interface {
	Login(ctx context.Context, email string, password string) (string, string, models.User, error)
	ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error)
	RegisterNewUser(ctx context.Context, email string, password string, name string) (userID int64, err error)
	Logout(ctx context.Context, jti string, exp time.Time, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	GetProfile(ctx context.Context, userID int64) (models.User, error)
	UpdateProfile(ctx context.Context, userID int64, name string) (models.User, error)
	UpdateAvatar(ctx context.Context, userID int64, file multipart.File, filename string) (models.User, error)
}

// AuthHandlers содержит обработчики для аутентификации
type AuthHandlers struct {
	log        *slog.Logger
	services   Services
	tokenTTL   time.Duration
	refreshTTL time.Duration
	secret     string
}

// AdsHandlers содержит обработчики для объявлений
type AdsHandlers struct {
	log      *slog.Logger
	services Services
	tokenTTL time.Duration
}

// CartHandlers содержит обработчики для корзины
type CartHandlers struct {
	log      *slog.Logger
	services Services
	tokenTTL time.Duration
}

// NewAuthHandlers создает новый экземпляр AuthHandlers
func NewAuthHandlers(log *slog.Logger, services Services, tokenTTL time.Duration, refreshTTL time.Duration, secret string) *AuthHandlers {
	return &AuthHandlers{
		log:        log,
		services:   services,
		tokenTTL:   tokenTTL,
		refreshTTL: refreshTTL,
		secret:     secret,
	}
}

// NewAdsHandlers создает новый экземпляр AdsHandlers
func NewAdsHandlers(log *slog.Logger, services Services, tokenTTL time.Duration) *AdsHandlers {
	return &AdsHandlers{
		log:      log,
		services: services,
		tokenTTL: tokenTTL,
	}
}

// NewCartHandlers создает новый экземпляр CartHandlers
func NewCartHandlers(log *slog.Logger, services Services, tokenTTL time.Duration) *CartHandlers {
	return &CartHandlers{
		log:      log,
		services: services,
		tokenTTL: tokenTTL,
	}
}

// setTokenCookie устанавливает cookie с JWT-токеном
func setTokenCookie(w http.ResponseWriter, token string, tokenTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(tokenTTL.Seconds()),
	})
}

// setCsrfCookie устанавливает cookie с CSRF-токеном
func setCsrfCookie(w http.ResponseWriter, csrfToken string, tokenTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		HttpOnly: false,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(tokenTTL.Seconds()),
	})
}

// setAuthCookie устанавливает cookie с JWT-токеном и CSRF-токеном
func (h *AuthHandlers) setAuthCookie(w http.ResponseWriter, token string, csrfToken string) {
	setTokenCookie(w, token, h.tokenTTL)
	setCsrfCookie(w, csrfToken, h.tokenTTL)
}

// setRefreshCookie устанавливает cookie с refresh-токеном
func (h *AuthHandlers) setRefreshCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.refreshTTL.Seconds()),
	})
}

// respondWithUser отправляет успешный ответ с данными пользователя
func (h *AuthHandlers) respondWithUser(w http.ResponseWriter, user models.User) {
	responser.RespondWithJSON(w, http.StatusOK, dto.LoginResponse{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	})
}
