package handlers

// Package handlers содержит HTTP-обработчики для всех маршрутов приложения
import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

//go:generate mockgen -source=handlers.go -destination=mocks/mock_handlers.go -package=mocks

// ошибки HTTP-обработчиков
const (
	ErrInvalidRequestBody   = "invalid request body"
	ErrUserAlreadyExists    = "user already exists"
	ErrFailedToRegisterUser = "failed to register user"
	ErrAutoLoginFailed      = "registered, but failed to login"
	ErrFailedToLogin        = "failed to login"
	ErrInternalError        = "internal error"
	ErrFailedToLogout       = "failed to logout"
	ErrMethodNotAllowed     = "Method not allowed"
	ErrInvalidUserID        = "invalid user id"
	ErrFailedToGetUserAds   = "failed to get user ads"
	ErrUnauthorized         = "unauthorized"
)

// Services объединяет все бизнес-сервисы приложения, необходимые хендлерам
type Services struct {
	Ads  Ads
	Auth Auth
}

// Ads описывает методы сервиса объявлений
type Ads interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error)
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
func NewAdsHandlers(log *slog.Logger, services Services) *AdsHandlers {
	return &AdsHandlers{
		log:      log,
		services: services,
	}
}

// setAuthCookie устанавливает cookie с токеном
func (h *AuthHandlers) setAuthCookie(w http.ResponseWriter, token string, csrfToken string) {
	// JWT-токен
	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: token,
		// Domain: "clover-go.ru", // Убран хардкод домена для работы на localhost
		HttpOnly: true, // JS не увидит куку
		// Secure:   true,                      // передача только по HTTPS
		Path:     "/",                       // доступна везде
		SameSite: http.SameSiteLaxMode,      // защита от CSRF атак
		MaxAge:   int(h.tokenTTL.Seconds()), // время жизни
	})

	// CSRF-токен
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		HttpOnly: false, // JS должен иметь доступ
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.tokenTTL.Seconds()),
	})
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
